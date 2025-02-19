// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricstrait

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"
	"time"

	oamrt "github.com/crossplane/crossplane-runtime/apis/core/v1alpha1"
	oamcore "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/go-logr/logr"
	"github.com/golang/mock/gomock"
	asserts "github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"github.com/verrazzano/verrazzano/application-operator/mocks"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"go.uber.org/zap"
	k8sapps "k8s.io/api/apps/v1"
	k8score "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	k8smeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

// TestReconcilerSetupWithManager test the creation of the metrics trait reconciler.
// GIVEN a controller implementation
// WHEN the controller is created
// THEN verify no error is returned
func TestReconcilerSetupWithManager(t *testing.T) {
	assert := asserts.New(t)

	var mocker *gomock.Controller
	var mgr *mocks.MockManager
	var cli *mocks.MockClient
	var scheme *runtime.Scheme
	var reconciler Reconciler
	var err error

	mocker = gomock.NewController(t)
	mgr = mocks.NewMockManager(mocker)
	cli = mocks.NewMockClient(mocker)
	scheme = runtime.NewScheme()
	_ = vzapi.AddToScheme(scheme)
	reconciler = Reconciler{Client: cli, Scheme: scheme, Scraper: "istio-system/prometheus"}
	mgr.EXPECT().GetControllerOptions().AnyTimes()
	mgr.EXPECT().GetScheme().Return(scheme)
	mgr.EXPECT().GetLogger().Return(logr.Discard())
	mgr.EXPECT().SetFields(gomock.Any()).Return(nil).AnyTimes()
	mgr.EXPECT().Add(gomock.Any()).Return(nil).AnyTimes()
	err = reconciler.SetupWithManager(mgr)
	mocker.Finish()
	assert.NoError(err)
}

// TestMetricsTraitCreatedForContainerizedWorkload tests the creation of a metrics trait related to a containerized workload.
// GIVEN a metrics trait that has been created
// AND the metrics trait is related to a containerized workload
// WHEN the metrics trait Reconcile method is invoked
// THEN verify that metrics trait finalizer is added
// AND verify that pod annotations are updated
// AND verify that the scraper configmap is updated
// AND verify that the scraper pod is restarted
func TestMetricsTraitCreatedForContainerizedWorkload(t *testing.T) {
	assert := asserts.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	testDeployment := k8sapps.Deployment{
		TypeMeta: k8smeta.TypeMeta{
			APIVersion: k8sapps.SchemeGroupVersion.Identifier(),
			Kind:       "Deployment",
		},
		ObjectMeta: k8smeta.ObjectMeta{
			Name:              "test-deployment-name",
			Namespace:         "test-namespace",
			CreationTimestamp: k8smeta.Now(),
			OwnerReferences: []k8smeta.OwnerReference{{
				APIVersion: oamcore.SchemeGroupVersion.Identifier(),
				Kind:       oamcore.ContainerizedWorkloadKind,
				Name:       "test-workload-name",
				UID:        "test-workload-uid"}}}}
	// Expect a call to get the trait resource.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-namespace", Name: "test-trait-name"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, trait *vzapi.MetricsTrait) error {
			trait.TypeMeta = k8smeta.TypeMeta{
				APIVersion: vzapi.SchemeGroupVersion.Identifier(),
				Kind:       vzapi.MetricsTraitKind}
			trait.ObjectMeta = k8smeta.ObjectMeta{
				Namespace: name.Namespace,
				Name:      name.Name}
			trait.Spec.WorkloadReference = oamrt.TypedReference{
				APIVersion: oamcore.SchemeGroupVersion.Identifier(),
				Kind:       oamcore.ContainerizedWorkloadKind,
				Name:       "test-workload-name"}
			return nil
		})
	// Expect a call to update the trait resource with a finalizer.
	mock.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, trait *vzapi.MetricsTrait, opts ...client.UpdateOption) error {
			assert.Equal("test-namespace", trait.Namespace)
			assert.Equal("test-trait-name", trait.Name)
			assert.Len(trait.Finalizers, 1)
			assert.Equal("metricstrait.finalizers.verrazzano.io", trait.Finalizers[0])
			return nil
		})
	// Expect a call to get the containerized workload resource
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-namespace", Name: "test-workload-name"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *unstructured.Unstructured) error {
			workload.SetGroupVersionKind(oamcore.ContainerizedWorkloadGroupVersionKind)
			workload.SetNamespace(name.Namespace)
			workload.SetName(name.Name)
			workload.SetUID("test-workload-uid")
			return nil
		})
	// Expect a call to get the prometheus configuration.
	mock.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, deployment *k8sapps.Deployment) error {
			assert.Equal("istio-system", name.Namespace)
			assert.Equal("prometheus", name.Name)
			deployment.APIVersion = k8sapps.SchemeGroupVersion.Identifier()
			deployment.Kind = deploymentKind
			deployment.Namespace = name.Namespace
			deployment.Name = name.Name
			return nil
		})
	// Expect a call to get the containerized workload resource definition
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "", Name: "containerizedworkloads.core.oam.dev"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workloadDef *oamcore.WorkloadDefinition) error {
			workloadDef.Namespace = name.Namespace
			workloadDef.Name = name.Name
			workloadDef.Spec.ChildResourceKinds = []oamcore.ChildResourceKind{
				{APIVersion: "apps/v1", Kind: "Deployment", Selector: nil},
				{APIVersion: "v1", Kind: "Service", Selector: nil},
			}
			return nil
		})
	// Expect a call to list the child Deployment resources of the containerized workload definition
	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *unstructured.UnstructuredList, opts ...client.ListOption) error {
			assert.Equal("Deployment", list.GetKind())
			return appendAsUnstructured(list, testDeployment)
		})
	// Expect a call to list the child Service resources of the containerized workload definition
	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *unstructured.UnstructuredList, opts ...client.ListOption) error {
			assert.Equal("Service", list.GetKind())
			return nil
		})
	// Expect a call to get the deployment definition
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-namespace", Name: "test-deployment-name"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, deployment *k8sapps.Deployment) error {
			deployment.ObjectMeta = testDeployment.ObjectMeta
			deployment.Spec = testDeployment.Spec
			return nil
		})
	// Expect a call to update the prometheus config
	mock.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, deployment *k8sapps.Deployment, opts ...client.UpdateOption) error {
			scrape, ok := deployment.Spec.Template.Annotations["verrazzano.io/metricsEnabled"]
			assert.True(ok)
			assert.Equal("true", scrape)
			target, ok := deployment.Spec.Template.Annotations["verrazzano.io/metricsPath"]
			assert.True(ok)
			assert.Equal("/metrics", target)
			port, ok := deployment.Spec.Template.Annotations["verrazzano.io/metricsPort"]
			assert.True(ok)
			assert.Equal("8080", port)
			return nil
		})
	// Expect a call to get the status writer
	mock.EXPECT().Status().Return(mockStatus).AnyTimes()
	// Expect a call to update the status of the trait status
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, trait *vzapi.MetricsTrait, opts ...client.UpdateOption) error {
			assert.Len(trait.Status.Conditions, 1)
			return nil
		})

	// Create and make the request
	request := ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "test-namespace", Name: "test-trait-name"}}

	reconciler := newMetricsTraitReconciler(mock)
	result, err := reconciler.Reconcile(nil, request)

	// Validate the results
	mocker.Finish()
	assert.NoError(err)
	assert.Equal(true, result.Requeue)
	assert.Equal(time.Duration(0), result.RequeueAfter)
}

// TestMetricsTraitCreatedForVerrazzanoWorkload tests the creation of a metrics trait related to a Verrazzano workload.
// The Verrazzano workload contains the real workload so we need to unwrap it.
// GIVEN a metrics trait that has been created
// AND the metrics trait is related to a Verrazzano workload
// WHEN the metrics trait Reconcile method is invoked
// THEN the contained workload should be unwrapped
// AND verify that metrics trait finalizer is added
// AND verify that pod annotations are updated
// AND verify that the scraper configmap is updated
// AND verify that the scraper pod is restarted
func TestMetricsTraitCreatedForVerrazzanoWorkload(t *testing.T) {
	assert := asserts.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	testDeployment := k8sapps.Deployment{
		TypeMeta: k8smeta.TypeMeta{
			APIVersion: k8sapps.SchemeGroupVersion.Identifier(),
			Kind:       "Deployment",
		},
		ObjectMeta: k8smeta.ObjectMeta{
			Name:              "test-deployment-name",
			Namespace:         "test-namespace",
			CreationTimestamp: k8smeta.Now(),
			OwnerReferences: []k8smeta.OwnerReference{{
				APIVersion: oamcore.SchemeGroupVersion.Identifier(),
				Kind:       oamcore.ContainerizedWorkloadKind,
				Name:       "test-workload-name",
				UID:        "test-workload-uid"}}}}
	// Expect a call to get the trait resource.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-namespace", Name: "test-trait-name"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, trait *vzapi.MetricsTrait) error {
			trait.TypeMeta = k8smeta.TypeMeta{
				APIVersion: vzapi.SchemeGroupVersion.Identifier(),
				Kind:       vzapi.MetricsTraitKind}
			trait.ObjectMeta = k8smeta.ObjectMeta{
				Namespace: name.Namespace,
				Name:      name.Name}
			trait.Spec.WorkloadReference = oamrt.TypedReference{
				APIVersion: oamcore.SchemeGroupVersion.Identifier(),
				Kind:       oamcore.ContainerizedWorkloadKind,
				Name:       "test-workload-name"}
			return nil
		})
	// Expect a call to update the trait resource with a finalizer.
	mock.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, trait *vzapi.MetricsTrait, opts ...client.UpdateOption) error {
			assert.Equal("test-namespace", trait.Namespace)
			assert.Equal("test-trait-name", trait.Name)
			assert.Len(trait.Finalizers, 1)
			assert.Equal("metricstrait.finalizers.verrazzano.io", trait.Finalizers[0])
			return nil
		})

	containedName := "test-contained-workload-name"
	containedResource := map[string]interface{}{
		"apiVersion": "coherence.oracle.com/v1",
		"kind":       "Coherence",
		"metadata": map[string]interface{}{
			"name": containedName,
		},
	}

	// Expect a call to get the Verrazzano workload resource
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-namespace", Name: "test-workload-name"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *unstructured.Unstructured) error {
			workload.SetAPIVersion("oam.verrazzano.io/v1alpha1")
			workload.SetKind("VerrazzanoCoherenceWorkload")
			workload.SetNamespace(name.Namespace)
			workload.SetName(name.Name)
			unstructured.SetNestedMap(workload.Object, containedResource, "spec", "template")
			workload.SetUID("test-workload-uid")
			return nil
		})
	// Expect a call to get the contained Coherence resource
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-namespace", Name: containedName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *unstructured.Unstructured) error {
			workload.SetUnstructuredContent(containedResource)
			workload.SetNamespace(name.Namespace)
			workload.SetUID("test-workload-uid")
			return nil
		})
	// Expect a call to get the prometheus configuration.
	mock.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, deployment *k8sapps.Deployment) error {
			assert.Equal("istio-system", name.Namespace)
			assert.Equal("prometheus", name.Name)
			deployment.APIVersion = k8sapps.SchemeGroupVersion.Identifier()
			deployment.Kind = deploymentKind
			deployment.Namespace = name.Namespace
			deployment.Name = name.Name
			return nil
		})
	// Expect a call to get the Coherence workload resource definition
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "", Name: "coherences.coherence.oracle.com"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workloadDef *oamcore.WorkloadDefinition) error {
			workloadDef.Namespace = name.Namespace
			workloadDef.Name = name.Name
			workloadDef.Spec.ChildResourceKinds = []oamcore.ChildResourceKind{
				{APIVersion: "apps/v1", Kind: "Deployment", Selector: nil},
				{APIVersion: "v1", Kind: "Service", Selector: nil},
			}
			return nil
		})
	// Expect a call to list the child Deployment resources of the Coherence workload definition
	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *unstructured.UnstructuredList, opts ...client.ListOption) error {
			assert.Equal("Deployment", list.GetKind())
			return appendAsUnstructured(list, testDeployment)
		})
	// Expect a call to list the child Service resources of the Coherence workload definition
	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *unstructured.UnstructuredList, opts ...client.ListOption) error {
			assert.Equal("Service", list.GetKind())
			return nil
		})
	// Expect a call to get the deployment definition
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-namespace", Name: "test-deployment-name"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, deployment *k8sapps.Deployment) error {
			deployment.ObjectMeta = testDeployment.ObjectMeta
			deployment.Spec = testDeployment.Spec
			return nil
		})
	// Expect a call to update the prometheus config
	mock.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, deployment *k8sapps.Deployment, opts ...client.UpdateOption) error {
			scrape, ok := deployment.Spec.Template.Annotations["verrazzano.io/metricsEnabled"]
			assert.True(ok)
			assert.Equal("true", scrape)
			target, ok := deployment.Spec.Template.Annotations["verrazzano.io/metricsPath"]
			assert.True(ok)
			assert.Equal("/metrics", target)
			port, ok := deployment.Spec.Template.Annotations["verrazzano.io/metricsPort"]
			assert.True(ok)
			assert.Equal("9612", port)
			return nil
		})
	// Expect a call to get the status writer
	mock.EXPECT().Status().Return(mockStatus).AnyTimes()
	// Expect a call to update the status of the trait status
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, trait *vzapi.MetricsTrait, opts ...client.UpdateOption) error {
			assert.Len(trait.Status.Conditions, 1)
			return nil
		})

	// Create and make the request
	request := ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "test-namespace", Name: "test-trait-name"}}

	reconciler := newMetricsTraitReconciler(mock)
	result, err := reconciler.Reconcile(nil, request)

	// Validate the results
	mocker.Finish()
	assert.NoError(err)
	assert.Equal(true, result.Requeue)
	assert.Equal(time.Duration(0), result.RequeueAfter)
}

