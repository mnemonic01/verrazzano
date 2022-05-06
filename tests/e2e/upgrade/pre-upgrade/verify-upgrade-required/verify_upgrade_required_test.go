// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verify

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/semver"
	vzalpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/pkg/test/framework/metrics"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

var upgradeVersion = os.Getenv("VERRAZZANO_UPGRADE_VERSION")

const (
	preventUpdatesDuringUpgradeFileEnvVar = "PREVENT_UPDATES_DURING_UPGRADE_FILE"
)

var t = framework.NewTestFramework("verify")

var _ = t.BeforeSuite(func() {
	start := time.Now()
	beforeSuitePassed = true
	metrics.Emit(t.Metrics.With("before_suite_elapsed_time", time.Since(start).Milliseconds()))
})

var failed = false
var beforeSuitePassed = false

var _ = t.AfterEach(func() {
	failed = failed || framework.VzCurrentGinkgoTestDescription().Failed()
})

var _ = t.AfterSuite(func() {
	start := time.Now()
	if failed || !beforeSuitePassed {
		pkg.ExecuteClusterDumpWithEnvVarConfig()
	}
	metrics.Emit(t.Metrics.With("after_suite_elapsed_time", time.Since(start).Milliseconds()))
})

var _ = t.Describe("Verify upgrade required before update is allowed", Label("f:platform-lcm.upgrade", "f:observability.monitoring.prom"), func() {

	// This is a very-specific check, which expects to run in the situation where we've updated the VPO to a
	// newer version but have not yet run an upgrade.

	// This is only valid for Release 1.3+, but we have no way to guard that in code since we don't have visibility into the
	// BOM version in the platform operator; we only have what's in the Spec and the Status version fields.

	// Verify that an edit to the system configuration is rejected when an upgrade is available but not yet applied
	// GIVEN a Verrazzano install
	// WHEN an edit is made without an upgrade, but an upgrade to a newer version is available
	// THEN the edit is rejected by the webhook
	t.Context("Verify upgrade-required checks", func() {
		t.It("Upgrade required check", func() {
			if upgradeVersion != "" {
				upgradeSemVer, err := semver.NewSemVersion(upgradeVersion)
				if err != nil {
					t.Fail(fmt.Sprintf("Error constructing upgrade semantic version %s: %s", upgradeVersion, err.Error()))
					return
				}
				minimumVersion := "1.3.0"
				minimumVerrazzanoVersion, err := semver.NewSemVersion(minimumVersion)
				if err != nil {
					t.Fail(fmt.Sprintf("Error constructing minimum semantic version %s: %s", minimumVersion, err.Error()))
					return
				}
				if upgradeSemVer.IsLessThan(minimumVerrazzanoVersion) {
					_, err = os.Create(preventUpdatesDuringUpgradeFileEnvVar)
					if err != nil {
						t.Fail(fmt.Sprintf("Failed to create file %s", os.Getenv(preventUpdatesDuringUpgradeFileEnvVar)))
					}
					Skip(fmt.Sprintf("Skipping the upgrade-required check spec since the upgrade Verrazzano "+
						"version %s is less than version %s", upgradeVersion, minimumVersion))
				}
			}
			vz, err := pkg.GetVerrazzano()
			if err != nil {
				t.Fail(fmt.Sprintf("Error getting Verrazzano instance: %s", err.Error()))
				return
			}

			if vz.Spec.Components.Istio == nil {
				vz.Spec.Components.Istio = &vzalpha1.IstioComponent{}
			}
			istio := vz.Spec.Components.Istio
			if istio.Ingress == nil {
				istio.Ingress = &vzalpha1.IstioIngressSection{
					Kubernetes: &vzalpha1.IstioKubernetesSection{},
				}
			}
			if istio.Egress == nil {
				istio.Egress = &vzalpha1.IstioEgressSection{
					Kubernetes: &vzalpha1.IstioKubernetesSection{},
				}
			}
			istio.Ingress.Kubernetes.Replicas = 3
			istio.Egress.Kubernetes.Replicas = 3

			vzclient, err := pkg.GetVerrazzanoClientset()
			if err != nil {
				t.Fail(fmt.Sprintf("Error getting Verrazzano client: %s", err.Error()))
				return
			}

			// This should fail with a webhook validation error
			_, err = vzclient.VerrazzanoV1alpha1().Verrazzanos(vz.Namespace).Update(context.TODO(), vz, v1.UpdateOptions{})
			Expect(err).Should(Not(BeNil()))
		})
	})
})
