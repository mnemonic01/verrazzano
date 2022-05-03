// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package keycloak

import (
	"fmt"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/pkg/test/framework/metrics"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

var waitTimeout = 1 * time.Minute
var pollingInterval = 30 * time.Second
var shortPollingInterval = 10 * time.Second

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


var _ = t.Describe("Verify users exist in Keycloak", Label("f:platform-lcm.install"), func() {
	isManagedClusterProfile := pkg.IsManagedClusterProfile()
	t.It("Verifying user in master realm", func() {
		if !isManagedClusterProfile {
			Eventually(verifyUserExists("master", os.Getenv(pkg.TestKeycloakMasterUserid)), waitTimeout, pollingInterval).Should(BeTrue())
		}
	})
	t.It("Verifying user in verrazzano-system realm", func() {
		if !isManagedClusterProfile {
			Eventually(verifyUserExists("verrazzano-system", os.Getenv(pkg.TestKeycloakVzUserid)), waitTimeout, pollingInterval).Should(BeTrue())
		}
	})
})

func verifyUserExists(realm, userID string) bool {
	kc, err := pkg.NewKeycloakAdminRESTClient()
	if err != nil {
		t.Logs.Error(fmt.Printf("Failed to create Keycloak REST client: %v\n", err))
		return false
	}

	exists, err := kc.VerifyUserExists(realm, userID)
	if err != nil {
		t.Logs.Info(fmt.Printf("Failed to verify user %s/%s: %v\n", realm, userID, err))
		return false
	}
	return exists
}