// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package k8sutil_test

import (
	"fmt"
	spdyfake "github.com/verrazzano/verrazzano/pkg/k8sutil/fake"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"istio.io/api/networking/v1alpha3"
	istiov1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	"k8s.io/client-go/util/homedir"
)

const envVarHome = "HOME"
const dummyKubeConfig = "dummy-kubeconfig"
const dummyk8sHost = "http://localhost"
const appConfigName = "test"
const appConfigNamespace = "test"

func TestGetKubeConfigLocationEnvVarTestKubeconfig(t *testing.T) {
	asserts := assert.New(t)
	// Preserve previous env var value
	prevEnvVar := os.Getenv(k8sutil.EnvVarTestKubeConfig)
	randomKubeConfig := "/home/testing/somerandompath"
	// Test using environment variable
	err := os.Setenv(k8sutil.EnvVarTestKubeConfig, randomKubeConfig)
	asserts.NoError(err)
	kubeConfigLoc, err := k8sutil.GetKubeConfigLocation()
	asserts.NoError(err)
	asserts.Equal(randomKubeConfig, kubeConfigLoc)
	// Reset env variable
	err = os.Setenv(k8sutil.EnvVarTestKubeConfig, prevEnvVar)
	asserts.NoError(err)

}

func TestGetKubeConfigLocationEnvVar(t *testing.T) {
	asserts := assert.New(t)
	// Preserve previous env var value
	prevEnvVar := os.Getenv(k8sutil.EnvVarKubeConfig)
	randomKubeConfig := "/home/xyz/somerandompath"
	// Test using environment variable
	err := os.Setenv(k8sutil.EnvVarKubeConfig, randomKubeConfig)
	asserts.NoError(err)
	kubeConfigLoc, err := k8sutil.GetKubeConfigLocation()
	asserts.NoError(err)
	asserts.Equal(randomKubeConfig, kubeConfigLoc)
	// Reset env variable
	err = os.Setenv(k8sutil.EnvVarKubeConfig, prevEnvVar)
	asserts.NoError(err)

}
func TestGetKubeConfigLocationHomeDir(t *testing.T) {
	asserts := assert.New(t)
	// Preserve previous env var value
	prevEnvVar := os.Getenv(k8sutil.EnvVarKubeConfig)
	// Test without environment variable
	err := os.Setenv(k8sutil.EnvVarKubeConfig, "")
	asserts.NoError(err)
	kubeConfigLoc, err := k8sutil.GetKubeConfigLocation()
	asserts.NoError(err)
	asserts.Equal(kubeConfigLoc, homedir.HomeDir()+"/.kube/config")
	// Reset env variable
	err = os.Setenv(k8sutil.EnvVarKubeConfig, prevEnvVar)
	asserts.NoError(err)
}

func TestGetKubeConfigLocationReturnsError(t *testing.T) {
	asserts := assert.New(t)
	// Preserve previous env var value
	prevEnvVarHome := os.Getenv(envVarHome)
	prevEnvVarKubeConfig := os.Getenv(k8sutil.EnvVarKubeConfig)
	// Unset HOME environment variable
	err := os.Setenv(envVarHome, "")
	asserts.NoError(err)
	// Unset KUBECONFIG environment variable
	err = os.Setenv(k8sutil.EnvVarKubeConfig, "")
	asserts.NoError(err)
	_, err = k8sutil.GetKubeConfigLocation()
	asserts.Error(err)
	// Reset env variables
	err = os.Setenv(k8sutil.EnvVarKubeConfig, prevEnvVarKubeConfig)
	asserts.NoError(err)
	err = os.Setenv(envVarHome, prevEnvVarHome)
	asserts.NoError(err)
}

func TestGetKubeConfigReturnsError(t *testing.T) {
	asserts := assert.New(t)
	// Preserve previous env var value
	prevEnvVarHome := os.Getenv(envVarHome)
	prevEnvVarKubeConfig := os.Getenv(k8sutil.EnvVarKubeConfig)
	// Unset HOME environment variable
	err := os.Setenv(envVarHome, "")
	asserts.NoError(err)
	// Unset KUBECONFIG environment variable
	err = os.Setenv(k8sutil.EnvVarKubeConfig, "")
	asserts.NoError(err)
	_, err = k8sutil.GetKubeConfig()
	asserts.Error(err)
	// Reset env variables
	err = os.Setenv(k8sutil.EnvVarKubeConfig, prevEnvVarKubeConfig)
	asserts.NoError(err)
	err = os.Setenv(envVarHome, prevEnvVarHome)
	asserts.NoError(err)
}

