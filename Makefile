ROOT_DIR=$(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
SRC_DIR=$(ROOT_DIR)/src
MIGRATIONS_DIR=$(ROOT_DIR)/migrations
CONFIG_FILE=$(ROOT_DIR)/config.yml
BIN=$(ROOT_DIR)/bin/bot
REVISION=$(shell git describe --tags 2>/dev/null || git log --format="v0.0-%h" -n 1 || echo "v0.0-unknown")

fmt:
	@echo "==> Running gofmt"
	@gofmt -l -s -w $(SRC_DIR)

deps:
	@echo "==> Installing dependencies"
	@go mod tidy

build: deps fmt
	@echo "==> Building"
	@cd $(SRC_DIR) && CGO_ENABLED=0 go build -o $(BIN) -ldflags "-X common.build=${REVISION}" .
	@echo $(BIN)

migrate: build
	${BIN} --config $(CONFIG_FILE) migrate -p $(MIGRATIONS_DIR)

migrate_down: build
	@${BIN} --config $(CONFIG_FILE) migrate -v down

run: migrate
	@echo "==> Running"
	@${BIN} --config $(CONFIG_FILE) run

test: deps fmt
	@echo "==> Running tests"
	@cd $(SRC_DIR) && go test ./... -v -cpu 2 -cover -race
