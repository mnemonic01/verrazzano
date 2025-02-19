# Copyright (C) 2020, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

include ../make/quality.mk
include ../make/generate.mk
include ../make/retry.mk

SCRIPT_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))/../build
TOOLS_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))/../tools

NAME:=verrazzano-application-operator
REPO_NAME:=verrazzano-application-operator

CONTROLLER_GEN_VERSION ?= $(shell go list -m -f '{{.Version}}' sigs.k8s.io/controller-tools)
CREATE_LATEST_TAG=0

CRD_OPTIONS ?= "crd:crdVersions=v1"
ifdef KUBECONFIG
  KUBECONFIG ?= ${KUBECONFIG}
else
  KUBECONFIG ?= ${HOME}/.kube/config
endif

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
ENV_NAME=verrazzano-application-operator
GO ?= CGO_ENABLED=0 GO111MODULE=on GOPRIVATE=github.com/verrazzano go
GO_LDFLAGS ?= -extldflags -static -X main.buildVersion=${BUILDVERSION} -X main.buildDate=${BUILDDATE}
KIND_CONFIG ?= kind-config.yaml
CRD_PATH=../platform-operator/helm_config/charts/verrazzano-application-operator/crds
OAM_PATH=../platform-operator/thirdparty/charts/oam-kubernetes-runtime/crds

CLUSTER_DUMP_LOCATION ?= application-operator-integ-cluster-dump

.PHONY: build
build: go-fmt go-vet
	go build -o bin/manager main.go

# Disable the application operator in the local Kubernetes cluster
# Deletes all of the webhook configuration for the application operator
# Scales the in-cluster application operator deployment to replicas 0
.PHONY: disable-operator
disable-operator:
	kubectl delete validatingwebhookconfigurations verrazzano-application-ingresstrait-validator || true
	kubectl delete mutatingwebhookconfigurations verrazzano-application-istio-defaulter || true
	kubectl delete mutatingwebhookconfigurations verrazzano-application-appconfig-defaulter || true
	kubectl delete validatingwebhookconfigurations verrazzano-application-verrazzanoproject || true
	kubectl delete mutatingwebhookconfigurations verrazzano-application-scrape-generator || true
	kubectl scale deployment verrazzano-application-operator --replicas=0 -n verrazzano-system

# Run against the configured Kubernetes cluster in ~/.kube/config
.PHONY: run
run:
	# To enable webhooks: $(GO) run main.go --kubeconfig=${KUBECONFIG} --enable-webhooks=true --metrics-addr=localhost:0 --cert-dir=build/webhook-certs
	$(GO) run main.go --kubeconfig=${KUBECONFIG} --enable-webhooks=false --metrics-addr=localhost:0

.PHONY: install-oam-crds
install-oam-crds:
	kubectl apply -f ${OAM_PATH}

.PHONY: install-verrazzano-crds
install-verrazzano-crds:
	kubectl apply -f ${CRD_PATH}

# Install CRDs into a cluster
.PHONY: install-crds
install-crds: manifests install-oam-crds install-verrazzano-crds

# Uninstall CRDs from a cluster
.PHONY: uninstall-crds
uninstall-crds: manifests
	kubectl delete -f ${CRD_PATH}

# Generate mocks
.PHONY: mock-gen
mock-gen:
	mockgen -destination=mocks/controller_client_mock.go -package=mocks -copyright_file=hack/boilerplate.go.txt sigs.k8s.io/controller-runtime/pkg/client Client,StatusWriter
	mockgen -destination=mocks/controller_manager_mock.go -package=mocks -copyright_file=hack/boilerplate.go.txt sigs.k8s.io/controller-runtime Manager

.PHONY: manifests
manifests: application-manifests
#
# Go build related tasks
#
.PHONY: go-build
go-build:
	$(GO) build \
		-ldflags "${GO_LDFLAGS}" \
		-o out/$(shell uname)_$(shell uname -m)/verrazzano-application-operator \
		main.go

