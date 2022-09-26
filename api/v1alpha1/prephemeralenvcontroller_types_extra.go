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

// The Github Token Secret
type SecretRef struct {
	// Name of the referent.
	// +required
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Key       string `json:"key"`
}

// The Github Repository, PRs against which will trigger the creation of the ephemeral environment
type GithubPRRepository struct {

	// User is the GitHub user name
	// +required
	User string `json:"user"`

	// Repo specifies the name of the githuh repository.
	// +required
	Repo string `json:"repo,omitempty"`

	// SecretRef specifies the Token Secret containing authentication credentials for
	// the Github Repository.
	// +required
	TokenSecretRef *SecretRef `json:"tokenSecretRef,omitempty"`
}

// EnvHelmRepo defines the Helm Repository for Infrastructure manifests
type EnvCreationHelmRepo struct {

	// Name of the Flux Source Repository containing the Helm Chart
	// +required
	FluxSourceRepoName string `json:"fluxSourceRepoName"`

	// The folder name in the Helm Repository containing the manifest templates
	// +required
	HelmChartPath string `json:"helmChartPath"`

	// The Chart version in semver format
	// +required
	// +kubebuilder:default="0.1.0"
	// +kubeubuilder:validation:Pattern="^v?([0-9]+)(\.[0-9]+)?(\.[0-9]+)?(-[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?(\+[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$"
	ChartVersion string `json:"chartVersion"`

	// The Kubernetes Namespace where the manifests will be deployed
	// +required
	// +kubebuilder:default="pr-helm-releases"
	DestinationNamespace string `json:"destinationNamespace"`
}
