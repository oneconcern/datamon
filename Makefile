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
GIT_HOME = $(shell git rev-parse --show-toplevel)
VERSION=$(shell git describe --tags)
COMMIT=$(shell git rev-parse HEAD)
RELEASE_TAG=$(shell go run ./hack/release_tag.go)
RELEASE_TAG_LATEST := $(shell go run ./hack/release_tag.go -l)
GITDIRTY=$(shell git diff --quiet || echo 'dirty')

# Version of pre-built base image to use. Increment manually only.
SIDECAR_BASE_VERSION ?= 20200307

UPX_VERSION=$(shell type upx>/dev/null && upx --version|head -1|cut -d' ' -f2)
UPX_MAJOR=$(shell echo $(UPX_VERSION)|cut -d'.' -f1)
UPX_MINOR=$(shell echo $(UPX_VERSION)|cut -d'.' -f2)
# upx >=3.96 supports osx 64bit binaries
UPX_FOR_OSX=$(shell if [[ -n "$(UPX_MAJOR)" && -n $(UPX_MINOR) && "$(UPX_MAJOR)" -ge 3 && "$(UPX_MINOR)" -ge 96 ]] ; then echo "1"; else echo "" ;fi)

# go-gettable tools used for build and test
# NOTE: we don't put packr2 in that list, to stick to the version in go.mod (no automatic upgrade)
TOOLS ?= github.com/mitchellh/gox \
	github.com/golangci/golangci-lint/cmd/golangci-lint \
	gotest.tools/gotestsum@latest \
	github.com/matryer/moq@latest \
	golang.org/x/tools/cmd/goimports

# Build tagging parameters
VERSION_INFO_IMPORT_PATH ?= github.com/oneconcern/datamon/cmd/datamon/cmd.
BASE_LD_FLAGS ?= -s -w
VERSION_LD_FLAG_VERSION ?= -X '$(VERSION_INFO_IMPORT_PATH)Version=$(VERSION)'
VERSION_LD_FLAG_DATE ?= -X '$(VERSION_INFO_IMPORT_PATH)BuildDate=$(shell date -u -R)'
VERSION_LD_FLAG_COMMIT ?= -X '$(VERSION_INFO_IMPORT_PATH)GitCommit=$(COMMIT)'
VERSION_LD_FLAG_STATE ?= -X '$(VERSION_INFO_IMPORT_PATH)GitState=$(GITDIRTY)'
VERSION_LD_FLAGS ?= $(VERSION_LD_FLAG_VERSION) $(VERSION_LD_FLAG_DATE) $(VERSION_LD_FLAG_COMMIT) $(VERSION_LD_FLAG_STATE)

REPOSITORY ?= gcr.io/onec-co
LOCALREPO ?= reg.onec.co
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
		helpMessage = match(lastLine, /^##\s*(.*)/); \
		if (helpMessage) { \
			helpCommand = substr($$1, 0, index($$1, ":")-1); \
			helpMessage = substr(lastLine, RSTART + 3, RLENGTH); \
			helpTopic = match(helpMessage, /^([A-Z]+):\s+/); \
			if (helpTopic != "") { \
				helpTopic = substr(helpMessage, RSTART,RLENGTH-1); \
				helpMessage = substr(helpMessage,RLENGTH+1) ; \
				if (helpTopic != previousTopic) { \
				  printf "\nTopic ${GREEN}%s${RESET}\n", helpTopic; \
					previousTopic = helpTopic; \
				} \
			} \
			printf "  ${YELLOW}%-$(TARGET_MAX_CHAR_NUM)s${RESET} ${GREEN}%s${RESET}\n", helpCommand, helpMessage; \
		} \
	} \
	{ lastLine = $$0 }' < $(MAKEFILE_LIST)

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
## SIDECAR: Build container with all datamon binaries for sidecars
build-datamon-binaries: export BUILD_TARGET=$(LOCALREPO)/datamon-binaries
build-datamon-binaries: export DOCKERFILE=dockerfiles/binaries.Dockerfile
build-datamon-binaries: export BUILD_ARGS=$(VERSION_ARGS)
build-datamon-binaries: export TAGS=
build-datamon-binaries: build-alpine-base
	$(MAKE) build-target