func TestGetKubeConfigDummyKubeConfig(t *testing.T) {
	asserts := assert.New(t)
	// Preserve previous env var value
	prevEnvVarKubeConfig := os.Getenv(k8sutil.EnvVarKubeConfig)
	// Unset KUBECONFIG environment variable
	wd, err := os.Getwd()
	asserts.NoError(err)
	err = os.Setenv(k8sutil.EnvVarKubeConfig, fmt.Sprintf("%s/%s", wd, dummyKubeConfig))
	asserts.NoError(err)
	kubeconfig, err := k8sutil.GetKubeConfig()
	asserts.NoError(err)
	asserts.NotNil(kubeconfig)
	asserts.Equal(kubeconfig.Host, dummyk8sHost)
	// Reset env variables
	err = os.Setenv(k8sutil.EnvVarKubeConfig, prevEnvVarKubeConfig)
	asserts.NoError(err)
}

func TestGetKubernetesClientsetReturnsError(t *testing.T) {
	asserts := assert.New(t)
	// Preserve previous env var value
	prevEnvVarHome := os.Getenv(envVarHome)
	prevEnvVarKubeConfig := os.Getenv(k8sutil.EnvVarKubeConfig)
	// Unset HOME environment variable
	err := os.Setenv(envVarHome, "")
	asserts.NoError(err)
	// Unset KUBECONFIG environment variable
	err = os.Setenv(k8sutil.EnvVarKubeConfig, "")
	asserts.NoError(err)
	_, err = k8sutil.GetKubernetesClientset()
	asserts.Error(err)
	// Reset env variables
	err = os.Setenv(k8sutil.EnvVarKubeConfig, prevEnvVarKubeConfig)
	asserts.NoError(err)
	err = os.Setenv(envVarHome, prevEnvVarHome)
	asserts.NoError(err)
}

func TestGetKubernetesClientsetDummyKubeConfig(t *testing.T) {
	asserts := assert.New(t)
	// Preserve previous env var value
	prevEnvVarKubeConfig := os.Getenv(k8sutil.EnvVarKubeConfig)
	// Unset KUBECONFIG environment variable
	wd, err := os.Getwd()
	asserts.NoError(err)
	err = os.Setenv(k8sutil.EnvVarKubeConfig, fmt.Sprintf("%s/%s", wd, dummyKubeConfig))
	asserts.NoError(err)
	clientset, err := k8sutil.GetKubernetesClientset()
	asserts.NoError(err)
	asserts.NotNil(clientset)
	// Reset env variables
	err = os.Setenv(k8sutil.EnvVarKubeConfig, prevEnvVarKubeConfig)
	asserts.NoError(err)
}

func TestGetIstioClientsetReturnsError(t *testing.T) {
	asserts := assert.New(t)
	// Preserve previous env var value
	prevEnvVarHome := os.Getenv(envVarHome)
	prevEnvVarKubeConfig := os.Getenv(k8sutil.EnvVarKubeConfig)
	// Unset HOME environment variable
	err := os.Setenv(envVarHome, "")
	asserts.NoError(err)
	// Unset KUBECONFIG environment variable
	err = os.Setenv(k8sutil.EnvVarKubeConfig, "")
	asserts.NoError(err)
	_, err = k8sutil.GetIstioClientset()
	asserts.Error(err)
	// Reset env variables
	err = os.Setenv(k8sutil.EnvVarKubeConfig, prevEnvVarKubeConfig)
	asserts.NoError(err)
	err = os.Setenv(envVarHome, prevEnvVarHome)
	asserts.NoError(err)
}

func TestGetIstioClientsetDummyKubeConfig(t *testing.T) {
	asserts := assert.New(t)
	// Preserve previous env var value
	prevEnvVarKubeConfig := os.Getenv(k8sutil.EnvVarKubeConfig)
	// Unset KUBECONFIG environment variable
	wd, err := os.Getwd()
	asserts.NoError(err)
	err = os.Setenv(k8sutil.EnvVarKubeConfig, fmt.Sprintf("%s/%s", wd, dummyKubeConfig))
	asserts.NoError(err)
	istioClientSet, err := k8sutil.GetIstioClientset()
	asserts.NoError(err)
	asserts.NotNil(istioClientSet)
	// Reset env variables
	err = os.Setenv(k8sutil.EnvVarKubeConfig, prevEnvVarKubeConfig)
	asserts.NoError(err)
}

