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

package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	fluxhelmrelease "github.com/fluxcd/helm-controller/api/v2beta1"
	prcontrollerephemeralenviov1alpha1 "github.com/manisbindra/pr-ephemeral-env-controller/api/v1alpha1"
)

const (
	releasePrefix       = "relpr-"
	pollInterval        = 5 * time.Minute
	sourceKind          = "GitRepository"
	sourceRepoNameSpace = "flux-system"
)

// PREphemeralEnvControllerReconciler reconciles a PREphemeralEnvController object
type PREphemeralEnvControllerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Record record.EventRecorder
}

func (r *PREphemeralEnvControllerReconciler) getGHToken(ctx context.Context, prController prcontrollerephemeralenviov1alpha1.PREphemeralEnvController) (string, error) {
	logger := log.FromContext(ctx)
	secretName := types.NamespacedName{
		Namespace: prController.Spec.GithubPRRepository.TokenSecretRef.Namespace,
		Name:      prController.Spec.GithubPRRepository.TokenSecretRef.Name,
	}

	secret := &corev1.Secret{}

	if err := r.Client.Get(ctx, secretName, secret); err != nil {
		logger.Error(err, "unable to fetch Secret")
		return "", client.IgnoreNotFound(err)
	}

	return string(secret.Data[prController.Spec.GithubPRRepository.TokenSecretRef.Key]), nil
}

// find PRNumber and PR HEAD SHA associated with the flux helm release passed, and return a
// corresponding PRDetails struct
func getPRDetailsForHelmRelease(ctx context.Context, helmRelease fluxhelmrelease.HelmRelease) (PRDetails, error) {
	// logger := log.FromContext(ctx)
	prDetails := PRDetails{}
	values := helmRelease.Spec.Values
	type helmRelValues struct {
		PrNumber int    `json:"prNumber"`
		PrSHA    string `json:"prSHA"`
	}
	hlmRlVal := helmRelValues{}

	if values == nil {
		return prDetails, fmt.Errorf("values is nil")
	}
	raw := values.Raw
	if raw == nil {
		return prDetails, fmt.Errorf("raw is nil")
	}
	if err := json.Unmarshal(raw, &hlmRlVal); err != nil {
		return prDetails, err
	}
	prDetails.HeadSHA = hlmRlVal.PrSHA
	prDetails.Number = hlmRlVal.PrNumber
	return prDetails, nil
}