// TestMetricsTraitCreatedForDeploymentWorkload tests the creation of a metrics trait related to a native Kubernetes Deployment workload.
// GIVEN a metrics trait that has been created
// AND the metrics trait is related to a k8s deployment workload
// WHEN the metrics trait Reconcile method is invoked
// THEN verify that metrics trait finalizer is added
// AND verify that pod annotations are updated
// AND verify that the scraper configmap is updated
// AND verify that the scraper pod is restarted
func TestMetricsTraitCreatedForDeploymentWorkload(t *testing.T) {
	assert := asserts.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	testDeployment := k8sapps.Deployment{
		TypeMeta: k8smeta.TypeMeta{
			APIVersion: k8sapps.SchemeGroupVersion.Identifier(),
			Kind:       "Deployment",
		},
		ObjectMeta: k8smeta.ObjectMeta{
			Name:              "test-deployment-name",
			Namespace:         "test-namespace",
			UID:               "test-workload-uid",
			CreationTimestamp: k8smeta.Now(),
			OwnerReferences: []k8smeta.OwnerReference{{
				APIVersion: oamcore.SchemeGroupVersion.Identifier(),
				Kind:       oamcore.ApplicationConfigurationKind,
				Name:       "test-workload-name",
				UID:        "wrong-workload-uid"}}}}
	// Expect a call to get the trait resource.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-namespace", Name: "test-trait-name"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, trait *vzapi.MetricsTrait) error {
			trait.TypeMeta = k8smeta.TypeMeta{
				APIVersion: vzapi.SchemeGroupVersion.Identifier(),
				Kind:       vzapi.MetricsTraitKind}
			trait.ObjectMeta = k8smeta.ObjectMeta{
				Namespace: name.Namespace,
				Name:      name.Name}
			trait.Spec.WorkloadReference = oamrt.TypedReference{
				APIVersion: k8sapps.SchemeGroupVersion.Identifier(),
				Kind:       "Deployment",
				Name:       "test-workload-name"}
			return nil
		})
	// Expect a call to update the trait resource with a finalizer.
	mock.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, trait *vzapi.MetricsTrait, opts ...client.UpdateOption) error {
			assert.Equal("test-namespace", trait.Namespace)
			assert.Equal("test-trait-name", trait.Name)
			assert.Len(trait.Finalizers, 1)
			assert.Equal("metricstrait.finalizers.verrazzano.io", trait.Finalizers[0])
			return nil
		})
	// Expect a call to get the containerized workload resource
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-namespace", Name: "test-workload-name"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *unstructured.Unstructured) error {
			workload.SetAPIVersion(k8sapps.SchemeGroupVersion.Identifier())
			workload.SetKind("Deployment")
			workload.SetNamespace(name.Namespace)
			workload.SetName(name.Name)
			workload.SetUID("test-workload-uid")
			return nil
		})
	// Expect a call to get the prometheus configuration.
	mock.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, deployment *k8sapps.Deployment) error {
			assert.Equal("istio-system", name.Namespace)
			assert.Equal("prometheus", name.Name)
			deployment.APIVersion = k8sapps.SchemeGroupVersion.Identifier()
			deployment.Kind = deploymentKind
			deployment.Namespace = name.Namespace
			deployment.Name = name.Name
			return nil
		})
	// Expect a call to get the containerized workload resource definition
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "", Name: "deployments.apps"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workloadDef *oamcore.WorkloadDefinition) error {
			workloadDef.Namespace = name.Namespace
			workloadDef.Name = name.Name
			workloadDef.Spec.ChildResourceKinds = []oamcore.ChildResourceKind{
				{APIVersion: "apps/v1", Kind: "Deployment", Selector: nil},
			}
			return nil
		})
	// Expect a call to list the child Deployment resources of the containerized workload definition
	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *unstructured.UnstructuredList, opts ...client.ListOption) error {
			assert.Equal("Deployment", list.GetKind())
			return appendAsUnstructured(list, testDeployment)
		})
	// Expect a call to get the deployment definition
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-namespace", Name: "test-deployment-name"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, deployment *k8sapps.Deployment) error {
			deployment.ObjectMeta = testDeployment.ObjectMeta
			deployment.Spec = testDeployment.Spec
			return nil
		})
	// Expect a call to update the prometheus config
	mock.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, deployment *k8sapps.Deployment, opts ...client.UpdateOption) error {
			scrape, ok := deployment.Spec.Template.Annotations["verrazzano.io/metricsEnabled"]
			assert.True(ok)
			assert.Equal("true", scrape)
			target, ok := deployment.Spec.Template.Annotations["verrazzano.io/metricsPath"]
			assert.True(ok)
			assert.Equal("/metrics", target)
			port, ok := deployment.Spec.Template.Annotations["verrazzano.io/metricsPort"]
			assert.True(ok)
			assert.Equal("8080", port)
			return nil
		})
	// Expect a call to get the status writer
	mock.EXPECT().Status().Return(mockStatus).AnyTimes()
	// Expect a call to update the status of the trait status
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, trait *vzapi.MetricsTrait, opts ...client.UpdateOption) error {
			assert.Len(trait.Status.Conditions, 1)
			return nil
		})

	// Create and make the request
	request := ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "test-namespace", Name: "test-trait-name"}}

	reconciler := newMetricsTraitReconciler(mock)
	result, err := reconciler.Reconcile(nil, request)

	// Validate the results
	mocker.Finish()
	assert.NoError(err)
	assert.Equal(true, result.Requeue)
	assert.Equal(time.Duration(0), result.RequeueAfter)
}

// TestMetricsTraitDeletedForContainerizedWorkload tests deletion of a metrics trait related to a containerized workload.
// GIVEN a metrics trait with a non-zero deletion time
// WHEN the metrics trait Reconcile method is invoked
// THEN verify that metrics trait finalizer is removed
// AND verify that pod annotations are cleaned up
// AND verify that the scraper configmap is cleanup up
// AND verify that the scraper pod is restarted
func TestMetricsTraitDeletedForContainerizedWorkload(t *testing.T) {
	assert := asserts.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	// mockStatus := mocks.NewMockStatusWriter(mocker)
	var err error

	params := map[string]string{
		"##OAM_APP_NAME##":         "test-oam-app-name",
		"##OAM_COMP_NAME##":        "test-oam-comp-name",
		"##TRAIT_NAME##":           "test-trait-name",
		"##TRAIT_NAMESPACE##":      "test-namespace",
		"##WORKLOAD_APIVER##":      "core.oam.dev/v1alpha2",
		"##WORKLOAD_KIND##":        "ContainerizedWorkload",
		"##WORKLOAD_NAME##":        "test-workload-name",
		"##PROMETHEUS_NAME##":      "vmi-system-prometheus-0",
		"##PROMETHEUS_NAMESPACE##": "verrazzano-system",
		"##DEPLOYMENT_NAMESPACE##": "test-namespace",
		"##DEPLOYMENT_NAME##":      "test-workload-name",
	}

	// 1. Expect a call to get the deleted trait resource.
	mock.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).DoAndReturn(func(ctx context.Context, name types.NamespacedName, trait *vzapi.MetricsTrait) error {
		assert.Equal("test-namespace", name.Namespace)
		assert.Equal("test-trait-name", name.Name)
		assert.NoError(updateObjectFromYAMLTemplate(trait, "test/templates/containerized_workload_metrics_trait_deleted.yaml", params))
		return nil
	})
	// 2. Expect a call to get the child resource
	mock.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).DoAndReturn(func(ctx context.Context, name types.NamespacedName, obj *k8sapps.Deployment) error {
		assert.Equal("test-namespace", name.Namespace)
		assert.Equal("test-workload-name", name.Name)
		assert.NoError(updateObjectFromYAMLTemplate(obj, "test/templates/containerized_workload_deployment.yaml", params))
		assert.Contains(obj.Spec.Template.Annotations, "verrazzano.io/metricsEnabled")
		assert.Contains(obj.Spec.Template.Annotations, "verrazzano.io/metricsPath")
		assert.Contains(obj.Spec.Template.Annotations, "verrazzano.io/metricsPort")
		return nil
	})
	// 3. Expect a call to update the child resource to remove the annotations
	mock.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, obj *k8sapps.Deployment, opts ...client.UpdateOption) error {
		assert.Equal("test-namespace", obj.Namespace)
		assert.Equal("test-workload-name", obj.Name)
		assert.NotContains(obj.Spec.Template.Annotations, "verrazzano.io/metricsEnabled")
		assert.NotContains(obj.Spec.Template.Annotations, "verrazzano.io/metricsPath")
		assert.NotContains(obj.Spec.Template.Annotations, "verrazzano.io/metricsPort")
		return nil
	})
	// 6. Expect a call to get the prometheus deployment.
	mock.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, name types.NamespacedName, deployment *k8sapps.Deployment) error {
		assert.Equal("verrazzano-system", name.Namespace)
		assert.Equal("vmi-system-prometheus-0", name.Name)
		assert.NoError(updateObjectFromYAMLTemplate(deployment, "test/templates/prometheus_deployment.yaml", params))
		return nil
	})
	// 7. Expect a call to get the prometheus configmap.
	mock.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, name types.NamespacedName, configmap *k8score.ConfigMap) error {
		assert.Equal("verrazzano-system", name.Namespace)
		assert.Equal("vmi-system-prometheus-0", name.Name)
		assert.NoError(updateObjectFromYAMLTemplate(configmap, "test/templates/prometheus_configmap.yaml", params))
		return nil
	})
	// 8. Expect a call to update the prometheus configmap
	mock.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, obj *k8score.ConfigMap, opts ...client.UpdateOption) error {
		assert.Equal("verrazzano-system", obj.Namespace)
		assert.Equal("vmi-system-prometheus-0", obj.Name)
		return nil
	})
	// 9. Expect a call to get the owner application configuration
	mock.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, name types.NamespacedName, appConfig *oamcore.ApplicationConfiguration) error {
		assert.Equal("test-namespace", name.Namespace)
		assert.Equal("test-oam-app-name", name.Name)
		assert.NoError(updateObjectFromYAMLTemplate(appConfig, "test/templates/appconf_metrics_trait_deleted.yaml", params))
		return nil
	})
	// 10. Expect a call to update the owner application configuration
	mock.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, appConfig *oamcore.ApplicationConfiguration, opts ...client.UpdateOption) error {
		assert.Equal("test-namespace", appConfig.Namespace)
		assert.Equal("test-oam-app-name", appConfig.Name)
		return nil
	})
	// 11. Expect a call to update the metrics trait to remove the finalizer
	mock.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, obj *vzapi.MetricsTrait, opts ...client.UpdateOption) error {
		assert.Equal("test-namespace", obj.Namespace)
		assert.Equal("test-trait-name", obj.Name)
		assert.Len(obj.Finalizers, 0)
		return nil
	})

	// Create and make the request
	request := ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "test-namespace", Name: "test-trait-name"}}
	reconciler := newMetricsTraitReconciler(mock)
	result, err := reconciler.Reconcile(nil, request)

	// Validate the results
	mocker.Finish()
	assert.NoError(err)
	assert.Equal(true, result.Requeue)
	assert.GreaterOrEqual(result.RequeueAfter.Seconds(), 45.0)
}

// TestMetricsTraitDeletedForContainerizedWorkload tests deletion of a metrics trait related to a containerized workload.
// GIVEN a metrics trait with a non-zero deletion time
// GIVEN the related deployment resource no longer exists
// WHEN the metrics trait Reconcile method is invoked
// THEN verify that metrics trait finalizer is removed
// AND verify that the scraper configmap is cleanup up
// AND verify that the scraper pod is restarted
// AND verify that the finalizer is removed
func TestMetricsTraitDeletedForContainerizedWorkloadWhenDeploymentDeleted(t *testing.T) {
	assert := asserts.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	// mockStatus := mocks.NewMockStatusWriter(mocker)
	var err error

	params := map[string]string{
		"##OAM_APP_NAME##":         "test-oam-app-name",
		"##OAM_COMP_NAME##":        "test-oam-comp-name",
		"##TRAIT_NAME##":           "test-trait-name",
		"##TRAIT_NAMESPACE##":      "test-namespace",
		"##WORKLOAD_APIVER##":      "core.oam.dev/v1alpha2",
		"##WORKLOAD_KIND##":        "ContainerizedWorkload",
		"##WORKLOAD_NAME##":        "test-workload-name",
		"##PROMETHEUS_NAME##":      "vmi-system-prometheus-0",
		"##PROMETHEUS_NAMESPACE##": "verrazzano-system",
		"##DEPLOYMENT_NAMESPACE##": "test-namespace",
		"##DEPLOYMENT_NAME##":      "test-workload-name",
	}

	// Expect a call to get the deleted trait resource.
	mock.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).DoAndReturn(func(ctx context.Context, name types.NamespacedName, trait *vzapi.MetricsTrait) error {
		assert.Equal("test-namespace", name.Namespace)
		assert.Equal("test-trait-name", name.Name)
		assert.NoError(updateObjectFromYAMLTemplate(trait, "test/templates/containerized_workload_metrics_trait_deleted.yaml", params))
		return nil
	})
	// Expect a call to get the child deployment resource
	// Do not modify the provided Deployment object to simulate it no longer existing.
	mock.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).DoAndReturn(func(ctx context.Context, name types.NamespacedName, obj *k8sapps.Deployment) error {
		assert.Equal("test-namespace", name.Namespace)
		assert.Equal("test-workload-name", name.Name)
		assert.True(obj.CreationTimestamp.IsZero(), "Expected creationTimestamp to be zero.")
		return nil
	})
	// Expect a call to get the prometheus deployment.
	mock.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, name types.NamespacedName, deployment *k8sapps.Deployment) error {
		assert.Equal("verrazzano-system", name.Namespace)
		assert.Equal("vmi-system-prometheus-0", name.Name)
		assert.NoError(updateObjectFromYAMLTemplate(deployment, "test/templates/prometheus_deployment.yaml", params))
		return nil
	})
	// Expect a call to get the prometheus configmap.
	mock.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, name types.NamespacedName, configmap *k8score.ConfigMap) error {
		assert.Equal("verrazzano-system", name.Namespace)
		assert.Equal("vmi-system-prometheus-0", name.Name)
		assert.NoError(updateObjectFromYAMLTemplate(configmap, "test/templates/prometheus_configmap.yaml", params))
		return nil
	})
	// Expect a call to update the prometheus configmap
	mock.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, obj *k8score.ConfigMap, opts ...client.UpdateOption) error {
		assert.Equal("verrazzano-system", obj.Namespace)
		assert.Equal("vmi-system-prometheus-0", obj.Name)
		return nil
	})
	// Expect a call to get the owner application configuration
	mock.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, name types.NamespacedName, appConfig *oamcore.ApplicationConfiguration) error {
		assert.Equal("test-namespace", name.Namespace)
		assert.Equal("test-oam-app-name", name.Name)
		assert.NoError(updateObjectFromYAMLTemplate(appConfig, "test/templates/appconf_metrics_trait_deleted.yaml", params))
		return nil
	})
	// Expect a call to update the owner application configuration
	mock.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, appConfig *oamcore.ApplicationConfiguration, opts ...client.UpdateOption) error {
		assert.Equal("test-namespace", appConfig.Namespace)
		assert.Equal("test-oam-app-name", appConfig.Name)
		return nil
	})
	// Expect a call to update the metrics trait to remove the finalizer
	mock.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, obj *vzapi.MetricsTrait, opts ...client.UpdateOption) error {
		assert.Equal("test-namespace", obj.Namespace)
		assert.Equal("test-trait-name", obj.Name)
		assert.Len(obj.Finalizers, 0)
		return nil
	})

	// Create and make the request
	request := ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "test-namespace", Name: "test-trait-name"}}
	reconciler := newMetricsTraitReconciler(mock)
	result, err := reconciler.Reconcile(nil, request)

	// Validate the results
	mocker.Finish()
	assert.NoError(err)
	assert.Equal(true, result.Requeue)
	assert.GreaterOrEqual(result.RequeueAfter.Seconds(), 45.0)
}

