// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mchelidon

import (
	"fmt"
	"os"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/multicluster/examples"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

const (
	longPollingInterval          = 20 * time.Second
	longWaitTimeout              = 10 * time.Minute
	pollingInterval              = 5 * time.Second
	waitTimeout                  = 5 * time.Minute
	consistentlyDuration         = 1 * time.Minute
	projectfile                  = "tests/e2e/multicluster/examples/helidon-deprecated/testdata/verrazzano-project.yaml"
	compFile                     = "tests/e2e/multicluster/examples/helidon-deprecated/testdata/mc-hello-helidon-comp.yaml"
	appFile                      = "tests/e2e/multicluster/examples/helidon-deprecated/testdata/mc-hello-helidon-app.yaml"
	changePlacementToAdminFile   = "tests/e2e/multicluster/examples/helidon-deprecated/testdata/patch-change-placement-to-admin.yaml"
	changePlacementToManagedFile = "tests/e2e/multicluster/examples/helidon-deprecated/testdata/patch-return-placement-to-managed1.yaml"
	testNamespace                = "hello-helidon-dep"
	testProjectName              = "hello-helidon-dep"
)

var clusterName = os.Getenv("MANAGED_CLUSTER_NAME")
var adminKubeconfig = os.Getenv("ADMIN_KUBECONFIG")
var managedKubeconfig = os.Getenv("MANAGED_KUBECONFIG")

// failed indicates whether any of the tests has failed
var failed = false

var _ = AfterEach(func() {
	// set failed to true if any of the tests has failed
	failed = failed || CurrentSpecReport().Failed()
})

// set the kubeconfig to use the admin cluster kubeconfig and deploy the example resources
var _ = BeforeSuite(func() {
	// deploy the VerrazzanoProject
	Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFileInCluster(projectfile, adminKubeconfig)
	}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())

	// wait for the namespace to be created on the cluster before deploying app
	Eventually(func() bool {
		return examples.HelidonNamespaceExists(adminKubeconfig, testNamespace)
	}, waitTimeout, pollingInterval).Should(BeTrue())

	// deploy the multicluster components
	Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFileInCluster(compFile, adminKubeconfig)
	}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())

	// deploy the multicluster app
	Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFileInCluster(appFile, adminKubeconfig)
	}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())
})

