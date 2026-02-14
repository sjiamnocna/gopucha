APP_NAME ?= gopucha
GO ?= go

UNAME_S := $(shell uname -s | tr '[:upper:]' '[:lower:]')
UNAME_M := $(shell uname -m)

GOOS ?= $(UNAME_S)
GOARCH ?= $(if $(filter x86_64,$(UNAME_M)),amd64,$(if $(filter aarch64 arm64,$(UNAME_M)),arm64,$(UNAME_M)))
CGO_ENABLED ?= 1

export GOOS
export GOARCH
export CGO_ENABLED

BIN_DIR := bin
BIN_PATH := $(BIN_DIR)/$(APP_NAME)

# Build flags for optimization
BUILD_FLAGS := -ldflags="-s -w" -trimpath

.PHONY: build build-optimized run dev test clean

# Standard build
build: $(BIN_DIR)
	$(GO) build -o "$(BIN_PATH)" ./cmd/$(APP_NAME)

run: build
	./$(BIN_PATH)

dev:
	$(GO) run ./cmd/$(APP_NAME) -no-monsters

# Optimized build with size reduction flags
build-optimized: $(BIN_DIR)
	$(GO) build $(BUILD_FLAGS) -o "$(BIN_PATH)" ./cmd/$(APP_NAME)

# Run unit tests
test:
	$(GO) test -v ./...

$(BIN_DIR):
	mkdir -p $(BIN_DIR)

clean:
	rm -rf $(BIN_DIR)
