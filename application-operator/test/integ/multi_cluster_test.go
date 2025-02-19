// Copyright (C) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package integ_test

import (
	"fmt"
	"reflect"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	clusterstest "github.com/verrazzano/verrazzano/application-operator/controllers/clusters/test"
	"github.com/verrazzano/verrazzano/application-operator/test/integ/util"
)

const (
	multiclusterTestNamespace = "multiclustertest"
	managedClusterName        = "managed1"
	crdDir                    = "../../../platform-operator/helm_config/charts/verrazzano-application-operator/crds"
	timeout                   = 2 * time.Minute
	pollInterval              = 40 * time.Millisecond
	applicationOperator       = "verrazzano-application-operator"
	duration                  = 1 * time.Minute
	existingNamespace         = "test-namespace-exists"
)

var (
	multiclusterCrds = []string{
		fmt.Sprintf("%v/clusters.verrazzano.io_multiclustersecrets.yaml", crdDir),
		fmt.Sprintf("%v/clusters.verrazzano.io_multiclusterconfigmaps.yaml", crdDir),
		fmt.Sprintf("%v/clusters.verrazzano.io_multiclustercomponents.yaml", crdDir),
		fmt.Sprintf("%v/clusters.verrazzano.io_multiclusterapplicationconfigurations.yaml", crdDir),
		fmt.Sprintf("%v/clusters.verrazzano.io_verrazzanoprojects.yaml", crdDir),
	}
)

var _ = ginkgo.Describe("Testing Multi-Cluster CRDs", func() {
	ginkgo.It("MultiCluster CRDs can be applied", func() {
		for _, crd := range multiclusterCrds {
			_, stderr := util.Kubectl(fmt.Sprintf("apply -f %v", crd))
			gomega.Expect(stderr).To(gomega.Equal(""), fmt.Sprintf("Failed to apply CRD %v", crd))
		}
	})
	ginkgo.It("Apply MultiClusterSecret creates K8S secret", func() {
		_, stderr := util.Kubectl("apply -f testdata/multi-cluster/verrazzanoproject_sample.yaml")
		gomega.Expect(stderr).To(gomega.Equal(""))
		_, stderr = util.Kubectl("apply -f testdata/multi-cluster/multicluster_secret_sample.yaml")
		gomega.Expect(stderr).To(gomega.Equal(""))
		mcsecret, err := K8sClient.GetMultiClusterSecret(multiclusterTestNamespace, "mymcsecret")
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Eventually(func() bool {
			return secretExistsWithFields(multiclusterTestNamespace, "mymcsecret", mcsecret)
		}, timeout, pollInterval).Should(gomega.BeTrue())
	})
	ginkgo.It("Apply MultiClusterSecret with 2 placements remains pending", func() {
		_, stderr := util.Kubectl("apply -f testdata/multi-cluster/verrazzanoproject_sample.yaml")
		gomega.Expect(stderr).To(gomega.Equal(""))
		_, stderr = util.Kubectl("apply -f testdata/multi-cluster/multicluster_secret_2placements.yaml")
		gomega.Expect(stderr).To(gomega.Equal(""))
		mcsecret, err := K8sClient.GetMultiClusterSecret(multiclusterTestNamespace, "mymcsecret2")
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Eventually(func() bool {
			return secretExistsWithFields(multiclusterTestNamespace, "mymcsecret2", mcsecret)
		}, timeout, pollInterval).Should(gomega.BeTrue())
		gomega.Eventually(func() bool {
			// Verify we have the expected status update
			mcRetrievedSecret, err := K8sClient.GetMultiClusterSecret(multiclusterTestNamespace, "mymcsecret2")
			return err == nil && mcRetrievedSecret.Status.State == clustersv1alpha1.Pending &&
				isStatusAsExpected(mcRetrievedSecret.Status, clustersv1alpha1.DeployComplete, clustersv1alpha1.Succeeded, "managed1")
		}, timeout, pollInterval).Should(gomega.BeTrue())
	})
	ginkgo.It("Apply MultiClusterComponent creates OAM component ", func() {
		_, stderr := util.Kubectl("apply -f testdata/multi-cluster/verrazzanoproject_sample.yaml")
		gomega.Expect(stderr).To(gomega.Equal(""))
		_, stderr = util.Kubectl("apply -f testdata/multi-cluster/multicluster_component_sample.yaml")
		gomega.Expect(stderr).To(gomega.Equal(""))
		mcComp, err := K8sClient.GetMultiClusterComponent(multiclusterTestNamespace, "mymccomp")
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Eventually(func() bool {
			return componentExistsWithFields(multiclusterTestNamespace, "mymccomp", mcComp)
		}, timeout, pollInterval).Should(gomega.BeTrue())
	})
})