// TestMetricsTraitDeletedForContainerizedWorkload tests deletion of a metrics trait related to a containerized workload.
// GIVEN a metrics trait with a non-zero deletion time
// WHEN the metrics trait Reconcile method is invoked
// THEN verify that metrics trait finalizer is removed
// AND verify that pod annotations are cleaned up
// AND verify that the scraper configmap is cleanup up
// AND verify that the scraper pod is restarted
func TestMetricsTraitDeletedForDeploymentWorkload(t *testing.T) {
	assert := asserts.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	// mockStatus := mocks.NewMockStatusWriter(mocker)
	var err error

	params := map[string]string{
		"##OAM_APP_NAME##":         "deploymetrics-appconf",
		"##OAM_COMP_NAME##":        "deploymetrics-deployment",
		"##TRAIT_NAME##":           "deploymetrics-deployment-trait",
		"##TRAIT_NAMESPACE##":      "deploymetrics",
		"##WORKLOAD_APIVER##":      "apps/v1",
		"##WORKLOAD_KIND##":        "Deployment",
		"##WORKLOAD_NAME##":        "deploymetrics-workload",
		"##PROMETHEUS_NAME##":      "vmi-system-prometheus-0",
		"##PROMETHEUS_NAMESPACE##": "verrazzano-system",
		"##DEPLOYMENT_NAMESPACE##": "deploymetrics",
		"##DEPLOYMENT_NAME##":      "deploymetrics-workload",
	}

	// 1. Expect a call to get the deleted trait resource.
	mock.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).DoAndReturn(func(ctx context.Context, name types.NamespacedName, trait *vzapi.MetricsTrait) error {
		assert.Equal("deploymetrics", name.Namespace)
		assert.Equal("deploymetrics-deployment-trait", name.Name)
		assert.NoError(updateObjectFromYAMLTemplate(trait, "test/templates/deployment_workload_metrics_trait_deleted.yaml", params))
		return nil
	})
	// 2. Expect a call to get the child resource
	mock.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).DoAndReturn(func(ctx context.Context, name types.NamespacedName, obj *k8sapps.Deployment) error {
		assert.Equal("deploymetrics", name.Namespace)
		assert.Equal("deploymetrics-workload", name.Name)
		assert.NoError(updateObjectFromYAMLTemplate(obj, "test/templates/containerized_workload_deployment.yaml", params))
		assert.Contains(obj.Spec.Template.Annotations, "verrazzano.io/metricsEnabled")
		assert.Contains(obj.Spec.Template.Annotations, "verrazzano.io/metricsPath")
		assert.Contains(obj.Spec.Template.Annotations, "verrazzano.io/metricsPort")
		return nil
	})
	// 3. Expect a call to update the child resource to remove the annotations
	mock.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, obj *k8sapps.Deployment, opts ...client.UpdateOption) error {
		assert.Equal("deploymetrics", obj.Namespace)
		assert.Equal("deploymetrics-workload", obj.Name)
		assert.NotContains(obj.Spec.Template.Annotations, "verrazzano.io/metricsEnabled")
		assert.NotContains(obj.Spec.Template.Annotations, "verrazzano.io/metricsPath")
		assert.NotContains(obj.Spec.Template.Annotations, "verrazzano.io/metricsPort")
		return nil
	})
	// 6. Expect a call to get the prometheus deployment.
	mock.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, name types.NamespacedName, deployment *k8sapps.Deployment) error {
		assert.Equal("verrazzano-system", name.Namespace)
		assert.Equal("vmi-system-prometheus-0", name.Name)
		assert.NoError(updateObjectFromYAMLTemplate(deployment, "test/templates/prometheus_deployment.yaml", params))
		return nil
	})
	// 7. Expect a call to get the prometheus configmap.
	mock.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, name types.NamespacedName, configmap *k8score.ConfigMap) error {
		assert.Equal("verrazzano-system", name.Namespace)
		assert.Equal("vmi-system-prometheus-0", name.Name)
		assert.NoError(updateObjectFromYAMLTemplate(configmap, "test/templates/deployment_prometheus_configmap.yaml", params))
		return nil
	})
	// 8. Expect a call to update the prometheus configmap
	mock.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, obj *k8score.ConfigMap, opts ...client.UpdateOption) error {
		assert.Equal("verrazzano-system", obj.Namespace)
		assert.Equal("vmi-system-prometheus-0", obj.Name)
		return nil
	})
	// 9. Expect a call to get the owner application configuration
	mock.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, name types.NamespacedName, appConfig *oamcore.ApplicationConfiguration) error {
		assert.Equal("deploymetrics", name.Namespace)
		assert.Equal("deploymetrics-appconf", name.Name)
		assert.NoError(updateObjectFromYAMLTemplate(appConfig, "test/templates/appconf_metrics_trait_deleted.yaml", params))
		return nil
	})
	// 10. Expect a call to update the owner application configuration
	mock.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, appConfig *oamcore.ApplicationConfiguration, opts ...client.UpdateOption) error {
		assert.Equal("deploymetrics", appConfig.Namespace)
		assert.Equal("deploymetrics-appconf", appConfig.Name)
		return nil
	})
	// 11. Expect a call to update the metrics trait to remove the finalizer
	mock.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, obj *vzapi.MetricsTrait, opts ...client.UpdateOption) error {
		assert.Equal("deploymetrics", obj.Namespace)
		assert.Equal("deploymetrics-deployment-trait", obj.Name)
		assert.Len(obj.Finalizers, 0)
		return nil
	})

	// Create and make the request
	request := ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "deploymetrics", Name: "deploymetrics-deployment-trait"}}
	reconciler := newMetricsTraitReconciler(mock)
	result, err := reconciler.Reconcile(nil, request)

	// Validate the results
	mocker.Finish()
	assert.NoError(err)
	assert.Equal(true, result.Requeue)
	assert.GreaterOrEqual(result.RequeueAfter.Seconds(), 45.0)
}

// TestFetchTraitError tests a failure to fetch the trait during reconcile.
// GIVEN a valid new metrics trait
// WHEN the the metrics trait Reconcile method is invoked
// AND a failure occurs fetching the metrics trait
// THEN verify the metrics trait finalizer is added
// AND verify the error is propigated to the caller
func TestFetchTraitError(t *testing.T) {
	assert := asserts.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	// Expect a call to get the trait resource and return an error.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-namespace", Name: "test-trait-name"}, gomock.Not(gomock.Nil())).
		Return(fmt.Errorf("test-error"))

	// Create and make the request
	reconciler := newMetricsTraitReconciler(mock)
	request := ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "test-namespace", Name: "test-trait-name"}}
	result, err := reconciler.Reconcile(nil, request)

	// Validate the results
	mocker.Finish()
	assert.Nil(err)
	assert.Equal(true, result.Requeue)
}

// TestWorkloadFetchError tests failing to fetch the workload during reconcile.
// GIVEN a valid new metrics trait
// WHEN the the metrics trait Reconcile method is invoked
// AND a failure occurs fetching the metrics trait
// THEN verify the metrics trait finalizer is added
// AND verify the error is propigated to the caller
func TestWorkloadFetchError(t *testing.T) {
	assert := asserts.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	// Expect a call to get the trait resource.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-namespace", Name: "test-trait-name"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, trait *vzapi.MetricsTrait) error {
			trait.TypeMeta = k8smeta.TypeMeta{
				APIVersion: vzapi.SchemeGroupVersion.Identifier(),
				Kind:       vzapi.MetricsTraitKind}
			trait.ObjectMeta = k8smeta.ObjectMeta{
				Namespace: name.Namespace,
				Name:      name.Name}
			trait.Spec.WorkloadReference = oamrt.TypedReference{
				APIVersion: oamcore.SchemeGroupVersion.Identifier(),
				Kind:       oamcore.ContainerizedWorkloadKind,
				Name:       "test-workload-name"}
			return nil
		})
	// Expect a call to update the trait resource with a finalizer.
	mock.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, trait *vzapi.MetricsTrait, opts ...client.UpdateOption) error {
			assert.Equal("test-namespace", trait.Namespace)
			assert.Equal("test-trait-name", trait.Name)
			assert.Len(trait.Finalizers, 1)
			assert.Equal("metricstrait.finalizers.verrazzano.io", trait.Finalizers[0])
			return nil
		})
	// Expect a call to get the containerized workload resource
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-namespace", Name: "test-workload-name"}, gomock.Not(gomock.Nil())).
		Return(fmt.Errorf("test-error"))

	// Create and make the request
	request := ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "test-namespace", Name: "test-trait-name"}}
	reconciler := newMetricsTraitReconciler(mock)
	result, err := reconciler.Reconcile(nil, request)

	// Validate the results
	mocker.Finish()
	assert.NotNil(err)
	assert.Equal(true, result.Requeue)
}

// TestDeploymentUpdateError testing failing to update a workload child deployment during reconcile.
// GIVEN a metrics trait that has been updated
// WHEN the metrics trait Reconcile method is invoked
// AND an error occurs updating the scraper deployment
// THEN verify an error is recorded in the status
func TestDeploymentUpdateError(t *testing.T) {
	assert := asserts.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	testDeployment := k8sapps.Deployment{
		TypeMeta: k8smeta.TypeMeta{
			APIVersion: k8sapps.SchemeGroupVersion.Identifier(),
			Kind:       "Deployment",
		},
		ObjectMeta: k8smeta.ObjectMeta{
			Name:              "test-deployment-name",
			Namespace:         "test-namespace",
			CreationTimestamp: k8smeta.Now(),
			OwnerReferences: []k8smeta.OwnerReference{{
				APIVersion: oamcore.SchemeGroupVersion.Identifier(),
				Kind:       oamcore.ContainerizedWorkloadKind,
				Name:       "test-workload-name",
				UID:        "test-workload-uid"}}}}
	// Expect a call to get the trait resource.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-namespace", Name: "test-trait-name"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, trait *vzapi.MetricsTrait) error {
			trait.TypeMeta = k8smeta.TypeMeta{
				APIVersion: vzapi.SchemeGroupVersion.Identifier(),
				Kind:       vzapi.MetricsTraitKind}
			trait.ObjectMeta = k8smeta.ObjectMeta{
				Namespace: name.Namespace,
				Name:      name.Name}
			trait.Spec.WorkloadReference = oamrt.TypedReference{
				APIVersion: oamcore.SchemeGroupVersion.Identifier(),
				Kind:       oamcore.ContainerizedWorkloadKind,
				Name:       "test-workload-name"}
			return nil
		})
	// Expect a call to update the trait resource with a finalizer.
	mock.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, trait *vzapi.MetricsTrait, opts ...client.UpdateOption) error {
			assert.Equal("test-namespace", trait.Namespace)
			assert.Equal("test-trait-name", trait.Name)
			assert.Len(trait.Finalizers, 1)
			assert.Equal("metricstrait.finalizers.verrazzano.io", trait.Finalizers[0])
			return nil
		})
	// Expect a call to get the containerized workload resource
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-namespace", Name: "test-workload-name"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *unstructured.Unstructured) error {
			workload.SetGroupVersionKind(oamcore.ContainerizedWorkloadGroupVersionKind)
			workload.SetNamespace(name.Namespace)
			workload.SetName(name.Name)
			workload.SetUID("test-workload-uid")
			return nil
		})
	// Expect a call to get the prometheus configuration.
	mock.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, deployment *k8sapps.Deployment) error {
			assert.Equal("istio-system", name.Namespace)
			assert.Equal("prometheus", name.Name)
			deployment.APIVersion = k8sapps.SchemeGroupVersion.Identifier()
			deployment.Kind = deploymentKind
			deployment.Namespace = name.Namespace
			deployment.Name = name.Name
			return nil
		})
	// Expect a call to get the containerized workload resource definition
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "", Name: "containerizedworkloads.core.oam.dev"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workloadDef *oamcore.WorkloadDefinition) error {
			workloadDef.Namespace = name.Namespace
			workloadDef.Name = name.Name
			workloadDef.Spec.ChildResourceKinds = []oamcore.ChildResourceKind{
				{APIVersion: "apps/v1", Kind: "Deployment", Selector: nil},
				{APIVersion: "v1", Kind: "Service", Selector: nil},
			}
			return nil
		})
	// Expect a call to list the child Deployment resources of the containerized workload definition
	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *unstructured.UnstructuredList, opts ...client.ListOption) error {
			assert.Equal("Deployment", list.GetKind())
			return appendAsUnstructured(list, testDeployment)
		})
	// Expect a call to list the child Service resources of the containerized workload definition
	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *unstructured.UnstructuredList, opts ...client.ListOption) error {
			assert.Equal("Service", list.GetKind())
			return nil
		})
	// Expect a call to get the deployment definition
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-namespace", Name: "test-deployment-name"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, deployment *k8sapps.Deployment) error {
			deployment.ObjectMeta = testDeployment.ObjectMeta
			deployment.Spec = testDeployment.Spec
			return nil
		})
	// Expect a call to update the child with annotations but return an error.
	mock.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("test-error"))
	// Expect a call to get the status writer and return a mock.
	mock.EXPECT().Status().Return(mockStatus).AnyTimes()
	// Expect a call to update the status of the ingress trait.
	// The status should include the error updating the deployment
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, trait *vzapi.MetricsTrait, opts ...client.UpdateOption) error {
			assert.Len(trait.Status.Conditions, 1)
			assert.Equal(oamrt.ReasonReconcileError, trait.Status.Conditions[0].Reason)
			return nil
		})

	// Create and make the request
	request := ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "test-namespace", Name: "test-trait-name"}}

	reconciler := newMetricsTraitReconciler(mock)
	result, err := reconciler.Reconcile(nil, request)

	// Validate the results
	mocker.Finish()
	assert.NoError(err)
	assert.Equal(true, result.Requeue)
	assert.Equal(time.Duration(0), result.RequeueAfter)
}

