// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package ingresstrait

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/app/v1alpha1"
	vzoam "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Test_createIngressTraitReconcileRequests tests the createIngressTraitReconcileRequests func for the following use case.
// GIVEN a request to createIngressTraitReconcileRequests
// THEN the correct set of reconcile requests is returned based on the number if IngressTraits across all namespaces
func Test_createIngressTraitReconcileRequests(t *testing.T) {

	asserts := assert.New(t)

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = vzapi.AddToScheme(scheme)
	_ = vzoam.AddToScheme(scheme)
	client := fake.NewClientBuilder().WithScheme(scheme).Build()

	reconciler := newIngressTraitReconciler(client)

	asserts.Len(reconciler.createIngressTraitReconcileRequests(), 0)

	_ = client.Create(context.TODO(), &vzoam.IngressTrait{ObjectMeta: metav1.ObjectMeta{Name: "trait1", Namespace: "traitns1"}})
	_ = client.Create(context.TODO(), &vzoam.IngressTrait{ObjectMeta: metav1.ObjectMeta{Name: "trait2", Namespace: "traitns1"}})
	_ = client.Create(context.TODO(), &vzoam.IngressTrait{ObjectMeta: metav1.ObjectMeta{Name: "trait1", Namespace: "traitns2"}})
	_ = client.Create(context.TODO(), &vzoam.IngressTrait{ObjectMeta: metav1.ObjectMeta{Name: "trait1", Namespace: "traitns3"}})
	_ = client.Create(context.TODO(), &vzoam.IngressTrait{ObjectMeta: metav1.ObjectMeta{Name: "trait2", Namespace: "traitns3"}})

	expectedRequests := []reconcile.Request{
		{NamespacedName: types.NamespacedName{Name: "trait1", Namespace: "traitns1"}},
		{NamespacedName: types.NamespacedName{Name: "trait2", Namespace: "traitns1"}},
		{NamespacedName: types.NamespacedName{Name: "trait1", Namespace: "traitns2"}},
		{NamespacedName: types.NamespacedName{Name: "trait1", Namespace: "traitns3"}},
		{NamespacedName: types.NamespacedName{Name: "trait2", Namespace: "traitns3"}},
	}
	actualRequests := reconciler.createIngressTraitReconcileRequests()
	asserts.Len(actualRequests, 5)
	asserts.Equal(expectedRequests, actualRequests)
}

// TestReconciler_shouldUpdateIngressTraits tests the isConsoleIngressUpdated func for the following use case.
// GIVEN a request to isConsoleIngressUpdated
// WHEN the only the Verrazzano Console ingress has changed
// THEN true is returned only when the TLS fields differ, false otherwise
func TestReconciler_shouldUpdateIngressTraits(t *testing.T) {
	tests := []struct {
		name string
		old  *v1alpha1.Verrazzano
		new  *v1alpha1.Verrazzano
		want bool
	}{
		{
			name: "no change",
			old:  &v1alpha1.Verrazzano{},
			new:  &v1alpha1.Verrazzano{},
			want: false,
		},
		{
			name: "not-change-env",
			old:  &v1alpha1.Verrazzano{},
			new: &v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					EnvironmentName: "default",
				},
			},
			want: false,
		},
		{
			name: "change-env",
			old:  &v1alpha1.Verrazzano{},
			new: &v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					EnvironmentName: "test-env",
				},
			},
			want: true,
		},
		{
			name: "change-oci-dns",
			old:  &v1alpha1.Verrazzano{},
			new: &v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						DNS: &v1alpha1.DNSComponent{
							OCI: &v1alpha1.OCI{
								DNSZoneName: "test-domain",
							},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "change-external-dns",
			old:  &v1alpha1.Verrazzano{},
			new: &v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						DNS: &v1alpha1.DNSComponent{
							External: &v1alpha1.External{
								Suffix: "test-suffix",
							},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "change-wildcard-dns",
			old:  &v1alpha1.Verrazzano{},
			new: &v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						DNS: &v1alpha1.DNSComponent{
							Wildcard: &v1alpha1.Wildcard{
								Domain: "xip.io",
							},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "change-istio-install-args",
			old:  &v1alpha1.Verrazzano{},
			new: &v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						Istio: &v1alpha1.IstioComponent{
							IstioInstallArgs: []v1alpha1.InstallArgs{{Name: "foo", Value: "bar"}},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "change-istio-type-to-nodeport-without-externalIPs",
			old:  &v1alpha1.Verrazzano{},
			new: &v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						Istio: &v1alpha1.IstioComponent{
							Ingress: &v1alpha1.IstioIngressSection{
								Type: v1alpha1.NodePort,
							},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "change-istio-type-to-nodeport-with-externalIPs",
			old:  &v1alpha1.Verrazzano{},
			new: &v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						Istio: &v1alpha1.IstioComponent{
							Ingress: &v1alpha1.IstioIngressSection{
								Type: v1alpha1.NodePort,
							},
							IstioInstallArgs: []v1alpha1.InstallArgs{
								{
									Name:      "gateways.istio-ingressgateway.externalIPs",
									ValueList: []string{testExternalIP},
								},
							},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "change-istio-type-from-nodeport",
			old: &v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						Istio: &v1alpha1.IstioComponent{
							Ingress: &v1alpha1.IstioIngressSection{
								Type: v1alpha1.NodePort,
							},
						},
					},
				},
			},
			new: &v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						Istio: &v1alpha1.IstioComponent{
							Ingress: &v1alpha1.IstioIngressSection{
								Type: v1alpha1.LoadBalancer,
							},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "change-istio-ports",
			old:  &v1alpha1.Verrazzano{},
			new: &v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						Istio: &v1alpha1.IstioComponent{
							Ingress: &v1alpha1.IstioIngressSection{
								Ports: []corev1.ServicePort{{Name: "https2", NodePort: 30057}},
							},
						},
					},
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			_ = corev1.AddToScheme(scheme)
			_ = vzapi.AddToScheme(scheme)
			_ = vzoam.AddToScheme(scheme)
			client := fake.NewClientBuilder().WithScheme(scheme).Build()

			r := newIngressTraitReconciler(client)
			if got := r.shouldUpdateIngressTraits(event.UpdateEvent{
				ObjectOld: tt.old,
				ObjectNew: tt.new,
			}); got != tt.want {
				t.Errorf("shouldUpdateIngressTraits() = %v, want %v", got, tt.want)
			}
		})
	}
}