var _ = ginkgo.Describe("Testing MultiClusterConfigMap", func() {
	ginkgo.It("Apply MultiClusterConfigMap creates a ConfigMap ", func() {
		_, stderr := util.Kubectl("apply -f testdata/multi-cluster/verrazzanoproject_sample.yaml")
		gomega.Expect(stderr).To(gomega.Equal(""), "VerrazzanoProject should be created successfully")
		_, stderr = util.Kubectl("apply -f testdata/multi-cluster/multicluster_configmap_sample.yaml")
		gomega.Expect(stderr).To(gomega.Equal(""))
		mcConfigMap, err := K8sClient.GetMultiClusterConfigMap(multiclusterTestNamespace, "mymcconfigmap")
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Eventually(func() bool {
			return configMapExistsMatchingMCConfigMap(
				multiclusterTestNamespace,
				"mymcconfigmap",
				mcConfigMap,
			)
		}, timeout, pollInterval).Should(gomega.BeTrue())
		gomega.Eventually(func() bool {
			// Verify we have the expected status update
			mcConfigMap, err := K8sClient.GetMultiClusterConfigMap(multiclusterTestNamespace, "mymcconfigmap")
			return err == nil && isStatusAsExpected(mcConfigMap.Status, clustersv1alpha1.DeployComplete, clustersv1alpha1.Succeeded, managedClusterName)
		}, timeout, pollInterval).Should(gomega.BeTrue())
	})
	ginkgo.It("Apply Invalid MultiClusterConfigMap results in Failed Status", func() {
		_, stderr := util.Kubectl("apply -f testdata/multi-cluster/verrazzanoproject_sample.yaml")
		gomega.Expect(stderr).To(gomega.Equal(""), "VerrazzanoProject should be created successfully")
		_, stderr = util.Kubectl("apply -f testdata/multi-cluster/multicluster_configmap_INVALID.yaml")
		gomega.Expect(stderr).To(gomega.Equal(""))
		gomega.Eventually(func() bool {
			// Expecting a failed state value in the MultiClusterConfigMap since creation of
			// underlying config map should fail for invalid config map
			mcConfigMap, err := K8sClient.GetMultiClusterConfigMap(multiclusterTestNamespace, "invalid-mccm")
			return err == nil && mcConfigMap.Status.State == clustersv1alpha1.Failed
		}, timeout, pollInterval).Should(gomega.BeTrue())
		gomega.Consistently(func() bool {
			// Verify the controller is not updating the status more than once with the failure,
			// and is adding exactly one cluster level status entry
			mcConfigMap, err := K8sClient.GetMultiClusterConfigMap(multiclusterTestNamespace, "invalid-mccm")
			return err == nil && isStatusAsExpected(mcConfigMap.Status, clustersv1alpha1.DeployFailed, clustersv1alpha1.Failed, managedClusterName)
		}, duration, pollInterval).Should(gomega.BeTrue())
	})
})

