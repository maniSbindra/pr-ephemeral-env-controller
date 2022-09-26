/*
Copyright 2022.

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PREphemeralEnvControllerSpec defines the desired state of PREphemeralEnvController
type PREphemeralEnvControllerSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// The Github Repository, PRs against which will trigger the creation of the ephemeral environment
	// +required
	GithubPRRepository *GithubPRRepository `json:"githubPRRepository,omitempty"`

	// Helm Repository for Infrastructure manifests
	// +required
	EnvCreationHelmRepo *EnvCreationHelmRepo `json:"envCreationHelmRepo,omitempty"`

	// Interval at which to check the GitRepository for PR updates.
	// +kubebuilder:default="60s"
	Interval metav1.Duration `json:"interval"`

	// Ephemeral Environment Health Check URL Template to be used to check the health of the ephemeral environment. If specified, the controller will check the health of the ephemeral environment and Update the Github PR status when environment is ready.
	// <<PR_NUMBER>> will be replaced with the PR number
	// <<PR_HEAD_SHA>> will be replaced with the PR head SHA
	EnvHealthCheckURLTemplate string `json:"envHealthCheckURLTemplate,omitempty"`
}

// PREphemeralEnvControllerStatus defines the observed state of PREphemeralEnvController
type PREphemeralEnvControllerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Conditions holds the conditions for the GitRepository.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	Message    string             `json:"message,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// PREphemeralEnvController is the Schema for the prephemeralenvcontrollers API
type PREphemeralEnvController struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PREphemeralEnvControllerSpec   `json:"spec,omitempty"`
	Status PREphemeralEnvControllerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PREphemeralEnvControllerList contains a list of PREphemeralEnvController
type PREphemeralEnvControllerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PREphemeralEnvController `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PREphemeralEnvController{}, &PREphemeralEnvControllerList{})
}
