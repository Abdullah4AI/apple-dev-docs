.PHONY: build install clean test deps run validate-app validate-lines

BINARY_NAME=appledev
BUILD_DIR=./bin
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-s -w -X github.com/Abdullah4AI/apple-developer-toolkit/swiftship/internal/commands.Version=$(VERSION)

build:
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/appledev

install: build
	mkdir -p $(HOME)/.local/bin
	cp $(BUILD_DIR)/$(BINARY_NAME) $(HOME)/.local/bin/$(BINARY_NAME)

clean:
	rm -rf $(BUILD_DIR) dist

test:
	go test ./... -v

deps:
	go mod tidy

run:
	go run ./cmd/appledev $(ARGS)

validate-app:
	@if [ -z "$(PROJECT_DIR)" ] || [ -z "$(APP_NAME)" ]; then \
		echo "Usage: make validate-app PROJECT_DIR=<dir> APP_NAME=<name>"; \
		exit 1; \
	fi
	./scripts/validate-app.sh $(PROJECT_DIR) $(APP_NAME)

validate-lines:
	@if [ -z "$(PROJECT_DIR)" ]; then \
		echo "Usage: make validate-lines PROJECT_DIR=<dir>"; \
		exit 1; \
	fi
	./scripts/validate-line-limits.sh $(PROJECT_DIR)
