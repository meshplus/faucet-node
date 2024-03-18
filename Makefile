
SHELL := /bin/bash
CURRENT_PATH = $(shell pwd)
APP_NAME = faucet

# build with verison infos
BUILD_CONST_DIR = github.com/axiomesh/${APP_NAME}/pkg/repo
BUILD_DATE = $(shell date +%FT%T)
GIT_COMMIT = $(shell git log --pretty=format:'%h' -n 1)
GIT_BRANCH = $(shell git rev-parse --abbrev-ref HEAD)

ifeq ($(version),)
	# not specify version: make install
	APP_VERSION = $(shell git describe --abbrev=0 --tag)
	ifeq ($(APP_VERSION),)
		APP_VERSION = dev
	endif
else
	# specify version: make install version=v0.6.1-dev
	APP_VERSION = $(version)
endif


GOLDFLAGS += -X "${BUILD_CONST_DIR}.BuildDate=${BUILD_DATE}"
GOLDFLAGS += -X "${BUILD_CONST_DIR}.BuildCommit=${GIT_COMMIT}"
GOLDFLAGS += -X "${BUILD_CONST_DIR}.BuildBranch=${GIT_BRANCH}"
GOLDFLAGS += -X "${BUILD_CONST_DIR}.BuildVersion=${APP_VERSION}"

STATIC_LDFLAGS += ${GOLDFLAGS}
STATIC_LDFLAGS += -linkmode external -extldflags -static

GO = GO111MODULE=on go
TEST_PKGS := $(shell $(GO) list ./... | grep -v 'cmd' | grep -v 'mock_*' | grep -v 'proto' | grep -v 'imports' | grep -v 'internal/app' | grep -v 'api')

RED=\033[0;31m
GREEN=\033[0;32m
BLUE=\033[0;34m
NC=\033[0m

GOARCH := $(or $(GOARCH),$(shell go env GOARCH))
GOOS := $(or $(GOOS),$(shell go env GOOS))

.PHONY: test

help: Makefile
	@echo "Choose a command run:"
	@sed -n 's/^##//p' $< | column -t -s ':' | sed -e 's/^/ /'

## make test: Run go unittest
test:
	go generate ./...
	@$(GO) test ${TEST_PKGS} -count=1

## make install: Go install the project (hpc)
install:
	$(GO) install -ldflags '${GOLDFLAGS}' ./cmd/${APP_NAME}
	@printf "${GREEN}Build Faucet successfully${NC}\n"

build:
	@mkdir -p bin
	$(GO) build -ldflags '${GOLDFLAGS}' ./cmd/${APP_NAME}
	@mv ./faucet bin
	@printf "${GREEN}Build Faucet successfully!${NC}\n"

package:build
	cp -f bin/${APP_NAME} ./scripts/package/tools/bin/${APP_NAME}
	tar czvf ./${APP_NAME}-${APP_VERSION}-${GOARCH}-${GOOS}.tar.gz -C ./scripts/package/ .
