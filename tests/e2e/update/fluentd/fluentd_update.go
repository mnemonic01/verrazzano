// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package fluentd

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"
	"time"

	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	pcons "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/update"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	fluentdName     = "fluentd"
	cacerts         = "cacerts"
	oneMinute       = 1 * time.Minute
	fiveMinutes     = 5 * time.Minute
	pollingInterval = 5 * time.Second
)

//type FluentdExtLogCollectorModifier struct {
//	ExtLogCollectorSec string
//	ExtLogCollectorURL string
//}
//
//type FluentdOciLoggingModifier struct {
//	APISec     string
//	DefaultLog string
//	SystemLog  string
//}

type FluentdModifier struct {
	Component vzapi.FluentdComponent
}

//func (u FluentdExtLogCollectorModifier) ModifyCR(cr *vzapi.Verrazzano) {
//	cr.Spec.Components.Fluentd = &vzapi.FluentdComponent{
//		ElasticsearchSecret: u.ExtLogCollectorSec,
//		ElasticsearchURL:    u.ExtLogCollectorURL,
//	}
//}
//
//func (u FluentdOciLoggingModifier) ModifyCR(cr *vzapi.Verrazzano) {
//	cr.Spec.Components.Fluentd = &vzapi.FluentdComponent{
//		OCI: &vzapi.OciLoggingConfiguration{
//			DefaultAppLogID: u.DefaultLog,
//			SystemLogID:     u.SystemLog,
//			APISecret:       u.APISec,
//		},
//	}
//}

func (u FluentdModifier) ModifyCR(cr *vzapi.Verrazzano) {
	fmt.Printf("~~~~ using \n%v\n \n%v\n", u.Component, &u.Component)
	cr.Spec.Components.Fluentd = &u.Component
}

func ValidateUpdate(m update.CRModifier, expectedError string) {
	gomega.Expect(func() bool {
		err := update.UpdateCR(m)
		if err != nil {
			pkg.Log(pkg.Info, fmt.Sprintf("Update error: %v", err))
		}
		fmt.Printf("Update error: %v\n", err)
		if expectedError == "" {
			return err == nil
		}
		if err == nil {
			return false
		}
		return strings.Contains(err.Error(), expectedError)
	}()).Should(gomega.BeTrue(), fmt.Sprintf("expected error %v", expectedError))
}

func ValidateDaemonset(osURL, osSec, apiSec string) {
	gomega.Eventually(func() bool {
		ds, err := pkg.GetDaemonSet(constants.VerrazzanoSystemNamespace, fluentdName)
		if err != nil {
			return false
		}
		validateCacertsVolume(ds)
		for _, cntr := range ds.Spec.Template.Spec.Containers {
			if !validateFluentdContainer(cntr, osURL, osSec) {
				return false
			}
		}
		if !validateOciSec(ds, apiSec) {
			return false
		}
		return ds.Status.NumberReady > 0
	}, fiveMinutes, pollingInterval).Should(gomega.BeTrue(), fmt.Sprintf("DaemonSet %s is not ready", osURL))
}

func validateFluentdContainer(cntr corev1.Container, osURL, osSec string) bool {
	if cntr.Name == fluentdName {
		if !validateCacerts(cntr) {
			return false
		}
		if !validateEsURL(cntr, osURL) {
			return false
		}
		if !validateEsSec(cntr, osSec) {
			return false
		}
	}
	return true
}

func validateEsURL(cntr corev1.Container, esURL string) bool {
	if esURL != "" {
		env := findEnv(cntr, "ELASTICSEARCH_URL")
		pkg.Log(pkg.Info, fmt.Sprintf("Found env: %v", env))
		if env == nil || env.Value != esURL {
			return false
		}
	}
	return true
}

func validateEsSec(cntr corev1.Container, esSec string) bool {
	if esSec != "" {
		env := findEnv(cntr, "ELASTICSEARCH_USER")
		pkg.Log(pkg.Info, fmt.Sprintf("Found env: %v", env))
		if env == nil || env.ValueFrom.SecretKeyRef.Name != esSec {
			return false
		}
	}
	return true
}

