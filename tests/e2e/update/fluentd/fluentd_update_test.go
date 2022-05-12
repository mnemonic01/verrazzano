// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package fluentd

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	pcons "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

const (
	opensearchURL = "https://opensearch.example.com:9200"
)

var (
	t        = framework.NewTestFramework("update fluentd")
	tempuuid = uuid.NewString()[:7]
	extEsSec = "my-extsec-" + tempuuid
	wrongSec = "wrong-sec-" + tempuuid
	ociLgSec = "my-ocilog-" + tempuuid
	sysLogID = "my-sysLog-" + tempuuid
	defLogID = "my-defLog-" + tempuuid
)

var _ = t.AfterSuite(func() {
	pkg.DeleteSecret(pcons.VerrazzanoInstallNamespace, extEsSec)
	pkg.DeleteSecret(pcons.VerrazzanoInstallNamespace, wrongSec)
	m := FluentdDefaultModifier{}
	ValidateUpdate(m, "")
	ValidateDaemonset(pkg.VmiESURL, pkg.VmiESInternalSecret, "")
})

var _ = t.Describe("Update Fluentd", Label("f:platform-lcm.update"), func() {
	t.Describe("fluentd verify", Label("f:platform-lcm.fluentd-verify"), func() {
		t.It("fluentd default config", func() {
			ValidateDaemonset(pkg.VmiESURL, pkg.VmiESInternalSecret, "")
		})
	})

	t.Describe("Validate external Opensearch config", Label("f:platform-lcm.fluentd-update-validation"), func() {
		t.It("secret validation", func() {
			m := FluentdExtLogCollectorModifier{ExtLogCollectorSec: extEsSec + "missing", ExtLogCollectorURL: opensearchURL}
			ValidateUpdate(m, "must be created")
		})
	})

	t.Describe("Update external Opensearch", Label("f:platform-lcm.fluentd-external-opensearch"), func() {
		t.It("external Opensearch", func() {
			pkg.CreateCredentialsSecret(pcons.VerrazzanoInstallNamespace, extEsSec, "user", "pw", map[string]string{})
			m := FluentdExtLogCollectorModifier{ExtLogCollectorSec: extEsSec, ExtLogCollectorURL: opensearchURL}
			ValidateUpdate(m, "")
			ValidateDaemonset(opensearchURL, extEsSec, "")
		})
	})

	t.Describe("Validate OCI logging config", Label("f:platform-lcm.fluentd-update-validation"), func() {
		t.It("secret validation", func() {
			m := FluentdOciLoggingModifier{APISec: wrongSec}
			ValidateUpdate(m, "must be created")
			pkg.CreateCredentialsSecret(pcons.VerrazzanoInstallNamespace, wrongSec, "api", "pw", map[string]string{})
			ValidateUpdate(m, "Did not find OCI configuration")
		})
	})

	t.Describe("Update OCI logging", Label("f:platform-lcm.fluentd-oci-logging"), func() {
		t.It(" OCI logging", func() {
			createOciLoggingSecret(ociLgSec)
			m := FluentdOciLoggingModifier{APISec: ociLgSec, SystemLog: sysLogID, DefaultLog: defLogID}
			ValidateUpdate(m, "")
			ValidateDaemonset("", "", ociLgSec)
			ValidateConfigMap(sysLogID, defLogID)
		})
	})
})
