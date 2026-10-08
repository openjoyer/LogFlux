GO ?= go
PROTOC ?= protoc
PROTOC_GEN_GO_VERSION ?= v1.36.6
PROTOC_GEN_GO_GRPC_VERSION ?= v1.5.1
PROTO_TOOLS_DIR := $(CURDIR)/bin/proto-tools
PROTO_FILES := $(shell find contracts/proto -name '*.proto' -type f | sort)
MODULES := contracts services/ingestion services/logs services/projects services/alerting

.PHONY: build test vet tidy fmt check proto-install proto

build:
	mkdir -p bin
	cd services/ingestion && GOWORK=off $(GO) build -mod=readonly -o ../../bin/ingestion ./cmd/server
	cd services/logs && GOWORK=off $(GO) build -mod=readonly -o ../../bin/logs ./cmd/server

test:
	@set -e; for module in $(MODULES); do \
	  echo "Testing $$module"; \
	  (cd $$module && packages=$$(GOWORK=off $(GO) list ./...) && \
	    if [ -n "$$packages" ]; then GOWORK=off $(GO) test -mod=readonly -race -count=1 -timeout=3m ./...; fi); \
	done

vet:
	@set -e; for module in $(MODULES); do \
	  (cd $$module && packages=$$(GOWORK=off $(GO) list ./...) && \
	    if [ -n "$$packages" ]; then GOWORK=off $(GO) vet -mod=readonly ./...; fi); \
	done

tidy:
	@set -e; for module in $(MODULES); do (cd $$module && GOWORK=off $(GO) mod tidy); done

fmt:
	$(GO) fmt ./contracts/... ./services/ingestion/... ./services/logs/...

check: vet test build

proto-install:
	@command -v $(GO) >/dev/null 2>&1 || { echo "Go not found; install Go or set GO=/path/to/go"; exit 1; }
	@if ! command -v $(PROTOC) >/dev/null 2>&1; then \
		command -v brew >/dev/null 2>&1 || { echo "Install protoc or Homebrew, then retry"; exit 1; }; \
		brew install protobuf; \
	fi
	@$(PROTOC) --version
	mkdir -p "$(PROTO_TOOLS_DIR)"
	GOBIN="$(PROTO_TOOLS_DIR)" GOWORK=off $(GO) install google.golang.org/protobuf/cmd/protoc-gen-go@$(PROTOC_GEN_GO_VERSION)
	GOBIN="$(PROTO_TOOLS_DIR)" GOWORK=off $(GO) install google.golang.org/grpc/cmd/protoc-gen-go-grpc@$(PROTOC_GEN_GO_GRPC_VERSION)

proto: proto-install
	@test -n "$(PROTO_FILES)" || { echo "No .proto files found in contracts/proto"; exit 1; }
	mkdir -p contracts/gen/go
	$(PROTOC) -I contracts/proto \
		--plugin=protoc-gen-go="$(PROTO_TOOLS_DIR)/protoc-gen-go" \
		--plugin=protoc-gen-go-grpc="$(PROTO_TOOLS_DIR)/protoc-gen-go-grpc" \
		--go_out=contracts/gen/go --go_opt=paths=source_relative \
		--go-grpc_out=contracts/gen/go --go-grpc_opt=paths=source_relative \
		$(PROTO_FILES)
