# Needed to bring in private packages from GitHub
ifndef GITHUB_USER
$(error "Must set GITHUB_USER") # this is a Make error
endif
ifndef GITHUB_TOKEN
$(error "Must set GITHUB_TOKEN") # this is a Make error
endif

# COLORS
GREEN  := $(shell tput -Txterm setaf 2)
YELLOW := $(shell tput -Txterm setaf 3)
WHITE  := $(shell tput -Txterm setaf 7)
RESET  := $(shell tput -Txterm sgr0)
GIT_BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
VERSION=$(shell git describe --tags)
COMMIT=$(shell git rev-parse HEAD)
GITDIRTY=$(shell git diff --quiet || echo 'dirty')
LOCAL_KUBECTX ?= "docker-for-desktop"
TARGET_MAX_CHAR_NUM=25

# Version and repo
VERSION=$(shell git describe --tags)
COMMIT=$(shell git rev-parse HEAD)
GITDIRTY=$(shell git diff --quiet || echo 'dirty')
REPOSITORY ?= "gcr.io/onec-co"

## Show help
help:
	@echo ''
	@echo 'Usage:'
	@echo '  ${YELLOW}make${RESET} ${GREEN}<target>${RESET}'
	@echo ''
	@echo 'Targets:'
	@awk '/^[a-zA-Z\-\_0-9]+:/ { \
		helpMessage = match(lastLine, /^## (.*)/); \
		if (helpMessage) { \
			helpCommand = substr($$1, 0, index($$1, ":")-1); \
			helpMessage = substr(lastLine, RSTART + 3, RLENGTH); \
			printf "  ${YELLOW}%-$(TARGET_MAX_CHAR_NUM)s${RESET} ${GREEN}%s${RESET}\n", helpCommand, helpMessage; \
		} \
	} \
	{ lastLine = $$0 }' $(MAKEFILE_LIST)

.PHONY: build-and-push-fuse-sidecar
## build sidecar container used in fuse demo
build-and-push-fuse-sidecar:
	@echo 'building fuse demo container'
	docker build \
		--progress plain \
		-t gcr.io/onec-co/datamon-fuse-sidecar \
		-t gcr.io/onec-co/datamon-fuse-sidecar:${GITHUB_USER}-$$(date '+%Y%m%d') \
		-t gcr.io/onec-co/datamon-fuse-sidecar:$(subst /,_,$(GIT_BRANCH)) \
		--ssh default \
		-f sidecar.Dockerfile \
		.
	docker push gcr.io/onec-co/datamon-fuse-sidecar

.PHONY: fuse-demo-build-shell
## build shell container used in fuse demo
fuse-demo-build-shell:
	@echo 'building fuse demo container'
	docker build \
		--progress plain \
		-t gcr.io/onec-co/datamon-fuse-demo-shell \
		--ssh default \
		-f ./hack/fuse-demo/shell.Dockerfile \
		.

.PHONY: fuse-demo-build-sidecar
## build sidecar container used in fuse demo
fuse-demo-build-sidecar:
	@echo 'building fuse demo container'
	docker build \
		--progress plain \
		--build-arg github_user=$(GITHUB_USER) \
		--build-arg github_token=$(GITHUB_TOKEN) \
		-t gcr.io/onec-co/datamon-fuse-sidecar-ritesh \
		--ssh default \
		-f ./hack/fuse-demo/sidecar.Dockerfile \
		.

.PHONY: build-datamon
## Build datamon docker container (datamon)
build-datamon:
	@echo 'building ${YELLOW}datamon${RESET} container'
	@docker build \
		--pull \
		--build-arg github_user=$(GITHUB_USER) \
		--build-arg github_token=$(GITHUB_TOKEN) \
		--build-arg version=$(VERSION) \
		--build-arg commit=$(COMMIT) \
		--build-arg dirty=$(GITDIRTY) \
		-t reg.onec.co/datamon:${GITHUB_USER}-$$(date '+%Y%m%d') \
		-t reg.onec.co/datamon:$(subst /,_,$(GIT_BRANCH)) \
		.

## demonstrate a fuse read-only filesystem
fuse-demo-ro: fuse-demo-build-shell fuse-demo-build-sidecar
	@docker image push gcr.io/onec-co/datamon-fuse-demo-shell
	@docker image push gcr.io/onec-co/datamon-fuse-demo-sidecar
	@./hack/fuse-demo/create_ro_pod.sh
	@sleep 8 # dumb timeout on container startup
	@./hack/fuse-demo/run_shell.sh

.PHONY: build-all
## Build all the containers
build-all: clean build-datamon build-migrate

.PHONY: build-migrate
## Build migrate tool
build-migrate:
	@echo 'building ${YELLOW}migrate${RESET} container'
	@docker build --pull \
		--build-arg github_user=$(GITHUB_USER) \
		--build-arg github_token=$(GITHUB_TOKEN) \
		-t reg.onec.co/migrate:${GITHUB_USER}-$$(date '+%Y%m%d') \
		-t reg.onec.co/migrate:$(subst /,_,$(GIT_BRANCH)) \
		-f migrate.Dockerfile .

.PHONY: build-csi
build-csi:
	@echo 'building ${yello}csi${RESET} container'
	@docker build --pull --build-arg github_user=$(GITHUB_USER) \
		--build-arg github_token=$(GITHUB_TOKEN) \
		-t ${REPOSITORY}/datamon-csi:${GITHUB_USER}-$$(date '+%Y%m%d') \
		-t ${REPOSITORY}/datamon-csi:$(subst /,_,$(GIT_BRANCH)) \
		-f csi.Dockerfile .

.PHONY: push-all
## Push all the containers
push-all: push-datamon

.PHONY: setup
## Setup for testing
setup: install-minio

.PHONY: install-minio
## Run minio locally for tests
install-minio:
	@docker run --name minio-test -d \
	-p 9000:9000  \
	-e "MINIO_ACCESS_KEY=access-key" \
	-e "MINIO_SECRET_KEY=secret-key-thing" \
	-e "MINIO_BROWSER=off" \
	-e "MINIO_DOMAIN=s3.local"  \
	-e "MINIO_HTTP_TRACE=/tmp/minio.log" \
	minio/minio server /data > /dev/null

.PHONY: install-minio-k8s
## Install minio in local kubernetes
install-minio-k8s:
	kubectl --context $(LOCAL_KUBECTX) create -f ./k8s/minio.yaml

.SILENT: clean
## Clean up post running tests
clean:
	-rm -rf testdata 2>&1 | true
	-docker stop minio-test 2>&1 | true
	-docker rm minio-test 2>&1 | true

.PHONY: test
## Setup, run all tests and clean
test: clean setup runtests clean

.PHONY: runtests
runtests:
	@go test ./...

.PHONY: gofmt
## Run gofmt on the cmd and pkg packages
gofmt:
	@gofmt -s -w ./cmd ./pkg

.PHONY: goimports
## Run goimports on the cmd and pkg packages
goimports:
	@goimports -w ./cmd ./pkg

.PHONY: check
## Runs static code analysis checks (golangci-lint)
check: gofmt goimports
	@golangci-lint run --build-tags fuse_cli --max-same-issues 0 --verbose
