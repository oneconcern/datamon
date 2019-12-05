SHELL=/bin/bash
# COLORS
GREEN  := $(shell tput -Txterm setaf 2)
YELLOW := $(shell tput -Txterm setaf 3)
WHITE  := $(shell tput -Txterm setaf 7)
RESET  := $(shell tput -Txterm sgr0)

LOCAL_KUBECTX ?= "docker-for-desktop"
TARGET_MAX_CHAR_NUM=30

# Version and repo
GIT_BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
VERSION=$(shell git describe --tags)
COMMIT=$(shell git rev-parse HEAD)
RELEASE_TAG=$(shell go run ./hack/release_tag.go)
GITDIRTY=$(shell git diff --quiet || echo 'dirty')

# Build tagging parameters
VERSION_INFO_IMPORT_PATH ?= github.com/oneconcern/datamon/cmd/datamon/cmd.
BASE_LD_FLAGS ?= -s -w -linkmode external
VERSION_LD_FLAG_VERSION ?= -X '$(VERSION_INFO_IMPORT_PATH)Version=$(VERSION)'
VERSION_LD_FLAG_DATE ?= -X '$(VERSION_INFO_IMPORT_PATH)BuildDate=$(shell date -u -R)'
VERSION_LD_FLAG_COMMIT ?= -X '$(VERSION_INFO_IMPORT_PATH)GitCommit=$(COMMIT)'
VERSION_LD_FLAG_VERSION ?= -X '$(VERSION_INFO_IMPORT_PATH)GitState=$(GITDIRTY)'
VERSION_LD_FLAGS ?= $(VERSION_LD_FLAG_VERSION) $(VERSION_LD_FLAG_DATE) $(VERSION_LD_FLAG_COMMIT) $(VERSION_LD_FLAG_VERSION)

REPOSITORY ?= gcr.io/onec-co
VERSION_ARGS := --build-arg version_import_path=$(VERSION_INFO_IMPORT_PATH) --build-arg version=$(VERSION) --build-arg commit=$(COMMIT) --build-arg dirty=$(GITDIRTY)
BUILD_USER := $(subst @,_,$(GITHUB_USER))
USER_TAG := $(shell if [[ -n "$(BUILD_USER)" && "$(BUILD_USER)" != "onecrobot" ]] ; then echo $(BUILD_USER)-$$(date '+%Y%m%d') ; fi)
BRANCH_TAG := $(subst /,_,$(GIT_BRANCH))

# Docker args
BUILDER := DOCKER_BUILDKIT=1 docker build --ssh default --progress plain

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

.PHONY: build-target
build-target:
	@echo '$(GREEN)building $(YELLOW)$(BUILD_TARGET)$(GREEN) container...$(RESET)'
	@echo "$(GREEN)with tags: $(TAGS)$(RESET)"
	$(BUILDER) -f $(DOCKERFILE) $(BUILD_ARGS) -t $(BUILD_TARGET) \
	$(shell for tag in $(TAGS) ; do echo "-t $(BUILD_TARGET):$$tag" ; done) \
	.

.PHONY: push-target
push-target:
	docker push $(BUILD_TARGET)

.PHONY: build-datamon-binaries
build-datamon-binaries:
	$(MAKE) build-target \
		BUILD_TARGET=datamon-binaries \
		DOCKERFILE=binaries.Dockerfile BUILD_ARGS="--pull $(VERSION_ARGS)" TAGS=""

.PHONY: build-and-push-fuse-sidecar
## build FUSE sidecar container used in Argo workflows
build-and-push-fuse-sidecar: export BUILD_TARGET=$(REPOSITORY)/datamon-fuse-sidecar
build-and-push-fuse-sidecar: export DOCKERFILE=sidecar.Dockerfile
build-and-push-fuse-sidecar: export BUILD_ARGS=
build-and-push-fuse-sidecar: export RELEASE_TAG_LATEST=$(shell go run ./hack/release_tag.go -l)
build-and-push-fuse-sidecar: export TAGS=$(RELEASE_TAG) $(RELEASE_TAG_LATEST) $(USER_TAG) $(BRANCH_TAG)
build-and-push-fuse-sidecar: build-datamon-binaries
	$(MAKE) build-target
	$(MAKE) push-target

