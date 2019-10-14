// Copyright © 2019 Banzai Cloud
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

package features

import (
	"context"

	pkgHelm "github.com/banzaicloud/pipeline/pkg/helm"
)

//go:generate sh -c "which mockery > /dev/null && mockery -name HelmService -inpkg || true"
// HelmService provides an interface for using Helm on a specific cluster.
type HelmService interface {
	ApplyDeployment(
		ctx context.Context,
		clusterID uint,
		namespace string,
		deploymentName string,
		releaseName string,
		values []byte,
		chartVersion string,
	) error

	// DeleteDeployment deletes a deployment from a specific cluster.
	DeleteDeployment(ctx context.Context, clusterID uint, releaseName string) error

	// GetDeployment gets a deployment by release name from a specific cluster.
	GetDeployment(ctx context.Context, clusterID uint, releaseName string) (*pkgHelm.GetDeploymentResponse, error)
}