.PHONY: go-build-linux
go-build-linux:
	GOOS=linux GOARCH=amd64 $(GO) build \
		-ldflags "${GO_LDFLAGS}" \
		-o out/linux_amd64/verrazzano-application-operator \
		main.go

.PHONY: go-install
go-install:
	$(GO) install

#
# Docker-related tasks
#
.PHONY: docker-clean
docker-clean:
	rm -rf ${DIST_DIR}

.PHONY: docker-build
docker-build: go-build-linux
	# the TPL file needs to be copied into this dir so it is in the docker build context
	cp ../THIRD_PARTY_LICENSES.txt .
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

#
# Test-related tasks
#
.PHONY: unit-test
unit-test: go-install
	$(GO) test -v ./apis/... ./controllers/... ./internal/... ./mcagent/...

#
# Test-related tasks
#
CLUSTER_NAME = verrazzano
VERRAZZANO_NS = verrazzano-system
DEPLOY = deploy
OPERATOR_SETUP = test/operatorsetup
OAM_RUNTIME_IMAGE="ghcr.io/verrazzano/oam-kubernetes-runtime:v0.3.0-20210222205541-9e8d4fb"

# These exports are needed for the install.sh
export VERRAZZANO_KUBECONFIG=${KUBECONFIG}
export VERRAZZANO_APP_OP_IMAGE=${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG}

.PHONY: setup-cluster
setup-cluster: create-cluster
	echo 'Load Docker image for the Verrazzano application operator...'
	time kind load docker-image --name ${CLUSTER_NAME} ${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG}

	# The application operator creates network policies in the istio-system namespace, so make sure it exists
	kubectl create ns istio-system

	echo 'Install OAM runtime and Verrazzano application operator...'
	time installer/install.sh || (echo 'Application operator install failed, capturing kind cluster dump'; ../tools/scripts/k8s-dump-cluster.sh -d ${CLUSTER_DUMP_LOCATION} -r ${CLUSTER_DUMP_LOCATION}/analysis.report; exit 1)

.PHONY: integ-test
integ-test: setup-cluster
	echo 'Run tests...'
	ginkgo -v --keep-going -cover test/integ/... || (echo 'Application operator tests failed, capturing kind cluster dump'; ../tools/scripts/k8s-dump-cluster.sh -d ${CLUSTER_DUMP_LOCATION} -r ${CLUSTER_DUMP_LOCATION}/analysis.report; exit 1)

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
ifdef JENKINS_URL
	# Get the ip address of the container running the kube apiserver
	# and update the kubeconfig file to point to that address, instead of localhost
	sed -i -e "s|127.0.0.1.*|`docker inspect ${CLUSTER_NAME}-control-plane | jq '.[].NetworkSettings.Networks[].IPAddress' | sed 's/"//g'`:6443|g" ${KUBECONFIG}
	cat ${KUBECONFIG} | grep server
	$$(X=$$(docker inspect $$(docker ps | grep "jenkins-runner" | awk '{ print $$1 }') | jq '.[].NetworkSettings.Networks' | grep -q kind ; echo $$?); if [[ ! $$X -eq "0" ]]; then docker network connect kind $$(docker ps | grep "jenkins-runner" | awk '{ print $$1 }'); fi)
endif

.PHONY: delete-cluster
delete-cluster:
	kind delete cluster --name ${CLUSTER_NAME}

#
# Kubernetes-related tasks
#
.PHONY: push-tag
push-tag:
	PUBLISH_TAG="${DOCKER_IMAGE_TAG}"; \
	echo "Tagging and pushing image ${DOCKER_IMAGE_FULLNAME}:$$PUBLISH_TAG"; \
	docker pull "${DOCKER_IMAGE_FULLNAME}:${DOCKER_IMAGE_TAG}"; \
	docker tag "${DOCKER_IMAGE_FULLNAME}:${DOCKER_IMAGE_TAG}" "${DOCKER_IMAGE_FULLNAME}:$$PUBLISH_TAG"; \
	$(call retry_docker_push,"${DOCKER_IMAGE_FULLNAME}:$$PUBLISH_TAG")