var _ = ginkgo.Describe("Testing MultiClusterApplicationConfiguration", func() {
	ginkgo.It("MultiClusterApplicationConfiguration can be created ", func() {
		_, stderr := util.Kubectl("apply -f testdata/multi-cluster/verrazzanoproject_sample.yaml")
		gomega.Expect(stderr).To(gomega.Equal(""))
		// First apply the hello-component referenced in this MultiCluster application config
		_, stderr = util.Kubectl("apply -f testdata/multi-cluster/multicluster_appconf_sample.yaml")
		gomega.Expect(stderr).To(gomega.Equal(""), "multicluster app config should be applied successfully")
		mcAppConfig, err := K8sClient.GetMultiClusterAppConfig(multiclusterTestNamespace, "mymcappconf")
		gomega.Expect(err).To(gomega.BeNil(), "multicluster app config mymcappconf should exist")
		gomega.Eventually(func() bool {
			return appConfigExistsWithFields(multiclusterTestNamespace, "mymcappconf", mcAppConfig)
		}, timeout, pollInterval).Should(gomega.BeTrue())
	})
})

var _ = ginkgo.Describe("Testing VerrazzanoProject validation", func() {
	ginkgo.It("VerrazzanoProject invalid namespace ", func() {
		// Apply VerrazzanoProject resource and expect to fail due to invalid namespace
		_, stderr := util.Kubectl("apply -f testdata/multi-cluster/verrazzanoproject_invalid_namespace.yaml")
		gomega.Expect(stderr).To(gomega.ContainSubstring(fmt.Sprintf("Namespace for the resource must be %q", constants.VerrazzanoMultiClusterNamespace)))
	})
	ginkgo.It("VerrazzanoProject invalid namespaces list", func() {
		// Apply VerrazzanoProject resource and expect to fail due to invalid namespaces list
		_, stderr := util.Kubectl("apply -f testdata/multi-cluster/verrazzanoproject_invalid_namespaces_list.yaml")
		gomega.Expect(stderr).To(gomega.ContainSubstring("missing required field \"namespaces\""))
	})
})

