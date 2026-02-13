APP_NAME ?= gopucha
IMAGE ?= golang:1.26-alpine
WORKDIR ?= /src
GO_CMD ?= /usr/local/go/bin/go

UNAME_S := $(shell uname -s | tr '[:upper:]' '[:lower:]')
UNAME_M := $(shell uname -m)
USER_ID := $(shell id -u)
GROUP_ID := $(shell id -g)

GOOS ?= $(UNAME_S)
GOARCH ?= $(if $(filter x86_64,$(UNAME_M)),amd64,$(if $(filter aarch64 arm64,$(UNAME_M)),arm64,$(UNAME_M)))

BIN_DIR := bin
BIN_PATH := $(BIN_DIR)/$(APP_NAME)

# Build flags for optimization
BUILD_FLAGS := -ldflags="-s -w" -trimpath

.PHONY: build build-optimized run test clean

DEPS_CMD := apk add --no-cache \
		build-base pkgconf mesa-dev xorgproto \
		libx11-dev libxcursor-dev libxrandr-dev libxinerama-dev libxi-dev

# Standard build
build: $(BIN_DIR)
	docker run --rm \
		-e GOOS=$(GOOS) \
		-e GOARCH=$(GOARCH) \
		-e CGO_ENABLED=1 \
		-e GOCACHE=/tmp/gocache \
		-e GOMODCACHE=/tmp/gomod \
		-v "$(CURDIR)":"$(WORKDIR)" \
		-v "$(CURDIR)/$(BIN_DIR)":"$(WORKDIR)/$(BIN_DIR)" \
		-w "$(WORKDIR)" \
		$(IMAGE) \
		sh -lc "$(DEPS_CMD) && $(GO_CMD) build -tags gui -o \"$(BIN_PATH)\" ./cmd/$(APP_NAME) && chown -R $(USER_ID):$(GROUP_ID) \"$(BIN_DIR)\""

# Optimized build with size reduction flags
build-optimized: $(BIN_DIR)
	docker run --rm \
		-e GOOS=$(GOOS) \
		-e GOARCH=$(GOARCH) \
		-e CGO_ENABLED=1 \
		-e GOCACHE=/tmp/gocache \
		-e GOMODCACHE=/tmp/gomod \
		-v "$(CURDIR)":"$(WORKDIR)" \
		-v "$(CURDIR)/$(BIN_DIR)":"$(WORKDIR)/$(BIN_DIR)" \
		-w "$(WORKDIR)" \
		$(IMAGE) \
		sh -lc "$(DEPS_CMD) && $(GO_CMD) build $(BUILD_FLAGS) -tags gui -o \"$(BIN_PATH)\" ./cmd/$(APP_NAME) && chown -R $(USER_ID):$(GROUP_ID) \"$(BIN_DIR)\""
# Run the game with default map
run: build
	./$(BIN_PATH)

# Run unit tests
test:
	docker run --rm \
		-e GOCACHE=/tmp/gocache \
		-e GOMODCACHE=/tmp/gomod \
		-v "$(CURDIR)":"$(WORKDIR)" \
		-w "$(WORKDIR)" \
		$(IMAGE) \
		sh -lc "$(DEPS_CMD) && $(GO_CMD) test -v ./..."

$(BIN_DIR):
	mkdir -p $(BIN_DIR)

clean:
	rm -rf $(BIN_DIR)