.PHONY: build-and-push-pg-sidecar
## build postgres sidecar container used in Argo workflows
build-and-push-pg-sidecar: export BUILD_TARGET=$(REPOSITORY)/datamon-pg-sidecar
build-and-push-pg-sidecar: export DOCKERFILE=sidecar-pg.Dockerfile
build-and-push-pg-sidecar: export BUILD_ARGS=
build-and-push-pg-sidecar: export TAGS=$(RELEASE_TAG) $(RELEASE_TAG_LATEST) $(USER_TAG) $(BRANCH_TAG)
build-and-push-pg-sidecar: build-datamon-binaries
	$(MAKE) build-target
	$(MAKE) push-target

.PHONY: build-and-push-datamover
## build sidecar container used in Argo workflows
build-and-push-datamover: export BUILD_TARGET=$(REPOSITORY)/datamon-datamover
build-and-push-datamover: export DOCKERFILE=datamover.Dockerfile
build-and-push-datamover: export BUILD_ARGS=
build-and-push-datamover: export RELEASE_TAG_LATEST=$(shell go run ./hack/release_tag.go -l -i gcr.io/onec-co/datamon-datamover)
build-and-push-datamover: export TAGS=$(RELEASE_TAG) $(RELEASE_TAG_LATEST) $(USER_TAG) $(BRANCH_TAG)
build-and-push-datamover: build-datamon-binaries
	$(MAKE) build-target
	$(MAKE) push-target

.PHONY: build-and-push-release-images
## build all docker images associated with a release
build-and-push-release-images: build-and-push-fuse-sidecar build-and-push-datamover

.PHONY: build-datamon
## Build datamon docker container on local registry
build-datamon: export BUILD_TARGET=reg.onec.co/datamon
build-datamon: export DOCKERFILE=Dockerfile
build-datamon: export BUILD_ARGS=--pull
build-datamon: export TAGS=$(USER_TAG) $(BRANCH_TAG)
build-datamon:
	$(MAKE) build-target

.PHONY: build-assets
## Prepare static assets for embedding in compiled binary
build-assets:
	go get -u github.com/gobuffalo/packr/v2/packr2
	(cd pkg/web && packr2 clean && packr2)

.PHONY: compile-datamon
## Build datamon executable on local OS
compile-datamon: export LDFLAGS=${BASE_LD_FLAGS} ${VERSION_LD_FLAGS}
compile-datamon: build-assets
	@echo 'building ${YELLOW}datamon${RESET} executable'
	@echo "${VERSION_LD_FLAGS}"
	@echo "${LDFLAGS}"
	go build -o $(TARGET) --ldflags "${LDFLAGS}" ./cmd/datamon

.PHONY: build-datamon-local
## DEMO: Build datamon executable in-place for local os.  Goes with internal operationalization demos.
build-datamon-local: export TARGET=cmd/datamon/datamon
build-datamon-local:
	$(MAKE) compile-datamon

.PHONY: build-datamon-mac
## Build datamon executable for mac os x (on mac os x)
build-datamon-mac: export TARGET=out/datamon.mac
build-datamon-mac:
	$(MAKE) compile-datamon

.PHONY: build-all
## Build all the containers
build-all: clean build-datamon build-migrate

.PHONY: build-migrate
## Build migrate tool
build-migrate: export BUILD_TARGET=reg.onec.co/migrate
build-migrate: export DOCKERFILE=migrate.Dockerfile
build-migrate: export BUILD_ARGS=--pull
build-migrate: export TAGS=$(USER_TAG) $(BRANCH_TAG)
build-migrate:
	$(MAKE) build-target

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

.PHONY: mocks
## Generate mocks for unit testing
mocks:
	@hack/go-generate.sh

.PHONY: runtests
runtests: mocks
	@go test ./...

.PHONY: gofmt
## Run gofmt on the cmd and pkg packages
gofmt:
	@gofmt -s -w ./cmd ./pkg ./internal ./hack/fuse-demo

.PHONY: goimports
## Run goimports on the cmd and pkg packages
goimports:
	@goimports -w ./cmd ./pkg ./hack/fuse-demo

.PHONY: check
## Runs static code analysis checks (golangci-lint)
check: gofmt goimports
	@golangci-lint run --build-tags fuse_cli --max-same-issues 0 --verbose