// TestUnsupportedWorkloadType tests a metrics trait with an unsupported workload type
// GIVEN a metrics trait has an unsupported workload type of ConfigMap
// WHEN the metrics trait Reconcile method is invoked
// THEN verify the trait is deleted
func TestUnsupportedWorkloadType(t *testing.T) {
	assert := asserts.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	// Expect a call to get the trait resource.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-namespace", Name: "test-trait-name"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, trait *vzapi.MetricsTrait) error {
			trait.TypeMeta = k8smeta.TypeMeta{
				APIVersion: vzapi.SchemeGroupVersion.Identifier(),
				Kind:       vzapi.MetricsTraitKind}
			trait.ObjectMeta = k8smeta.ObjectMeta{
				Namespace: name.Namespace,
				Name:      name.Name}
			trait.Spec.WorkloadReference = oamrt.TypedReference{
				APIVersion: "v1",
				Kind:       "ConfigMap",
				Name:       "test-workload-name"}
			return nil
		})
	// Expect a call to update the trait resource with a finalizer.
	mock.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, trait *vzapi.MetricsTrait, opts ...client.UpdateOption) error {
			assert.Equal("test-namespace", trait.Namespace)
			assert.Equal("test-trait-name", trait.Name)
			assert.Len(trait.Finalizers, 1)
			assert.Equal("metricstrait.finalizers.verrazzano.io", trait.Finalizers[0])
			return nil
		})
	// Expect a call to get the ConfigMap workload resource
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-namespace", Name: "test-workload-name"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *unstructured.Unstructured) error {
			workload.SetGroupVersionKind(schema.GroupVersionKind{Group: "", Version: "v1", Kind: "ConfigMap"})
			workload.SetNamespace(name.Namespace)
			workload.SetName(name.Name)
			workload.SetUID("test-workload-uid")
			return nil
		})
	// Expect a call to delete the trait resource.
	mock.EXPECT().
		Delete(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, trait *vzapi.MetricsTrait, opts ...client.DeleteOption) error {
			return nil
		})
	// Create and make the request
	request := ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "test-namespace", Name: "test-trait-name"}}

	reconciler := newMetricsTraitReconciler(mock)
	result, err := reconciler.Reconcile(nil, request)

	// Validate the results
	mocker.Finish()
	assert.NoError(err)
	assert.Equal(false, result.Requeue)
}

// TestNoUpdatesRequired tests a reconcile where no updates to any resources was required.
// GIVEN a metrics trait that has not been updated
// WHEN the metrics trait Reconcile method is invoked
// THEN verify no updates are made
func TestNoUpdatesRequired(t *testing.T) {
	assert := asserts.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	testDeployment := k8sapps.Deployment{
		TypeMeta: k8smeta.TypeMeta{
			APIVersion: k8sapps.SchemeGroupVersion.Identifier(),
			Kind:       "Deployment",
		},
		ObjectMeta: k8smeta.ObjectMeta{
			Name:              "test-deployment-name",
			Namespace:         "test-namespace",
			CreationTimestamp: k8smeta.Now(),
			OwnerReferences: []k8smeta.OwnerReference{{
				APIVersion: oamcore.SchemeGroupVersion.Identifier(),
				Kind:       oamcore.ContainerizedWorkloadKind,
				Name:       "test-workload-name",
				UID:        "test-workload-uid"}},
			Labels: map[string]string{
				appObjectMetaLabel:  "test-app-name",
				compObjectMetaLabel: "test-comp-name"}}}
	testDeployment.Spec.Template.ObjectMeta.Labels = map[string]string{
		appObjectMetaLabel:  "test-app-name",
		compObjectMetaLabel: "test-comp-name"}
	annotations := map[string]string{
		"verrazzano.io/metricsEnabled": "true",
		"verrazzano.io/metricsPort":    "8080",
		"verrazzano.io/metricsPath":    "/metrics"}
	testDeployment.Spec.Template.ObjectMeta.Annotations = annotations

	testNamespace := k8score.Namespace{
		TypeMeta: k8smeta.TypeMeta{
			Kind: "Namespace",
		},
		ObjectMeta: k8smeta.ObjectMeta{
			Name: "test-namespace",
		},
	}

	// Expect a call to get the trait resource.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-namespace", Name: "test-trait-name"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, trait *vzapi.MetricsTrait) error {
			trait.TypeMeta = k8smeta.TypeMeta{
				APIVersion: vzapi.SchemeGroupVersion.Identifier(),
				Kind:       vzapi.MetricsTraitKind}
			trait.ObjectMeta = k8smeta.ObjectMeta{
				Namespace: name.Namespace,
				Name:      name.Name,
				Labels: map[string]string{
					appObjectMetaLabel:  "test-app-name",
					compObjectMetaLabel: "test-comp-name"}}
			trait.Spec.WorkloadReference = oamrt.TypedReference{
				APIVersion: oamcore.SchemeGroupVersion.Identifier(),
				Kind:       oamcore.ContainerizedWorkloadKind,
				Name:       "test-workload-name"}
			trait.Status.Conditions = []oamrt.Condition{{
				Type: "Synced", Status: "True", Reason: "ReconcileSuccess", Message: ""}}
			trait.Status.Resources = []vzapi.QualifiedResourceRelation{
				{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Namespace:  "test-namespace",
					Name:       "test-deployment-name",
					Role:       "source",
				},
				{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Namespace:  "istio-system",
					Name:       "prometheus",
					Role:       "scraper",
				}}
			return nil
		})
	// Expect a call to update the trait resource with a finalizer.
	mock.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, trait *vzapi.MetricsTrait, opts ...client.UpdateOption) error {
			assert.Equal("test-namespace", trait.Namespace)
			assert.Equal("test-trait-name", trait.Name)
			assert.Len(trait.Finalizers, 1)
			assert.Equal("metricstrait.finalizers.verrazzano.io", trait.Finalizers[0])
			return nil
		})
	// Expect a call to get the containerized workload resource
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-namespace", Name: "test-workload-name"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *unstructured.Unstructured) error {
			workload.SetGroupVersionKind(oamcore.ContainerizedWorkloadGroupVersionKind)
			workload.SetNamespace(name.Namespace)
			workload.SetName(name.Name)
			workload.SetUID("test-workload-uid")
			return nil
		})
	// Expect a call to get the prometheus deployment.
	mock.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(&k8sapps.Deployment{})).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, deployment *k8sapps.Deployment) error {
			assert.Equal("istio-system", name.Namespace)
			assert.Equal("prometheus", name.Name)
			deployment.APIVersion = k8sapps.SchemeGroupVersion.Identifier()
			deployment.Kind = deploymentKind
			deployment.Namespace = name.Namespace
			deployment.Name = name.Name
			deployment.Spec.Template.Spec.Volumes = []k8score.Volume{{
				Name: "config-volume",
				VolumeSource: k8score.VolumeSource{
					ConfigMap: &k8score.ConfigMapVolumeSource{
						LocalObjectReference: k8score.LocalObjectReference{Name: "prometheus"}}}}}
			return nil
		})
	// Expect a call to get the containerized workload resource definition
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "", Name: "containerizedworkloads.core.oam.dev"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workloadDef *oamcore.WorkloadDefinition) error {
			workloadDef.Namespace = name.Namespace
			workloadDef.Name = name.Name
			workloadDef.Spec.ChildResourceKinds = []oamcore.ChildResourceKind{
				{APIVersion: "apps/v1", Kind: "Deployment", Selector: nil},
				{APIVersion: "v1", Kind: "Service", Selector: nil},
			}
			return nil
		})
	// Expect a call to list the child Deployment resources of the containerized workload definition
	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *unstructured.UnstructuredList, opts ...client.ListOption) error {
			assert.Equal("Deployment", list.GetKind())
			return appendAsUnstructured(list, testDeployment)
		})
	// Expect a call to list the child Service resources of the containerized workload definition
	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *unstructured.UnstructuredList, opts ...client.ListOption) error {
			assert.Equal("Service", list.GetKind())
			return nil
		})
	// Expect a call to get the deployment definition
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-namespace", Name: "test-deployment-name"}, gomock.AssignableToTypeOf(&k8sapps.Deployment{})).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, deployment *k8sapps.Deployment) error {
			deployment.ObjectMeta = testDeployment.ObjectMeta
			deployment.Spec = testDeployment.Spec
			return nil
		})
	// Expect a call to get the prometheus configuration.
	mock.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(&k8score.ConfigMap{})).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, configmap *k8score.ConfigMap) error {
			assert.Equal("istio-system", name.Namespace)
			assert.Equal("prometheus", name.Name)
			configmap.APIVersion = k8sapps.SchemeGroupVersion.Identifier()
			configmap.Kind = configMapKind
			configmap.Namespace = name.Namespace
			configmap.Name = name.Name
			params := map[string]string{
				jobNameHolder:       "test-app-name_default_test-namespace_test-comp-name",
				namespaceHolder:     "test-namespace",
				appNameHolder:       "test-app-name",
				compNameHolder:      "test-comp-name",
				sslProtocolHolder:   "scheme: http",
				vzClusterNameHolder: "thiscluster"}
			scrapeConfigs, err := readTemplate("test/templates/prometheus_scrape_configs.yaml", params)
			scrapeConfigs = removeHeaderLines(scrapeConfigs, 2)
			assert.NoError(err)
			configmap.Data = map[string]string{prometheusConfigKey: scrapeConfigs}
			return nil
		})

	// Expect calls related to getting the Verrazzano cluster name
	// registration secret, failing which, the local registration secret
	expectVerrazzanoClusterNameCalls(mock)

	// Expect a call to get the namespace definition
	mock.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(&k8score.Namespace{})).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, namespace *k8score.Namespace) error {
			namespace.TypeMeta = testNamespace.TypeMeta
			namespace.ObjectMeta = testNamespace.ObjectMeta
			return nil
		})
	// Create and make the request
	request := ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "test-namespace", Name: "test-trait-name"}}
	reconciler := newMetricsTraitReconciler(mock)
	result, err := reconciler.Reconcile(nil, request)

	// Validate the results
	mocker.Finish()
	assert.NoError(err)
	assert.Equal(true, result.Requeue)
	assert.GreaterOrEqual(result.RequeueAfter.Seconds(), 45.0)
}

func expectVerrazzanoClusterNameCalls(mock *mocks.MockClient) {
	localSecretData := map[string][]byte{constants.ClusterNameData: []byte("thiscluster")}
	regSecretFullName := types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace,
		Name: constants.MCRegistrationSecret}
	localSecretFullName := types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace,
		Name: constants.MCLocalRegistrationSecret}
	mock.EXPECT().
		Get(gomock.Any(), regSecretFullName, gomock.Not(gomock.Nil())).
		Return(kerr.NewNotFound(schema.ParseGroupResource("Secret"), constants.MCRegistrationSecret))
	mock.EXPECT().
		Get(gomock.Any(), localSecretFullName, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *k8score.Secret) error {
			secret.Name = localSecretFullName.Name
			secret.Namespace = localSecretFullName.Namespace
			secret.Data = localSecretData
			return nil
		})
}

