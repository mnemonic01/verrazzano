// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package os

import (
	"go.uber.org/zap"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"

	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
)

// CreateTempFile creates a temp file from a filename pattern and data
func CreateTempFile(log vzlog.VerrazzanoLogger, filenamePattern string, data []byte) (*os.File, error) {
	var tmpFile *os.File
	tmpFile, err := ioutil.TempFile(os.TempDir(), filenamePattern)
	if err != nil {
		return tmpFile, log.ErrorfNewErr("Failed to create temporary file: %v", err)
	}

	if _, err = tmpFile.Write(data); err != nil {
		return tmpFile, log.ErrorfNewErr("Failed to write to temporary file: %v", err)
	}

	// Close the file
	if err := tmpFile.Close(); err != nil {
		return tmpFile, log.ErrorfNewErr("Failed to close temporary file: %v", err)
	}
	return tmpFile, nil
}

func RemoveTempFiles(log *zap.SugaredLogger, regexPattern string) error {
	files, err := ioutil.ReadDir(os.TempDir())
	if err != nil {
		log.Errorf("Unable to read temp directory: %v", err)
		return err
	}
	matcher, err := regexp.Compile(regexPattern)
	if err != nil {
		log.Errorf("Unable to compile regex pattern: %s: %v", regexPattern, err)
		return err
	}
	for _, file := range files {
		if !file.IsDir() && matcher.Match([]byte(file.Name())) {
			fullPath := filepath.Join(os.TempDir(), file.Name())
			log.Debugf("Deleting temp file %s", fullPath)
			if err := os.Remove(fullPath); err != nil {
				log.Errorf("Error deleting temp file %s: %v", fullPath, err)
				return err
			}
		}
	}
	return nil
}
