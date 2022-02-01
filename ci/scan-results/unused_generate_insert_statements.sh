#!/bin/bash
#
# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

# This script can be used to generate insert statements from the CSV file data
# if LOAD DATA INFILE is deemed unsuitable for use in the JenkinsfileDB loading script.
if [ $# -ne 1 ] ; then
  echo "Usage: $0 <scan results csv file>"
  exit 1
fi

CSV_FILE=$1

while IFS="," read -r commit branch release scantime jobnum scanner vulnid sev target; do
  echo """
      INSERT INTO CONSOLIDATED_SCAN_RESULT(
        VERRAZZANO_COMMIT_HASH,BRANCH_NAME,RELEASE_TAG,SCAN_TIME,JOB_NUMBER,SCANNER_NAME,VULNERABILITY_ID,SEVERITY,SCAN_TARGET)
      VALUES(
        '$commit','$branch','$release',STR_TO_DATE('$scantime','%Y%m%d%H%i%s'),$jobnum,'$scanner','$vulnid','$sev','$target'
      );
  """
done < <(cat $CSV_FILE)