// TestNoUpdatesRequired tests a reconcile where no updates to any resources was required.
// GIVEN a metrics trait that has not been updated
// WHEN the metrics trait Reconcile method is invoked
// THEN verify no updates are made
func TestSSLNoUpdatesRequired(t *testing.T) {
	assert := asserts.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	testDeployment := k8sapps.Deployment{
		TypeMeta: k8smeta.TypeMeta{
			APIVersion: k8sapps.SchemeGroupVersion.Identifier(),
			Kind:       "Deployment",
		},
		ObjectMeta: k8smeta.ObjectMeta{
			Name:              "test-deployment-name",
			Namespace:         "test-namespace",
			CreationTimestamp: k8smeta.Now(),
			OwnerReferences: []k8smeta.OwnerReference{{
				APIVersion: oamcore.SchemeGroupVersion.Identifier(),
				Kind:       oamcore.ContainerizedWorkloadKind,
				Name:       "test-workload-name",
				UID:        "test-workload-uid"}},
			Labels: map[string]string{
				appObjectMetaLabel:  "test-app-name",
				compObjectMetaLabel: "test-comp-name"}}}
	testDeployment.Spec.Template.ObjectMeta.Labels = map[string]string{
		appObjectMetaLabel:  "test-app-name",
		compObjectMetaLabel: "test-comp-name"}
	annotations := map[string]string{
		"verrazzano.io/metricsEnabled": "true",
		"verrazzano.io/metricsPort":    "8080",
		"verrazzano.io/metricsPath":    "/metrics"}
	testDeployment.Spec.Template.ObjectMeta.Annotations = annotations

	testNamespace := k8score.Namespace{
		TypeMeta: k8smeta.TypeMeta{
			Kind: "Namespace",
		},
		ObjectMeta: k8smeta.ObjectMeta{
			Name: "test-namespace",
		},
	}
	labels := map[string]string{
		"istio-injection": "enabled",
	}
	testNamespace.ObjectMeta.Labels = labels

	// Expect a call to get the trait resource.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-namespace", Name: "test-trait-name"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, trait *vzapi.MetricsTrait) error {
			trait.TypeMeta = k8smeta.TypeMeta{
				APIVersion: vzapi.SchemeGroupVersion.Identifier(),
				Kind:       vzapi.MetricsTraitKind}
			trait.ObjectMeta = k8smeta.ObjectMeta{
				Namespace: name.Namespace,
				Name:      name.Name,
				Labels: map[string]string{
					appObjectMetaLabel:  "test-app-name",
					compObjectMetaLabel: "test-comp-name"}}
			trait.Spec.WorkloadReference = oamrt.TypedReference{
				APIVersion: oamcore.SchemeGroupVersion.Identifier(),
				Kind:       oamcore.ContainerizedWorkloadKind,
				Name:       "test-workload-name"}
			trait.Status.Conditions = []oamrt.Condition{{
				Type: "Synced", Status: "True", Reason: "ReconcileSuccess", Message: ""}}
			trait.Status.Resources = []vzapi.QualifiedResourceRelation{
				{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Namespace:  "test-namespace",
					Name:       "test-deployment-name",
					Role:       "source",
				},
				{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Namespace:  "istio-system",
					Name:       "prometheus",
					Role:       "scraper",
				}}
			return nil
		})
	// Expect a call to update the trait resource with a finalizer.
	mock.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, trait *vzapi.MetricsTrait, opts ...client.UpdateOption) error {
			assert.Equal("test-namespace", trait.Namespace)
			assert.Equal("test-trait-name", trait.Name)
			assert.Len(trait.Finalizers, 1)
			assert.Equal("metricstrait.finalizers.verrazzano.io", trait.Finalizers[0])
			return nil
		})
	// Expect a call to get the containerized workload resource
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-namespace", Name: "test-workload-name"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *unstructured.Unstructured) error {
			workload.SetGroupVersionKind(oamcore.ContainerizedWorkloadGroupVersionKind)
			workload.SetNamespace(name.Namespace)
			workload.SetName(name.Name)
			workload.SetUID("test-workload-uid")
			return nil
		})
	// Expect a call to get the prometheus deployment.
	mock.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(&k8sapps.Deployment{})).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, deployment *k8sapps.Deployment) error {
			assert.Equal("istio-system", name.Namespace)
			assert.Equal("prometheus", name.Name)
			deployment.APIVersion = k8sapps.SchemeGroupVersion.Identifier()
			deployment.Kind = deploymentKind
			deployment.Namespace = name.Namespace
			deployment.Name = name.Name
			deployment.Spec.Template.Spec.Volumes = []k8score.Volume{{
				Name: "config-volume",
				VolumeSource: k8score.VolumeSource{
					ConfigMap: &k8score.ConfigMapVolumeSource{
						LocalObjectReference: k8score.LocalObjectReference{Name: "prometheus"}}}}}
			return nil
		})
	// Expect a call to get the containerized workload resource definition
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "", Name: "containerizedworkloads.core.oam.dev"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workloadDef *oamcore.WorkloadDefinition) error {
			workloadDef.Namespace = name.Namespace
			workloadDef.Name = name.Name
			workloadDef.Spec.ChildResourceKinds = []oamcore.ChildResourceKind{
				{APIVersion: "apps/v1", Kind: "Deployment", Selector: nil},
				{APIVersion: "v1", Kind: "Service", Selector: nil},
			}
			return nil
		})
	// Expect a call to list the child Deployment resources of the containerized workload definition
	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *unstructured.UnstructuredList, opts ...client.ListOption) error {
			assert.Equal("Deployment", list.GetKind())
			return appendAsUnstructured(list, testDeployment)
		})
	// Expect a call to list the child Service resources of the containerized workload definition
	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *unstructured.UnstructuredList, opts ...client.ListOption) error {
			assert.Equal("Service", list.GetKind())
			return nil
		})
	// Expect a call to get the deployment definition
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-namespace", Name: "test-deployment-name"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, deployment *k8sapps.Deployment) error {
			deployment.ObjectMeta = testDeployment.ObjectMeta
			deployment.Spec = testDeployment.Spec
			return nil
		})
	// Expect a call to get the prometheus configuration.
	mock.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(&k8score.ConfigMap{})).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, configmap *k8score.ConfigMap) error {
			assert.Equal("istio-system", name.Namespace)
			assert.Equal("prometheus", name.Name)
			configmap.APIVersion = k8sapps.SchemeGroupVersion.Identifier()
			configmap.Kind = configMapKind
			configmap.Namespace = name.Namespace
			configmap.Name = name.Name
			params := map[string]string{
				jobNameHolder:       "test-app-name_default_test-namespace_test-comp-name",
				namespaceHolder:     "test-namespace",
				appNameHolder:       "test-app-name",
				compNameHolder:      "test-comp-name",
				vzClusterNameHolder: "thiscluster",
				sslProtocolHolder:   "scheme: https\n  tls_config:\n    ca_file: /etc/istio-certs/root-cert.pem\n    cert_file: /etc/istio-certs/cert-chain.pem\n    insecure_skip_verify: true\n    key_file: /etc/istio-certs/key.pem"}
			scrapeConfigs, err := readTemplate("test/templates/prometheus_scrape_configs.yaml", params)
			scrapeConfigs = removeHeaderLines(scrapeConfigs, 2)
			assert.NoError(err)
			configmap.Data = map[string]string{prometheusConfigKey: scrapeConfigs}
			return nil
		})

	// Expect calls to get Verrazzano cluster name
	expectVerrazzanoClusterNameCalls(mock)

	// Expect a call to get the namespace definition
	mock.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(&k8score.Namespace{})).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, namespace *k8score.Namespace) error {
			namespace.TypeMeta = testNamespace.TypeMeta
			namespace.ObjectMeta = testNamespace.ObjectMeta
			return nil
		})
	// Create and make the request
	request := ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "test-namespace", Name: "test-trait-name"}}
	reconciler := newMetricsTraitReconciler(mock)
	result, err := reconciler.Reconcile(nil, request)

	// Validate the results
	mocker.Finish()
	assert.NoError(err)
	assert.Equal(true, result.Requeue)
	assert.GreaterOrEqual(result.RequeueAfter.Seconds(), 45.0)
}

// TestMetricsTraitCreatedForWLSWorkload tests creation of a metrics trait related to a WLS workload.
// GIVEN a metrics trait that has been created
// WHEN the metrics trait Reconcile method is invoked
// THEN verify that metrics trait finalizer is added
// AND verify that pod annotations are updated
// AND verify that the scraper configmap is updated
// AND verify that the scraper pod is restarted
func TestMetricsTraitCreatedForWLSWorkload(t *testing.T) {
	assert := asserts.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	var err error

	params := map[string]string{
		"##OAM_APP_NAME##":         "test-oam-app-name",
		"##OAM_COMP_NAME##":        "test-oam-comp-name",
		"##TRAIT_NAME##":           "test-trait-name",
		"##TRAIT_NAMESPACE##":      "test-namespace",
		"##WORKLOAD_APIVER##":      "weblogic.oracle/v8",
		"##WORKLOAD_KIND##":        "Deployment",
		"##WORKLOAD_NAME##":        "test-workload-name",
		"##PROMETHEUS_NAME##":      "prometheus",
		"##PROMETHEUS_NAMESPACE##": "istio-system",
		"##DOMAIN_NAME##":          "test-workload-name",
		"##DOMAIN_NAMESPACE##":     "test-namespace",
		"##SECRET_NAME##":          "test-secret-namedomain-weblogic-credentials",
		"##SECRET_NAMESPACE##":     "test-namespace",
		"##SECRET_USERNAME##":      base64.StdEncoding.EncodeToString([]byte("test-secret-username")),
		"##SECRET_PASSWORD##":      base64.StdEncoding.EncodeToString([]byte("test-secret-password")),
		"##POD_NAMESPACE##":        "test-namespace",
		"##POD_NAME##":             "test-pod-name",
	}

	testNamespace := k8score.Namespace{
		TypeMeta: k8smeta.TypeMeta{
			Kind: "Namespace",
		},
		ObjectMeta: k8smeta.ObjectMeta{
			Name: "test-namespace",
		},
	}

	// Expect a call to get the trait resource.
	mock.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(&vzapi.MetricsTrait{})).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, trait *vzapi.MetricsTrait) error {
			assert.Equal("test-namespace", name.Namespace)
			assert.Equal("test-trait-name", name.Name)
			assert.NoError(updateObjectFromYAMLTemplate(trait, "test/templates/wls_workload_metrics_trait_created.yaml", params))
			return nil
		})
	// Expect a call to update the trait resource with a finalizer.
	mock.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, trait *vzapi.MetricsTrait, opts ...client.UpdateOption) error {
			assert.Equal("test-namespace", trait.Namespace)
			assert.Equal("test-trait-name", trait.Name)
			assert.Len(trait.Finalizers, 1)
			assert.Equal("metricstrait.finalizers.verrazzano.io", trait.Finalizers[0])
			return nil
		})
	// Expect a call to get the WLS domain workload resource
	mock.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *unstructured.Unstructured) error {
			assert.Equal("test-namespace", name.Namespace)
			assert.Equal("test-workload-name", name.Name)
			assert.NoError(updateUnstructuredFromYAMLTemplate(workload, "test/templates/wls_domain.yaml", params))
			return nil
		})
	// Expect a call to get the prometheus deployment.
	mock.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(&k8sapps.Deployment{})).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, deployment *k8sapps.Deployment) error {
			assert.Equal("istio-system", name.Namespace)
			assert.Equal("prometheus", name.Name)
			assert.NoError(updateObjectFromYAMLTemplate(deployment, "test/templates/prometheus_deployment.yaml", params))
			return nil
		})
	// Expect a call to get the workload definition
	mock.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(&oamcore.WorkloadDefinition{})).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workloadDef *oamcore.WorkloadDefinition) error {
			assert.Equal("", name.Namespace)
			assert.Equal("domains.weblogic.oracle", name.Name)
			assert.NoError(updateObjectFromYAMLTemplate(workloadDef, "deploy/workloaddefinition_wls.yaml", params))
			return nil
		})
	// Expect a call to list the child Pod resources of the workload
	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *unstructured.UnstructuredList, opts ...client.ListOption) error {
			assert.Equal("Pod", list.GetKind())
			pod := k8score.Pod{}
			assert.NoError(updateObjectFromYAMLTemplate(&pod, "test/templates/wls_pod.yaml", params))
			return appendAsUnstructured(list, pod)
		})
	// Expect a call to list the child Service resources of the workload
	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *unstructured.UnstructuredList, opts ...client.ListOption) error {
			assert.Equal("Service", list.GetKind())
			// Don't add any services to the list
			return nil
		})
	// Expect a call to get the child Pod resource
	mock.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(&k8score.Pod{})).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, pod *k8score.Pod) error {
			assert.Equal("test-namespace", pod.Namespace)
			assert.Equal("test-pod-name", pod.Name)
			assert.NoError(updateObjectFromYAMLTemplate(pod, "test/templates/wls_pod.yaml", params))
			return nil
		})
	// Expect a call to get the WLS domain credentials
	mock.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(&k8score.Secret{})).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *k8score.Secret) error {
			assert.Equal("test-namespace", name.Namespace)
			assert.Equal("tododomain-weblogic-credentials", name.Name)
			assert.NoError(updateObjectFromYAMLTemplate(secret, "test/templates/secret_opaque.yaml", params))
			return nil
		})
	// Expect a call to get the prometheus configuration.
	mock.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(&k8score.ConfigMap{})).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, configmap *k8score.ConfigMap) error {
			assert.Equal("istio-system", name.Namespace)
			assert.Equal("prometheus", name.Name)
			assert.NoError(updateObjectFromYAMLTemplate(configmap, "test/templates/prometheus_configmap.yaml", params))
			return nil
		})

	// Expect calls to get Verrazzano Cluster name
	expectVerrazzanoClusterNameCalls(mock)

	// Expect a call to get update the child pod
	mock.EXPECT().
		Update(gomock.Any(), gomock.AssignableToTypeOf(&k8score.Pod{}), gomock.Any()).
		DoAndReturn(func(ctx context.Context, obj *k8score.Pod, opts ...client.UpdateOption) error {
			assert.Equal("test-namespace", obj.Namespace)
			assert.Equal("test-pod-name", obj.Name)
			return nil
		})
	// Expect a call to get update the prometheus configmap
	mock.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, obj *k8score.ConfigMap, opts ...client.UpdateOption) error {
			assert.Equal("istio-system", obj.Namespace)
			assert.Equal("prometheus", obj.Name)
			assert.Contains(obj.Data["prometheus.yml"], "target_label: "+prometheusClusterNameLabel)
			assert.Contains(obj.Data["prometheus.yml"], "replacement: thiscluster")
			return nil
		})
	// Expect a call to get the status writer
	mock.EXPECT().Status().Return(mockStatus).AnyTimes()
	// Expect a call to update the status of the trait status
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, trait *vzapi.MetricsTrait, opts ...client.UpdateOption) error {
			assert.Len(trait.Status.Conditions, 1)
			return nil
		})
	// Expect a call to get the namespace definition
	mock.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(&k8score.Namespace{})).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, namespace *k8score.Namespace) error {
			namespace.TypeMeta = testNamespace.TypeMeta
			namespace.ObjectMeta = testNamespace.ObjectMeta
			return nil
		})

	// Create and make the request
	request := ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "test-namespace", Name: "test-trait-name"}}
	reconciler := newMetricsTraitReconciler(mock)
	result, err := reconciler.Reconcile(nil, request)

	// Validate the results
	mocker.Finish()
	assert.NoError(err)
	assert.Equal(true, result.Requeue)
	assert.GreaterOrEqual(result.RequeueAfter.Seconds(), 45.0)
}

