APP_NAME ?= gopucha
IMAGE ?= golang:alpine
WORKDIR ?= /src

UNAME_S := $(shell uname -s | tr '[:upper:]' '[:lower:]')
UNAME_M := $(shell uname -m)

GOOS ?= $(UNAME_S)
GOARCH ?= $(if $(filter x86_64,$(UNAME_M)),amd64,$(if $(filter aarch64 arm64,$(UNAME_M)),arm64,$(UNAME_M)))

BIN_DIR := bin
BIN_PATH := $(BIN_DIR)/$(APP_NAME)

# Build flags for optimization
BUILD_FLAGS := -ldflags="-s -w" -trimpath

.PHONY: build build-optimized run test clean

# Standard build
build: $(BIN_DIR)
	go build -o "$(BIN_PATH)" ./cmd/$(APP_NAME)

# Optimized build with size reduction flags
build-optimized: $(BIN_DIR)
	go build $(BUILD_FLAGS) -o "$(BIN_PATH)" ./cmd/$(APP_NAME)

# Run the game with default map
run: build
	./$(BIN_PATH)

# Run unit tests
test:
	go test -v ./...

$(BIN_DIR):
	mkdir -p $(BIN_DIR)

clean:
	rm -rf $(BIN_DIR)