var _ = ginkgo.Describe("Testing VerrazzanoProject namespace generation", func() {
	ginkgo.It("Apply VerrazzanoProject with default namespace labels", func() {
		_, stderr := util.Kubectl("apply -f testdata/multi-cluster/verrazzanoproject_namespace_default_labels.yaml")
		gomega.Expect(stderr).To(gomega.Equal(""), "VerrazzanoProject should be created successfully")
		gomega.Eventually(func() bool {
			namespace, err := K8sClient.GetNamespace("test-namespace-1")
			if err == nil {
				return namespace.Labels[constants.LabelIstioInjection] == constants.LabelIstioInjectionDefault &&
					namespace.Labels[constants.LabelVerrazzanoManaged] == constants.LabelVerrazzanoManagedDefault &&
					namespace.Labels["label1"] == "test1" &&
					len(namespace.Labels) == 3
			}
			return false
		}, timeout, pollInterval).Should(gomega.BeTrue())
		gomega.Eventually(func() bool {
			namespace, err := K8sClient.GetNamespace("test-namespace-2")
			if err == nil {
				return namespace.Labels[constants.LabelIstioInjection] == constants.LabelIstioInjectionDefault &&
					namespace.Labels[constants.LabelVerrazzanoManaged] == constants.LabelVerrazzanoManagedDefault &&
					namespace.Labels["label2"] == "test2" &&
					len(namespace.Labels) == 3
			}
			return false
		}, timeout, pollInterval).Should(gomega.BeTrue())
		gomega.Eventually(func() bool {
			vp, err := K8sClient.GetVerrazzanoProject(constants.VerrazzanoMultiClusterNamespace, "test-default-labels")
			return err == nil && isStatusAsExpected(vp.Status, clustersv1alpha1.DeployComplete, clustersv1alpha1.Succeeded, "managed1")
		}, timeout, pollInterval).Should(gomega.BeTrue())
	})
	ginkgo.It("Apply VerrazzanoProject to override default Verrazzano labels", func() {
		_, stderr := util.Kubectl("apply -f testdata/multi-cluster/verrazzanoproject_namespace_override_labels.yaml")
		gomega.Expect(stderr).To(gomega.Equal(""), "VerrazzanoProject should be updated successfully")
		gomega.Eventually(func() bool {
			namespace, err := K8sClient.GetNamespace("test-namespace-11")
			if err == nil {
				return namespace.Labels[constants.LabelIstioInjection] == "disabled" &&
					namespace.Labels[constants.LabelVerrazzanoManaged] == "false" &&
					namespace.Labels["label1"] == "test1" &&
					len(namespace.Labels) == 3
			}
			return false
		}, timeout, pollInterval).Should(gomega.BeTrue())
		gomega.Eventually(func() bool {
			namespace, err := K8sClient.GetNamespace("test-namespace-12")
			if err == nil {
				return namespace.Labels[constants.LabelIstioInjection] == constants.LabelIstioInjectionDefault &&
					namespace.Labels[constants.LabelVerrazzanoManaged] == constants.LabelVerrazzanoManagedDefault &&
					namespace.Labels["label2"] == "test2" &&
					len(namespace.Labels) == 3
			}
			return false
		}, timeout, pollInterval).Should(gomega.BeTrue())
	})
	ginkgo.It("Apply VerrazzanoProject with namespace already exists", func() {
		_, stderr := util.Kubectl("create ns " + existingNamespace)
		if stderr != "" {
			ginkgo.Fail(fmt.Sprintf("failed to create namespace %s", existingNamespace))
		}
		_, stderr = util.Kubectl("apply -f testdata/multi-cluster/verrazzanoproject_namespace_exists.yaml")
		gomega.Expect(stderr).To(gomega.Equal(""), "VerrazzanoProject should be created successfully")
		gomega.Eventually(func() bool {
			namespace, err := K8sClient.GetNamespace(existingNamespace)
			if err == nil {
				return namespace.Labels[constants.LabelIstioInjection] == constants.LabelIstioInjectionDefault &&
					namespace.Labels[constants.LabelVerrazzanoManaged] == constants.LabelVerrazzanoManagedDefault &&
					namespace.Labels["label1"] == "test1" &&
					len(namespace.Labels) == 3
			}
			return false
		}, timeout, pollInterval).Should(gomega.BeTrue())
		gomega.Eventually(func() bool {
			vp, err := K8sClient.GetVerrazzanoProject(constants.VerrazzanoMultiClusterNamespace, "test-default-labels")
			return err == nil && isStatusAsExpected(vp.Status, clustersv1alpha1.DeployComplete, clustersv1alpha1.Succeeded, "managed1")
		}, timeout, pollInterval).Should(gomega.BeTrue())
	})
})

