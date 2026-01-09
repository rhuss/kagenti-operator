/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package tekton provides detection of Tekton Pipelines availability
// to enable conditional registration of the AgentBuild controller.
package tekton

import (
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	logger = ctrl.Log.WithName("tekton")
)

// TektonAPIGroupVersion is the API group version for Tekton Pipelines
const TektonAPIGroupVersion = "tekton.dev/v1"

// IsAvailable checks if Tekton Pipelines CRDs are installed in the cluster.
// It uses the Kubernetes discovery client to check for the tekton.dev/v1 API group.
func IsAvailable(config *rest.Config) bool {
	dc, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		logger.Error(err, "Failed to create discovery client, assuming Tekton not available")
		return false
	}

	return IsAvailableWithDiscovery(dc)
}

// IsAvailableWithDiscovery checks Tekton availability using a provided discovery client.
// This function is exported to allow testing with mock discovery clients.
func IsAvailableWithDiscovery(dc discovery.DiscoveryInterface) bool {
	_, err := dc.ServerResourcesForGroupVersion(TektonAPIGroupVersion)
	if err != nil {
		logger.Info("Tekton Pipelines not detected - AgentBuild functionality will be disabled")
		return false
	}

	logger.Info("Tekton Pipelines detected - AgentBuild functionality enabled")
	return true
}
