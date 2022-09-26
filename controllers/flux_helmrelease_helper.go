package controllers

import (
	"context"
	"fmt"

	"time"

	fluxhelmrelease "github.com/fluxcd/helm-controller/api/v2beta1"
	prcontrollerephemeralenviov1alpha1 "github.com/manisbindra/pr-ephemeral-env-controller/api/v1alpha1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	FLUX_HELM_RELEASE_PREFIX    = "relpr-"
	FLUX_POLL_INTERVAL          = 5 * time.Minute
	FLUX_SOURCE_KIND            = "GitRepository"
	FLUX_SOURCE_REPO_NAME_SPACE = "flux-system"
)

// Creates a Flux HelmRelease for the PR, the resource is created in the namespace specified in the CRD
func (r *PREphemeralEnvControllerReconciler) CreateFluxHelmRelease(ctx context.Context, prDetails PRDetails) error {

	prNumberStr := fmt.Sprintf("%d", prDetails.Number)
	releaseName := fmt.Sprintf("%s%d", FLUX_HELM_RELEASE_PREFIX, prDetails.Number)
	helmRelease := &fluxhelmrelease.HelmRelease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      releaseName,
			Namespace: r.EnvCreationHelmRepo.DestinationNamespace,
		},
		Spec: fluxhelmrelease.HelmReleaseSpec{
			Chart: fluxhelmrelease.HelmChartTemplate{
				Spec: fluxhelmrelease.HelmChartTemplateSpec{
					Chart: r.EnvCreationHelmRepo.HelmChartPath,
					SourceRef: fluxhelmrelease.CrossNamespaceObjectReference{
						Kind:      FLUX_SOURCE_KIND,
						Name:      r.EnvCreationHelmRepo.FluxSourceRepoName,
						Namespace: FLUX_SOURCE_REPO_NAME_SPACE,
					},
					Version: r.EnvCreationHelmRepo.ChartVersion,
				},
			},
			Values: &apiextensionsv1.JSON{
				Raw: []byte(`{"prNumber":` + prNumberStr + `, "prSHA":"` + prDetails.HeadSHA + `"}`),
			},
			Interval:    metav1.Duration{Duration: FLUX_POLL_INTERVAL},
			ReleaseName: releaseName,
		},
	}

	if err := r.Create(ctx, helmRelease); err != nil {
		return err
	}
	return nil
}

// Updates a Flux HelmRelease for the PR, this is called when new commit is pushed to the PR. The Flux Helm release is updated and results in
// the commit SHA being updated in the HelmRelease values.
func (r *PREphemeralEnvControllerReconciler) UpdateFluxHelmRelease(ctx context.Context, helmRel fluxhelmrelease.HelmRelease, prDetail PRDetails) error {
	logger := log.FromContext(ctx)
	logger.Info("updating helm release...")
	helmRel.Spec.Values.Raw = []byte(fmt.Sprintf(`{"prNumber": %d, "prSHA": "%s"}`, prDetail.Number, prDetail.HeadSHA))
	if err := r.Client.Update(ctx, &helmRel); err != nil {
		logger.Error(err, "unable to update HelmRelease")
		return err
	}

	return nil
}

// The function deletes FLUX HelmReleases for which PRs are no longer open. It is passed a list of Flux
func (r *PREphemeralEnvControllerReconciler) DeleteFluxHelmRelease(ctx context.Context, helmReleases map[int]fluxhelmrelease.HelmRelease, prDetails map[int]PRDetails, prController *prcontrollerephemeralenviov1alpha1.PREphemeralEnvController) error {
	logger := log.FromContext(ctx)
	logger.Info("Checking if any flux helm releases need to be deleted...")
	for prNumber, helmRel := range helmReleases {
		if prDet, ok := prDetails[prNumber]; !ok {
			// Update status of PR on Github
			r.UpdatePRStatus(ctx, prNumber, prDet.HeadSHA, "closed", "PR closed, deleting ephemeral environment")
			mesg := fmt.Sprintf("Deletion request submitted for flux HelmRelease of prNumber: %d", prNumber)
			r.Record.Event(prController, "Normal", "DelReqSubmitted", mesg)
			logger.Info(mesg, "prNumber", prNumber)
			if err := r.Client.Delete(ctx, &helmRel); err != nil {
				mesg := fmt.Sprintf("unable to delete flux HelmRelease for prNumber: %d", prNumber)
				r.Record.Event(prController, "Warning", "DeleteFailed", mesg)
				logger.Error(err, mesg)
				return err
			}

			mesg = fmt.Sprintf("Deletion request submitted for flux HelmRelease of prNumber: %d", prNumber)
			logger.Info(mesg, "prNumber", prNumber)
		}
	}

	return nil
}
