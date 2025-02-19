#!/bin/bash
#
# Copyright (c) 2020, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
INSTALL_DIR=$SCRIPT_DIR/../../install
UNINSTALL_DIR=$SCRIPT_DIR/..

. $INSTALL_DIR/common.sh
. $INSTALL_DIR/config.sh
. $UNINSTALL_DIR/uninstall-utils.sh

set -o pipefail

TMP_DIR=$(mktemp -d)
trap 'rc=$?; rm -rf ${TMP_DIR} || true; _logging_exit_handler $rc' EXIT

CONFIG_DIR=$INSTALL_DIR/config

function uninstall_istio() {

  # Make attempt at calling helm to cleanup.  Do you fail on error, we cleanup manually below
  helm uninstall -n istio-system istiod || true
  helm uninstall -n istio-system istio-egress || true
  helm uninstall -n istio-system istio-ingress || true
  helm uninstall -n istio-system istiod || true

  # Make attempt to delete using istioctl
  istioctl x uninstall --purge -y || true

  # delete webhook configurations
  log "Removing Istio Webhook Configurations"
  kubectl delete MutatingWebhookConfiguration istio-sidecar-injector --ignore-not-found=true || err_return $? "Could not delete MutatingWebhookConfiguration from Istio" || return $?
  kubectl delete ValidatingWebhookConfiguration istiod-istio-system  --ignore-not-found=true || err_return $? "Could not delete ValidatingWebhookConfiguration from Istio" || return $?

  # delete istio api services
  log "Deleting Istio API Services"
  delete_k8s_resources apiservice ":metadata.name" "Could not delete APIServices from Istio" '/istio.io/ {print $1}' \
    || return $? # return on pipefail

  # delete istio cluster role bindings
  log "Deleting Istio Cluster Role Bindings"
  delete_k8s_resources clusterrolebinding ":metadata.name" "Could not delete ClusterRoleBindings from Istio" '/istio-system|istio-multi/ {print $1}' \
    || return $? # return on pipefail

  # delete istio cluster roles
  log "Deleting Istio Cluster Roles"
  delete_k8s_resources clusterrole ":metadata.name" "Could not delete ClusterRoles from Istio" '/istio-system|istio-reader|istiocoredns/ {print $1}' \
    || return $? # return on pipefail

      # delete istio crds
  log "Deleting Istio Custom Resource Definitions"
  delete_k8s_resources crd ":metadata.name" "Could not delete CustomResourceDefinition from Istio" '/istio.io/ {print $1}' \
    || return $? # return on pipefail

}

function delete_secrets() {
  # Delete istio.default in all namespaces
  log "Retrieving istio secrets for deletion"
  kubectl delete secret istio.default --ignore-not-found=true || err_return $? "Could not delete secret from Istio in namespace default" || return $?
  kubectl delete secret istio.default -n kube-public --ignore-not-found=true || err_return $? "Could not delete secret from Istio in namespace kube-public" || return $?
  kubectl delete secret istio.default -n kube-node-lease --ignore-not-found=true || err_return $? "Could not delete secret from Istio in namespace kube-node-lease" || return $?

  # delete secrets left over in kube-system
  delete_k8s_resources secrets ":metadata.name,:metadata.annotations" "Could not delete secrets from Istio in namespace kube-system" '/istio./ {print $1}' "kube-system" \
    || return $? # return on pipefail
}

function delete_istio_namepsace() {
  log "Deleting istio-system finalizers"
  patch_k8s_resources namespaces ":metadata.name" "Could not remove finalizers from namespace istio-system" '/istio-system/ {print $1}' '{"metadata":{"finalizers":null}}' \
    || return $? # return on pipefail

  log "Deleting istio-system namespace"
  kubectl delete namespace istio-system --ignore-not-found=true || err_return $? "Could not delete namespace istio-system" || return $?
}

function delete_external_dns() {
  log "Deleting external-dns"

  # delete all ExternalDNS ingresses before deleting ExternalDNS
  delete_k8s_resources ingress ":metadata.name,:metadata.annotations" "Could not delete Ingresses managed by ExternalDNS" '/external-dns/ {print $1}' \

  helm ls -n cert-manager \
    | awk '/external-dns/ {print $1}' \
    | xargsr helm uninstall -n cert-manager \
    || err_return $? "Could not delete external-dns from helm" || return $? # return on pipefail

  # delete clusterrole and clusterrolebinding
  log "Deleting ClusterRoles and ClusterRoleBindings for external-dns"
  kubectl delete clusterrole external-dns --ignore-not-found=true || err_return $? "Could not delete ClusterRole external-dns" || return $?
  kubectl delete clusterrolebinding external-dns --ignore-not-found=true || err_return $? "Could not delete ClusterRoleBinding external-dns" || return $?
}

function finalize() {
  # Removing possible reference to verrazzano in clusterroles and clusterrolebindings
  log "Removing Verrazzano ClusterRoles and ClusterRoleBindings"
  delete_k8s_resources clusterrolebinding ":metadata.name" "Could not delete ClusterRoleBindings" '/verrazzano/ && ! /verrazzano-platform-operator/ && ! /verrazzano-install/ && ! /verrazzano-managed-cluster/' \
    || return $? # return on pipefail

  delete_k8s_resources clusterrole ":metadata.name" "Could not delete ClusterRoles" '/verrazzano/ && ! /verrazzano-managed-cluster/' \
    || return $? # return on pipefail

  log "Deleting configmap istio-ca-root-cert from all namespaces"
  IFS=$'\n' read -r -d '' -a namespaces < <( kubectl get namespaces --no-headers -o custom-columns=":metadata.name" && printf '\0' )
  for ns in "${namespaces[@]}" ; do
    if kubectl get configmap istio-ca-root-cert > /dev/null 2>&1 ; then
      kubectl delete configmap istio-ca-root-cert --namespace ${ns} --ignore-not-found=true
    fi
  done

  # Grab all leftover Helm repos and delete resources
  log "Deleting Helm repos"
  local helm_ls
  helm_ls=$(helm repo ls >/dev/null 2>&1)
  if [ $? -eq 0 ]; then
    echo "$helm_ls" \
      | awk '/istio.io|stable|jetstack|rancher-stable|codecentric/ {print $1}' \
      | xargsr -I name helm repo remove name \
      || err_return $? "Could delete Helm Repos" || return $? # return on pipefail
  fi

}

action "Deleting Istio Components" uninstall_istio || exit 1
action "Deleting Istio Secrets" delete_secrets || exit 1
action "Deleting Istio Namespace" delete_istio_namepsace || exit 1
action "Deleting External DNS Components" delete_external_dns || exit 1
action "Finalizing Uninstall" finalize || exit 1
