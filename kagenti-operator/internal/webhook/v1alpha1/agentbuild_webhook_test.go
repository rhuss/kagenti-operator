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

package v1alpha1

import (
	"context"
	"strings"
	"testing"

	agentv1alpha1 "github.com/kagenti/operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestValidateCreate_TektonUnavailable(t *testing.T) {
	validator := &AgentBuildValidator{TektonAvailable: false}

	agentBuild := &agentv1alpha1.AgentBuild{
		ObjectMeta: metav1.ObjectMeta{Name: "test-build", Namespace: "default"},
		Spec: agentv1alpha1.AgentBuildSpec{
			SourceSpec: agentv1alpha1.SourceSpec{
				SourceRepository: "github.com/example/repo",
			},
		},
	}

	_, err := validator.ValidateCreate(context.Background(), agentBuild)
	if err == nil {
		t.Error("Expected error when Tekton is unavailable")
	}
	if !strings.Contains(err.Error(), "Tekton Pipelines") {
		t.Errorf("Expected Tekton-related error, got: %v", err)
	}
}

func TestValidateCreate_TektonAvailable(t *testing.T) {
	validator := &AgentBuildValidator{TektonAvailable: true}

	// Create a valid AgentBuild that passes all validation
	enabled := true
	agentBuild := &agentv1alpha1.AgentBuild{
		ObjectMeta: metav1.ObjectMeta{Name: "test-build", Namespace: "default"},
		Spec: agentv1alpha1.AgentBuildSpec{
			SourceSpec: agentv1alpha1.SourceSpec{
				SourceRepository: "github.com/example/repo",
			},
			BuildOutput: &agentv1alpha1.BuildOutput{
				Image:         "test-image",
				ImageTag:      "v1.0.0",
				ImageRegistry: "ghcr.io/example",
			},
			Pipeline: &agentv1alpha1.PipelineSpec{
				Namespace: "default",
				Steps: []agentv1alpha1.PipelineStepSpec{
					{
						Name:      "build",
						ConfigMap: "build-task",
						Enabled:   &enabled,
					},
				},
			},
		},
	}

	_, err := validator.ValidateCreate(context.Background(), agentBuild)
	// Should pass all validation when Tekton is available
	if err != nil {
		t.Errorf("Expected no error for valid AgentBuild when Tekton is available, got: %v", err)
	}
}

func TestValidateUpdate_TektonUnavailable(t *testing.T) {
	validator := &AgentBuildValidator{TektonAvailable: false}

	agentBuild := &agentv1alpha1.AgentBuild{
		ObjectMeta: metav1.ObjectMeta{Name: "test-build", Namespace: "default"},
		Spec: agentv1alpha1.AgentBuildSpec{
			SourceSpec: agentv1alpha1.SourceSpec{
				SourceRepository: "github.com/example/repo",
			},
		},
	}

	_, err := validator.ValidateUpdate(context.Background(), agentBuild, agentBuild)
	if err == nil {
		t.Error("Expected error when Tekton is unavailable on update")
	}
	if !strings.Contains(err.Error(), "Tekton Pipelines") {
		t.Errorf("Expected Tekton-related error, got: %v", err)
	}
}

func TestValidateDelete_TektonUnavailable(t *testing.T) {
	validator := &AgentBuildValidator{TektonAvailable: false}

	agentBuild := &agentv1alpha1.AgentBuild{
		ObjectMeta: metav1.ObjectMeta{Name: "test-build", Namespace: "default"},
	}

	// Delete should always be allowed, even if Tekton is unavailable
	_, err := validator.ValidateDelete(context.Background(), agentBuild)
	if err != nil {
		t.Errorf("Delete should be allowed even when Tekton is unavailable: %v", err)
	}
}

func TestValidateAgentBuild_TektonAvailable_ValidSpec(t *testing.T) {
	validator := &AgentBuildValidator{TektonAvailable: true}

	// Create a valid AgentBuild with all required fields
	enabled := true
	agentBuild := &agentv1alpha1.AgentBuild{
		ObjectMeta: metav1.ObjectMeta{Name: "test-build", Namespace: "default"},
		Spec: agentv1alpha1.AgentBuildSpec{
			SourceSpec: agentv1alpha1.SourceSpec{
				SourceRepository: "github.com/example/repo",
			},
			BuildOutput: &agentv1alpha1.BuildOutput{
				Image:         "test-image",
				ImageTag:      "v1.0.0",
				ImageRegistry: "ghcr.io/example",
			},
			Pipeline: &agentv1alpha1.PipelineSpec{
				Namespace: "default",
				Steps: []agentv1alpha1.PipelineStepSpec{
					{
						Name:      "build",
						ConfigMap: "build-task",
						Enabled:   &enabled,
					},
				},
			},
		},
	}

	err := validator.validateAgentBuild(agentBuild)
	if err != nil {
		t.Errorf("Expected no error for valid AgentBuild, got: %v", err)
	}
}
