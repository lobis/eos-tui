APP_NAME := eos-tui
CMD_PATH := .
BIN_DIR := ./bin
LOCAL_BIN := $(BIN_DIR)/$(APP_NAME)
LINUX_BIN := $(BIN_DIR)/$(APP_NAME)-linux-amd64

GO ?= go
GOOS ?= linux
GOARCH ?= amd64
GO_BUILD_FLAGS ?= -a

REMOTE_HOST ?= lobis-eos-dev
REMOTE_HOST_SECONDARY ?= eospilot
REMOTE_DIR ?= /root
REMOTE_BIN ?= $(REMOTE_DIR)/$(APP_NAME)
REMOTE_TMP ?= $(REMOTE_BIN).new
REMOTE_ARGS ?=
EOSPILOT ?=
SSH_OPTS ?= -o LogLevel=ERROR
SCP_OPTS ?= -o LogLevel=ERROR

.PHONY: build build-local test deploy-remote deploy-both deploy-remote-eospilot deploy-eospilot run-remote dev-remote smoke-remote clean help

help:
	@echo "Available targets:"
	@echo "  build           - Build linux-amd64 binary"
	@echo "  build-local     - Build local binary"
	@echo "  test            - Run tests"
	@echo "  fmt             - Format code"
	@echo "  deploy-remote   - Deploy to dev host"
	@echo "  deploy-eospilot - Deploy to eospilot host"
	@echo "  clean           - Remove build artifacts"

build:
	mkdir -p $(BIN_DIR)
	GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build $(GO_BUILD_FLAGS) -o $(LINUX_BIN) $(CMD_PATH)

build-local:
	mkdir -p $(BIN_DIR)
	$(GO) build $(GO_BUILD_FLAGS) -o $(LOCAL_BIN) $(CMD_PATH)

test:
	$(GO) test ./...

fmt:
	$(GO) fmt ./...

deploy-remote: build
	ssh $(SSH_OPTS) $(REMOTE_HOST) 'mkdir -p $(REMOTE_DIR)'
	scp $(SCP_OPTS) $(LINUX_BIN) $(REMOTE_HOST):$(REMOTE_TMP)
	ssh $(SSH_OPTS) $(REMOTE_HOST) 'install -m 0755 $(REMOTE_TMP) $(REMOTE_BIN) && rm -f $(REMOTE_TMP)'

deploy-both: build
	ssh $(SSH_OPTS) $(REMOTE_HOST) 'mkdir -p $(REMOTE_DIR)'
	scp $(SCP_OPTS) $(LINUX_BIN) $(REMOTE_HOST):$(REMOTE_TMP)
	ssh $(SSH_OPTS) $(REMOTE_HOST) 'install -m 0755 $(REMOTE_TMP) $(REMOTE_BIN) && rm -f $(REMOTE_TMP)'
	ssh $(SSH_OPTS) $(REMOTE_HOST_SECONDARY) 'mkdir -p $(REMOTE_DIR)'
	scp $(SCP_OPTS) $(LINUX_BIN) $(REMOTE_HOST_SECONDARY):$(REMOTE_TMP)
	ssh $(SSH_OPTS) $(REMOTE_HOST_SECONDARY) 'install -m 0755 $(REMOTE_TMP) $(REMOTE_BIN) && rm -f $(REMOTE_TMP)'

deploy-eospilot: build
	ssh $(SSH_OPTS) $(REMOTE_HOST_SECONDARY) 'mkdir -p $(REMOTE_DIR)'
	scp $(SCP_OPTS) $(LINUX_BIN) $(REMOTE_HOST_SECONDARY):$(REMOTE_TMP)
	ssh $(SSH_OPTS) $(REMOTE_HOST_SECONDARY) 'install -m 0755 $(REMOTE_TMP) $(REMOTE_BIN) && rm -f $(REMOTE_TMP)'

run-remote:
	ssh $(SSH_OPTS) -tt $(REMOTE_HOST) 'TERM=$${TERM:-xterm-256color} $(REMOTE_BIN) $(REMOTE_ARGS)'

dev-remote: deploy-remote run-remote

smoke-remote: deploy-remote
	ssh $(SSH_OPTS) -tt $(REMOTE_HOST) 'TERM=$${TERM:-xterm-256color} timeout 3 $(REMOTE_BIN) $(REMOTE_ARGS)' || test $$? -eq 124

clean:
	rm -rf $(BIN_DIR)
	rm -f eos-tui
