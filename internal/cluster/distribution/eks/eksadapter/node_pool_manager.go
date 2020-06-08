// Copyright © 2020 Banzai Cloud
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package eksadapter

import (
	"context"
	"strconv"
	"strings"
	"time"

	"emperror.dev/errors"
	"go.uber.org/cadence/client"

	"github.com/aws/aws-sdk-go/service/cloudformation"

	"github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksprovider/workflow"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksworkflow"
	"github.com/banzaicloud/pipeline/internal/global"
	"github.com/banzaicloud/pipeline/internal/kubernetes"
	"github.com/banzaicloud/pipeline/pkg/kubernetes/custom/npls"
)

type nodePoolManager struct {
	workflowClient client.Client
	enterprise     bool
}

// NewNodePoolManager returns a new eks.NodePoolManager
// that manages node pools asynchronously via Cadence workflows.
func NewNodePoolManager(workflowClient client.Client, enterprise bool) eks.NodePoolManager {
	return nodePoolManager{
		workflowClient: workflowClient,
		enterprise:     enterprise,
	}
}

func (n nodePoolManager) UpdateNodePool(
	ctx context.Context,
	c cluster.Cluster,
	nodePoolName string,
	nodePoolUpdate eks.NodePoolUpdate,
) (string, error) {
	taskList := "pipeline"
	if n.enterprise {
		taskList = "pipeline-enterprise"
	}

	workflowOptions := client.StartWorkflowOptions{
		TaskList:                     taskList,
		ExecutionStartToCloseTimeout: 30 * 24 * 60 * time.Minute,
	}

	input := eksworkflow.UpdateNodePoolWorkflowInput{
		ProviderSecretID: c.SecretID.String(),
		Region:           c.Location,

		StackName: generateNodePoolStackName(c.Name, nodePoolName),

		ClusterID:       c.ID,
		ClusterSecretID: c.ConfigSecretID.String(),
		ClusterName:     c.Name,
		NodePoolName:    nodePoolName,
		OrganizationID:  c.OrganizationID,

		NodeImage: nodePoolUpdate.Image,

		Options: eks.NodePoolUpdateOptions{
			MaxSurge:       nodePoolUpdate.Options.MaxSurge,
			MaxBatchSize:   nodePoolUpdate.Options.MaxBatchSize,
			MaxUnavailable: nodePoolUpdate.Options.MaxUnavailable,
			Drain: eks.NodePoolUpdateDrainOptions{
				Timeout:     nodePoolUpdate.Options.Drain.Timeout,
				FailOnError: nodePoolUpdate.Options.Drain.FailOnError,
				PodSelector: nodePoolUpdate.Options.Drain.PodSelector,
			},
		},
	}

	e, err := n.workflowClient.StartWorkflow(ctx, workflowOptions, eksworkflow.UpdateNodePoolWorkflowName, input)
	if err != nil {
		return "", errors.WrapWithDetails(err, "failed to start workflow", "workflow", eksworkflow.UpdateNodePoolWorkflowName)
	}

	return e.ID, nil
}

// TODO: this is temporary
func generateNodePoolStackName(clusterName string, poolName string) string {
	return "pipeline-eks-nodepool-" + clusterName + "-" + poolName
}

// ListNodePools is for listing node pools from CloudsetFormation and NodePoolLabelSets
func (n nodePoolManager) ListNodePools(
	ctx context.Context,
	c cluster.Cluster,
	st eks.SecretStore,
	dcf kubernetes.DynamicClientFactory,
) ([]eks.NodePool, error) {
	// CloudsetFormation
	sessionFactory := workflow.NewAWSSessionFactory(st)
	client, err := sessionFactory.New(c.OrganizationID, c.SecretID.String(), c.Location)
	if err != nil {
		return nil, err
	}

	cfClient := cloudformation.New(client)
	describeStacksInput := cloudformation.DescribeStacksInput{}
	stacksOutput, err := cfClient.DescribeStacks(&describeStacksInput)
	if err != nil {
		return nil, err
	}

	var relevantStacks []*cloudformation.Stack
	for _, stack := range stacksOutput.Stacks {
		// TODO: Later a better filtering method will be needed
		if strings.HasPrefix(*stack.StackName, "pipeline-eks-nodepool-"+c.Name) {
			relevantStacks = append(relevantStacks, stack)
		}
	}

	stackMap := map[string]*cloudformation.Stack{}
	for _, stack := range relevantStacks {
		stackMap[*stack.StackName] = stack
	}

	// NodePoolLabelSets
	clusterClient, err := dcf.FromSecret(ctx, c.SecretID.String())
	if err != nil {
		return nil, err
	}

	manager := npls.NewManager(clusterClient, global.Config.Cluster.Namespace)
	labelSets, err := manager.GetAll()
	if err != nil {
		return nil, err
	}

	// Aggregate NodePoolLabelSets and CloudsetFormation Stacks
	var nodePools []eks.NodePool
	for labelSetName, labelSet := range labelSets {
		stack := stackMap[labelSetName]

		parameterMap := map[string]string{}
		for _, parameter := range stack.Parameters {
			parameterMap[*parameter.ParameterKey] = *parameter.ParameterValue
		}

		AutoscalingEnabled, err := strconv.ParseBool(parameterMap["ClusterAutoscalerEnabled"])
		if err != nil {
			return nil, err
		}

		AutoscalingMinSize, err := strconv.Atoi(parameterMap["NodeAutoScalingGroupMinSize"])
		if err != nil {
			return nil, err
		}

		AutoscalingMaxSize, err := strconv.Atoi(parameterMap["NodeAutoScalingGroupMaxSize"])
		if err != nil {
			return nil, err
		}

		NodeAutoScalingInitSize, err := strconv.Atoi(parameterMap["NodeAutoScalingInitSize"])
		if err != nil {
			return nil, err
		}

		nodePool := eks.NodePool{
			Name:   labelSetName,
			Labels: labelSet,
			Size:   NodeAutoScalingInitSize,
			Autoscaling: struct {
				Enabled bool
				MinSize int
				MaxSize int
			}{
				Enabled: AutoscalingEnabled,
				MinSize: AutoscalingMinSize,
				MaxSize: AutoscalingMaxSize,
			},
			InstanceType: parameterMap["NodeInstanceType"],
			Image:        parameterMap["NodeImageId"],
			SpotPrice:    parameterMap["NodeSpotPrice"],
		}

		nodePools = append(nodePools, nodePool)
	}

	return nodePools, nil
}
