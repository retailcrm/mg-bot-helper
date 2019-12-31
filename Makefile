ROOT_DIR=$(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
SRC_DIR=$(ROOT_DIR)/src
MIGRATIONS_DIR=$(ROOT_DIR)/src/migrations
CONFIG_FILE=$(ROOT_DIR)/config.yml
CONFIG_TEST_FILE=$(ROOT_DIR)/config_test.yml
BIN=$(ROOT_DIR)/bin/bot
GO_BIN_DIR=$(shell go env GOPATH)/bin
REVISION=$(shell git describe --tags 2>/dev/null || git log --format="v0.0-%h" -n 1 || echo "v0.0-unknown")

build: packr_install deps fmt
	@echo "==> Building"
	@cd $(SRC_DIR) && $(GO_BIN_DIR)/packr2 && CGO_ENABLED=0 go build -o $(BIN) -ldflags "-X common.build=${REVISION}" .
	@cd $(SRC_DIR) && $(GO_BIN_DIR)/packr2 clean
	@echo $(BIN)

run: migrate
	@echo "==> Running"
	@${BIN} --config $(CONFIG_FILE) run

test: deps fmt
	@echo "==> Running tests"
	@cd $(SRC_DIR) && go test ./... -v -cpu 2 -cover -race

jenkins_test: migrate_test
	@echo "==> Running tests (result in test-report.xml)"
	@go get -v -u github.com/jstemmer/go-junit-report
	@cd $(SRC_DIR) && go test ./... -v -cpu 2 -cover -race | go-junit-report -set-exit-code > $(ROOT_DIR)/test-report.xml
	@echo "==> Cleanup dependencies"
	@go mod tidy

fmt:
	@echo "==> Running gofmt"
	@gofmt -l -s -w $(SRC_DIR)

deps:
	@echo "==> Installing dependencies"
	@go mod tidy

migration: transport_tool_install
	@$(GO_BIN_DIR)/transport-core-tool migration -d $(MIGRATIONS_DIR)

migrate: build
	${BIN} --config $(CONFIG_FILE) migrate

migrate_test: build
	@${BIN} --config $(CONFIG_TEST_FILE) migrate

migrate_down: build
	@${BIN} --config $(CONFIG_FILE) migrate -v down

transport_tool_install:
ifeq (, $(shell command -v transport-core-tool 2> /dev/null))
	@echo "==> Installing migration generator..."
	@go get -u github.com/retailcrm/mg-transport-core/cmd/transport-core-tool
endif

packr_install:
ifeq (, $(shell command -v packr2 2> /dev/null))
	@go get github.com/gobuffalo/packr/v2/packr2@v2.7.1
endif