var _ = Describe("Multi-cluster verify hello-helidon", func() {
	Context("Admin Cluster", func() {
		// GIVEN an admin cluster and at least one managed cluster
		// WHEN the example application has been deployed to the admin cluster
		// THEN expect that the multi-cluster resources have been created on the admin cluster
		It("Has multi cluster resources", func() {
			Eventually(func() bool {
				return examples.VerifyMCResourcesV100(adminKubeconfig, true, false, testNamespace)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})
		// GIVEN an admin cluster
		// WHEN the multi-cluster example application has been created on admin cluster but not placed there
		// THEN expect that the app is not deployed to the admin cluster consistently for some length of time
		It("Does not have application placed", func() {
			Consistently(func() bool {
				return examples.VerifyHelloHelidonInCluster(adminKubeconfig, true, false, testProjectName, testNamespace)
			}, consistentlyDuration, pollingInterval).Should(BeTrue())
		})
	})

	Context("Managed Cluster", func() {
		// GIVEN an admin cluster and at least one managed cluster
		// WHEN the example application has been deployed to the admin cluster
		// THEN expect that the multi-cluster resources have been created on the managed cluster
		It("Has multi cluster resources", func() {
			Eventually(func() bool {
				return examples.VerifyMCResourcesV100(managedKubeconfig, false, true, testNamespace)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})
		// GIVEN an admin cluster and at least one managed cluster
		// WHEN the multi-cluster example application has been created on admin cluster and placed in managed cluster
		// THEN expect that the app is deployed to the managed cluster
		It("Has application placed", func() {
			Eventually(func() bool {
				return examples.VerifyHelloHelidonInCluster(managedKubeconfig, false, true, testProjectName, testNamespace)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})
	})

	Context("Remaining Managed Clusters", func() {
		clusterCountStr := os.Getenv("CLUSTER_COUNT")
		if clusterCountStr == "" {
			// skip tests
			return
		}
		clusterCount, err := strconv.Atoi(clusterCountStr)
		if err != nil {
			// skip tests
			return
		}

		kubeconfigDir := os.Getenv("KUBECONFIG_DIR")
		for i := 3; i <= clusterCount; i++ {
			kubeconfig := kubeconfigDir + "/" + fmt.Sprintf("%d", i) + "/kube_config"
			It("Does not have multi cluster resources", func() {
				Eventually(func() bool {
					return examples.VerifyMCResourcesV100(kubeconfig, false, false, testNamespace)
				}, waitTimeout, pollingInterval).Should(BeTrue())
			})
			It("Does not have application placed", func() {
				Eventually(func() bool {
					return examples.VerifyHelloHelidonInCluster(kubeconfig, false, false, testProjectName, testNamespace)
				}, waitTimeout, pollingInterval).Should(BeTrue())
			})
		}
	})

	Context("Logging", func() {
		indexName := "verrazzano-namespace-hello-helidon-dep"

		// GIVEN an admin cluster and at least one managed cluster
		// WHEN the example application has been deployed to the admin cluster
		// THEN expect the Elasticsearch index for the app exists on the admin cluster Elasticsearch
		It("Verify Elasticsearch index exists on admin cluster", func() {
			Eventually(func() bool {
				return pkg.LogIndexFoundInCluster(indexName, adminKubeconfig)
			}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find log index for hello helidon")
		})

		// GIVEN an admin cluster and at least one managed cluster
		// WHEN the example application has been deployed to the admin cluster
		// THEN expect recent Elasticsearch logs for the app exist on the admin cluster Elasticsearch
		It("Verify recent Elasticsearch log record exists on admin cluster", func() {
			Eventually(func() bool {
				return pkg.LogRecordFoundInCluster(indexName, time.Now().Add(-24*time.Hour), map[string]string{
					"kubernetes.labels.app_oam_dev\\/component": "hello-helidon-component",
					"kubernetes.labels.app_oam_dev\\/name":      "hello-helidon-appconf",
					"kubernetes.container_name":                 "hello-helidon-container",
				}, adminKubeconfig)
			}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find a recent log record")
		})
	})

	// GIVEN an admin cluster and at least one managed cluster
	// WHEN the example application has been deployed to the admin cluster
	// THEN expect Prometheus metrics for the app to exist in Prometheus on the admin cluster
	Context("Metrics", func() {
		It("Verify Prometheus metrics exist on admin cluster", func() {
			clusterNameMetricsLabel, _ := pkg.GetClusterNameMetricLabel()
			Eventually(func() bool {
				var m = map[string]string{clusterNameMetricsLabel: clusterName}
				return pkg.MetricsExistInCluster("base_jvm_uptime_seconds", m, adminKubeconfig)
			}, longWaitTimeout, longPollingInterval).Should(BeTrue())
		})
	})

	Context("Change Placement of app to Admin Cluster", func() {
		It("Apply patch to change placement to admin cluster", func() {
			Eventually(func() error {
				return examples.ChangePlacementV100(adminKubeconfig, changePlacementToAdminFile, testNamespace, testProjectName)
			}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())
		})

		It("MC Resources should be removed from managed cluster", func() {
			Eventually(func() bool {
				// app should not be placed in the managed cluster
				return examples.VerifyMCResourcesV100(managedKubeconfig, false, false, testNamespace)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})

		It("App should be removed from managed cluster", func() {
			Eventually(func() bool {
				// app should not be placed in the managed cluster
				return examples.VerifyHelloHelidonInCluster(managedKubeconfig, false, false, testProjectName, testNamespace)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})

		It("App should be placed in admin cluster", func() {
			Eventually(func() bool {
				// app should be placed in the admin cluster
				return examples.VerifyHelloHelidonInCluster(adminKubeconfig, true, true, testProjectName, testProjectName)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})
	})

	// Ensure that if we change placement again, back to the original managed cluster, everything functions
	// as expected. This is needed because the change of placement to admin cluster and the change of placement to
	// a managed cluster are different, and we want to ensure we test the case where the destination cluster is
	// each of the 2 types - admin and managed
	Context("Return the app to Managed Cluster", func() {
		It("Apply patch to change placement back to managed cluster", func() {
			Eventually(func() error {
				return examples.ChangePlacementV100(adminKubeconfig, changePlacementToManagedFile, testNamespace, testProjectName)
			}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())
		})

		// GIVEN an admin cluster
		// WHEN the multi-cluster example application has changed placement from admin back to managed cluster
		// THEN expect that the app is not deployed to the admin cluster
		It("Admin cluster does not have application placed", func() {
			Eventually(func() bool {
				return examples.VerifyHelloHelidonInCluster(adminKubeconfig, true, false, testProjectName, testNamespace)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})

		// GIVEN a managed cluster
		// WHEN the multi-cluster example application has changed placement to this managed cluster
		// THEN expect that the app is now deployed to the cluster
		It("Managed cluster again has application placed", func() {
			Eventually(func() bool {
				return examples.VerifyHelloHelidonInCluster(managedKubeconfig, false, true, testProjectName, testNamespace)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})
	})

	Context("Delete resources", func() {
		It("Delete resources on admin cluster", func() {
			Eventually(func() error {
				return cleanUp(adminKubeconfig)
			}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())
		})

		It("Verify deletion on admin cluster", func() {
			Eventually(func() bool {
				return examples.VerifyHelloHelidonDeletedAdminCluster(adminKubeconfig, false, testNamespace, testProjectName)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})

		It("Verify automatic deletion on managed cluster", func() {
			Eventually(func() bool {
				return examples.VerifyHelloHelidonDeletedInManagedCluster(managedKubeconfig, testNamespace, testProjectName)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})

		It("Delete test namespace on managed cluster", func() {
			Eventually(func() error {
				return pkg.DeleteNamespaceInCluster(testNamespace, managedKubeconfig)
			}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())
		})

		It("Delete test namespace on admin cluster", func() {
			Eventually(func() error {
				return pkg.DeleteNamespaceInCluster(testNamespace, adminKubeconfig)
			}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())
		})
	})
})

var _ = AfterSuite(func() {
	if failed {
		pkg.ExecuteClusterDumpWithEnvVarConfig()
	}
})

func cleanUp(kubeconfigPath string) error {
	if err := pkg.DeleteResourceFromFileInCluster(projectfile, kubeconfigPath); err != nil {
		return fmt.Errorf("failed to delete multi-cluster hello-helidon application resource: %v", err)
	}

	if err := pkg.DeleteResourceFromFileInCluster(compFile, kubeconfigPath); err != nil {
		return fmt.Errorf("failed to delete multi-cluster hello-helidon component resources: %v", err)
	}

	if err := pkg.DeleteResourceFromFileInCluster(appFile, kubeconfigPath); err != nil {
		return fmt.Errorf("failed to delete hello-helidon project resource: %v", err)
	}
	return nil
}
