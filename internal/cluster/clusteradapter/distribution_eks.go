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

package clusteradapter

import (
	"context"

	"emperror.dev/errors"
	"github.com/mitchellh/mapstructure"

	"github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks"
)

// NewEKSService returns a new EKS distribution service.
func NewEKSService(service eks.Service) cluster.Service {
	return eksService{
		service: service,
	}
}

type eksService struct {
	service eks.Service
}

func (s eksService) DeleteCluster(ctx context.Context, clusterIdentifier cluster.Identifier, options cluster.DeleteClusterOptions) (deleted bool, err error) {
	panic("implement me")
}

func (s eksService) CreateNodePool(ctx context.Context, clusterID uint, rawNodePool cluster.NewRawNodePool) error {
	panic("implement me")
}

func (s eksService) UpdateNodePool(ctx context.Context, clusterID uint, nodePoolName string, rawNodePoolUpdate cluster.RawNodePoolUpdate) (string, error) {
	var nodePoolUpdate eks.NodePoolUpdate

	err := mapstructure.Decode(rawNodePoolUpdate, &nodePoolUpdate)
	if err != nil {
		// TODO: return a service error
		return "", errors.Wrap(err, "failed to decode node pool update")
	}

	return s.service.UpdateNodePool(ctx, clusterID, nodePoolName, nodePoolUpdate)
}

func (s eksService) DeleteNodePool(ctx context.Context, clusterID uint, name string) (deleted bool, err error) {
	panic("implement me")
}

func (s eksService) ListNodePools(ctx context.Context, clusterID uint) ([]cluster.NodePool, error) {
	panic("implement me")
}
