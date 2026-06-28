.PHONY: build build-server build-client run-server run-client test clean

BIN_DIR := bin

SERVER := $(BIN_DIR)/server
CLIENT := $(BIN_DIR)/client

# данные о сборке подставляются в бинарники Клиента и Сервера через ldflags
BUILD_VERSION ?= v0.0.1
BUILD_DATE ?= $(shell date +%Y-%m-%d)
BUILD_COMMIT ?= $(shell git rev-parse --short HEAD)

LDFLAGS := \
	-X main.buildVersion=$(BUILD_VERSION) \
	-X main.buildDate=$(BUILD_DATE) \
	-X main.buildCommit=$(BUILD_COMMIT)

# собрать Сервер и Клиент
build: build-server build-client

# собрать Сервер с информацией о сборке
build-server:
	@mkdir -p $(BIN_DIR)
	go build \
		-ldflags "$(LDFLAGS)" \
		-o $(SERVER) \
		./cmd/server

# собрать Клиент с информацией о сборке
build-client:
	@mkdir -p $(BIN_DIR)
	go build \
		-ldflags "$(LDFLAGS)" \
		-o $(CLIENT) \
		./cmd/client

# стартовать Сервер
run-server: build-server
	$(SERVER)

# стартовать Клиент
run-client: build-client
	$(CLIENT)

# запустить тесты
test:
	go test ./...

# удалить артефакты сборки
clean:
	rm -rf $(BIN_DIR)
