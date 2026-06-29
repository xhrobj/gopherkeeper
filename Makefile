.PHONY: \
	build build-server build-client \
	db-up db-down db-connect db-erase \
	run-server run-client \
	test \
	clean

# брать локальные параметры из env-файла (если он есть)
ENV_FILE ?= .env

-include $(ENV_FILE)

# параметры логирования
LOG_LEVEL ?= info
export LOG_LEVEL

# данные о сборке подставляются в бинарники Клиента и Сервера через ldflags
BUILD_VERSION ?= v0.0.1
BUILD_DATE ?= $(shell date +%Y-%m-%d)
BUILD_COMMIT ?= $(shell git rev-parse --short HEAD)

LDFLAGS := \
	-X main.buildVersion=$(BUILD_VERSION) \
	-X main.buildDate=$(BUILD_DATE) \
	-X main.buildCommit=$(BUILD_COMMIT)

# каталог для артефактов сборки и пути к бинарникам
BIN_DIR := bin

SERVER := $(BIN_DIR)/server
CLIENT := $(BIN_DIR)/client

# команда Docker Compose с выбранным env-файлом
# !!!: для целей db-* требуется env-файл; для других команд он опционален
# NOTE: создать локальный env-файл: cp .env.example .env
COMPOSE := docker compose --env-file $(ENV_FILE)

# параметры локального запуска Сервера и Клиента
ADDRESS ?= localhost:8080

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

# создать (при необходимости) и запустить локальный PostgreSQL
db-up:
	$(COMPOSE) up -d postgres

# остановить и удалить контейнер PostgreSQL без удаления данных
db-down:
	$(COMPOSE) down

# подключиться к PostgreSQL через psql
db-connect:
	$(COMPOSE) exec postgres psql -U $(POSTGRES_USER) -d $(POSTGRES_DB)

# удалить контейнер PostgreSQL и локальные данные
db-erase:
	$(COMPOSE) down -v

# стартовать Сервер
run-server: build-server
	$(SERVER) -a $(ADDRESS)

# стартовать Клиент
run-client: build-client
	$(CLIENT) -a $(ADDRESS)

# запустить тесты
test:
	go test ./...

# удалить артефакты сборки
clean:
	rm -rf $(BIN_DIR)
