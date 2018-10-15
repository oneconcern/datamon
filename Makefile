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
LOCAL_KUBECTX ?= "docker-for-desktop"

TARGET_MAX_CHAR_NUM=25
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

.PHONY: build-datamon
## Build datamon docker container (datamon)
build-datamon:
	@echo 'building ${YELLOW}datamon${RESET} container'
	@docker build --pull --build-arg github_user=$(GITHUB_USER) --build-arg github_token=$(GITHUB_TOKEN) -t reg.onec.co/datamon:$$(date '+%Y%m%d') -t reg.onec.co/datamon:$(subst /,_,$(GIT_BRANCH)) .

.PHONY: push-datamon
## Push datamon docker container
push-datamon:
	@docker push reg.onec.co/datamon

.PHONY: build-all
## Build all the containers
build-all: build-datamon

.PHONY: push-all
## Push all the containers
push-all: push-datamon

.PHONY: setup
## Setup for local development
setup: install-minio

.PHONY: install-minio
## Install minio in local kubernetes
install-minio:
	kubectl --context $(LOCAL_KUBECTX) create -f ./k8s/minio.yaml