var _ = ginkgo.Describe("Testing VerrazzanoProject rolebinding generation", func() {
	ginkgo.It("Apply VerrazzanoProject and validate rolebindings are created", func() {
		_, stderr := util.Kubectl("apply -f testdata/multi-cluster/verrazzanoproject_sample.yaml")
		gomega.Expect(stderr).To(gomega.Equal(""), "VerrazzanoProject should be created successfully")

		// expect two admin and two monitor rolebindings
		gomega.Eventually(func() bool {
			return K8sClient.DoesRoleBindingContainSubject("verrazzano-project-admin", "multiclustertest", "User", "test-user")
		}, timeout, pollInterval).Should(gomega.BeTrue())
		gomega.Eventually(func() bool {
			return K8sClient.DoesRoleBindingContainSubject("admin", "multiclustertest", "User", "test-user")
		}, timeout, pollInterval).Should(gomega.BeTrue())
		gomega.Eventually(func() bool {
			return K8sClient.DoesRoleBindingContainSubject("verrazzano-project-monitor", "multiclustertest", "Group", "test-viewers")
		}, timeout, pollInterval).Should(gomega.BeTrue())
		gomega.Eventually(func() bool {
			return K8sClient.DoesRoleBindingContainSubject("view", "multiclustertest", "Group", "test-viewers")
		}, timeout, pollInterval).Should(gomega.BeTrue())
	})
	ginkgo.It("Apply VerrazzanoProject and validate rolebindings are updated", func() {
		_, stderr := util.Kubectl("apply -f testdata/multi-cluster/verrazzanoproject_sample.yaml")
		gomega.Expect(stderr).To(gomega.Equal(""), "VerrazzanoProject should be created successfully")

		_, stderr = util.Kubectl("apply -f testdata/multi-cluster/verrazzanoproject_update_rolebindings.yaml")
		gomega.Expect(stderr).To(gomega.Equal(""), "VerrazzanoProject should be updated successfully")

		// expect two admin and two monitor rolebindings and check that the subjects were updated
		gomega.Eventually(func() bool {
			return K8sClient.DoesRoleBindingContainSubject("verrazzano-project-admin", "multiclustertest", "User", "test-NEW-user")
		}, timeout, pollInterval).Should(gomega.BeTrue())
		gomega.Eventually(func() bool {
			return K8sClient.DoesRoleBindingContainSubject("admin", "multiclustertest", "User", "test-NEW-user")
		}, timeout, pollInterval).Should(gomega.BeTrue())
		gomega.Eventually(func() bool {
			return K8sClient.DoesRoleBindingContainSubject("verrazzano-project-monitor", "multiclustertest", "Group", "test-NEW-viewers")
		}, timeout, pollInterval).Should(gomega.BeTrue())
		gomega.Eventually(func() bool {
			return K8sClient.DoesRoleBindingContainSubject("view", "multiclustertest", "Group", "test-NEW-viewers")
		}, timeout, pollInterval).Should(gomega.BeTrue())
	})
})

func appConfigExistsWithFields(namespace string, name string, multiClusterAppConfig *clustersv1alpha1.MultiClusterApplicationConfiguration) bool {
	fmt.Printf("Looking for OAM app config %v/%v\n", namespace, name)
	appConfig, err := K8sClient.GetOAMAppConfig(namespace, name)
	if err != nil {
		return false
	}
	areEqual := true
	for i, expectedComp := range multiClusterAppConfig.Spec.Template.Spec.Components {
		areEqual = areEqual &&
			appConfig.Spec.Components[i].ComponentName == expectedComp.ComponentName
	}
	if !areEqual {
		fmt.Println("Retrieved app config spec doesn't match multi cluster app config spec: components mismatch")
		return false
	}
	// check annotations
	areEqual = true
	for i, expectedAnnotation := range multiClusterAppConfig.Spec.Template.Metadata.Annotations {
		areEqual = areEqual &&
			appConfig.Annotations[i] == expectedAnnotation
	}
	if !areEqual {
		fmt.Println("Retrieved app config spec doesn't match multi cluster app config spec:  annotations mismatch")
		return false
	}
	return true
}

func componentExistsWithFields(namespace string, name string, multiClusterComp *clustersv1alpha1.MultiClusterComponent) bool {
	fmt.Printf("Looking for OAM Component %v/%v\n", namespace, name)
	component, err := K8sClient.GetOAMComponent(namespace, name)
	if err != nil {
		return false
	}
	areEqual := reflect.DeepEqual(component.Spec.Parameters, multiClusterComp.Spec.Template.Spec.Parameters)
	if !areEqual {
		fmt.Println("Retrieved component parameters don't match multi cluster component parameters")
		return false
	}
	compWorkload, err := clusterstest.ReadContainerizedWorkload(component.Spec.Workload)
	if err != nil {
		fmt.Printf("Retrieved OAM component workload could not be read %v\n", err.Error())
		return false
	}
	mcCompWorkload, err := clusterstest.ReadContainerizedWorkload(multiClusterComp.Spec.Template.Spec.Workload)
	if err != nil {
		fmt.Printf("MultiClusterComponent workload could not be read: %v\n", err.Error())
	}

	if reflect.DeepEqual(compWorkload, mcCompWorkload) {
		return true
	}
	fmt.Println("MultiClusterComponent Workload does not match retrieved OAM Component Workload")
	return false
}

