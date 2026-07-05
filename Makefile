.PHONY: \
	show-coverage \
	gen-tls-certs gen-jwt-secret \
	check-client-ca \
	build build-server build-client build-client-cross \
	db-up db-down db-connect db-erase \
	run-server run-client \
	run-client-health \
	run-client-register run-client-login run-client-whoami \
	test-all test test-race test-integration \
	coverage \
	vet lint ci \
	clean clean-gen

# брать локальные параметры из env-файла (если он есть)
ENV_FILE ?= .env

-include $(ENV_FILE)

# TLS-сертификаты для локальной разработки
TLS_CERT_DIR := .certs

TLS_CA_CERT := $(TLS_CERT_DIR)/ca.pem
TLS_SERVER_CERT := $(TLS_CERT_DIR)/server.pem
TLS_SERVER_KEY := $(TLS_CERT_DIR)/server-key.pem

# JWT-параметры локального запуска Сервера читаются из env-файла
# и передаются Серверу через окружение.
export JWT_SECRET
export JWT_TTL

# путь к локальному файлу online-сессии Клиента можно задать через env-файл.
export SESSION_FILE

# параметры логирования
LOG_LEVEL ?= info
export LOG_LEVEL

# данные о сборке подставляются в бинарники Клиента и Сервера через ldflags
BUILD_VERSION ?= v0.3.0
BUILD_DATE ?= $(shell date +%Y-%m-%d)
BUILD_COMMIT ?= $(shell git rev-parse --short HEAD)

LDFLAGS := \
	-X main.buildVersion=$(BUILD_VERSION) \
	-X main.buildDate=$(BUILD_DATE) \
	-X main.buildCommit=$(BUILD_COMMIT)

# каталоги для артефактов сборки и пути к бинарникам
BIN_DIR := bin
DIST_DIR := dist

SERVER_NAME := gopherkeeper-server
CLIENT_NAME := gkeep

SERVER := $(BIN_DIR)/$(SERVER_NAME)
CLIENT := $(BIN_DIR)/$(CLIENT_NAME)

# команда Docker Compose с выбранным env-файлом
# !!!: для целей db-*, run-server, test-integration, coverage, test-all и ci
# требуется env-файл с переменными POSTGRES_*
# NOTE: создать локальный env-файл: `cp .env.example .env`
COMPOSE := docker compose --env-file $(ENV_FILE)

# параметры локального запуска Сервера и Клиента
ADDRESS ?= localhost:8080
DATABASE_DSN ?= postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable

# !!!: строка подключения к локальному PostgreSQL собирается из POSTGRES_* и передается Серверу через окружение
export DATABASE_DSN

# обновить профиль покрытия и вывести общий процент
show-coverage: coverage
	go tool cover -func=coverage.out | tail -n 1

# сгенерировать (при необходимости) локальный CA и TLS-сертификат Сервера
gen-tls-certs:
	./scripts/generate-tls-certs.sh

# сгенерировать случайный JWT secret для локальной разработки
# значение нужно скопировать в JWT_SECRET локального .env-файла
gen-jwt-secret:
	@openssl rand -base64 32

check-client-ca:
	@if [ ! -f "$(TLS_CA_CERT)" ]; then \
		echo "(+_+) CA certificate not found: $(TLS_CA_CERT)"; \
		echo "1. start the local Server first with 'make run-server'"; \
		echo "2. or copy the Server CA certificate and set TLS_CA_CERT"; \
		exit 1; \
	fi

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
# и дождаться его готовности
db-up:
	$(COMPOSE) up -d --wait postgres

# остановить и удалить контейнер PostgreSQL без удаления данных
db-down:
	$(COMPOSE) down

# подключиться к PostgreSQL через psql
db-connect:
	$(COMPOSE) exec postgres psql -U $(POSTGRES_USER) -d $(POSTGRES_DB)

# удалить контейнер PostgreSQL и локальные данные
db-erase:
	$(COMPOSE) down -v

# собрать и запустить Сервер с локальным PostgreSQL
# !!!: требуется:
# 1. env-файл с переменными POSTGRES_*
# 2. запущенный Docker
run-server: db-up gen-tls-certs build-server
	$(SERVER) \
		-a $(ADDRESS) \
		--tls-cert $(TLS_SERVER_CERT) \
		--tls-key $(TLS_SERVER_KEY)

# собрать и запустить Клиент с выводом общей справки
run-client: build-client
	$(CLIENT)

# запустить Клиент и выполнить health-запрос к Серверу
run-client-health: check-client-ca
	$(CLIENT) health \
		-a $(ADDRESS) \
		--ca-cert $(TLS_CA_CERT)

# запустить Клиент для регистрации пользователя
# LOGIN нужно передать через окружение или командную строку make
# примеры:
# - `LOGIN=bob make run-client-register`
# - `make run-client-register LOGIN=bob`
run-client-register: check-client-ca
	$(CLIENT) register \
		--login $(LOGIN) \
		-a $(ADDRESS) \
		--ca-cert $(TLS_CA_CERT)

# запустить Клиент для входа пользователя
# LOGIN нужно передать через окружение или командную строку make
run-client-login: check-client-ca
	$(CLIENT) login \
		--login $(LOGIN) \
		-a $(ADDRESS) \
		--ca-cert $(TLS_CA_CERT)

# запустить Клиент и вывести текущего пользователя online-сессии
run-client-whoami: check-client-ca
	$(CLIENT) whoami \
		-a $(ADDRESS) \
		--ca-cert $(TLS_CA_CERT)

# запустить полный набор тестов
# !!!: для интеграционных тестов требуется запущенный Docker
test-all: test-race test-integration

# запустить обычные тесты
test:
	go test ./...

# запустить тесты с детектором гонок данных
test-race:
	go test -race ./...

# запустить интеграционные тесты с локальным PostgreSQL
test-integration: db-up
	go test -tags=integration -count=1 ./...

# запустить обычные и интеграционные тесты
# и сохранить атомарный профиль покрытия всего проекта
coverage: db-up
	go test \
		-count=1 \
		-tags=integration \
		-coverpkg=./... \
		-covermode=atomic \
		-coverprofile=coverage.out \
		./...

# выполнить стандартный статический анализ Go-кода
vet:
	go vet ./...

# проверить проект набором линтеров golangci-lint
lint:
	golangci-lint run ./...

# собрать проект и выполнить полный набор CI-проверок
ci: build test-all vet lint

# очистить артефакты сборки и coverage
clean:
	rm -rf $(BIN_DIR) $(DIST_DIR) coverage.out

# удалить сгенерированные TLS-сертификаты
clean-gen:
	rm -rf $(TLS_CERT_DIR)
