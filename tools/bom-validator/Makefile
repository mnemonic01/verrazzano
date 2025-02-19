# Copyright (C) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

include ../../make/quality.mk
include ../../make/generate.mk
include ../../make/retry.mk

MAKEFILE_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))

NAME:=bom-validator
REPO_NAME:=bom-validator

CREATE_LATEST_TAG=0

ifndef DOCKER_IMAGE_FULLNAME
DOCKER_IMAGE_NAME ?= ${NAME}-dev
DOCKER_IMAGE_FULLNAME=${DOCKER_IMAGE_NAME}
ifeq ($(MAKECMDGOALS),$(filter $(MAKECMDGOALS),docker-push push-tag))
ifndef DOCKER_REPO
    $(error DOCKER_REPO must be defined as the name of the docker repository where image will be pushed)
endif
ifndef DOCKER_NAMESPACE
    $(error DOCKER_NAMESPACE must be defined as the name of the docker namespace where image will be pushed)
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
ANALYSIS_TOOL_VERSION = ${DOCKER_IMAGE_TAG}
ifdef RELEASE_VERSION
	OPERATOR_VERSION = ${RELEASE_VERSION}
	ANALYSIS_TOOL_VERSION = ${RELEASE_VERSION}
endif
ifndef RELEASE_BRANCH
	RELEASE_BRANCH=$(shell git rev-parse --abbrev-ref HEAD)
endif

DIST_DIR:=dist
EXTRA_PARAMS=
INTEG_RUN_ID=
ENV_NAME=bom-validator
GO = GO111MODULE=on GOPRIVATE=github.com/verrazzano/* go
GO_LDFLAGS ?= -extldflags -static -X main.buildVersion=${BUILDVERSION} -X main.buildDate=${BUILDDATE}

#
#
#
.PHONY: run
run:
	$(GO) run ${GOPATH}/src/github.com/verrazzano/verrazzano/tools/analysis/main.go --zap-log-level=debug

#
# Go build related tasks
#
.PHONY: go-build
go-build:
	GOOS=darwin GOARCH=amd64 $(GO) build \
		-ldflags "${GO_LDFLAGS} -X main.analysisToolVersion=${ANALYSIS_TOOL_VERSION}" \
		-o out/darwin_amd64/bom-validator \
		${GOPATH}/src/github.com/verrazzano/verrazzano/tools/bom-validator
	GOOS=linux GOARCH=amd64 $(GO) build \
        -ldflags "${GO_LDFLAGS} -X main.analysisToolVersion=${ANALYSIS_TOOL_VERSION}" \
        -o out/linux_amd64/bom-validator \
        ${GOPATH}/src/github.com/verrazzano/verrazzano/tools/bom-validator

.PHONY: go-install
go-install:
	$(GO) install ./...

#
# Docker-related tasks
#
.PHONY: docker-clean
docker-clean:
	rm -rf ${DIST_DIR}

.PHONY: docker-build
docker-build: go-build
	@echo Building bom-validator image ${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG}
	# the TPL file needs to be copied into this dir so it is in the docker build context
	cp ../../THIRD_PARTY_LICENSES.txt .
	docker build --pull -f Dockerfile \
		-t ${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG} .

.PHONY: docker-push
docker-push: docker-build
	docker tag ${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG} ${DOCKER_IMAGE_FULLNAME}:${DOCKER_IMAGE_TAG}
	$(call retry_docker_push,${DOCKER_IMAGE_FULLNAME}:${DOCKER_IMAGE_TAG})

ifeq ($(CREATE_LATEST_TAG), "1")
	docker tag ${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG} ${DOCKER_IMAGE_FULLNAME}:latest;
	$(call retry_docker_push,${DOCKER_IMAGE_FULLNAME}:latest);
endif

#
# Test-related tasks
#
.PHONY: unit-test
unit-test: go-install
	$(GO) test -v  ./...

.PHONY: push-tag
push-tag:
	PUBLISH_TAG="${DOCKER_IMAGE_TAG}"; \
	echo "Tagging and pushing image ${DOCKER_IMAGE_FULLNAME}:$$PUBLISH_TAG"; \
	docker pull "${DOCKER_IMAGE_FULLNAME}:${DOCKER_IMAGE_TAG}"; \
	docker tag "${DOCKER_IMAGE_FULLNAME}:${DOCKER_IMAGE_TAG}" "${DOCKER_IMAGE_FULLNAME}:$$PUBLISH_TAG"; \
	$(call retry_docker_push,"${DOCKER_IMAGE_FULLNAME}:$$PUBLISH_TAG")