func secretExistsWithFields(namespace, name string, mcsecret *clustersv1alpha1.MultiClusterSecret) bool {
	fmt.Printf("Looking for Kubernetes secret %v/%v\n", namespace, name)
	secret, err := K8sClient.GetSecret(namespace, name)
	return err == nil && reflect.DeepEqual(secret.Data, mcsecret.Spec.Template.Data) &&
		reflect.DeepEqual(secret.Labels, mcsecret.Spec.Template.Metadata.Labels)
}

func configMapExistsMatchingMCConfigMap(namespace, name string, mcConfigMap *clustersv1alpha1.MultiClusterConfigMap) bool {
	fmt.Printf("Looking for Kubernetes ConfigMap %v/%v\n", namespace, name)
	configMap, err := K8sClient.GetConfigMap(namespace, name)
	return err == nil &&
		reflect.DeepEqual(configMap.Data, mcConfigMap.Spec.Template.Data) &&
		reflect.DeepEqual(configMap.BinaryData, mcConfigMap.Spec.Template.BinaryData) &&
		reflect.DeepEqual(configMap.Labels, mcConfigMap.Spec.Template.Metadata.Labels)
}

func createRegistrationSecret() {
	createSecret := fmt.Sprintf(
		"create secret generic %s --from-literal=%s=%s -n %s",
		constants.MCRegistrationSecret,
		constants.ClusterNameData,
		managedClusterName,
		constants.VerrazzanoSystemNamespace)

	_, stderr := util.Kubectl(createSecret)
	if stderr != "" {
		ginkgo.Fail(fmt.Sprintf("failed to create secret %v: %v", constants.MCRegistrationSecret, stderr))
	}
}

func isStatusAsExpected(status clustersv1alpha1.MultiClusterResourceStatus,
	expectedConditionType clustersv1alpha1.ConditionType,
	expectedClusterState clustersv1alpha1.StateType,
	expectedClusterName string) bool {
	matchingConditionCount := 0
	matchingClusterStatusCount := 0
	for _, condition := range status.Conditions {
		if condition.Type == expectedConditionType {
			matchingConditionCount++
		}
	}
	for _, clusterStatus := range status.Clusters {
		if clusterStatus.State == expectedClusterState &&
			clusterStatus.Name == expectedClusterName &&
			clusterStatus.LastUpdateTime != "" {
			matchingClusterStatusCount++
		}
	}
	return matchingConditionCount == 1 && matchingClusterStatusCount == 1
}

func setupMultiClusterTest() {
	isPodRunningYet := func() bool {
		return K8sClient.IsPodRunning(applicationOperator, constants.VerrazzanoSystemNamespace)
	}
	gomega.Eventually(isPodRunningYet, "2m", "5s").Should(gomega.BeTrue(),
		fmt.Sprintf("The pod %s in namespace %s should be in the Running state", applicationOperator, constants.VerrazzanoSystemNamespace))

	_, stderr := util.Kubectl("create ns " + constants.VerrazzanoMultiClusterNamespace)
	if stderr != "" {
		ginkgo.Fail(fmt.Sprintf("failed to create namespace %v", constants.VerrazzanoMultiClusterNamespace))
	}

	_, stderr = util.Kubectl("create ns " + multiclusterTestNamespace)
	if stderr != "" {
		ginkgo.Fail(fmt.Sprintf("failed to create namespace %v", multiclusterTestNamespace))
	}

	createRegistrationSecret()
}
