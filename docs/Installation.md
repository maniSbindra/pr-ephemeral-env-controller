# Installation

## Pre-rerequisites
* A Kubernetes Cluster is needed where the controller will run
* Flux needs to be configured on the Cluster
* For each configured "fluxSourceRepoName: infra-repo-public" in PREphemeralEnvController's applied to the Kubernetes cluster, a flux source needs to be added, like
    ```
    export GITHUB_INFRA_REPOSITORY="https://github.com/maniSbindra/ephemeral-env-infra"
    flux create source git infra-repo-public \
    --url ${GITHUB_INFRA_REPOSITORY} \
    --branch "main" \
    --username=${GITHUB_USER} --password=${GITHUB_TOKEN}
    ```

## Installing the controller on an existing cluster
Once the pre-requisites are setup on the kubernetes cluster, we first install the controller, with all its dependencies like, CRD Spec, deployment, roles, role bindings etc using the script below

    
  ```
  git clone https://github.com/maniSbindra/pr-ephemeral-env-controller.git
  cd pr-ephemeral-env-controller
  kubectl apply -f docs/install-controller.yaml
  ```

  After this we create a PREphemeralEnvController resource using the manifest. Please note 
  * A secret containing the Github PAT token needs to be created on the cluster as specified in the spec.tokenSecretRef section of the PREphemeralEnvController manifest file. 
  * Make sure that flux source has been created for the value specified in spec.envCreationHelmRepo.fluxSourceRepoName
  Once these configurations are done we can apply the manifest file

    ```
    kubectl apply -f docs/sample-pr-eph-env-controller-with-healthcheck.yaml
    ```

## Complete setup on kind cluster, including creation of new kind cluster

The following setup can be used to create isolated ephemreal environmens with isolated Kubernetes (AKS) cluster, and isolated Postgres Database (Azure Postgres), and the Application with PR changes deployed to that cluster

The [sample PREphemeralEnvController configuration](https://github.com/maniSbindra/ephemeral-mgmt/blob/main/mgmt-server-install-with-flux/ephemeral-prcontroller-CR.yaml) shown creates a new Ephmeral environment for each PR to the [sample application repository](https://github.com/maniSbindra/ephemeral-app), which is a simple todo API (CRUD for todo items), the tech stack is Java / Springboot, and the application needs a backend postgres database. In this case for each PR several resources are created, including a new resource group, a new AKS cluster on which the application deployement and service (corresponding to the PR SHA commit of the application) are created, a new Azure Postgres backend database to which the application points to read and persist data. To try this out you can bootstrap your Kubernetes cluster using these [steps](https://github.com/maniSbindra/ephemeral-mgmt/tree/main/mgmt-server-install-with-flux)

## Running locally and debugging
This project is a kubebuilder project. to run in locally

* First we clone the repo
    ```
    git clone https://github.com/maniSbindra/pr-ephemeral-env-controller.git
    cd pr-ephemeral-env-controller
    ```
* Next we install the crds
    ```
    make manifests
    ```
* After this we create a PREphemeralEnvController resource using the manifest. Please note 
  * A secret containing the Github PAT token needs to be created on the cluster as specified in the spec.tokenSecretRef section of the PREphemeralEnvController manifest file. 
  * Make sure that flux source has been created for the value specified in spec.envCreationHelmRepo.fluxSourceRepoName
  Once these configurations are done we can apply the manifest file

    ```
    kubectl apply -f docs/sample-pr-eph-env-controller-with-healthcheck.yaml
    ```
* And then we run the controller
    ```
    make run
    ```
    If we want to debug in vscode instead of running, we can go the debug menu and click on 'Launch File' configuration.