// TestMetricsTraitDeletedForWLSWorkload tests reconciling a deleted metrics trait related to a WLS workload.
// GIVEN a metrics trait with a non-zero deletion time
// WHEN the metrics trait Reconcile method is invoked
// THEN verify that metrics trait finalizer is removed
// AND verify that pod annotations are cleaned up
// AND verify that the scraper configmap is cleanup up
// AND verify that the scraper pod is restarted
func TestMetricsTraitDeletedForWLSWorkload(t *testing.T) {
	assert := asserts.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	// mockStatus := mocks.NewMockStatusWriter(mocker)
	var err error

	params := map[string]string{
		"##OAM_APP_NAME##":         "test-oam-app-name",
		"##OAM_COMP_NAME##":        "test-oam-comp-name",
		"##TRAIT_NAME##":           "test-trait-name",
		"##TRAIT_NAMESPACE##":      "test-namespace",
		"##WORKLOAD_APIVER##":      "weblogic.oracle/v8",
		"##WORKLOAD_KIND##":        "Deployment",
		"##WORKLOAD_NAME##":        "test-workload-name",
		"##PROMETHEUS_NAME##":      "prometheus",
		"##PROMETHEUS_NAMESPACE##": "istio-system",
		"##DOMAIN_NAME##":          "test-workload-name",
		"##DOMAIN_NAMESPACE##":     "test-namespace",
		"##SECRET_NAME##":          "test-secret-namedomain-weblogic-credentials",
		"##SECRET_NAMESPACE##":     "test-namespace",
		"##SECRET_USERNAME##":      base64.StdEncoding.EncodeToString([]byte("test-secret-username")),
		"##SECRET_PASSWORD##":      base64.StdEncoding.EncodeToString([]byte("test-secret-password")),
		"##POD_NAMESPACE##":        "test-namespace",
		"##POD_NAME##":             "test-pod-name",
	}

	// 1. Expect a call to get the deleted trait resource.
	mock.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).DoAndReturn(func(ctx context.Context, name types.NamespacedName, trait *vzapi.MetricsTrait) error {
		assert.Equal("test-namespace", name.Namespace)
		assert.Equal("test-trait-name", name.Name)
		assert.NoError(updateObjectFromYAMLTemplate(trait, "test/templates/wls_workload_metrics_trait_deleted.yaml", params))
		return nil
	})
	// 2. Expect a call to get the child resource.
	mock.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).DoAndReturn(func(ctx context.Context, name types.NamespacedName, obj *k8score.Pod) error {
		assert.Equal("test-namespace", name.Namespace)
		assert.Equal("test-workload-namedomain-adminserver", name.Name)
		podParams := map[string]string{"##POD_NAME##": name.Name}
		assert.NoError(updateObjectFromYAMLTemplate(obj, "test/templates/wls_pod.yaml", podParams, params))
		assert.Contains(obj.Annotations, "verrazzano.io/metricsEnabled")
		assert.Contains(obj.Annotations, "verrazzano.io/metricsPath")
		assert.Contains(obj.Annotations, "verrazzano.io/metricsPort")
		return nil
	})
	// 3. Expect a call to update the child resource to remove the annotations.
	mock.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, obj *k8score.Pod, opts ...client.UpdateOption) error {
		assert.Equal("test-namespace", obj.Namespace)
		assert.Equal("test-workload-namedomain-adminserver", obj.Name)
		assert.NotContains(obj.Annotations, "verrazzano.io/metricsEnabled")
		assert.NotContains(obj.Annotations, "verrazzano.io/metricsPath")
		assert.NotContains(obj.Annotations, "verrazzano.io/metricsPort")
		return nil
	})
	// 4. Expect a call to get the child resource.
	mock.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).DoAndReturn(func(ctx context.Context, name types.NamespacedName, obj *k8score.Pod) error {
		assert.Equal("test-namespace", name.Namespace)
		assert.Equal("test-workload-namedomain-managed-server1", name.Name)
		podParams := map[string]string{"##POD_NAME##": name.Name}
		assert.NoError(updateObjectFromYAMLTemplate(obj, "test/templates/wls_pod.yaml", podParams, params))
		assert.Contains(obj.Annotations, "verrazzano.io/metricsEnabled")
		assert.Contains(obj.Annotations, "verrazzano.io/metricsPath")
		assert.Contains(obj.Annotations, "verrazzano.io/metricsPort")
		return nil
	})
	// 5. Expect a call to update the child resource to remove the annotations.
	mock.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, obj *k8score.Pod, opts ...client.UpdateOption) error {
		assert.Equal("test-namespace", obj.Namespace)
		assert.Equal("test-workload-namedomain-managed-server1", obj.Name)
		assert.NotContains(obj.Annotations, "verrazzano.io/metricsEnabled")
		assert.NotContains(obj.Annotations, "verrazzano.io/metricsPath")
		assert.NotContains(obj.Annotations, "verrazzano.io/metricsPort")
		return nil
	})
	// 6. Expect a call to get the prometheus deployment.
	mock.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, name types.NamespacedName, deployment *k8sapps.Deployment) error {
		assert.Equal("istio-system", name.Namespace)
		assert.Equal("prometheus", name.Name)
		assert.NoError(updateObjectFromYAMLTemplate(deployment, "test/templates/prometheus_deployment.yaml", params))
		return nil
	})
	// 7. Expect a call to get the prometheus configmap.
	mock.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, name types.NamespacedName, configmap *k8score.ConfigMap) error {
		assert.Equal("istio-system", name.Namespace)
		assert.Equal("prometheus", name.Name)
		assert.NoError(updateObjectFromYAMLTemplate(configmap, "test/templates/prometheus_configmap.yaml", params))
		return nil
	})
	// 8. Expect a call to update the prometheus configmap.
	mock.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, obj *k8score.ConfigMap, opts ...client.UpdateOption) error {
		assert.Equal("istio-system", obj.Namespace)
		assert.Equal("prometheus", obj.Name)
		return nil
	})
	// 9. Expect a call to get the owner application configuration
	mock.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, name types.NamespacedName, appConfig *oamcore.ApplicationConfiguration) error {
		assert.Equal("test-namespace", name.Namespace)
		assert.Equal("test-oam-app-name", name.Name)
		assert.NoError(updateObjectFromYAMLTemplate(appConfig, "test/templates/appconf_metrics_trait_deleted.yaml", params))
		return nil
	})
	// 10. Expect a call to update the owner application configuration
	mock.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, appConfig *oamcore.ApplicationConfiguration, opts ...client.UpdateOption) error {
		assert.Equal("test-namespace", appConfig.Namespace)
		assert.Equal("test-oam-app-name", appConfig.Name)
		return nil
	})
	// 11. Expect a call to update the metrics trait to remove the finalizer
	mock.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, obj *vzapi.MetricsTrait, opts ...client.UpdateOption) error {
		assert.Equal("test-namespace", obj.Namespace)
		assert.Equal("test-trait-name", obj.Name)
		assert.Len(obj.Finalizers, 0)
		return nil
	})

	// Create and make the request
	request := ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "test-namespace", Name: "test-trait-name"}}
	reconciler := newMetricsTraitReconciler(mock)
	result, err := reconciler.Reconcile(nil, request)

	// Validate the results
	mocker.Finish()
	assert.NoError(err)
	assert.Equal(true, result.Requeue)
	assert.GreaterOrEqual(result.RequeueAfter.Seconds(), 45.0)
}

// TestMetricsTraitCreatedForCOHWorkload tests the creation of a metrics trait related to a Coherence workload.
// GIVEN a metrics trait that has been created
// AND the metrics trait is related to a Coherence workload
// WHEN the metrics trait Reconcile method is invoked
// THEN verify that metrics trait finalizer is added
// AND verify that pod annotations are updated
// AND verify that the scraper configmap is updated
// AND verify that the scraper pod is restarted
func TestMetricsTraitCreatedForCOHWorkload(t *testing.T) {
	assert := asserts.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	testDeployment := k8sapps.StatefulSet{
		TypeMeta: k8smeta.TypeMeta{
			APIVersion: k8sapps.SchemeGroupVersion.Identifier(),
			Kind:       "StatefulSet",
		},
		ObjectMeta: k8smeta.ObjectMeta{
			Name:              "test-stateful-set-name",
			Namespace:         "test-namespace",
			CreationTimestamp: k8smeta.Now(),
			OwnerReferences: []k8smeta.OwnerReference{{
				APIVersion: "coherence.oracle.com/v1",
				Kind:       "Coherence",
				Name:       "test-workload-name",
				UID:        "test-workload-uid"}}}}
	// Expect a call to get the trait resource.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-namespace", Name: "test-trait-name"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, trait *vzapi.MetricsTrait) error {
			trait.TypeMeta = k8smeta.TypeMeta{
				APIVersion: vzapi.SchemeGroupVersion.Identifier(),
				Kind:       vzapi.MetricsTraitKind}
			trait.ObjectMeta = k8smeta.ObjectMeta{
				Namespace: name.Namespace,
				Name:      name.Name}
			trait.Spec.WorkloadReference = oamrt.TypedReference{
				APIVersion: "coherence.oracle.com/v1",
				Kind:       "Coherence",
				Name:       "test-workload-name"}
			return nil
		})
	// Expect a call to update the trait resource with a finalizer.
	mock.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, trait *vzapi.MetricsTrait, opts ...client.UpdateOption) error {
			assert.Equal("test-namespace", trait.Namespace)
			assert.Equal("test-trait-name", trait.Name)
			assert.Len(trait.Finalizers, 1)
			assert.Equal("metricstrait.finalizers.verrazzano.io", trait.Finalizers[0])
			return nil
		})
	// Expect a call to get the Coherence workload resource
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-namespace", Name: "test-workload-name"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *unstructured.Unstructured) error {
			workload.SetGroupVersionKind(schema.GroupVersionKind{
				Group:   "coherence.oracle.com",
				Version: "v1",
				Kind:    "Coherence",
			})
			workload.SetNamespace(name.Namespace)
			workload.SetName(name.Name)
			workload.SetUID("test-workload-uid")
			return nil
		})
	// Expect a call to get the prometheus configuration.
	mock.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, deployment *k8sapps.Deployment) error {
			assert.Equal("istio-system", name.Namespace)
			assert.Equal("prometheus", name.Name)
			deployment.APIVersion = k8sapps.SchemeGroupVersion.Identifier()
			deployment.Kind = deploymentKind
			deployment.Namespace = name.Namespace
			deployment.Name = name.Name
			return nil
		})
	// Expect a call to get the Coherence workload resource definition
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "", Name: "coherences.coherence.oracle.com"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workloadDef *oamcore.WorkloadDefinition) error {
			workloadDef.Namespace = name.Namespace
			workloadDef.Name = name.Name
			workloadDef.Spec.ChildResourceKinds = []oamcore.ChildResourceKind{
				{APIVersion: "apps/v1", Kind: "StatefulSet", Selector: nil},
				{APIVersion: "v1", Kind: "Service", Selector: nil},
			}
			return nil
		})
	// Expect a call to list the child StatefulSet resources of the Coherence workload definition
	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *unstructured.UnstructuredList, opts ...client.ListOption) error {
			assert.Equal("StatefulSet", list.GetKind())
			return appendAsUnstructured(list, testDeployment)
		})
	// Expect a call to list the child Service resources of the Coherence workload definition
	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *unstructured.UnstructuredList, opts ...client.ListOption) error {
			assert.Equal("Service", list.GetKind())
			return nil
		})
	// Expect a call to get the deployment definition
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-namespace", Name: "test-stateful-set-name"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, statefulSet *k8sapps.StatefulSet) error {
			statefulSet.ObjectMeta = testDeployment.ObjectMeta
			statefulSet.Spec = testDeployment.Spec
			return nil
		})
	// Expect a call to update the prometheus config
	mock.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, deployment *k8sapps.StatefulSet, opts ...client.UpdateOption) error {
			scrape, ok := deployment.Spec.Template.Annotations["verrazzano.io/metricsEnabled"]
			assert.True(ok)
			assert.Equal("true", scrape)
			target, ok := deployment.Spec.Template.Annotations["verrazzano.io/metricsPath"]
			assert.True(ok)
			assert.Equal("/metrics", target)
			port, ok := deployment.Spec.Template.Annotations["verrazzano.io/metricsPort"]
			assert.True(ok)
			assert.Equal("9612", port)
			return nil
		})
	// Expect a call to get the status writer
	mock.EXPECT().Status().Return(mockStatus).AnyTimes()
	// Expect a call to update the status of the trait status
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, trait *vzapi.MetricsTrait, opts ...client.UpdateOption) error {
			assert.Len(trait.Status.Conditions, 1)
			return nil
		})

	// Create and make the request
	request := ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "test-namespace", Name: "test-trait-name"}}

	reconciler := newMetricsTraitReconciler(mock)
	result, err := reconciler.Reconcile(nil, request)

	// Validate the results
	mocker.Finish()
	assert.NoError(err)
	assert.Equal(true, result.Requeue)
	assert.Equal(time.Duration(0), result.RequeueAfter)
}

// TestMetricsTraitCreatedWithMultiplePorts tests the creation of a metrics trait related to a Coherence workload.
// GIVEN a metrics trait that has been created that specifies multiple metrics ports
// AND the metrics trait is related to a Coherence workload
// WHEN the metrics trait Reconcile method is invoked
// THEN verify that metrics trait finalizer is added
// AND verify that pod annotations are updated
// AND verify that the scraper configmap is updated
// AND verify that the scraper pod is restarted
func TestMetricsTraitCreatedWithMultiplePorts(t *testing.T) {
	assert := asserts.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	testDeployment := k8sapps.StatefulSet{
		TypeMeta: k8smeta.TypeMeta{
			APIVersion: k8sapps.SchemeGroupVersion.Identifier(),
			Kind:       "StatefulSet",
		},
		ObjectMeta: k8smeta.ObjectMeta{
			Name:              "test-stateful-set-name",
			Namespace:         "test-namespace",
			CreationTimestamp: k8smeta.Now(),
			OwnerReferences: []k8smeta.OwnerReference{{
				APIVersion: "coherence.oracle.com/v1",
				Kind:       "Coherence",
				Name:       "test-workload-name",
				UID:        "test-workload-uid"}}}}
	// Expect a call to get the trait resource.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-namespace", Name: "test-trait-name"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, trait *vzapi.MetricsTrait) error {
			trait.TypeMeta = k8smeta.TypeMeta{
				APIVersion: vzapi.SchemeGroupVersion.Identifier(),
				Kind:       vzapi.MetricsTraitKind}
			trait.ObjectMeta = k8smeta.ObjectMeta{
				Namespace: name.Namespace,
				Name:      name.Name}
			trait.Spec.WorkloadReference = oamrt.TypedReference{
				APIVersion: "coherence.oracle.com/v1",
				Kind:       "Coherence",
				Name:       "test-workload-name"}
			port1 := 8080
			port2 := 9612
			path2 := "/somemetrics"
			trait.Spec.Ports = []vzapi.PortSpec{{
				Port: &port1,
			}, {
				Port: &port2,
				Path: &path2,
			}}
			return nil
		})
	// Expect a call to update the trait resource with a finalizer.
	mock.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, trait *vzapi.MetricsTrait, opts ...client.UpdateOption) error {
			assert.Equal("test-namespace", trait.Namespace)
			assert.Equal("test-trait-name", trait.Name)
			assert.Len(trait.Finalizers, 1)
			assert.Equal("metricstrait.finalizers.verrazzano.io", trait.Finalizers[0])
			return nil
		})
	// Expect a call to get the Coherence workload resource
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-namespace", Name: "test-workload-name"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *unstructured.Unstructured) error {
			workload.SetGroupVersionKind(schema.GroupVersionKind{
				Group:   "coherence.oracle.com",
				Version: "v1",
				Kind:    "Coherence",
			})
			workload.SetNamespace(name.Namespace)
			workload.SetName(name.Name)
			workload.SetUID("test-workload-uid")
			return nil
		})
	// Expect a call to get the prometheus configuration.
	mock.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, deployment *k8sapps.Deployment) error {
			assert.Equal("istio-system", name.Namespace)
			assert.Equal("prometheus", name.Name)
			deployment.APIVersion = k8sapps.SchemeGroupVersion.Identifier()
			deployment.Kind = deploymentKind
			deployment.Namespace = name.Namespace
			deployment.Name = name.Name
			return nil
		})
	// Expect a call to get the Coherence workload resource definition
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "", Name: "coherences.coherence.oracle.com"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workloadDef *oamcore.WorkloadDefinition) error {
			workloadDef.Namespace = name.Namespace
			workloadDef.Name = name.Name
			workloadDef.Spec.ChildResourceKinds = []oamcore.ChildResourceKind{
				{APIVersion: "apps/v1", Kind: "StatefulSet", Selector: nil},
				{APIVersion: "v1", Kind: "Service", Selector: nil},
			}
			return nil
		})
	// Expect a call to list the child StatefulSet resources of the Coherence workload definition
	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *unstructured.UnstructuredList, opts ...client.ListOption) error {
			assert.Equal("StatefulSet", list.GetKind())
			return appendAsUnstructured(list, testDeployment)
		})
	// Expect a call to list the child Service resources of the Coherence workload definition
	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *unstructured.UnstructuredList, opts ...client.ListOption) error {
			assert.Equal("Service", list.GetKind())
			return nil
		})
	// Expect a call to get the deployment definition
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-namespace", Name: "test-stateful-set-name"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, statefulSet *k8sapps.StatefulSet) error {
			statefulSet.ObjectMeta = testDeployment.ObjectMeta
			statefulSet.Spec = testDeployment.Spec
			return nil
		})
	// Expect a call to update the prometheus config
	mock.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, deployment *k8sapps.StatefulSet, opts ...client.UpdateOption) error {
			scrape, ok := deployment.Spec.Template.Annotations["verrazzano.io/metricsEnabled"]
			assert.True(ok)
			assert.Equal("true", scrape)
			target, ok := deployment.Spec.Template.Annotations["verrazzano.io/metricsPath"]
			assert.True(ok)
			assert.Equal("/metrics", target)
			port, ok := deployment.Spec.Template.Annotations["verrazzano.io/metricsPort"]
			assert.True(ok)
			assert.Equal("8080", port)
			scrape, ok = deployment.Spec.Template.Annotations["verrazzano.io/metricsEnabled1"]
			assert.True(ok)
			assert.Equal("true", scrape)
			target, ok = deployment.Spec.Template.Annotations["verrazzano.io/metricsPath1"]
			assert.True(ok)
			assert.Equal("/somemetrics", target)
			port, ok = deployment.Spec.Template.Annotations["verrazzano.io/metricsPort1"]
			assert.True(ok)
			assert.Equal("9612", port)
			return nil
		})
	// Expect a call to get the status writer
	mock.EXPECT().Status().Return(mockStatus).AnyTimes()
	// Expect a call to update the status of the trait status
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, trait *vzapi.MetricsTrait, opts ...client.UpdateOption) error {
			assert.Len(trait.Status.Conditions, 1)
			return nil
		})

	// Create and make the request
	request := ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "test-namespace", Name: "test-trait-name"}}

	reconciler := newMetricsTraitReconciler(mock)
	result, err := reconciler.Reconcile(nil, request)

	// Validate the results
	mocker.Finish()
	assert.NoError(err)
	assert.Equal(true, result.Requeue)
	assert.Equal(time.Duration(0), result.RequeueAfter)
}

