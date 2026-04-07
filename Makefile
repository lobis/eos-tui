APP_NAME := eos-tui
CMD_PATH := ./cmd/eostui
BIN_DIR := ./bin
LOCAL_BIN := $(BIN_DIR)/$(APP_NAME)
LINUX_BIN := $(BIN_DIR)/$(APP_NAME)-linux-amd64

GO ?= go
GOOS ?= linux
GOARCH ?= amd64
GO_BUILD_FLAGS ?= -a

REMOTE_HOST ?= lobis-eos-dev
REMOTE_HOST_SECONDARY ?= eospilot
REMOTE_DIR ?= /root/lobisapa
REMOTE_BIN ?= $(REMOTE_DIR)/$(APP_NAME)
REMOTE_TMP ?= $(REMOTE_BIN).new
REMOTE_ARGS ?=
EOSPILOT ?=

.PHONY: build build-local test deploy-remote deploy-both deploy-remote-eospilot run-remote dev-remote smoke-remote clean

build:
	mkdir -p $(BIN_DIR)
	GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build $(GO_BUILD_FLAGS) -o $(LINUX_BIN) $(CMD_PATH)

build-local:
	mkdir -p $(BIN_DIR)
	$(GO) build $(GO_BUILD_FLAGS) -o $(LOCAL_BIN) $(CMD_PATH)

test:
	$(GO) test ./...

deploy-remote: build
	ssh $(REMOTE_HOST) 'mkdir -p $(REMOTE_DIR)'
	scp $(LINUX_BIN) $(REMOTE_HOST):$(REMOTE_TMP)
	ssh $(REMOTE_HOST) 'install -m 0755 $(REMOTE_TMP) $(REMOTE_BIN) && rm -f $(REMOTE_TMP)'

deploy-both: build
	ssh $(REMOTE_HOST) 'mkdir -p $(REMOTE_DIR)'
	scp $(LINUX_BIN) $(REMOTE_HOST):$(REMOTE_TMP)
	ssh $(REMOTE_HOST) 'install -m 0755 $(REMOTE_TMP) $(REMOTE_BIN) && rm -f $(REMOTE_TMP)'
	ssh $(REMOTE_HOST_SECONDARY) 'mkdir -p $(REMOTE_DIR)'
	scp $(LINUX_BIN) $(REMOTE_HOST_SECONDARY):$(REMOTE_TMP)
	ssh $(REMOTE_HOST_SECONDARY) 'install -m 0755 $(REMOTE_TMP) $(REMOTE_BIN) && rm -f $(REMOTE_TMP)'

deploy-remote-eospilot: build
	test -n "$(EOSPILOT)"
	$(EOSPILOT) $(LINUX_BIN) $(REMOTE_HOST):$(REMOTE_TMP)
	ssh $(REMOTE_HOST) 'install -m 0755 $(REMOTE_TMP) $(REMOTE_BIN) && rm -f $(REMOTE_TMP)'

run-remote:
	ssh -tt $(REMOTE_HOST) 'TERM=$${TERM:-xterm-256color} $(REMOTE_BIN) $(REMOTE_ARGS)'

dev-remote: deploy-remote run-remote

smoke-remote: deploy-remote
	ssh -tt $(REMOTE_HOST) 'TERM=$${TERM:-xterm-256color} timeout 3 $(REMOTE_BIN) $(REMOTE_ARGS)' || test $$? -eq 124

clean:
	rm -f $(LOCAL_BIN) $(LINUX_BIN)