//+kubebuilder:rbac:groups=prcontroller.ephemeralenv.io.prcontroller.ephemeralenv.io,resources=prephemeralenvcontrollers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=prcontroller.ephemeralenv.io.prcontroller.ephemeralenv.io,resources=prephemeralenvcontrollers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=prcontroller.ephemeralenv.io.prcontroller.ephemeralenv.io,resources=prephemeralenvcontrollers/finalizers,verbs=update
//+kubebuilder:rbac:groups=helm.toolkit.fluxcd.io,resources=helmreleases,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=helm.crossplane.io,resources=releases,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the PREphemeralEnvController object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
func (r *PREphemeralEnvControllerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// TODO(user): your logic here
	logger := log.FromContext(ctx)

	// TODO(user): your logic here

	var err error

	var prController prcontrollerephemeralenviov1alpha1.PREphemeralEnvController

	// List of PRDetails for all Open Github Pull Requests
	var prDetails []PRDetails

	// Map containing PR Number and PRDetails for all PRs which are currently open in Github
	PRNumPRDetailsMap := make(map[int]PRDetails)

	// List all Flux Helm Releases in the Cluster (Namespace specified in the CRD)
	var helmReleaseList fluxhelmrelease.HelmReleaseList

	// Map containing PR Number and associated Flux Helm Release
	// for all Flux Helm Releases in the Cluster (Namespace specified in the CRD)
	PRNumHelmReleaseMap := make(map[int]fluxhelmrelease.HelmRelease)

	// Map containing PR Number and associated PRDetails
	// for all Flux Helm Releases in the Cluster (Namespace specified in the CRD)
	PRNumPRDetailsMapForHelmReleases := make(map[int]PRDetails)

	if err := r.Get(ctx, req.NamespacedName, &prController); err != nil {
		logger.Error(err, "unable to fetch PRController")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Set initial status message for controller
	if prController.Status.Message == "" {
		prController.Status.Message = "Starting"
		_ = r.Status().Update(ctx, &prController)
	}

	// Get the github tokentoken from the secretref specified in the CRD
	token, err := r.getGHToken(ctx, prController)
	if err != nil || len(token) == 0 {
		logger.Error(err, "unable to fetch Token")
		prController.Status.Message = "TokenLoadFailed"
		_ = r.Status().Update(ctx, &prController)
		mesg := "Could not fetch Github Token"
		r.Record.Event(&prController, "Warning", "TokenLoadFailed", mesg)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Get Active Pull Requests from Github
	prDetails, err = GetActivePullRequests(prController.Spec.GithubPRRepository.User, prController.Spec.GithubPRRepository.Repo, token)

	// Fetch all Helm Releases in the Cluster, and in the namespace specified in the CRD
	if err := r.List(ctx, &helmReleaseList, client.InNamespace(prController.Spec.EnvCreationHelmRepo.DestinationNamespace)); err != nil {
		logger.Info("currently no HelmRelease objects found")
	}

	// Create Maps of PRNumber -> HelmRelease and PRNumber -> PRDetails
	// for all helm releases in the system
	for _, helmRelease := range helmReleaseList.Items {
		prDet, err := getPRDetailsForHelmRelease(ctx, helmRelease)
		if err != nil {
			logger.Error(err, "unable to get PR details for HelmRelease")
		}
		PRNumPRDetailsMapForHelmReleases[prDet.Number] = prDet
		PRNumHelmReleaseMap[prDet.Number] = helmRelease
	}

	if err != nil {
		mesg := "Unable to fetch active pull requests from Github"
		r.Record.Event(&prController, "Warning", "PRFetchFailed", mesg)
		logger.Error(err, "unable to get active pull requests")

		prController.Status.Message = "PRFetchFailed"
		_ = r.Status().Update(ctx, &prController)

		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	prController.Status.Message = "Ready"
	_ = r.Status().Update(context.Background(), &prController)

	// Create a map of PRNumber -> PRDetails for all PRs which are currently open in Github
	for _, pr := range prDetails {
		PRNumPRDetailsMap[pr.Number] = pr
	}

	logger.Info("Active pull requests count", "noOfOpenPRs", len(prDetails))

	if len(prDetails) == 0 {
		logger.Info("No active pull requests found")
		r.Record.Event(&prController, "Normal", "NoActivePRs", "No active PRs found")
	}
	// Create / Update Flux HelmRelease for each Active Github PR
	for _, pr := range prDetails {
		if prHelmRel, ok := PRNumPRDetailsMapForHelmReleases[pr.Number]; !ok {
			logger.Info("Creating Env Flux Helm Release for PR", "pr", pr)
			if err := r.CreateFluxHelmRelease(ctx, pr, prController.Spec.EnvCreationHelmRepo); err != nil {
				mesg := fmt.Sprintf("unable to create flux helm release for PR %d", pr.Number)
				r.Record.Event(&prController, "Warning", "UnableToCreateHelmRelease", mesg)
				logger.Error(err, "unable to create flux HelmRelease", "prDetails", prDetails)
			}
			mesg := fmt.Sprintf("New flux HelmRelease created for PR %d", pr.Number)
			r.Record.Event(&prController, "Normal", "FluxHelmReleaseCreated", mesg)
			err = UpdatePRStatus(ctx, prController.Spec.GithubPRRepository.User, prController.Spec.GithubPRRepository.Repo, token, pr.Number, pr.HeadSHA, "pending", "Creation of ephemeral environment for PR in progress")
			if err != nil {
				logger.Error(err, "unable to update PR status")
			}
		} else {
			if prHelmRel.HeadSHA != pr.HeadSHA {
				logger.Info("Updating Flux helm release for PR", "pr", pr)
				if err := r.UpdateFluxHelmRelease(ctx, PRNumHelmReleaseMap[pr.Number], pr); err != nil {
					mesg := fmt.Sprintf("unable to update flux helm release for PR %d", pr.Number)
					r.Record.Event(&prController, "Warning", "UnableToCreateHelmRelease", mesg)
					logger.Error(err, "unable to update flux HelmRelease", "prDetails", prDetails)
				}
				logger.Info("Updated Flux Helm Release for PR", "PR Number:", pr.Number, "PR SHA:", pr.HeadSHA)
				mesg := fmt.Sprintf("Flux HelmRelease updated for PR %d", pr.Number)
				r.Record.Event(&prController, "Normal", "FluxHelmReleaseCreated", mesg)
			} else {
				logger.Info("Flux HelmRelease already exists for PR and is up to date", "pr", pr)
				mesg := fmt.Sprintf("Flux HelmRelease already exists for PR and is up to date, PR %d", pr.Number)
				r.Record.Event(&prController, "Normal", "FluxHelmReleaseCreated", mesg)
				if IsEnvReady(pr.Number) {
					logger.Info("Environment is ready for PR", "pr", pr)
					mesg := fmt.Sprintf("Environment is ready for PR %d", pr.Number)
					r.Record.Event(&prController, "Normal", "EnvReady", mesg)
					err = UpdatePRStatus(ctx, prController.Spec.GithubPRRepository.User, prController.Spec.GithubPRRepository.Repo, token, pr.Number, pr.HeadSHA, "success", "Successully created ephemeral environment for PR")
					if err != nil {
						logger.Error(err, "unable to update PR status")
					}
				}
			}
		}

	}

	// Delete HelmRelease for closed PRs if any
	_ = r.DeleteFluxHelmRelease(ctx, PRNumHelmReleaseMap, PRNumPRDetailsMap, &prController)

	requeueAfter := prController.Spec.Interval.Duration
	if requeueAfter < 60*time.Second {
		requeueAfter = 60 * time.Second
	}

	return ctrl.Result{RequeueAfter: requeueAfter}, nil

	// ***************************

	// return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PREphemeralEnvControllerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&prcontrollerephemeralenviov1alpha1.PREphemeralEnvController{}).
		Complete(r)
}
