-include .env
export
SHELL := /bin/bash
#DEBUG := --debug
#VERBOSE := --verbose
APP_NAME := goNginxWeightChanger
OS := linux

# colors
GREEN = $(shell tput -Txterm setaf 2)
YELLOW = $(shell tput -Txterm setaf 3)
WHITE = $(shell tput -Txterm setaf 7)
RESET = $(shell tput -Txterm sgr0)
GRAY = $(shell tput -Txterm setaf 6)
TARGET_MAX_CHAR_NUM = 30

.EXPORT_ALL_VARIABLES:

.PHONY: build
all: help
## Show env vars

#foreach:
#	@$(foreach var,$(ENV_NAME), echo "\n\n======= Check $(var) =======\n\n" \
#	&& cmd $($(var)_NAMESPACE) $(CHART_PATH) $($(var)_VALUES) $($(var)_VALUES_SET) || exit;)

mod-tidy:
	go mod init main 2>/dev/null || exit 0;
	go mod tidy

run:
	go run ./

build: clean mod-tidy
	mkdir -p ./build 2>/dev/null || exit 0;
	go build -o build/$(APP_NAME) ./

clean:
	rm *.tar.xz 2> /dev/null || exit 0
	rm -rf ./build 2> /dev/null || exit 0

## Shows help. | Help
help:
	@echo ''
	@echo 'Usage:'
	@echo ''
	@echo '  ${YELLOW}make${RESET} ${GREEN}<target>${RESET}'
	@echo ''
	@echo 'Targets:'
	@awk '/^[a-zA-Z0-9\-_]+:/ { \
		helpMessage = match(lastLine, /^## (.*)/); \
		if (helpMessage) { \
		    if (index(lastLine, "|") != 0) { \
				stage = substr(lastLine, index(lastLine, "|") + 1); \
				printf "\n ${GRAY}%s: \n\n", stage;  \
			} \
			helpCommand = substr($$1, 0, index($$1, ":")-1); \
			helpMessage = substr(lastLine, RSTART + 3, RLENGTH); \
			if (index(lastLine, "|") != 0) { \
				helpMessage = substr(helpMessage, 0, index(helpMessage, "|")-1); \
			} \
			printf "  ${YELLOW}%-$(TARGET_MAX_CHAR_NUM)s${RESET} ${GREEN}%s${RESET}\n", helpCommand, helpMessage; \
		} \
	} \
	{ lastLine = $$0 }' $(MAKEFILE_LIST)
	@echo ''