func TestGetHostnameFromGatewayReturnsErrorNoKubeconfig(t *testing.T) {
	asserts := assert.New(t)
	// Preserve previous env var value
	prevEnvVarHome := os.Getenv(envVarHome)
	prevEnvVarKubeConfig := os.Getenv(k8sutil.EnvVarKubeConfig)
	// Unset HOME environment variable
	err := os.Setenv(envVarHome, "")
	asserts.NoError(err)
	// Unset KUBECONFIG environment variable
	err = os.Setenv(k8sutil.EnvVarKubeConfig, "")
	asserts.NoError(err)
	_, err = k8sutil.GetHostnameFromGateway(appConfigNamespace, appConfigName)
	asserts.Error(err)
	// Reset env variables
	err = os.Setenv(k8sutil.EnvVarKubeConfig, prevEnvVarKubeConfig)
	asserts.NoError(err)
	err = os.Setenv(envVarHome, prevEnvVarHome)
	asserts.NoError(err)
}

func TestGetHostnameFromGatewayErrorListGateways(t *testing.T) {
	asserts := assert.New(t)
	// Preserve previous env var value
	prevEnvVarKubeConfig := os.Getenv(k8sutil.EnvVarKubeConfig)
	// Unset KUBECONFIG environment variable
	wd, err := os.Getwd()
	asserts.NoError(err)
	err = os.Setenv(k8sutil.EnvVarKubeConfig, fmt.Sprintf("%s/%s", wd, dummyKubeConfig))
	asserts.NoError(err)
	_, err = k8sutil.GetHostnameFromGateway(appConfigNamespace, appConfigName)
	asserts.Error(err)
	// Reset env variables
	err = os.Setenv(k8sutil.EnvVarKubeConfig, prevEnvVarKubeConfig)
	asserts.NoError(err)
}

func TestGetHostnameFromGatewayGatewayForAppConfigDoesNotExist(t *testing.T) {
	asserts := assert.New(t)
	// Preserve previous env var value
	prevEnvVarKubeConfig := os.Getenv(k8sutil.EnvVarKubeConfig)
	// Unset KUBECONFIG environment variable
	wd, err := os.Getwd()
	asserts.NoError(err)
	err = os.Setenv(k8sutil.EnvVarKubeConfig, fmt.Sprintf("%s/%s", wd, dummyKubeConfig))
	asserts.NoError(err)

	gateway1 := istiov1alpha3.Gateway{}
	gateway1.Name = "test1"
	gateway2 := istiov1alpha3.Gateway{}
	gateway2.Name = "test2"
	hostname, _ := k8sutil.GetHostnameFromGateway(appConfigNamespace, appConfigName, gateway1, gateway2)
	asserts.Empty(hostname)
	// Reset env variables
	err = os.Setenv(k8sutil.EnvVarKubeConfig, prevEnvVarKubeConfig)
	asserts.NoError(err)
}

func TestGetHostnameFromGatewayGatewaysForAppConfigExists(t *testing.T) {
	asserts := assert.New(t)
	serverHost := "testhost"
	// Preserve previous env var value
	prevEnvVarKubeConfig := os.Getenv(k8sutil.EnvVarKubeConfig)
	// Unset KUBECONFIG environment variable
	wd, err := os.Getwd()
	asserts.NoError(err)
	err = os.Setenv(k8sutil.EnvVarKubeConfig, fmt.Sprintf("%s/%s", wd, dummyKubeConfig))
	asserts.NoError(err)

	gateway1 := istiov1alpha3.Gateway{}
	gateway1.Name = fmt.Sprintf("%s-%s-gw", appConfigNamespace, appConfigName)
	server := &v1alpha3.Server{}
	server.Hosts = append(server.Hosts, serverHost)
	gateway1.Spec.Servers = append(gateway1.Spec.Servers, server)

	gateway2 := istiov1alpha3.Gateway{}
	gateway2.Name = "test1"
	hostname, _ := k8sutil.GetHostnameFromGateway(appConfigNamespace, appConfigName, gateway1, gateway2)
	asserts.Equal(serverHost, hostname)
	// Reset env variables
	err = os.Setenv(k8sutil.EnvVarKubeConfig, prevEnvVarKubeConfig)
	asserts.NoError(err)
}

// TestExecPod tests running a command on a remote pod
// GIVEN a pod in a cluster and a command to run on that pod
//  WHEN ExecPod is called
//  THEN ExecPod return the stdout, stderr, and a nil error
func TestExecPod(t *testing.T) {
	k8sutil.NewPodExecutor = spdyfake.NewPodExecutor
	spdyfake.PodSTDOUT = "foobar"
	cfg, client := spdyfake.NewClientsetConfig()
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "name",
		},
	}
	stdout, _, err := k8sutil.ExecPod(client, cfg, pod, "container", []string{"run", "some", "command"})
	assert.Nil(t, err)
	assert.Equal(t, spdyfake.PodSTDOUT, stdout)
}
