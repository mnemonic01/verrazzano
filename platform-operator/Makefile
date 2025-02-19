# Copyright (C) 2020, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

include ../make/quality.mk
include ../make/generate.mk
include ../make/retry.mk

SCRIPT_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))/../build
TOOLS_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))/../tools

NAME:=verrazzano-platform-operator
REPO_NAME:=verrazzano-platform-operator

CONTROLLER_GEN_VERSION ?= $(shell go list -m -f '{{.Version}}' sigs.k8s.io/controller-tools)
CREATE_LATEST_TAG=0

CRD_OPTIONS ?= "crd:crdVersions=v1"
KUBECONFIG ?= ${HOME}/.kube/config

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

SHORT_COMMIT_HASH ?= $(shell git rev-parse --short=8 HEAD)
VERRAZZANO_DEV_VERSION ?= $(shell grep verrazzano-development-version ../.verrazzano-development-version | sed -e 's/verrazzano-development-version=//')
VERRAZZANO_VERSION ?= ${VERRAZZANO_DEV_VERSION}-local+${SHORT_COMMIT_HASH}

VERRAZZANO_APPLICATION_OPERATOR_IMAGE ?= verrazzano-application-operator-dev

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
ENV_NAME=verrazzano-platform-operator
GO ?= GO111MODULE=on GOPRIVATE=github.com/verrazzano go
GO_LDFLAGS ?= -extldflags -static -X main.buildVersion=${BUILDVERSION} -X main.buildDate=${BUILDDATE}
GO_BINDATA_VERSION ?= $(shell go list -m -f '{{.Version}}' 'github.com/go-bindata/go-bindata/v3' )

CRD_PATH=helm_config/charts/verrazzano-platform-operator/crds

CLUSTER_DUMP_LOCATION ?= platform-operator-integ-cluster-dump
SHELL = bash

# Run against the configured Kubernetes cluster in ~/.kube/config
.PHONY: run
run:
	$(GO) run main.go --kubeconfig=${KUBECONFIG} --zap-log-level=debug

# Install CRDs into a cluster
.PHONY: install-crds
install-crds:
	kubectl apply -f ${CRD_PATH}

# Uninstall CRDs from a cluster
.PHONY: uninstall-crds
uninstall-crds:
	kubectl delete -f ${CRD_PATH}


#
# Go build related tasks
#
.PHONY: go-build
go-build: assets
	$(GO) build \
		-ldflags "${GO_LDFLAGS}" \
		-o out/$(shell uname)_$(shell uname -m)/verrazzano-platform-operator \
		main.go

.PHONY: go-build-linux
go-build-linux:
	GOOS=linux GOARCH=amd64 $(GO) build \
		-ldflags "${GO_LDFLAGS}" \
		-o out/linux_amd64/verrazzano-platform-operator \
		main.go

.PHONY: go-install
go-install:
	$(GO) install ./...

.PHONY: manifests
manifests: platform-manifests

# Generate mocks
.PHONY: mock-gen
mock-gen:
	mockgen -destination=mocks/component_mock.go -package=mocks -copyright_file=hack/boilerplate.go.txt github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi ComponentContext,ComponentInfo,ComponentInstaller,ComponentUpgrader,Component
	mockgen -destination=mocks/controller_mock.go -package=mocks -copyright_file=hack/boilerplate.go.txt sigs.k8s.io/controller-runtime/pkg/client Client,StatusWriter

# find or download go-bindata
# download go-bindata if necessary
.PHONY: go-bindata
go-bindata:
ifeq (, $(shell command -v go-bindata))
	$(GO) get github.com/go-bindata/go-bindata/v3/...@${GO_BINDATA_VERSION}
	echo $(GOBIN)
	$(eval GO_BINDATA=$(GOBIN)/go-bindata)
else
	$(eval GO_BINDATA=$(shell command -v go-bindata))
endif
	@{ \
	set -eu; \
	ACTUAL_GO_BINDATA_VERSION=$$(${GO_BINDATA} --version | head -1 | awk '{print $$2}') ; \
	if [ "v$${ACTUAL_GO_BINDATA_VERSION}" != "${GO_BINDATA_VERSION}" ] ; then \
		echo  "Bad go-bindata version $${ACTUAL_GO_BINDATA_VERSION}, please install ${GO_BINDATA_VERSION}" ; \
	fi ; \
	}

.PHONY: assets
assets: go-bindata
	$(GO_BINDATA) -modtime 1647457998 -pkg grafana -o generated_assets.go ./manifests/dashboards/...
	echo '// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.' | cat - generated_assets.go > tmp && mv tmp generated_assets.go
	echo '// Copyright (C) 2022, Oracle and/or its affiliates.' | cat - generated_assets.go > tmp && mv tmp generated_assets.go
	mv ./generated_assets.go ./controllers/verrazzano/component/grafana/

#
# Docker-related tasks
#
.PHONY: docker-clean
docker-clean:
	rm -rf ${DIST_DIR}

.PHONY: docker-build
docker-build: generate-bom go-build-linux
	@echo Building verrazzano-platform-operator image ${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG}
	@echo using verrazzano-application-operator image ${VERRAZZANO_APPLICATION_OPERATOR_IMAGE}
	# the TPL file needs to be copied into this dir so it is in the docker build context
	cp ../THIRD_PARTY_LICENSES.txt .
	docker build --pull -f Dockerfile \
		--build-arg VERRAZZANO_APPLICATION_OPERATOR_IMAGE="${VERRAZZANO_APPLICATION_OPERATOR_IMAGE}" \
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
	$(GO) test -v  ./internal/... ./controllers/... ./apis/...