.PHONY: build-and-push-fuse-sidecar
## SIDECAR: Build FUSE sidecar container used in Argo workflows
build-and-push-fuse-sidecar: export BUILD_TARGET=$(REPOSITORY)/datamon-fuse-sidecar
build-and-push-fuse-sidecar: export DOCKERFILE=dockerfiles/sidecar.Dockerfile
build-and-push-fuse-sidecar: export BUILD_ARGS=--build-arg VERSION=$(SIDECAR_BASE_VERSION)
build-and-push-fuse-sidecar: export TAGS=$(RELEASE_TAG) $(RELEASE_TAG_LATEST) $(USER_TAG) $(BRANCH_TAG)
build-and-push-fuse-sidecar: build-datamon-binaries
	$(MAKE) build-target
	$(MAKE) push-target

.PHONY: build-and-push-pg-sidecar
## SIDECAR: Build postgres sidecar container used in Argo workflows
build-and-push-pg-sidecar: export BUILD_TARGET=$(REPOSITORY)/datamon-pg-sidecar
build-and-push-pg-sidecar: export DOCKERFILE=dockerfiles/sidecar-pg.Dockerfile
build-and-push-pg-sidecar: export BUILD_ARGS=--build-arg VERSION=$(SIDECAR_BASE_VERSION)
build-and-push-pg-sidecar: export TAGS=$(RELEASE_TAG) $(RELEASE_TAG_LATEST) $(USER_TAG) $(BRANCH_TAG)
build-and-push-pg-sidecar: build-datamon-binaries
	$(MAKE) build-target
	$(MAKE) push-target

.PHONY: ensure-gotools
## BUILD: Install all go-gettable tools
ensure-gotools:
	@mkdir -p ${GOPATH}/bin && \
	pushd ${GOPATH}/bin 1>/dev/null 2>&1 && \
	for tool in $(TOOLS) ; do \
	  echo "$(GREEN)INFO: ensuring $(YELLOW)$${tool}$(GREEN) is up to date$(RESET)" && \
		go get $${tool} 1>/dev/null 2>&1; \
	done && \
	popd 2>/dev/null

.PHONY: gofmt
## BUILD: Run gofmt on the cmd and pkg packages (ships with go)
gofmt:
	@gofmt -s -w .

.PHONY: goimports
## BUILD: Run goimports on the cmd and pkg packages
goimports:
	@TOOLS=golang.org/x/tools/cmd/goimports $(MAKE) ensure-gotools
	@goimports -w .

.PHONY: build-assets
## BUILD: Prepare static assets for embedding in compiled binary
build-assets:
	go get github.com/gobuffalo/packr/v2/packr2
	(cd pkg/web && packr2 clean && packr2)

.PHONY: compile-datamon
compile-datamon: export LDFLAGS=${BASE_LD_FLAGS} ${VERSION_LD_FLAGS}
compile-datamon: build-assets
	@echo '$(GREEN)INFO: building ${YELLOW}datamon${GREEN} executable$(RESET)'
	@if [[ -z "$(TARGET)" ]] ; then echo "You must specify a TARGET" && exit 1; fi
	go build -o $(TARGET) -ldflags "${LDFLAGS}" ./cmd/datamon

.PHONY: build-datamon-local
## BUILD: Build datamon executable for local os
build-datamon-local: export TARGET=out/datamon
build-datamon-local:
	@mkdir -p out
	$(MAKE) compile-datamon


.PHONY: build-datamon-mac
## BUILD: Build local datamon executable for mac os x in ./out directory (on mac os x)
build-datamon-mac: export TARGET=out/datamon.mac
build-datamon-mac:
	@mkdir -p out
	$(MAKE) compile-datamon

