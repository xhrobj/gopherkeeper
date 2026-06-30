.PHONY: \
	show-coverage \
	gen-tls-certs \
	build build-server build-client \
	db-up db-down db-connect db-erase \
	run-server run-client build-client-cross \
	test-all test test-race test-integration test-coverage \
	vet lint ci \
	clean clean-gen

# брать локальные параметры из env-файла (если он есть)
ENV_FILE ?= .env

-include $(ENV_FILE)

# TLS-сертификаты для локальной разработки
TLS_CERT_DIR := .certs

#TLS_CA_CERT := $(TLS_CERT_DIR)/ca.pem
TLS_SERVER_CERT := $(TLS_CERT_DIR)/server.pem
TLS_SERVER_KEY := $(TLS_CERT_DIR)/server-key.pem

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

# каталоги для артефактов сборки и пути к бинарникам
BIN_DIR := bin
DIST_DIR := dist

SERVER := $(BIN_DIR)/server
CLIENT := $(BIN_DIR)/client

# команда Docker Compose с выбранным env-файлом
# !!!: для целей db-* и run-server требуется env-файл с переменными POSTGRES_*
# NOTE: создать локальный env-файл: cp .env.example .env
COMPOSE := docker compose --env-file $(ENV_FILE)

# параметры локального запуска Сервера и Клиента
ADDRESS ?= localhost:8080
DATABASE_DSN ?= postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable

# !!!: строка подключения к локальному PostgreSQL собирается из POSTGRES_* и передается Серверу через окружение
export DATABASE_DSN

# обновить профиль покрытия и вывести общий процент
show-coverage: test-coverage
	go tool cover -func=coverage.out | tail -n 1

# сгенерировать (при необходимости) локальный CA и TLS-сертификат Сервера
gen-tls-certs:
	./scripts/generate-tls-certs.sh

# собрать Сервер и Клиент
build: build-server build-client

# собрать Сервер
build-server:
	@mkdir -p $(BIN_DIR)
	go build \
		-ldflags "$(LDFLAGS)" \
		-o $(SERVER) \
		./cmd/server

# собрать Клиент
build-client:
	@mkdir -p $(BIN_DIR)
	go build \
		-ldflags "$(LDFLAGS)" \
		-o $(CLIENT) \
		./cmd/client

# собрать Клиент для Linux, Windows и macOS
build-client-cross:
	@mkdir -p $(DIST_DIR)
	GOOS=linux GOARCH=amd64 go build \
		-ldflags "$(LDFLAGS)" \
		-o $(DIST_DIR)/gopherkeeper-client-linux-amd64 \
		./cmd/client
	GOOS=windows GOARCH=amd64 go build \
		-ldflags "$(LDFLAGS)" \
		-o $(DIST_DIR)/gopherkeeper-client-windows-amd64.exe \
		./cmd/client
	GOOS=darwin GOARCH=amd64 go build \
		-ldflags "$(LDFLAGS)" \
		-o $(DIST_DIR)/gopherkeeper-client-darwin-amd64 \
		./cmd/client
	GOOS=darwin GOARCH=arm64 go build \
		-ldflags "$(LDFLAGS)" \
		-o $(DIST_DIR)/gopherkeeper-client-darwin-arm64 \
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
run-server: gen-tls-certs build-server
	$(SERVER) \
		-a $(ADDRESS) \
		--tls-cert $(TLS_SERVER_CERT) \
		--tls-key $(TLS_SERVER_KEY)

# стартовать Клиент
run-client: build-client
	$(CLIENT) -a $(ADDRESS)

# запустить полный набор тестов, в конце показать покрытие
# !!!: для интеграционных тестов требуется запущенный Docker
test-all: test-race test-integration show-coverage

# запустить обычные тесты
test:
	go test ./...

# запустить тесты с детектором гонок данных
test-race:
	go test -race ./...

# запустить интеграционные тесты с локальным PostgreSQL
test-integration:
	$(COMPOSE) up -d --wait postgres
	go test -tags=integration -count=1 ./...

# запустить тесты и сохранить атомарный профиль покрытия
test-coverage:
	go test -covermode=atomic -coverprofile=coverage.out ./...

# выполнить стандартный статический анализ Go-кода
vet:
	go vet ./...

# проверить проект набором линтеров golangci-lint
lint:
	golangci-lint run ./...

# собрать проект и выполнить полный набор CI-проверок
ci: build test-race vet lint

# очистить артефакты сборки и coverage
clean:
	rm -rf $(BIN_DIR) $(DIST_DIR) coverage.out

# удалить сгенерированные TLS-сертификаты
clean-gen:
	rm -rf $(TLS_CERT_DIR)
