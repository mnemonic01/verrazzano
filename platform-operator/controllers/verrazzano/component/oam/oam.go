// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package oam

import (
	"context"
	"fmt"

	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

const (
	pvcClusterRoleName         = "oam-kubernetes-runtime-pvc"
	aggregateToControllerLabel = "rbac.oam.dev/aggregate-to-controller"
)

// isOAMReady checks if the OAM operator deployment is ready
func isOAMReady(context spi.ComponentContext) bool {
	deployments := []types.NamespacedName{
		{
			Name:      ComponentName,
			Namespace: ComponentNamespace,
		},
	}
	prefix := fmt.Sprintf("Component %s", context.GetComponent())
	return status.DeploymentsAreReady(context.Log(), context.Client(), deployments, 1, prefix)
}

// ensureClusterRoles creates or updates additional OAM cluster roles during install and upgrade
func ensureClusterRoles(ctx spi.ComponentContext) error {
	// add a cluster role that allows the OAM operator to manage persistent volume claim workloads
	pvcClusterRole := rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{Name: pvcClusterRoleName}}

	_, err := controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client(), &pvcClusterRole, func() error {
		if pvcClusterRole.Labels == nil {
			pvcClusterRole.Labels = make(map[string]string)
		}
		// this label triggers cluster role aggregation into the oam-kubernetes-runtime cluster role
		pvcClusterRole.Labels[aggregateToControllerLabel] = "true"
		pvcClusterRole.Rules = []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{corev1.ResourcePersistentVolumeClaims.String()},
				Verbs: []string{
					"create",
					"delete",
					"get",
					"list",
					"patch",
					"update",
					"deletecollection",
				},
			},
		}
		return nil
	})
	return err
}
