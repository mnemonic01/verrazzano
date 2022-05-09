// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package configmaps

import (
	"context"
	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"

	vzctrl "github.com/verrazzano/verrazzano/pkg/controller"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/controllers"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// VerrazzanoConfigMapsReconciler reconciles ConfigMaps.
// This controller manages Helm override sources from the Verrazzano CR
type VerrazzanoConfigMapsReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	log    vzlog.VerrazzanoLogger
}

// SetupWithManager creates a new controller and adds it to the manager
func (r *VerrazzanoConfigMapsReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.ConfigMap{}).
		Complete(r)
}

func (r *VerrazzanoConfigMapsReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	if ctx == nil {
		ctx = context.TODO()
	}

	vzList := &installv1alpha1.VerrazzanoList{}
	listOpts := []client.ListOption{
		client.InNamespace(""),
	}
	err := r.List(ctx, vzList, listOpts...)
	if err != nil {
		if errors.IsNotFound(err) {
			zap.S().Infof("VZ Not found ConfigMap")
			return reconcile.Result{}, nil
		}
		zap.S().Errorf("Failed to fetch Verrazzano resource: %v", err)
		return newRequeueWithDelay(), err
	}
	if vzList != nil && len(vzList.Items) > 0 {
		vz := &vzList.Items[0]
		_, err := r.reconcileHelmOverrideConfigMap(ctx, req, vz)
		if err != nil {
			zap.S().Errorf("Failed to reconcile ConfigMap: %v", err)
			return newRequeueWithDelay(), err
		}
	}
	return ctrl.Result{}, nil
}

func (r *VerrazzanoConfigMapsReconciler) reconcileHelmOverrideConfigMap(ctx context.Context, req ctrl.Request, vz *installv1alpha1.Verrazzano) (ctrl.Result, error) {

	// Get the ConfigMap present in the Verrazzano CR namespace
	configMap := &corev1.ConfigMap{}
	if vz.Namespace == req.Namespace {
		if err := r.Get(ctx, req.NamespacedName, configMap); err != nil {
			zap.S().Errorf("Failed to fetch ConfigMap in Verrazzano CR namespace: %v", err)
			return newRequeueWithDelay(), err
		}
		if result, err := r.initLogger(*configMap); err != nil {
			return result, err
		}

		componentCtx, err := spi.NewContext(r.log, r.Client, vz, false)
		if err != nil {
			r.log.Errorf("Failed to construct component context: %v", err)
			return newRequeueWithDelay(), err
		}

		if componentName, ok := controllers.VzContainsResource(componentCtx, configMap); ok {
			err := r.updateVerrazzanoForHelmOverrides(componentCtx, componentName)
			if err != nil {
				r.log.Errorf("Failed to reconcile ConfigMap: %v", err)
				return newRequeueWithDelay(), err
			}
			r.log.Infof("Updated VZ")
		}
	}
	return ctrl.Result{}, nil
}

func (r *VerrazzanoConfigMapsReconciler) updateVerrazzanoForHelmOverrides(componentCtx spi.ComponentContext, componentName string) error {
	cr := componentCtx.ActualCR()
	res, err := controllerutil.CreateOrUpdate(context.TODO(), r.Client, cr, func() error {
		componentStatus := cr.Status.Components[componentName]
		if componentStatus == nil {
			componentStatus = &installv1alpha1.ComponentStatusDetails{
				Name: componentName,
			}
		}
		componentStatus.ReconcilingGeneration = 0
		componentStatus.LastReconciledGeneration = 0
		componentStatus.State = installv1alpha1.CompStateReady
		cr.Status.Components[componentName] = componentStatus
		return nil
	})

	if err == nil {
		r.log.Infof("Verrazzano %s", res)
		return nil
	}
	return err
}

func (r *VerrazzanoConfigMapsReconciler) initLogger(cm corev1.ConfigMap) (ctrl.Result, error) {
	// Get the resource logger needed to log message using 'progress' and 'once' methods
	log, err := vzlog.EnsureResourceLogger(&vzlog.ResourceConfig{
		Name:           cm.Name,
		Namespace:      cm.Namespace,
		ID:             string(cm.UID),
		Generation:     cm.Generation,
		ControllerName: "ConfigMaps",
	})
	if err != nil {
		zap.S().Errorf("Failed to create resource logger for VerrazzanoConfigMap controller: %v", err)
		return newRequeueWithDelay(), err
	}
	r.log = log
	return ctrl.Result{}, nil
}

// Create a new Result that will cause a reconcile requeue after a short delay
func newRequeueWithDelay() ctrl.Result {
	return vzctrl.NewRequeueWithDelay(3, 5, time.Second)
}
