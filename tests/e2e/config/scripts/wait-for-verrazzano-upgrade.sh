#!/usr/bin/env bash
#
# Copyright (c) 2020, 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

resourceName=${1:-my-verrazzano}
resPath=verrazzano/${resourceName}

if ! kubectl get ${resPath}; then
  echo "Verrazzano resource ${resPath} not found"
  exit 1
fi

echo "Starting wait for upgrade operation at $(date)"

SECONDS=0
retval_success=1
retval_failed=1
while [[ $retval_success -ne 0 ]] && [[ $retval_failed -ne 0 ]]; do
  sleep 5
  output=$(kubectl wait --for=condition=UpgradeFailed ${resPath} --timeout=0 2>&1)
  retval_failed=$?
  output=$(kubectl wait --for=condition=UpgradeComplete ${resPath} --timeout=0 2>&1)
  retval_success=$?
done

if [ $retval_failed -eq 0 ]; then
    echo "Upgrade Failed"
    exit 1
fi

echo "Upgrade completed successfully at $(date).  Wait time: $SECONDS seconds"
