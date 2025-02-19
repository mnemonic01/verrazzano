# Copyright (C) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

include ../../make/quality.mk
include ../../make/generate.mk
include ../../make/retry.mk

SCRIPT_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))/../build
TOOLS_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))/../../tools

NAME:=verrazzano-weblogic-image-tool
REPO_NAME:=verrazzano-weblogic-image-tool

CREATE_LATEST_TAG=0

ifndef DOCKER_IMAGE_FULLNAME
DOCKER_IMAGE_NAME ?= ${NAME}-dev
DOCKER_IMAGE_FULLNAME=${DOCKER_IMAGE_NAME}
ifeq ($(MAKECMDGOALS),$(filter $(MAKECMDGOALS),docker-push push-tag))
	ifndef DOCKER_REPO
		$(error DOCKER_REPO must be defined as the name of the Docker repository where image will be pushed)
	endif
	ifndef DOCKER_NAMESPACE
		$(error DOCKER_NAMESPACE must be defined as the name of the Docker namespace where image will be pushed)
	endif
endif
ifdef DOCKER_NAMESPACE
DOCKER_IMAGE_FULLNAME := ${DOCKER_NAMESPACE}/${DOCKER_IMAGE_FULLNAME}
endif
ifdef DOCKER_REPO
DOCKER_IMAGE_FULLNAME := ${DOCKER_REPO}/${DOCKER_IMAGE_FULLNAME}
endif
endif

DOCKER_IMAGE_TAG ?= local-$(shell git rev-parse --short HEAD)

OPERATOR_VERSION = ${DOCKER_IMAGE_TAG}
ifdef RELEASE_VERSION
	OPERATOR_VERSION = ${RELEASE_VERSION}
endif
ifndef RELEASE_BRANCH
	RELEASE_BRANCH=$(shell git rev-parse --abbrev-ref HEAD)
endif

DIST_DIR:=dist
K8S_NAMESPACE:=default
WATCH_NAMESPACE:=
EXTRA_PARAMS=
INTEG_RUN_ID=
ENV_NAME=verrazzano-weblogic-image-tool

CLUSTER_DUMP_LOCATION ?= weblogic-image-tool-integ-cluster-dump

# for retrieving installers from oci bucket
OCI_CLI_REGION ?= us-phoenix-1
BUCKET_NAME ?= build-shared-files
JDK8_BUNDLE ?= jdk-8u281-linux-x64.tar.gz
WEBLOGIC_BUNDLE ?= fmw_12.2.1.4.0_wls.jar

.PHONY: get-installers
get-installers:
	cd ./installers; \
	export OCI_CLI_REGION=${OCI_CLI_REGION}; \
	(test -f "weblogic-deploy.zip" || wget https://github.com/oracle/weblogic-deploy-tooling/releases/download/release-1.9.15/weblogic-deploy.zip); \
	(test -f ${JDK8_BUNDLE} || oci os object get -bn ${BUCKET_NAME} --file ${JDK8_BUNDLE} --name ${JDK8_BUNDLE});

.PHONY: docker-clean
docker-clean:
	rm -rf ${DIST_DIR}

.PHONY: docker-build
docker-build: get-installers
	# the TPL file needs to be copied into this dir so it is in the docker build context
	cp ../../THIRD_PARTY_LICENSES.txt .
	docker build --pull \
		-t ${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG} .

.PHONY: docker-push
docker-push: docker-build
	docker tag ${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG} ${DOCKER_IMAGE_FULLNAME}:${DOCKER_IMAGE_TAG}
	$(call retry_docker_push,${DOCKER_IMAGE_FULLNAME}:${DOCKER_IMAGE_TAG})

ifeq ($(CREATE_LATEST_TAG), "1")
	docker tag ${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG} ${DOCKER_IMAGE_FULLNAME}:latest;
	$(call retry_docker_push,${DOCKER_IMAGE_FULLNAME}:latest);
endif
