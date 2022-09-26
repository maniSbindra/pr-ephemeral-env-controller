package controllers

import (
	"context"
	"fmt"

	fluxhelmrelease "github.com/fluxcd/helm-controller/api/v2beta1"
	prcontrollerephemeralenviov1alpha1 "github.com/manisbindra/pr-ephemeral-env-controller/api/v1alpha1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/log"

	cpv1beta1 "github.com/crossplane-contrib/provider-helm/apis/release/v1beta1"
)

// Creates a Flux HelmRelease for the PR, the resource is created in the namespace specified in the CRD
func (r *PREphemeralEnvControllerReconciler) CreateFluxHelmRelease(ctx context.Context, prDetails PRDetails, envCreationHelmRepo *prcontrollerephemeralenviov1alpha1.EnvCreationHelmRepo) error {

	prNumberStr := fmt.Sprintf("%d", prDetails.Number)
	releaseName := fmt.Sprintf("relpr-%d", prDetails.Number)
	helmRelease := &fluxhelmrelease.HelmRelease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      releaseName,
			Namespace: envCreationHelmRepo.DestinationNamespace,
		},
		Spec: fluxhelmrelease.HelmReleaseSpec{
			Chart: fluxhelmrelease.HelmChartTemplate{
				Spec: fluxhelmrelease.HelmChartTemplateSpec{
					Chart: envCreationHelmRepo.HelmChartPath,
					SourceRef: fluxhelmrelease.CrossNamespaceObjectReference{
						Kind:      sourceKind,
						Name:      envCreationHelmRepo.FluxSourceRepoName,
						Namespace: sourceRepoNameSpace,
					},
					Version: envCreationHelmRepo.ChartVersion,
				},
			},
			Values: &apiextensionsv1.JSON{
				Raw: []byte(`{"prNumber":` + prNumberStr + `, "prSHA":"` + prDetails.HeadSHA + `"}`),
			},
			Interval:    metav1.Duration{Duration: pollInterval},
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
		if _, ok := prDetails[prNumber]; !ok {
			// Update status of PR on Github
			mesg := fmt.Sprintf("submitting deletion request for flux helm release, PR %d ", prNumber)
			r.Record.Event(prController, "Normal", "DeletionRequestSubmitted", mesg)
			logger.Info("submitting deletion request for flux helm release...", "prNumber", prNumber)
			if err := r.Client.Delete(ctx, &helmRel); err != nil {
				mesg := fmt.Sprintf("unable to delete flux HelmRelease for prNumber: %d", prNumber)
				r.Record.Event(prController, "Warning", "DeleteFailed", mesg)
				logger.Error(err, "unable to delete flux HelmRelease")
				return err
			}

			// In some cases when using crossplane helm provider, the dependencies of crossplane helm release are deleted
			// before the crossplane helm release itself. This causes the crossplane helm release to be stuck in deleting state.
			// Adding workaround to remove the finalizer from the crossplane helm release, so that the deletion can proceed.
			// In case of Argo cd this was handled directly using Argo CD sync wave annotations in the crossplane Helm Release resource.
			rel := r.getCrossplaneHelmRelease(ctx, prNumber)
			rel.ObjectMeta.Finalizers = []string{}
			if err := r.Client.Update(ctx, rel); err != nil {
				logger.Error(err, "unable to update Crossplane HelmRelease and remove finalizer, to proceed with deletion")
				return err
			}
			logger.Info("deletion for helm release submitted", "prNumber", prNumber)
		}
	}

	return nil
}

func (r *PREphemeralEnvControllerReconciler) getCrossplaneHelmRelease(ctx context.Context, prNum int) *cpv1beta1.Release {
	logger := log.FromContext(ctx)
	appHelmRelName := fmt.Sprintf("apphelmpr%d", prNum)

	cpHlmRel := &cpv1beta1.Release{}

	err := r.Get(ctx, types.NamespacedName{Name: appHelmRelName, Namespace: ""}, cpHlmRel)
	if err != nil {
		logger.Error(err, "unable to get HelmRelease")
	}

	return cpHlmRel
}
