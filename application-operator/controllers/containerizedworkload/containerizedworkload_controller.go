// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package containerizedworkload

import (
	"context"
	"fmt"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"

	"github.com/crossplane/oam-kubernetes-runtime/pkg/oam"
	"github.com/verrazzano/verrazzano/application-operator/controllers/appconfig"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"

	oamv1 "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/go-logr/logr"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type Reconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// SetupWithManager registers our controller with the manager
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&oamv1.ContainerizedWorkload{}).
		Complete(r)
}

// Reconcile checks restart version annotations on an ContainerizedWorkload and
// restarts as needed.
func (r *Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("containerizedworkload", req.NamespacedName)

	// We do not want any resource to get reconciled if it is in namespace kube-system
	// This is due to a bug found in OKE, it should not affect functionality of any vz operators
	// If this is the case then return success
	if req.Namespace == vzconst.KubeSystem {
		log.Info("Containerized workload resource should not be reconciled in kube-system namespace, ignoring")
		return reconcile.Result{}, nil
	}

	ctx := context.Background()
	log.Info("Reconciling ContainerizedWorkload")

	// fetch the ContainerizedWorkload
	var workload oamv1.ContainerizedWorkload
	if err := r.Client.Get(ctx, req.NamespacedName, &workload); err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info("ContainerizedWorkload has been deleted")
		} else {
			log.Error(err, "Failed to fetch ContainerizedWorkload")
		}
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}

	// get the user-specified restart version - if it's missing then there's nothing to do here
	restartVersion, ok := workload.Annotations[vzconst.RestartVersionAnnotation]
	if !ok || len(restartVersion) == 0 {
		log.Info("No restart version annotation found, nothing to do")
		return reconcile.Result{}, nil
	}

	if err := r.restartWorkload(ctx, restartVersion, &workload, log); err != nil {
		return reconcile.Result{}, err
	}

	log.Info("Successfully reconciled ContainerizedWorkload")
	return reconcile.Result{}, nil
}

func (r *Reconciler) restartWorkload(ctx context.Context, restartVersion string, workload *oamv1.ContainerizedWorkload, log logr.Logger) error {
	log.Info(fmt.Sprintf("Marking container %s with restart-version %s", workload.Name, restartVersion))
	var deploymentList appsv1.DeploymentList
	componentNameReq, _ := labels.NewRequirement(oam.LabelAppComponent, selection.Equals, []string{workload.ObjectMeta.Labels[oam.LabelAppComponent]})
	appNameReq, _ := labels.NewRequirement(oam.LabelAppName, selection.Equals, []string{workload.ObjectMeta.Labels[oam.LabelAppName]})
	selector := labels.NewSelector()
	selector = selector.Add(*componentNameReq, *appNameReq)
	err := r.Client.List(ctx, &deploymentList, &client.ListOptions{Namespace: workload.Namespace, LabelSelector: selector})
	if err != nil {
		return err
	}
	for index := range deploymentList.Items {
		deployment := &deploymentList.Items[index]
		if err := appconfig.DoRestartDeployment(ctx, r.Client, restartVersion, deployment, log); err != nil {
			return err
		}
	}
	return nil
}