// TestMetricsTraitCreatedWithMultiplePortsAndPort tests the creation of a metrics trait related to a Coherence workload.
// GIVEN a metrics trait that has been created that specifies multiple metrics ports and a single port
// AND the metrics trait is related to a Coherence workload
// WHEN the metrics trait Reconcile method is invoked
// THEN verify that metrics trait finalizer is added
// AND verify that pod annotations are updated
// AND verify that the scraper configmap is updated
// AND verify that the scraper pod is restarted
func TestMetricsTraitCreatedWithMultiplePortsAndPort(t *testing.T) {
	assert := asserts.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	testDeployment := k8sapps.StatefulSet{
		TypeMeta: k8smeta.TypeMeta{
			APIVersion: k8sapps.SchemeGroupVersion.Identifier(),
			Kind:       "StatefulSet",
		},
		ObjectMeta: k8smeta.ObjectMeta{
			Name:              "test-stateful-set-name",
			Namespace:         "test-namespace",
			CreationTimestamp: k8smeta.Now(),
			OwnerReferences: []k8smeta.OwnerReference{{
				APIVersion: "coherence.oracle.com/v1",
				Kind:       "Coherence",
				Name:       "test-workload-name",
				UID:        "test-workload-uid"}}}}
	// Expect a call to get the trait resource.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-namespace", Name: "test-trait-name"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, trait *vzapi.MetricsTrait) error {
			trait.TypeMeta = k8smeta.TypeMeta{
				APIVersion: vzapi.SchemeGroupVersion.Identifier(),
				Kind:       vzapi.MetricsTraitKind}
			trait.ObjectMeta = k8smeta.ObjectMeta{
				Namespace: name.Namespace,
				Name:      name.Name}
			trait.Spec.WorkloadReference = oamrt.TypedReference{
				APIVersion: "coherence.oracle.com/v1",
				Kind:       "Coherence",
				Name:       "test-workload-name"}
			singlePort := 7001
			port1 := 8080
			port2 := 9612
			path2 := "/somemetrics"
			trait.Spec.Ports = []vzapi.PortSpec{{
				Port: &port1,
			}, {
				Port: &port2,
				Path: &path2,
			}}
			trait.Spec.Port = &singlePort
			return nil
		})
	// Expect a call to update the trait resource with a finalizer.
	mock.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, trait *vzapi.MetricsTrait, opts ...client.UpdateOption) error {
			assert.Equal("test-namespace", trait.Namespace)
			assert.Equal("test-trait-name", trait.Name)
			assert.Len(trait.Finalizers, 1)
			assert.Equal("metricstrait.finalizers.verrazzano.io", trait.Finalizers[0])
			return nil
		})
	// Expect a call to get the Coherence workload resource
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-namespace", Name: "test-workload-name"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *unstructured.Unstructured) error {
			workload.SetGroupVersionKind(schema.GroupVersionKind{
				Group:   "coherence.oracle.com",
				Version: "v1",
				Kind:    "Coherence",
			})
			workload.SetNamespace(name.Namespace)
			workload.SetName(name.Name)
			workload.SetUID("test-workload-uid")
			return nil
		})
	// Expect a call to get the prometheus configuration.
	mock.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, deployment *k8sapps.Deployment) error {
			assert.Equal("istio-system", name.Namespace)
			assert.Equal("prometheus", name.Name)
			deployment.APIVersion = k8sapps.SchemeGroupVersion.Identifier()
			deployment.Kind = deploymentKind
			deployment.Namespace = name.Namespace
			deployment.Name = name.Name
			return nil
		})
	// Expect a call to get the Coherence workload resource definition
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "", Name: "coherences.coherence.oracle.com"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workloadDef *oamcore.WorkloadDefinition) error {
			workloadDef.Namespace = name.Namespace
			workloadDef.Name = name.Name
			workloadDef.Spec.ChildResourceKinds = []oamcore.ChildResourceKind{
				{APIVersion: "apps/v1", Kind: "StatefulSet", Selector: nil},
				{APIVersion: "v1", Kind: "Service", Selector: nil},
			}
			return nil
		})
	// Expect a call to list the child StatefulSet resources of the Coherence workload definition
	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *unstructured.UnstructuredList, opts ...client.ListOption) error {
			assert.Equal("StatefulSet", list.GetKind())
			return appendAsUnstructured(list, testDeployment)
		})
	// Expect a call to list the child Service resources of the Coherence workload definition
	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *unstructured.UnstructuredList, opts ...client.ListOption) error {
			assert.Equal("Service", list.GetKind())
			return nil
		})
	// Expect a call to get the deployment definition
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-namespace", Name: "test-stateful-set-name"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, statefulSet *k8sapps.StatefulSet) error {
			statefulSet.ObjectMeta = testDeployment.ObjectMeta
			statefulSet.Spec = testDeployment.Spec
			return nil
		})
	// Expect a call to update the prometheus config
	mock.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, deployment *k8sapps.StatefulSet, opts ...client.UpdateOption) error {
			scrape, ok := deployment.Spec.Template.Annotations["verrazzano.io/metricsEnabled"]
			assert.True(ok)
			assert.Equal("true", scrape)
			target, ok := deployment.Spec.Template.Annotations["verrazzano.io/metricsPath"]
			assert.True(ok)
			assert.Equal("/metrics", target)
			port, ok := deployment.Spec.Template.Annotations["verrazzano.io/metricsPort"]
			assert.True(ok)
			assert.Equal("8080", port)
			scrape, ok = deployment.Spec.Template.Annotations["verrazzano.io/metricsEnabled1"]
			assert.True(ok)
			assert.Equal("true", scrape)
			target, ok = deployment.Spec.Template.Annotations["verrazzano.io/metricsPath1"]
			assert.True(ok)
			assert.Equal("/somemetrics", target)
			port, ok = deployment.Spec.Template.Annotations["verrazzano.io/metricsPort1"]
			assert.True(ok)
			assert.Equal("9612", port)
			scrape, ok = deployment.Spec.Template.Annotations["verrazzano.io/metricsEnabled2"]
			assert.True(ok)
			assert.Equal("true", scrape)
			target, ok = deployment.Spec.Template.Annotations["verrazzano.io/metricsPath2"]
			assert.True(ok)
			assert.Equal("/metrics", target)
			port, ok = deployment.Spec.Template.Annotations["verrazzano.io/metricsPort2"]
			assert.True(ok)
			assert.Equal("7001", port)
			return nil
		})
	// Expect a call to get the status writer
	mock.EXPECT().Status().Return(mockStatus).AnyTimes()
	// Expect a call to update the status of the trait status
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, trait *vzapi.MetricsTrait, opts ...client.UpdateOption) error {
			assert.Len(trait.Status.Conditions, 1)
			return nil
		})

	// Create and make the request
	request := ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "test-namespace", Name: "test-trait-name"}}

	reconciler := newMetricsTraitReconciler(mock)
	result, err := reconciler.Reconcile(nil, request)

	// Validate the results
	mocker.Finish()
	assert.NoError(err)
	assert.Equal(true, result.Requeue)
	assert.Equal(time.Duration(0), result.RequeueAfter)
}

// TestMetricsTraitDeletedForCOHWorkload tests deletion of a metrics trait related to a coherence workload.
// GIVEN a metrics trait with a non-zero deletion time
// WHEN the metrics trait Reconcile method is invoked
// THEN verify that metrics trait finalizer is removed
// AND verify that pod annotations are cleaned up
// AND verify that the scraper configmap is cleanup up
// AND verify that the scraper pod is restarted
func TestMetricsTraitDeletedForCOHWorkload(t *testing.T) {
	assert := asserts.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	var err error

	params := map[string]string{
		"##OAM_APP_NAME##":          "test-oam-app-name",
		"##OAM_COMP_NAME##":         "test-oam-comp-name",
		"##TRAIT_NAME##":            "test-trait-name",
		"##TRAIT_NAMESPACE##":       "test-namespace",
		"##WORKLOAD_APIVER##":       "coherence.oracle.com/v1",
		"##WORKLOAD_KIND##":         "Coherence",
		"##WORKLOAD_NAME##":         "test-workload-name",
		"##PROMETHEUS_NAME##":       "vmi-system-prometheus-0",
		"##PROMETHEUS_NAMESPACE##":  "verrazzano-system",
		"##STATEFULSET_NAMESPACE##": "test-namespace",
		"##STATEFULSET_NAME##":      "test-workload-name",
	}

	// 1. Expect a call to get the deleted trait resource.
	mock.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).DoAndReturn(func(ctx context.Context, name types.NamespacedName, trait *vzapi.MetricsTrait) error {
		assert.Equal("test-namespace", name.Namespace)
		assert.Equal("test-trait-name", name.Name)
		assert.NoError(updateObjectFromYAMLTemplate(trait, "test/templates/coherence_workload_metrics_trait_deleted.yaml", params))
		return nil
	})
	// 2. Expect a call to get the child resource
	mock.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).DoAndReturn(func(ctx context.Context, name types.NamespacedName, obj *k8sapps.StatefulSet) error {
		assert.Equal("test-namespace", name.Namespace)
		assert.Equal("test-workload-name", name.Name)
		assert.NoError(updateObjectFromYAMLTemplate(obj, "test/templates/coherence_workload_statefulset.yaml", params))
		assert.Contains(obj.Spec.Template.Annotations, "verrazzano.io/metricsEnabled")
		assert.Contains(obj.Spec.Template.Annotations, "verrazzano.io/metricsPath")
		assert.Contains(obj.Spec.Template.Annotations, "verrazzano.io/metricsPort")
		return nil
	})
	// 3. Expect a call to update the child resource to remove the annotations
	mock.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, obj *k8sapps.StatefulSet, opts ...client.UpdateOption) error {
		assert.Equal("test-namespace", obj.Namespace)
		assert.Equal("test-workload-name", obj.Name)
		assert.NotContains(obj.Spec.Template.Annotations, "verrazzano.io/metricsEnabled")
		assert.NotContains(obj.Spec.Template.Annotations, "verrazzano.io/metricsPath")
		assert.NotContains(obj.Spec.Template.Annotations, "verrazzano.io/metricsPort")
		return nil
	})
	// 6. Expect a call to get the prometheus deployment.
	mock.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, name types.NamespacedName, deployment *k8sapps.Deployment) error {
		assert.Equal("verrazzano-system", name.Namespace)
		assert.Equal("vmi-system-prometheus-0", name.Name)
		assert.NoError(updateObjectFromYAMLTemplate(deployment, "test/templates/prometheus_deployment.yaml", params))
		return nil
	})
	// 7. Expect a call to get the prometheus configmap.
	mock.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, name types.NamespacedName, configmap *k8score.ConfigMap) error {
		assert.Equal("verrazzano-system", name.Namespace)
		assert.Equal("vmi-system-prometheus-0", name.Name)
		assert.NoError(updateObjectFromYAMLTemplate(configmap, "test/templates/prometheus_configmap.yaml", params))
		return nil
	})
	// 8. Expect a call to update the prometheus configmap
	mock.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, obj *k8score.ConfigMap, opts ...client.UpdateOption) error {
		assert.Equal("verrazzano-system", obj.Namespace)
		assert.Equal("vmi-system-prometheus-0", obj.Name)
		return nil
	})
	// 9. Expect a call to get the owner application configuration
	mock.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, name types.NamespacedName, appConfig *oamcore.ApplicationConfiguration) error {
		assert.Equal("test-namespace", name.Namespace)
		assert.Equal("test-oam-app-name", name.Name)
		assert.NoError(updateObjectFromYAMLTemplate(appConfig, "test/templates/appconf_metrics_trait_deleted.yaml", params))
		return nil
	})
	// 10. Expect a call to update the owner application configuration
	mock.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, appConfig *oamcore.ApplicationConfiguration, opts ...client.UpdateOption) error {
		assert.Equal("test-namespace", appConfig.Namespace)
		assert.Equal("test-oam-app-name", appConfig.Name)
		return nil
	})
	// 11. Expect a call to update the metrics trait to remove the finalizer
	mock.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, obj *vzapi.MetricsTrait, opts ...client.UpdateOption) error {
		assert.Equal("test-namespace", obj.Namespace)
		assert.Equal("test-trait-name", obj.Name)
		assert.Len(obj.Finalizers, 0)
		return nil
	})

	// Create and make the request
	request := ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "test-namespace", Name: "test-trait-name"}}
	reconciler := newMetricsTraitReconciler(mock)
	result, err := reconciler.Reconcile(nil, request)

	// Validate the results
	mocker.Finish()
	assert.NoError(err)
	assert.Equal(true, result.Requeue)
	assert.GreaterOrEqual(result.RequeueAfter.Seconds(), 45.0)
}

