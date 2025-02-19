#!/bin/bash
#
# Copyright (c) 2020, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

set -e

TMP_DIR=$(mktemp -d)
trap 'rc=$?; rm -rf ${TMP_DIR} || true' EXIT

OCI_CONFIG_SECRET_NAME=oci
VERRAZZANO_INSTALL_NS=verrazzano-install

if [ "${OCI_DNS_AUTH}" != "instance_principal" ]; then
  # perform these validations when instance principal is not used
  # Validate expected environment variables exist
  if [ -z "${OCI_CLI_REGION}" ]; then
    echo "OCI_REGION environment variable must be set"
    exit 1
  fi
  if [ -z "${OCI_CLI_TENANCY}" ]; then
    echo "OCI_TENANCY_OCID environment variable must be set"
    exit 1
  fi
  if [ -z "${OCI_CLI_USER}" ]; then
    echo "OCI_USER_OCID environment variable must be set"
    exit 1
  fi
  if [ -z "${OCI_CLI_FINGERPRINT}" ]; then
    echo "OCI_FINGERPRINT environment variable must be set"
    exit 1
  fi
  if [ -z "${OCI_CLI_KEY_FILE}" ]; then
    echo "OCI_PRIVATE_KEY_FILE environment variable must be set"
    exit 1
  fi
fi


OUTPUT_FILE=$TMP_DIR/oci-config.yaml
echo "OCI_DNS_AUTH value = " ${OCI_DNS_AUTH}
# Generate the yaml file
if [ "${OCI_DNS_AUTH}" == "instance_principal" ]; then
  echo "auth:" > $OUTPUT_FILE
  echo "  authtype: instance_principal" >> $OUTPUT_FILE
else
  echo "auth:" > $OUTPUT_FILE
  echo "  region: ${OCI_CLI_REGION}" >> $OUTPUT_FILE
  echo "  tenancy: ${OCI_CLI_TENANCY}" >> $OUTPUT_FILE
  echo "  user: ${OCI_CLI_USER}" >> $OUTPUT_FILE
  echo "  key: |" >> $OUTPUT_FILE
  cat ${OCI_CLI_KEY_FILE} | sed 's/^/    /' >> $OUTPUT_FILE
  echo "  fingerprint: ${OCI_CLI_FINGERPRINT}" >> $OUTPUT_FILE
  if [[ ! -z "${OCI_PRIVATE_KEY_PASSPHRASE}" ]]; then
    echo "  passphrase: ${OCI_PRIVATE_KEY_PASSPHRASE}" >> $OUTPUT_FILE
  fi
fi
# create the secret in verrazzano-install namespace
if ! kubectl get namespace ${VERRAZZANO_INSTALL_NS} ; then
  echo "The namespace ${VERRAZZANO_INSTALL_NS} doesn't exit. Please install the Verrazzano platform operator and try again."
  exit 1
fi
kubectl create secret generic $OCI_CONFIG_SECRET_NAME -n $VERRAZZANO_INSTALL_NS --from-file=$OUTPUT_FILE