.PHONY: cross-compile-binaries
## BUILD: Build all binaries with cross compilation for linux and darwin platforms
cross-compile-binaries: export TARGET=out/bin
cross-compile-binaries: export OS=linux darwin
cross-compile-binaries: export ARCH=amd64
cross-compile-binaries: export LDFLAGS=${BASE_LD_FLAGS} ${VERSION_LD_FLAGS}
cross-compile-binaries: build-assets
	@TOOLS=github.com/mitchellh/gox $(MAKE) ensure-gotools
	@mkdir -p ${TARGET}

	# Ref: https://github.com/mitchellh/gox/issues/55 for CGO_ENABLED=0
	CGO_ENABLED=0 gox -os "${OS}" -arch "${ARCH}" -tags netgo -output "${TARGET}/{{.Dir}}_{{.OS}}_{{.Arch}}" -ldflags "${LDFLAGS}" ./...

	@if [[ -n "${UPX_FOR_OSX}" ]]; then \
		echo "INFO: compressing all binaries" && \
		upx ${TARGET}/* ; res=$$? || true ; \
		if [[ $${res} -ne 0 && $${res} -ne 2 ]] ; then \
		  echo "ERROR: upx failed" && exit 1 ; \
		fi ; \
	elif [[ -n "${UPX_VERSION}" ]] ; then \
	  echo "INFO: compressing linux binaries only. Install upx >=3.96 to compress darwin binaries" && \
		upx ${TARGET}/*_linux_* ; res=$$? || true ; \
		if [[ $${res} -ne 0 && $${res} -ne 2 ]] ; then \
		  echo "ERROR: upx failed" && exit 1 ; \
		fi ; \
	else \
	  echo "WARN:: upx not available in this environment. Binaries not compressed." ; \
	fi

.PHONY: build-and-push-datamon-builder
## BUILD: Build new base builder image for CI jobs
build-and-push-datamon-builder:
	$(BUILDER) -t $(REPOSITORY)/datamon-builder:${BASEVERSION} - < dockerfiles/builder.Dockerfile
	docker tag $(REPOSITORY)/datamon-builder:${BASEVERSION} $(REPOSITORY)/datamon-builder:latest

	docker push $(REPOSITORY)/datamon-builder:${BASEVERSION}
	docker push $(REPOSITORY)/datamon-builder:latest

.PHONY: setup
## TEST: Setup for local testing (install minio container)
setup: install-minio

.PHONY: install-minio
## TEST: Run minio locally for tests
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
## TEST: Install minio in local kubernetes
install-minio-k8s:
	kubectl --context $(LOCAL_KUBECTX) create -f ./k8s/minio.yaml

.SILENT: clean
## TEST: Clean up post running tests
clean:
	-rm -rf testdata out
	-docker stop minio-test 2>&1 | true
	-docker rm minio-test 2>&1 | true

CI_NS=datamon-ci

.PHONY: clean-pg-demo
## TEST: Ensure k8s pg sidecar demo resources are relinquished (used by CI)
clean-pg-demo:
	kubectl -n $(CI_NS) delete deployment datamon-pg-demo-$(RELEASE_TAG) || true
	kubectl -n $(CI_NS) get configmaps -l app=datamon-coord-pg-demo --output custom-columns=NAME:.metadata.name|\
	grep $(RELEASE_TAG)|while read -r config ; do kubectl -n $(CI_NS) delete configmap $${config} ; done
	kubectl -n $(CI_NS) get pods -l app=datamon-coord-pg-demo --output custom-columns=NAME:.metadata.name|tail +2|\
	while read -r pod ; do kubectl -n $(CI_NS) delete pod $${pod} --grace-period 0 --force ; done

# TODO(fred): there are some secrets etc to be removed too

.PHONY: clean-fuse-demo
## TEST: Ensure k8s fuse sidecar demo resources are relinquished (used by CI)
clean-fuse-demo:
	kubectl -n $(CI_NS) delete deployment datamon-fuse-demo-$(RELEASE_TAG) || true
	kubectl -n $(CI_NS) get configmaps -l app=datamon-coord-fuse-demo --output custom-columns=NAME:.metadata.name|\
	grep $(RELEASE_TAG)|while read -r config ; do kubectl -n $(CI_NS) delete configmap $${config} ; done
	kubectl -n $(CI_NS) get pods -l app=datamon-coord-fuse-demo --output custom-columns=NAME:.metadata.name|tail +2|\
	while read -r pod ; do kubectl -n $(CI_NS) delete pod $${pod} --grace-period 0 --force ; done

.PHONY: clean-all-demo
## TEST: Ensure all k8s sidecar demo resources are relinquished (manual use)
clean-all-demo:
	kubectl -n $(CI_NS) delete deployments --all
	kubectl -n $(CI_NS) delete configmaps --all
	kubectl -n $(CI_NS) get pods --output custom-columns=NAME:.metadata.name|tail +2|\
	while read -r pod ; do kubectl -n $(CI_NS) delete pod $${pod} --grace-period 0 --force ; done

.PHONY: test
## TEST: Setup, run all tests and clean
test: clean setup runtests clean

.PHONY: mocks
## TEST: Generate mocks for unit testing
mocks:
	@TOOLS=github.com/matryer/moq@latest $(MAKE) ensure-gotools
	@hack/go-generate.sh

.PHONY: runtests
runtests: mocks
	@TOOLS=gotest.tools/gotestsum@latest $(MAKE) ensure-gotools
	@gotestsum --format short-with-failures -- -timeout 15m -cover -covermode atomic ./...

.PHONY: lint
## TEST: Runs static code analysis checks (golangci-lint)
lint: gofmt goimports
	TOOLS=github.com/golangci/golangci-lint/cmd/golangci-lint $(MAKE) ensure-gotools
	@golangci-lint run --verbose

.PHONY: profile-metrics
## TEST: Build the metrics collection binary and write output
profile-metrics:
	@mkdir -p out/metrics
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

.PHONY: lint-scripts
## TEST: Scan all shells in repo, excluding zsh which is not linted
lint-scripts:
	@(cd $(GIT_HOME) && find . -name \*.sh|\
		while read -r arg ;do script=$$(head -1q "$${arg}"); \
		  if [[ ! $${script} =~ "zsh" ]] ; then echo "$${arg}"; fi; \
		done|\
		xargs shellcheck && \
	echo "INFO: shell scripts linted ok (zsh not scanned)")

.PHONY: lint-zsh
## TEST: Experimental shell linting: shellcheck v0.7 evaluates zsh as bash, using --shell=bash: informative output only for now
lint-zsh:
	@(cd $(GIT_HOME) && \
	docker run -v $$(pwd):/mnt koalaman/shellcheck:latest --shell=bash $$(find . -name \*.sh))

.PHONY: build-and-push-sidecar-base
## SIDECAR: Build new base images for sidecars and upload them to gcr.io repository
BASEVERSION=$(shell date +%Y%m%d)
build-and-push-sidecar-base:
	$(BUILDER) --build-arg VERSION=${BASEVERSION} -t $(REPOSITORY)/datamon-sidecar-base:${BASEVERSION} - < dockerfiles/sidecar-base.Dockerfile
	docker tag $(REPOSITORY)/datamon-sidecar-base:${BASEVERSION} $(REPOSITORY)/datamon-sidecar-base:latest

	$(BUILDER) --build-arg VERSION=${BASEVERSION} -t $(REPOSITORY)/datamon-pgsidecar-base:${BASEVERSION} - < dockerfiles/pgsidecar-base.Dockerfile
	docker tag $(REPOSITORY)/datamon-pgsidecar-base:${BASEVERSION} $(REPOSITORY)/datamon-pgsidecar-base:latest

	docker push $(REPOSITORY)/datamon-sidecar-base:${BASEVERSION}
	docker push $(REPOSITORY)/datamon-sidecar-base:latest
	docker push $(REPOSITORY)/datamon-pgsidecar-base:${BASEVERSION}
	docker push $(REPOSITORY)/datamon-pgsidecar-base:latest

### k8s demos
.PHONY: fuse-demo-ro
## DEMO: Demonstrate a fuse read-only filesystem on kubernetes
fuse-demo-ro: fuse-demo-build-shell fuse-demo-build-sidecar
	@./hack/fuse-demo/create_ro_pod.sh
	@./hack/fuse-demo/run_shell.sh

.PHONY: fuse-demo-build-shell
## DEMO: Build shell container used in fuse demo
fuse-demo-build-shell: export BUILD_TARGET=$(REPOSITORY)/datamon-fuse-demo-shell
fuse-demo-build-shell: export DOCKERFILE=hack/fuse-demo/shell.Dockerfile
fuse-demo-build-shell: export BUILD_ARGS=--build-arg VERSION=$(SIDECAR_BASE_VERSION)
fuse-demo-build-shell: export TAGS=latest $(RELEASE_TAG) $(USER_TAG) $(BRANCH_TAG)
fuse-demo-build-shell:
	$(MAKE) build-target
	$(MAKE) push-target

.PHONY: fuse-demo-coord-build-app
## DEMO: Build shell container used in fuse demo: mock application using fuse mount
fuse-demo-coord-build-app: export BUILD_TARGET=$(REPOSITORY)/datamon-fuse-demo-coord-app
fuse-demo-coord-build-app: export DOCKERFILE=hack/fuse-demo/coord-app.Dockerfile
fuse-demo-coord-build-app: export BUILD_ARGS=--build-arg VERSION=$(SIDECAR_BASE_VERSION)
fuse-demo-coord-build-app: export TAGS=latest $(RELEASE_TAG) $(USER_TAG) $(BRANCH_TAG)
fuse-demo-coord-build-app:
	$(MAKE) build-target
	$(MAKE) push-target

.PHONY: pg-demo-coord-build-app
## DEMO: Build shell container used in postgres demo: mock application using postgres
pg-demo-coord-build-app: export BUILD_TARGET=$(REPOSITORY)/datamon-pg-demo-coord-app
pg-demo-coord-build-app: export DOCKERFILE=hack/fuse-demo/coord-app-pg.Dockerfile
pg-demo-coord-build-app: export BUILD_ARGS=--build-arg VERSION=$(SIDECAR_BASE_VERSION)
pg-demo-coord-build-app: export TAGS=latest $(RELEASE_TAG) $(USER_TAG) $(BRANCH_TAG)
pg-demo-coord-build-app:
	$(MAKE) build-target
	$(MAKE) push-target

.PHONY: release-all-docker
## RELEASE: Push all containers to be released
release-all-docker: build-datamon build-migrate build-wrapper push-datamon push-migrate push-wrapper build-and-push-fuse-sidecar build-and-push-pg-sidecar

.PHONY: build-alpine-base
## RELEASE: Build local base datamon container on alpine linux
build-alpine-base: export BUILD_TARGET=$(LOCALREPO)/datamon-alpine-base
build-alpine-base: export DOCKERFILE=dockerfiles/alpine-base.Dockerfile
build-alpine-base: export BUILD_ARGS=$(VERSION_ARGS)
build-alpine-base: export TAGS=
build-alpine-base:
	$(MAKE) build-target

.PHONY: build-datamon
## RELEASE: Build datamon docker container to be released
build-datamon: export BUILD_TARGET=$(REPOSITORY)/datamon
build-datamon: export DOCKERFILE=dockerfiles/datamon.Dockerfile
build-datamon: export BUILD_ARGS=$(VERSION_ARGS)
build-datamon: export TAGS=latest $(RELEASE_TAG) $(RELEASE_TAG_LATEST) $(USER_TAG) $(BRANCH_TAG)
build-datamon: build-alpine-base
	$(MAKE) build-target

.PHONY: push-datamon
## RELEASE: Push released datamon container
push-datamon:
	docker push $(REPOSITORY)/datamon:${RELEASE_TAG}
	docker push $(REPOSITORY)/datamon:latest

.PHONY: build-migrate
## RELEASE: Build container for the migrate tool
build-migrate: export BUILD_TARGET=$(REPOSITORY)/datamon-migrate
build-migrate: export DOCKERFILE=dockerfiles/migrate.Dockerfile
build-migrate: export BUILD_ARGS=$(VERSION_ARGS)
build-migrate: export TAGS=latest $(RELEASE_TAG) $(RELEASE_TAG_LATEST) $(USER_TAG) $(BRANCH_TAG)
build-migrate: build-alpine-base
	$(MAKE) build-target

.PHONY: push-migrate
## RELEASE: Push released migrate container
push-migrate:
	docker push $(REPOSITORY)/datamon-migrate:${RELEASE_TAG}
	docker push $(REPOSITORY)/datamon-migrate:latest

.PHONY: build-wrapper
## RELEASE: Build application wrapper for ARGO workflows
build-wrapper: export BUILD_TARGET=$(REPOSITORY)/datamon-wrapper
build-wrapper: export DOCKERFILE=dockerfiles/wrapper.Dockerfile
build-wrapper: export BUILD_ARGS=
build-wrapper: export TAGS=latest $(RELEASE_TAG) $(RELEASE_TAG_LATEST) $(USER_TAG) $(BRANCH_TAG)
build-wrapper:
	$(MAKE) build-target

.PHONY: push-wrapper
## RELEASE: Push released wrapper container
push-wrapper:
	docker push $(REPOSITORY)/datamon-wrapper:${RELEASE_TAG}
	docker push $(REPOSITORY)/datamon-wrapper:latest

.PHONY: release
## RELEASE: Cut a new release. Example: make release TAG=v1.0.6
release:
	@if [[ -z "$(TAG)" ]] ; then echo "Must have arg: TAG=vX.Y.Z" && exit 1 ;fi
	@git tag -a -m "datamon release ${TAG}" ${TAG}
	git push origin ${TAG}
#
# NOTES:
# * atm git tags are not signed
# TODO:
# * [ ] bind this with git-chglog or similar
# * [ ] release dry-run on alternate upstream
