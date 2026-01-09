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

package tekton

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakediscovery "k8s.io/client-go/discovery/fake"
	fakeclientset "k8s.io/client-go/kubernetes/fake"
)

func TestIsAvailableWithDiscovery_TektonInstalled(t *testing.T) {
	// Create fake discovery client that returns tekton.dev/v1 resources
	fakeClient := fakeclientset.NewSimpleClientset()
	fakeDiscovery, ok := fakeClient.Discovery().(*fakediscovery.FakeDiscovery)
	if !ok {
		t.Fatal("Failed to get fake discovery client")
	}

	fakeDiscovery.Resources = []*metav1.APIResourceList{
		{
			GroupVersion: "tekton.dev/v1",
			APIResources: []metav1.APIResource{
				{Name: "pipelines", Kind: "Pipeline"},
				{Name: "pipelineruns", Kind: "PipelineRun"},
			},
		},
	}

	available := IsAvailableWithDiscovery(fakeDiscovery)
	if !available {
		t.Error("Expected Tekton to be available when CRDs exist")
	}
}

func TestIsAvailableWithDiscovery_TektonNotInstalled(t *testing.T) {
	// Create fake discovery client without tekton resources
	fakeClient := fakeclientset.NewSimpleClientset()
	fakeDiscovery, ok := fakeClient.Discovery().(*fakediscovery.FakeDiscovery)
	if !ok {
		t.Fatal("Failed to get fake discovery client")
	}

	// No Tekton resources
	fakeDiscovery.Resources = []*metav1.APIResourceList{}

	available := IsAvailableWithDiscovery(fakeDiscovery)
	if available {
		t.Error("Expected Tekton to be unavailable when CRDs don't exist")
	}
}

func TestIsAvailableWithDiscovery_OtherAPIsPresent(t *testing.T) {
	// Create fake discovery client with other APIs but not Tekton
	fakeClient := fakeclientset.NewSimpleClientset()
	fakeDiscovery, ok := fakeClient.Discovery().(*fakediscovery.FakeDiscovery)
	if !ok {
		t.Fatal("Failed to get fake discovery client")
	}

	fakeDiscovery.Resources = []*metav1.APIResourceList{
		{
			GroupVersion: "apps/v1",
			APIResources: []metav1.APIResource{
				{Name: "deployments", Kind: "Deployment"},
			},
		},
		{
			GroupVersion: "config.openshift.io/v1",
			APIResources: []metav1.APIResource{
				{Name: "clusteroperators", Kind: "ClusterOperator"},
			},
		},
	}

	available := IsAvailableWithDiscovery(fakeDiscovery)
	if available {
		t.Error("Expected Tekton to be unavailable when only other APIs are present")
	}
}
