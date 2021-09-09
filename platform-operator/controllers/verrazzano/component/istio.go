// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package component

import (
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/types"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

const istioGlobalHubKey = "global.hub"

// appendIstioOverrides appends the Keycloak theme for the key keycloak.extraInitContainers.
// A go template is used to replace the image in the init container spec.
func appendIstioOverrides(_ *zap.SugaredLogger, releaseName string, _ string, _ string, kvs []keyValue) ([]keyValue, error) {
	// Create a Bom and get the key value overrides
	bom, err := NewBom(DefaultBomFilePath())
	if err != nil {
		return nil, err
	}

	// Get the istio component
	sc, err := bom.GetSubcomponent(releaseName)
	if err != nil {
		return nil, err
	}

	registry := bom.resolveRegistry(sc)
	repo := bom.resolveRepo(sc)

	// Override the global.hub if either of the 2 env vars were defined
	if registry != bom.bomDoc.Registry || repo != sc.Repository {
		// Return a new key:value pair with the rendered value
		kvs = append(kvs, keyValue{
			key:   istioGlobalHubKey,
			value: registry + "/" + repo,
		})
	}

	return kvs, nil
}

// istiodReadyCheck Determines if istiod is up and has a minimum number of available replicas
func istiodReadyCheck(log *zap.SugaredLogger, client clipkg.Client, _ string, namespace string) bool {
	deployments := []types.NamespacedName{
		{Name: "istiod", Namespace: namespace},
	}
	return status.DeploymentsReady(log, client, deployments, 1)
}