// TestUseHTTPSForScrapeTargetFalseConditions tests that false is returned for the following conditions
// GIVEN a unlabeled Istio namespace or a workload of kind VerrazzanoCoherenceWorkload or a workload of kind Coherence
// WHEN the useHttpsForScrapeTarget method is invoked
// THEN verify that the false boolean value is returned since all those conditions require an http scrape target
func TestUseHTTPSForScrapeTargetFalseConditions(t *testing.T) {
	assert := asserts.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	mtrait := vzapi.MetricsTrait{
		TypeMeta: k8smeta.TypeMeta{
			Kind: "VerrazzanoCoherenceWorkload",
		},
	}

	testNamespace := k8score.Namespace{
		TypeMeta: k8smeta.TypeMeta{
			Kind: "Namespace",
		},
		ObjectMeta: k8smeta.ObjectMeta{
			Name: "test-namespace",
		},
	}

	// Expect a call to get the namespace definition
	mock.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, namespace *k8score.Namespace) error {
			namespace.TypeMeta = testNamespace.TypeMeta
			namespace.ObjectMeta = testNamespace.ObjectMeta
			return nil
		})

	mtrait.Spec.WorkloadReference.Kind = "VerrazzanoCoherenceWorkload"
	https, _ := useHTTPSForScrapeTarget(nil, nil, &mtrait)
	// Expect https to be false for scrape target of Kind VerrazzanoCoherenceWorkload
	assert.False(https, "Expected https to be false for Workload of Kind VerrazzanoCoherenceWorkload")

	mtrait.Spec.WorkloadReference.Kind = "Coherence"
	https, _ = useHTTPSForScrapeTarget(nil, nil, &mtrait)
	// Expect https to be false for scrape target of Kind Coherence
	assert.False(https, "Expected https to be false for Workload of Kind Coherence")

	reconciler := newMetricsTraitReconciler(mock)

	mtrait.Spec.WorkloadReference.Kind = ""
	https, _ = useHTTPSForScrapeTarget(nil, reconciler.Client, &mtrait)
	// Expect https to be false for namespaces NOT labeled for istio-injection
	assert.False(https, "Expected https to be false for namespace NOT labeled for istio injection")
	mocker.Finish()
}

// TestUseHTTPSForScrapeTargetTrueCondition tests that true is returned for namespaces marked for Istio injection
// GIVEN a labeled Istio namespace
// WHEN the useHttpsForScrapeTarget method is invoked
// THEN verify that the true boolean value is returned since pods in Istio namespaces require an https scrape target because of MTLS
func TestUseHTTPSForScrapeTargetTrueCondition(t *testing.T) {
	assert := asserts.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	mtrait := vzapi.MetricsTrait{
		TypeMeta: k8smeta.TypeMeta{
			Kind: "ContainerizedWorkload",
		},
	}

	testNamespace := k8score.Namespace{
		TypeMeta: k8smeta.TypeMeta{
			Kind: "Namespace",
		},
		ObjectMeta: k8smeta.ObjectMeta{
			Name: "test-namespace",
		},
	}

	// Expect a call to get the namespace definition
	mock.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, namespace *k8score.Namespace) error {
			namespace.TypeMeta = testNamespace.TypeMeta
			namespace.ObjectMeta = testNamespace.ObjectMeta
			return nil
		})

	reconciler := newMetricsTraitReconciler(mock)

	labels := map[string]string{
		"istio-injection": "enabled",
	}
	testNamespace.ObjectMeta.Labels = labels
	https, _ := useHTTPSForScrapeTarget(nil, reconciler.Client, &mtrait)
	// Expect https to be true for namespaces labeled for istio-injection
	assert.True(https, "Expected https to be true for namespaces labeled for Istio injection")
	mocker.Finish()
}

// newMetricsTraitReconciler creates a new reconciler for testing
// cli - The Kerberos client to inject into the reconciler
func newMetricsTraitReconciler(cli client.Client) Reconciler {
	scheme := runtime.NewScheme()
	vzapi.AddToScheme(scheme)
	reconciler := Reconciler{
		Client:  cli,
		Log:     zap.S(),
		Scheme:  scheme,
		Scraper: "istio-system/prometheus",
	}
	return reconciler
}

// convertToUnstructured converts an object to an Unstructured version
// object - The object to convert to Unstructured
func convertToUnstructured(object interface{}) (unstructured.Unstructured, error) {
	bytes, err := json.Marshal(object)
	if err != nil {
		return unstructured.Unstructured{}, err
	}
	var u map[string]interface{}
	json.Unmarshal(bytes, &u)
	return unstructured.Unstructured{Object: u}, nil
}

// appendAsUnstructured appends an object to the list after converting it to an Unstructured
// list - The list to append to.
// object - The object to convert to Unstructured and append to the list
func appendAsUnstructured(list *unstructured.UnstructuredList, object interface{}) error {
	u, err := convertToUnstructured(object)
	if err != nil {
		return err
	}
	list.Items = append(list.Items, u)
	return nil
}

// readTemplate reads a string template from a file and replaces values in the template from param maps
// template - The filename of a template
// params - a vararg of param maps
func readTemplate(template string, params ...map[string]string) (string, error) {
	bytes, err := ioutil.ReadFile("../../" + template)
	if err != nil {
		bytes, err = ioutil.ReadFile("../" + template)
		if err != nil {
			bytes, err = ioutil.ReadFile(template)
			if err != nil {
				return "", err
			}
		}
	}
	content := string(bytes)
	for _, p := range params {
		for k, v := range p {
			content = strings.ReplaceAll(content, k, v)
		}
	}
	return content, nil
}

// removeHeaderLines removes the top N lines from the text.
func removeHeaderLines(text string, lines int) string {
	line := 0
	output := ""
	scanner := bufio.NewScanner(strings.NewReader(text))
	for scanner.Scan() {
		if line >= lines {
			output += scanner.Text()
			output += "\n"
		}
		line++
	}
	return output
}

// updateUnstructuredFromYAMLTemplate updates an unstructured from a populated YAML template file.
// uns - The unstructured to update
// template - The template file
// params - The param maps to merge into the template
func updateUnstructuredFromYAMLTemplate(uns *unstructured.Unstructured, template string, params ...map[string]string) error {
	str, err := readTemplate(template, params...)
	if err != nil {
		return err
	}
	bytes, err := yaml.YAMLToJSON([]byte(str))
	if err != nil {
		return err
	}
	_, _, err = unstructured.UnstructuredJSONScheme.Decode(bytes, nil, uns)
	if err != nil {
		return err
	}
	return nil
}

// updateObjectFromYAMLTemplate updates an object from a populated YAML template file.
// uns - The unstructured to update
// template - The template file
// params - The param maps to merge into the template
func updateObjectFromYAMLTemplate(obj interface{}, template string, params ...map[string]string) error {
	uns := unstructured.Unstructured{}
	err := updateUnstructuredFromYAMLTemplate(&uns, template, params...)
	if err != nil {
		return err
	}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(uns.Object, obj)
	if err != nil {
		return err
	}
	return nil
}

// TestReconcileKubeSystem tests to make sure we do not reconcile
// Any resource that belong to the kube-system namespace
func TestReconcileKubeSystem(t *testing.T) {
	assert := asserts.New(t)

	var mocker = gomock.NewController(t)
	var cli = mocks.NewMockClient(mocker)

	// create a request and reconcile it
	request := ctrl.Request{NamespacedName: types.NamespacedName{Namespace: vzconst.KubeSystem, Name: "test-trait-name"}}
	reconciler := newMetricsTraitReconciler(cli)
	result, err := reconciler.Reconcile(nil, request)

	// Validate the results
	mocker.Finish()
	assert.Nil(err)
	assert.True(result.IsZero())
}

// TestMetricsTraitDisabledForContainerizedWorkload tests the creation of a metrics trait related to a containerized workload.
// GIVEN a metrics trait that has been disabled
// WHEN the metrics trait Reconcile method is invoked
// THEN verify that metrics trait finalizer is added
// AND verify that the scraper configmap is updated with the scrape job being removed
func TestMetricsTraitDisabledForContainerizedWorkload(t *testing.T) {
	assert := asserts.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	disabled := false
	ns := "test-namespace"
	metricsTrait := vzapi.MetricsTrait{
		ObjectMeta: k8smeta.ObjectMeta{
			Namespace:   ns,
			Name:        "my-metrics-trait",
			ClusterName: "my-cluster",
			Labels: map[string]string{
				appObjectMetaLabel:  "test-myapp",
				compObjectMetaLabel: "test-mycomp",
			}},
	}
	jobName, _ := createPrometheusScrapeConfigMapJobName(&metricsTrait, 0)
	promConfigYaml, _ := readTemplate("test/templates/prometheus_config_job_to_be_removed.yaml",
		map[string]string{"##JOB_NAME##": jobName})
	cv := k8score.Volume{Name: "config-volume"}
	cv.ConfigMap = &k8score.ConfigMapVolumeSource{}
	cv.ConfigMap.Name = "prom-config"
	testDeployment := k8sapps.Deployment{
		TypeMeta: k8smeta.TypeMeta{
			APIVersion: k8sapps.SchemeGroupVersion.Identifier(),
			Kind:       "Deployment",
		},
		ObjectMeta: k8smeta.ObjectMeta{
			Name:              "test-deployment-name",
			Namespace:         ns,
			CreationTimestamp: k8smeta.Now(),
			OwnerReferences: []k8smeta.OwnerReference{{
				APIVersion: oamcore.SchemeGroupVersion.Identifier(),
				Kind:       oamcore.ContainerizedWorkloadKind,
				Name:       "test-workload-name",
				UID:        "test-workload-uid"}}}}
	testDeployment.Spec.Template.Spec.Volumes = []k8score.Volume{cv}

	// Expect a call to get configMap
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "istio-system", Name: cv.ConfigMap.Name}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, cm *k8score.ConfigMap) error {
			cm.Data = map[string]string{
				prometheusConfigKey: promConfigYaml,
			}
			return nil
		})
	// Expect a call to get the trait resource.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: ns, Name: metricsTrait.Name}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, trait *vzapi.MetricsTrait) error {
			trait.TypeMeta = k8smeta.TypeMeta{
				APIVersion: vzapi.SchemeGroupVersion.Identifier(),
				Kind:       vzapi.MetricsTraitKind}
			trait.ObjectMeta = k8smeta.ObjectMeta{
				Namespace: name.Namespace,
				Name:      name.Name}
			trait.Spec.Enabled = &disabled
			trait.ClusterName = metricsTrait.ClusterName
			trait.Namespace = metricsTrait.Namespace
			trait.Labels = metricsTrait.Labels
			trait.Spec.WorkloadReference = oamrt.TypedReference{
				APIVersion: oamcore.SchemeGroupVersion.Identifier(),
				Kind:       oamcore.ContainerizedWorkloadKind,
				Name:       "test-workload-name"}
			return nil
		})
	// Expect a call to update the trait resource with a finalizer.
	mock.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, trait *vzapi.MetricsTrait, _ ...client.UpdateOption) error {
			assert.Len(trait.Finalizers, 1)
			assert.Equal("metricstrait.finalizers.verrazzano.io", trait.Finalizers[0])
			return nil
		})
	mock.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, trait *k8sapps.Deployment, _ ...client.UpdateOption) error {
			return nil
		})
	// Expect a call to update the configMap with the scrape job being removed
	mock.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, cm *k8score.ConfigMap, _ ...client.UpdateOption) error {
			config := cm.Data[prometheusConfigKey]
			// Verify the scrape job has been removed by disabling the metricsTrait
			assert.False(strings.Contains(config, jobName))
			return nil
		})
	// Expect a call to get the containerized workload resource
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: ns, Name: "test-workload-name"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *unstructured.Unstructured) error {
			workload.SetGroupVersionKind(oamcore.ContainerizedWorkloadGroupVersionKind)
			workload.SetNamespace(name.Namespace)
			workload.SetName(name.Name)
			workload.SetUID("test-workload-uid")
			return nil
		})
	// Expect a call to get the prometheus configuration.
	mock.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, deployment *k8sapps.Deployment) error {
			assert.Equal("istio-system", name.Namespace)
			assert.Equal("prometheus", name.Name)
			deployment.APIVersion = k8sapps.SchemeGroupVersion.Identifier()
			deployment.Kind = deploymentKind
			deployment.Namespace = name.Namespace
			deployment.Name = name.Name
			deployment.Spec = testDeployment.Spec
			return nil
		})
	// Expect a call to get the containerized workload resource definition
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "", Name: "containerizedworkloads.core.oam.dev"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workloadDef *oamcore.WorkloadDefinition) error {
			workloadDef.Namespace = name.Namespace
			workloadDef.Name = name.Name
			workloadDef.Spec.ChildResourceKinds = []oamcore.ChildResourceKind{
				{APIVersion: "apps/v1", Kind: "Deployment", Selector: nil},
				{APIVersion: "v1", Kind: "Service", Selector: nil},
			}
			return nil
		})
	// Expect a call to list the child Deployment resources of the containerized workload definition
	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *unstructured.UnstructuredList, opts ...client.ListOption) error {
			assert.Equal("Deployment", list.GetKind())
			return appendAsUnstructured(list, testDeployment)
		})
	// Expect a call to list the child Service resources of the containerized workload definition
	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *unstructured.UnstructuredList, opts ...client.ListOption) error {
			assert.Equal("Service", list.GetKind())
			return nil
		})
	// Expect a call to get the deployment definition
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: ns, Name: "test-deployment-name"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, deployment *k8sapps.Deployment) error {
			deployment.ObjectMeta = testDeployment.ObjectMeta
			deployment.Spec = testDeployment.Spec
			return nil
		})
	// Expect a call to get the status writer
	mock.EXPECT().Status().Return(mockStatus).AnyTimes()
	// Expect a call to update the status of the trait status
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, trait *vzapi.MetricsTrait, opts ...client.UpdateOption) error {
			assert.Len(trait.Status.Conditions, 1)
			assert.Equal(disabled, *trait.Spec.Enabled)
			return nil
		})

	// Create and make the request
	request := ctrl.Request{NamespacedName: types.NamespacedName{Namespace: ns, Name: metricsTrait.Name}}
	reconciler := newMetricsTraitReconciler(mock)
	result, err := reconciler.Reconcile(nil, request)
	// Validate the results
	mocker.Finish()
	assert.NoError(err)
	assert.Equal(true, result.Requeue)
}