.PHONY: profile-metrics
## Build the metrics collection binary and write output
profile-metrics:
	@go build -o out/metrics/metrics.out ./cmd/metrics
	./hack/metrics/xtime.sh \
		-l out/metrics/upload.log \
		-t out/metrics/upload_time.log \
		-- out/metrics/metrics.out \
		--cpuprof out/metrics/cpu.prof \
		--memprof out/metrics/mem.prof \
		upload
	@./hack/metrics/pprof_fmt_metrics.sh \
		out/metrics/metrics.out \
		out/metrics/cpu.prof \
		out/metrics/mem.prof \
		out/metrics

### k8s demos

# todo: scripts to datamon-fuse-sidecar Docker img, lint to CI
.PHONY: lint-init-scripts
## DEMO: build shell container used in fuse demo
fuse-demo-lint-init-scripts:
	shellcheck hack/fuse-demo/wrap_application.sh

.PHONY: fuse-demo-build-shell
## DEMO: build shell container used in fuse demo
fuse-demo-build-shell: export BUILD_TARGET=$(REPOSITORY)/datamon-fuse-demo-shell
fuse-demo-build-shell: export DOCKERFILE=hack/fuse-demo/shell.Dockerfile
fuse-demo-build-shell: export BUILD_ARGS=
fuse-demo-build-shell: export TAGS=latest
fuse-demo-build-shell:
	$(MAKE) build-target
	$(MAKE) push-target

.PHONY: fuse-demo-build-sidecar
## DEMO: build sidecar container used in fuse demo
fuse-demo-build-sidecar: export BUILD_TARGET=$(REPOSITORY)/datamon-fuse-demo-sidecar
fuse-demo-build-sidecar: export DOCKERFILE=hack/fuse-demo/sidecar.Dockerfile
fuse-demo-build-sidecar: export BUILD_ARGS=
fuse-demo-build-sidecar: export TAGS=latest
fuse-demo-build-sidecar: build-and-push-fuse-sidecar
	$(MAKE) build-target
	$(MAKE) push-target

.PHONY: fuse-demo-ro
## DEMO: demonstrate a fuse read-only filesystem
fuse-demo-ro: fuse-demo-build-shell fuse-demo-build-sidecar
	@./hack/fuse-demo/create_ro_pod.sh
	@./hack/fuse-demo/run_shell.sh

.PHONY: fuse-demo-coord-build-app
## DEMO: build shell container used in fuse demo
fuse-demo-coord-build-app: export BUILD_TARGET=$(REPOSITORY)/datamon-fuse-demo-coord-app
fuse-demo-coord-build-app: export DOCKERFILE=hack/fuse-demo/coord-app.Dockerfile
fuse-demo-coord-build-app: export BUILD_ARGS=
fuse-demo-coord-build-app: export TAGS=latest
fuse-demo-coord-build-app:
	$(MAKE) build-target
	$(MAKE) push-target

.PHONY: fuse-demo-coord-build-datamon
## DEMO: build shell container used in fuse demo
fuse-demo-coord-build-datamon: export BUILD_TARGET=$(REPOSITORY)/datamon-fuse-demo-coord-datamon
fuse-demo-coord-build-datamon: export DOCKERFILE=hack/fuse-demo/coord-datamon.Dockerfile
fuse-demo-coord-build-datamon: export BUILD_ARGS=
fuse-demo-coord-build-datamon: export TAGS=latest
fuse-demo-coord-build-datamon: export SERIALIZED_INPUT_FILE=hack/fuse-demo/gen/fuse-params.yaml
fuse-demo-coord-build-datamon:
	@go run hack/fuse-demo/write_fuse_params.go
	$(MAKE) build-target
	$(MAKE) push-target

.PHONY: pg-demo-coord-build-app
## DEMO: build shell container used in fuse demo
pg-demo-coord-build-app: export BUILD_TARGET=$(REPOSITORY)/datamon-pg-demo-coord-app
pg-demo-coord-build-app: export DOCKERFILE=hack/fuse-demo/coord-app-pg.Dockerfile
pg-demo-coord-build-app: export BUILD_ARGS=
pg-demo-coord-build-app: export TAGS=latest
pg-demo-coord-build-app:
	$(MAKE) build-target
	$(MAKE) push-target

.PHONY: release
## Cut a new release. Example: make release TAG=v1.0.6
release:
	@if [[ -z "$(TAG)" ]] ; then echo "Must have arg: TAG=vX.Y.Z" && exit 1 ;fi
	@git tag -a -m "datamon release ${TAG}" ${TAG}
	git push origin ${TAG}
#
# NOTES:
# * atm git tags are not signed
# TODO:
# * [ ] bind this with git-chglog or similar
