# COLORS
GREEN  := $(shell tput -Txterm setaf 2)
YELLOW := $(shell tput -Txterm setaf 3)
WHITE  := $(shell tput -Txterm setaf 7)
RESET  := $(shell tput -Txterm sgr0)
GIT_BRANCH := $(shell git rev-parse --abbrev-ref HEAD)

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

.PHONY: build-tpt
## Build tpt docker container (tpt)
build-tpt:
	@echo 'building ${YELLOW}tpt${RESET} container'
	@docker build --pull -t reg.onec.co/tpt:$$(date '+%Y%m%d') -t reg.onec.co/tpt:$(subst /,_,$(GIT_BRANCH)) .

.PHONY: push-tpt
## Push tpt docker container
push-tpt:
	@docker push reg.onec.co/tpt

.PHONY: build-flexvoldrivers
## Build flexvoldrivers docker container (flexvoldrivers)
build-flexvoldrivers:
	@echo 'building ${YELLOW}flexvoldrivers${RESET} container'
	@docker build --pull -t reg.onec.co/flexvoldrivers:$$(date '+%Y%m%d') -t reg.onec.co/flexvoldrivers:$(subst /,_,$(GIT_BRANCH)) -f flexvoldrivers/Dockerfile .

.PHONY: push-flexvoldrivers
## Push flexvoldrivers docker container
push-flexvoldrivers:
	@docker push reg.onec.co/flexvoldrivers

.PHONY: build-all
## Build all the containers
build-all: build-tpt build-flexvoldrivers

.PHONE: push-all
## Push all the containers
push-all: push-tpt push-flexvoldrivers