func validateCacerts(cntr corev1.Container) bool {
	vm := findVolMount(cntr, cacerts)
	pkg.Log(pkg.Info, fmt.Sprintf("Found %s VolumeMount: %v", cacerts, vm))
	return vm != nil && vm.MountPath == "/fluentd/cacerts"
}

func validateCacertsVolume(ds *appsv1.DaemonSet) bool {
	vol := findVol(ds, cacerts)
	pkg.Log(pkg.Info, fmt.Sprintf("Found %s Volume: %v", cacerts, vol))
	return vol != nil
}

func validateOciSec(ds *appsv1.DaemonSet, apiSec string) bool {
	if apiSec != "" {
		vol := findVol(ds, "oci-secret-volume")
		pkg.Log(pkg.Info, fmt.Sprintf("Found oci-secret-volume: %v", vol))
		if vol == nil || vol.VolumeSource.Secret.SecretName != apiSec {
			return false
		}
	}
	return true
}

func findVol(ds *appsv1.DaemonSet, name string) *corev1.Volume {
	for _, vol := range ds.Spec.Template.Spec.Volumes {
		if vol.Name == name {
			return &vol
		}
	}
	return nil
}

func findEnv(c corev1.Container, name string) *corev1.EnvVar {
	for _, env := range c.Env {
		if env.Name == name {
			return &env
		}
	}
	return nil
}

func findVolMount(c corev1.Container, name string) *corev1.VolumeMount {
	for _, vm := range c.VolumeMounts {
		if vm.Name == name {
			return &vm
		}
	}
	return nil
}

func createOciLoggingSecret(name string) (*corev1.Secret, error) {
	// Get the kubernetes clientset
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Failed to get clientset with error: %v", err))
		return nil, err
	}
	key, _ := rsa.GenerateKey(rand.Reader, 4096)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: pcons.VerrazzanoInstallNamespace,
		},
		Data: map[string][]byte{
			"config": []byte(`
[DEFAULT]
user=ocid1.user.oc1..testuser
tenancy=ocid1.tenancy.oc1..testtenancy
region=us-ashburn-1
fingerprint=test:fingerprint
key_file=/root/.oci/key
`),
			// Encode private key to PKCS#1 ASN.1 PEM.
			"key": pem.EncodeToMemory(
				&pem.Block{
					Type:  "RSA PRIVATE KEY",
					Bytes: x509.MarshalPKCS1PrivateKey(key),
				},
			),
		},
	}
	scr, err := clientset.CoreV1().Secrets(pcons.VerrazzanoInstallNamespace).Create(context.TODO(), secret, metav1.CreateOptions{})
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("CreateOciLoggingSecret %v error: %v", name, err))
	}
	return scr, err
}

func ValidateConfigMap(sysLog, defLog string) {
	cmName := fluentdName + "-config"
	gomega.Eventually(func() bool {
		cm, err := pkg.GetConfigMap(cmName, constants.VerrazzanoSystemNamespace)
		pkg.Log(pkg.Info, fmt.Sprintf("Found ConfigMap: %v %v", cmName, err))
		if err != nil {
			return false
		}
		if sysLog != "" {
			entry := "oci-logging-system.conf"
			conf, ok := cm.Data[entry]
			pkg.Log(pkg.Info, fmt.Sprintf("Found sysLog %v in ConfigMap: %v", entry, ok))
			if !ok || !strings.Contains(conf, sysLog) {
				return false
			}
		}
		if defLog != "" {
			entry := "oci-logging-default-app.conf"
			conf, ok := cm.Data[entry]
			pkg.Log(pkg.Info, fmt.Sprintf("Found defLog %v in ConfigMap: %v", entry, ok))
			if !ok || !strings.Contains(conf, defLog) {
				return false
			}
		}
		return true
	}, oneMinute, pollingInterval).Should(gomega.BeTrue(), fmt.Sprintf("ConfigMap %s is not ready", cmName))
}