#
# Test-related tasks
#
# On Jenkins we have pre-created clusters ready to go that are named specifically for the integ-test case
CLUSTER_NAME = verrazzano
VERRAZZANO_NS = verrazzano-install
BUILD_DEPLOY = build/deploy
OPERATOR_YAML ?= ${BUILD_DEPLOY}/operator.yaml
DEPLOY = deploy
OPERATOR_SETUP = test/operatorsetup

CLUSTER_DUMP_LOCATION ?= platform-operator-integ-cluster-dump
KIND_CONFIG ?= kind-config.yaml

.PHONY: integ-test
integ-test: create-cluster generate-operator-yaml
	echo 'Load docker image for the verrazzano-platform-operator...'
	kind load docker-image --name ${CLUSTER_NAME} ${DOCKER_IMAGE_FULLNAME}:${DOCKER_IMAGE_TAG}
	echo 'Deploy Verrazzano platform operator ...'
	kubectl apply -f ${OPERATOR_YAML} || (echo 'Platform operator install failed, capturing kind cluster dump'; ../tools/scripts/k8s-dump-cluster.sh -d ${CLUSTER_DUMP_LOCATION} -r ${CLUSTER_DUMP_LOCATION}/analysis.report; exit 1)
	echo 'Run tests...'
	ginkgo -v --keep-going -cover test/integ/... || (echo 'Platform operator tests failed, capturing kind cluster dump'; ../tools/scripts/k8s-dump-cluster.sh -d ${CLUSTER_DUMP_LOCATION} -r ${CLUSTER_DUMP_LOCATION}/analysis.report; exit 1)

.PHONY: generate-operator-yaml
generate-operator-yaml:
	OPERATOR_DIR=$$(dirname ${OPERATOR_YAML}) ; \
	mkdir -p $${OPERATOR_DIR} ; \
	env DOCKER_IMAGE=${DOCKER_IMAGE_FULLNAME}:${DOCKER_IMAGE_TAG} IMAGE_PULL_SECRETS=${IMAGE_PULL_SECRETS} APP_OPERATOR_IMAGE=${VERRAZZANO_APPLICATION_OPERATOR_IMAGE} ../tools/scripts/generate_operator_yaml.sh > ${OPERATOR_YAML}

.PHONY: cleanup-cluster
cleanup-cluster:
	./build/scripts/cleanup.sh ${CLUSTER_NAME}

.PHONY: create-cluster
create-cluster: WORKSPACE ?= ${CURDIR}
create-cluster:
	kind_clusters=$$(kind get clusters) ; \
    if [ -z "$${kind_clusters}" ] || ! kind get clusters | egrep "^${CLUSTER_NAME}\$$" > /dev/null ; then \
		echo 'Create cluster...' ;\
		echo 'KUBECONFIG=${KUBECONFIG}' ;\
		HTTP_PROXY="" HTTPS_PROXY="" http_proxy="" https_proxy="" time kind create cluster -v 1 \
			--name ${CLUSTER_NAME} \
			--wait 5m \
			--config=test/${KIND_CONFIG} ;\
		if [ $$? -ne 0 ] ; then \
			mkdir ${WORKSPACE}/kind-logs ;\
			kind export logs ${WORKSPACE}/kind-logs ;\
			echo "Kind cluster creation failed" ;\
			exit 1 ;\
		fi ;\
		kubectl config set-context kind-${CLUSTER_NAME} ;\
	fi

.PHONY: delete-cluster
delete-cluster:
	kind delete cluster --name ${CLUSTER_NAME}

.PHONY: push-tag
push-tag:
	PUBLISH_TAG="${DOCKER_IMAGE_TAG}"; \
	echo "Tagging and pushing image ${DOCKER_IMAGE_FULLNAME}:$$PUBLISH_TAG"; \
	docker pull "${DOCKER_IMAGE_FULLNAME}:${DOCKER_IMAGE_TAG}"; \
	docker tag "${DOCKER_IMAGE_FULLNAME}:${DOCKER_IMAGE_TAG}" "${DOCKER_IMAGE_FULLNAME}:$$PUBLISH_TAG"; \
	$(call retry_docker_push,"${DOCKER_IMAGE_FULLNAME}:$$PUBLISH_TAG")

.PHONY: create-test-deploy
create-test-deploy:
	if [ -n "${VZ_DEV_IMAGE}" ]; then \
		IMAGE=$$(echo $${VZ_DEV_IMAGE} | cut -f 1 -d :) ; \
		IMAGE_TAG=$$(echo $${VZ_DEV_IMAGE} | cut -f 2 -d :) ; \
		DOCKER_IMAGE_FULLNAME=$${IMAGE} DOCKER_IMAGE_TAG=$${IMAGE_TAG} VERRAZZANO_APPLICATION_OPERATOR_IMAGE=$${VZ_APP_OP_IMAGE} $(MAKE) generate-operator-yaml ; \
	else \
		echo "VZ_DEV_IMAGE not defined, please set it to a valid image name/tag"; \
	fi

.PHONY: generate-bom
generate-bom:
	@echo Generating BOM verrazzano-platform-operator image ${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG}
	@echo                verrazzano-application-operator image ${VERRAZZANO_APPLICATION_OPERATOR_IMAGE}
	mkdir out || true
	../tools/scripts/generate_bom.sh verrazzano-bom.json ${VERRAZZANO_VERSION} ${VERRAZZANO_APPLICATION_OPERATOR_IMAGE} ${DOCKER_IMAGE_NAME} ${DOCKER_IMAGE_TAG} out/generated-verrazzano-bom.json

