// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch_test

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/verrazzano-backup/lib/constants"
	"github.com/verrazzano/verrazzano/verrazzano-backup/lib/klog"
	"github.com/verrazzano/verrazzano/verrazzano-backup/lib/opensearch"
	"github.com/verrazzano/verrazzano/verrazzano-backup/lib/types"
	"go.uber.org/zap"
	"os"
	"strings"
	"testing"
)

func init() {
	os.Setenv(constants.DevKey, constants.TruthString)
}
func logHelper() (*zap.SugaredLogger, string) {
	file, err := os.CreateTemp(os.TempDir(), fmt.Sprintf("verrazzano-%s-hook-*.log", strings.ToLower("TEST")))
	if err != nil {
		fmt.Printf("Unable to create temp file")
		os.Exit(1)
	}
	defer file.Close()
	log, _ := klog.Logger(file.Name())
	return log, file.Name()
}

// TestEnsureOpenSearchIsReachable tests the EnsureOpenSearchIsReachable method for the following use case.
// GIVEN opensearch object
// WHEN invoked with opensearch URL
// THEN verifies whether opensearch is reachable or not
func TestEnsureOpenSearchIsReachable(t *testing.T) {
	log, f := logHelper()
	defer os.Remove(f)
	var c types.ConnectionData
	c.BackupName = "mango"
	o := opensearch.Opensearch(&opensearch.OpensearchImpl{})
	ok := o.EnsureOpenSearchIsReachable(constants.OpenSearchURL, &c, log)
	assert.Nil(t, ok)
	assert.Equal(t, false, false)
}

// TestRegisterSnapshotRepository tests the RegisterSnapshotRepository method for the following use case.
// GIVEN opensearch object
// WHEN invoked with snapshot data and creds
// THEN registers a repository to object store
func TestRegisterSnapshotRepository(t *testing.T) {
	log, f := logHelper()
	defer os.Remove(f)
	var objsecret types.ObjectStoreSecret
	objsecret.SecretName = "alpha"
	objsecret.SecretKey = "cloud"
	objsecret.ObjectAccessKey = "alphalapha"
	objsecret.ObjectSecretKey = "betabetabeta"
	var sdat types.ConnectionData
	sdat.Secret = objsecret
	sdat.BackupName = "mango"
	sdat.RegionName = "region"
	sdat.Endpoint = constants.OpenSearchURL

	o := opensearch.Opensearch(&opensearch.OpensearchImpl{})
	err := o.RegisterSnapshotRepository(&sdat, log)
	assert.NotNil(t, err)
}

// TestTriggerSnapshot tests the TriggerSnapshot method for the following use case.
// GIVEN opensearch object
// WHEN invoked with snapshot name
// THEN creates a snaphot in object store
func TestTriggerSnapshot(t *testing.T) {
	log, f := logHelper()
	defer os.Remove(f)

	o := opensearch.Opensearch(&opensearch.OpensearchImpl{})
	var c types.ConnectionData
	c.BackupName = "mango"
	err := o.TriggerSnapshot(&c, log)
	assert.NotNil(t, err)
}

// TestCheckSnapshotProgress tests the CheckSnapshotProgress method for the following use case.
// GIVEN opensearch object
// WHEN invoked with snapshot name
// THEN tracks snapshot progress towards completion
func TestCheckSnapshotProgress(t *testing.T) {
	log, f := logHelper()
	defer os.Remove(f)

	o := opensearch.Opensearch(&opensearch.OpensearchImpl{})
	var c types.ConnectionData
	c.BackupName = "mango"
	err := o.CheckSnapshotProgress(&c, log)
	assert.Nil(t, err)
}

// TestDeleteDataStreams tests the DeleteData method for the following use case.
// GIVEN opensearch object
// WHEN invoked with logger
// THEN deletes data from Opensearch cluster
func TestDeleteDataStreams(t *testing.T) {
	log, f := logHelper()
	defer os.Remove(f)

	o := opensearch.Opensearch(&opensearch.OpensearchImpl{})
	err := o.DeleteData(log)
	assert.NotNil(t, err)
}

// TestTriggerSnapshot tests the TriggerRestore method for the following use case.
// GIVEN opensearch object
// WHEN invoked with snapshot name
// THEN creates a restore from object store from given snapshot name
func TestTriggerRestore(t *testing.T) {
	log, f := logHelper()
	defer os.Remove(f)

	o := opensearch.Opensearch(&opensearch.OpensearchImpl{})
	var c types.ConnectionData
	c.BackupName = "mango"
	err := o.TriggerRestore(&c, log)
	assert.NotNil(t, err)
}
