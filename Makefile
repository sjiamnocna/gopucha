APP_NAME ?= gopucha
IMAGE ?= golang:alpine
WORKDIR ?= /src

UNAME_S := $(shell uname -s | tr '[:upper:]' '[:lower:]')
UNAME_M := $(shell uname -m)

GOOS ?= $(UNAME_S)
GOARCH ?= $(if $(filter x86_64,$(UNAME_M)),amd64,$(if $(filter aarch64 arm64,$(UNAME_M)),arm64,$(UNAME_M)))

BIN_DIR := bin
BIN_PATH := $(BIN_DIR)/$(APP_NAME)

.PHONY: build clean

build: $(BIN_DIR)
	docker run --rm \
		-e GOOS=$(GOOS) \
		-e GOARCH=$(GOARCH) \
			-v "$(CURDIR)":"$(WORKDIR)" \
			-v "$(CURDIR)/$(BIN_DIR)":"$(WORKDIR)/$(BIN_DIR)" \
		-w "$(WORKDIR)" \
		$(IMAGE) \
		go build -o "$(BIN_PATH)" ./cmd/$(APP_NAME)

$(BIN_DIR):
	mkdir -p $(BIN_DIR)

clean:
	rm -rf $(BIN_DIR)
