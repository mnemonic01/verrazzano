// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricstrait

import (
	"strings"

	"regexp"

	"github.com/Jeffail/gabs/v2"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

// updateStringMap updates a string key value pair in a map.
// strMap is the map to be updated.  It may be nil.
// key is the key to add to the map
// value is the value to add to the map
// Returns the provided or a new map if strMap was nil
func updateStringMap(strMap map[string]string, key string, value string) map[string]string {
	if strMap == nil {
		strMap = map[string]string{}
	}
	strMap[key] = value
	return strMap
}

// copyStringMapEntries copies key value pairs from one map to another.
// target is the map key value pairs are copied into
// source is the map key value pairs are copied from
// keys are a list of keys to copy from the source to the target map
// Returns the target map or a new map if the target was nil
func copyStringMapEntries(target map[string]string, source map[string]string, keys ...string) map[string]string {
	if target == nil {
		target = map[string]string{}
	}
	for _, key := range keys {
		value, found := source[key]
		if found {
			target[key] = value
		}
	}
	return target
}

// parseYAMLString parses a string into a internal representation.
// s is the YAML formatted string to parse.
// Returns an unstructured representation of the input YAML string.
// Returns and error if parsing fails.
func parseYAMLString(s string) (*gabs.Container, error) {
	prometheusJSON, _ := yaml.YAMLToJSON([]byte(s))
	return gabs.ParseJSON(prometheusJSON)
}

// writeYAMLString writes unstructured data to a YAML formatted string.
// c is the unstructured representation
// Returns a YAML format string version of the input
// Returns an error if the unstructured cannot be converted to a YAML string.
func writeYAMLString(c *gabs.Container) (string, error) {
	bytes, err := yaml.JSONToYAML(c.Bytes())
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// getClusterNameFromObjectMetaOrDefault extracts the customer name from object metadata.
// meta is the object metadata to extract the cluster from.
// Returns the cluster name or "default" if the cluster name is the empty string.
func getClusterNameFromObjectMetaOrDefault(meta metav1.ObjectMeta) string {
	name := meta.ClusterName
	if name == "" {
		return "default"
	}
	return name
}

// getNamespaceFromObjectMetaOrDefault extracts the namespace name from object metadata.
// meta is the object metadata to extract the namespace name from.
// Returns the namespace name of "default" if the namespace is the empty string.
func getNamespaceFromObjectMetaOrDefault(meta metav1.ObjectMeta) string {
	name := meta.Namespace
	if name == "" {
		return "default"
	}
	return name
}

// mergeTemplateWithContext merges a map of string into a string template.
// template is the string to merge the values from the context map into.
// context is a map of string to be merged into the template.
// Returns a string with all values from the context merged into the template.
func mergeTemplateWithContext(template string, context map[string]string) string {
	for key, value := range context {
		template = strings.ReplaceAll(template, key, value)
	}
	return template
}

// GetSupportedWorkloadType returns workload type corresponding to input API version and kind
// that is supported by MetricsTrait.
func GetSupportedWorkloadType(apiVerKind string) string {
	// Match any version of Group=weblogic.oracle and Kind=Domain
	if matched, _ := regexp.MatchString("^weblogic.oracle/.*\\.Domain$", apiVerKind); matched {
		return constants.WorkloadTypeWeblogic
	}
	// Match any version of Group=coherence.oracle and Kind=Coherence
	if matched, _ := regexp.MatchString("^coherence.oracle.com/.*\\.Coherence$", apiVerKind); matched {
		return constants.WorkloadTypeCoherence
	}

	// Match any version of Group=coherence.oracle and Kind=VerrazzanoHelidonWorkload or
	// In the case of Helidon, the workload isn't currently being unwrapped
	if matched, _ := regexp.MatchString("^oam.verrazzano.io/.*\\.VerrazzanoHelidonWorkload$", apiVerKind); matched {
		return constants.WorkloadTypeGeneric
	}

	// Match any version of Group=core.oam.dev and Kind=ContainerizedWorkload
	if matched, _ := regexp.MatchString("^core.oam.dev/.*\\.ContainerizedWorkload$", apiVerKind); matched {
		return constants.WorkloadTypeGeneric
	}

	// Match any version of Group=apps and Kind=Deployment
	if matched, _ := regexp.MatchString("^apps/.*\\.Deployment$", apiVerKind); matched {
		return constants.WorkloadTypeGeneric
	}

	return ""
}
