# 🔐 (^-^)/ GophKeeper. Менеджер паролей и приватных данных

[![(-_-) Go CI](https://github.com/xhrobj/gopherkeeper/actions/workflows/go-ci.yml/badge.svg)](https://github.com/xhrobj/gopherkeeper/actions/workflows/go-ci.yml)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=xhrobj_gophkeeper&metric=alert_status)](https://sonarcloud.io/summary/new_code?id=xhrobj_gophkeeper)
[![Coverage](https://sonarcloud.io/api/project_badges/measure?project=xhrobj_gophkeeper&metric=coverage)](https://sonarcloud.io/summary/new_code?id=xhrobj_gophkeeper)

[Техническое задание](SPECIFICATION.md) · [OpenAPI](api/openapi.yaml)

## Текущее поведение

- `client`, `client -h`, `client --help`, `client help` — баннер и общая справка;
- `client health --help`, `client help health` — справка команды `health` без баннера;
- `client -v`, `client --version` — баннер и полная информация о сборке;
- `client health` — только результат команды;
- `client register` — регистрация пользователя;
- `client login` — вход пользователя и сохранение локальной online-сессии;
- `client whoami` — проверка текущего пользователя по сохранённой online-сессии.

## Локальный запуск с нуля

### 1. Клонировать репозиторий

```bash
git clone https://github.com/xhrobj/gopherkeeper.git
cd gopherkeeper
```

### 2. Собрать Сервер и Клиент

```bash
make build
```

Команда соберёт бинарники в каталог `bin/`:

```text
bin/gopherkeeper-server
bin/gkeep
```

### 3. Сгенерировать локальные TLS-сертификаты

```bash
make gen-tls-certs
```

Команда создаёт локальный CA certificate и server certificate в каталоге `.certs/`. Для локального self-signed TLS Клиенту нужен файл `.certs/ca.pem`.

### 4. Настроить JSON-конфиг Клиента

Создать локальный конфиг Клиента:

```bash
cp configs/client.example.json configs/client.json
```

Указать в `configs/client.json` адрес Сервера и путь к локальному CA certificate:

```json
{
  "address": "localhost:8080",
  "ca_cert_file": ".certs/ca.pem",
  "session_file": ""
}
```

Если `session_file` оставить пустым, Клиент будет хранить online-сессию в системном пользовательском cache-каталоге:

```text
<user-cache-dir>/gopherkeeper/session.json
```

Для локальной разработки можно явно указать session-файл внутри проекта, например в каталоге `.session/`:

```json
{
  "address": "localhost:8080",
  "ca_cert_file": ".certs/ca.pem",
  "session_file": ".session/session.json"
}
```

Каталог `.session/` предназначен только для локального dev-запуска и не коммитится в репозиторий.

Клиент читает JSON-конфиг по флагу `--config` / `-c` или через переменную окружения `CONFIG`.

Если путь к конфигу не задан, Клиент работает от env-переменных и значений по умолчанию. Если путь задан явно, но файл не найден или содержит некорректный JSON, запуск завершается ошибкой.

Приоритет источников конфигурации Клиента:

```text
flag > env > config file > default
```

Для локальной работы удобно один раз экспортировать путь к конфигу:

```bash
export CONFIG=configs/client.json
```

После этого клиентские команды можно запускать без повторения `--config`.

### 5. Настроить env-файл Сервера

Создать локальный env-файл:

```bash
cp .env.example .env
```

Проверить и при необходимости заполнить в `.env` параметры PostgreSQL, JWT, серверного шифрования записей и логирования:

```dotenv
POSTGRES_USER=gopherkeeper
POSTGRES_PASSWORD=gopherkeeper
POSTGRES_DB=gopherkeeperdb
POSTGRES_HOST=localhost
POSTGRES_PORT=5432

JWT_SECRET=<secret>
JWT_TTL=1h

RECORD_MASTER_KEY=<secret>
RECORD_KEY_ID=primary

LOG_LEVEL=info
```

JWT secret можно сгенерировать командой:

```bash
make gen-jwt-secret
```

Скопировать выведенное значение в локальный `.env`:

```dotenv
JWT_SECRET=<generated-secret>
```

Record master key для серверного шифрования payload'ов можно сгенерировать командой:

```bash
make gen-record-master-key
```

Скопировать выведенное значение в локальный `.env`:

```dotenv
RECORD_MASTER_KEY=<generated-record-master-key>
RECORD_KEY_ID=primary
```

### 6. Запустить Docker

Перед запуском Сервера должен быть запущен Docker.

PostgreSQL поднимается автоматически командой `make run-server`, поэтому отдельно запускать `docker compose` для обычного локального сценария не нужно.

### 7. Запустить Сервер

Запустить Сервер через Makefile:

```bash
make run-server
```

Команда:

- поднимет локальный PostgreSQL через Docker Compose;
- сгенерирует локальные TLS certificates, если их ещё нет;
- соберёт бинарник Сервера;
- запустит Сервер с параметрами из `.env`.

После старта Сервер применит миграции и начнёт принимать HTTPS-запросы.

### 8. Проверить доступность Сервера

В другом терминале:

```bash
export CONFIG=configs/client.json
./bin/gkeep health
```

Ожидаемый результат:

```text
Server status: ok
```

Если Сервер не запущен или недоступен, команда завершится ошибкой:

```text
server unavailable: connection refused
```

### 9. Зарегистрировать пользователя

```bash
./bin/gkeep register -l alice
```

Клиент запросит пароль интерактивно:

```text
Password:
Repeat password:
User alice registered successfully.
```

Пароль не передаётся через аргументы процесса и не попадает в shell history.

Для CI и скриптов password можно передать одной строкой через stdin. Значение переменной должно поступать из безопасного хранилища секретов, а не записываться буквально в команду:

```bash
printf '%s\n' "$GKEEP_PASSWORD" | ./bin/gkeep register -l alice --password-stdin
```

### 10. Войти под пользователем

```bash
./bin/gkeep login -l alice
```

Ожидаемый результат:

```text
User alice logged in successfully.
```

После успешного входа Клиент сохраняет JWT bearer token в локальный session-файл. Token не выводится в stdout или stderr.

Для CI и скриптов:

```bash
printf '%s\n' "$GKEEP_PASSWORD" | ./bin/gkeep login -l alice --password-stdin
```

### 11. Проверить текущую online-сессию

```bash
./bin/gkeep whoami
```

Если пользователь вошёл:

```text
alice
```

Если локальной online-сессии нет:

```text
not logged in
```

Состояние `not logged in` не считается технической ошибкой команды `whoami`.


### 12. Создать text-запись

Подготовить файл с приватным текстом:

```bash
mkdir -p .session
printf 'secret note\n' > .session/note.txt
```

Создать запись:

```bash
./bin/gkeep records create-text --title 'my note' --text-file .session/note.txt
```

Ожидаемый результат:

```text
Created text record <record-id> with revision 1.
```

Приватный текст передаётся через файл, а не через аргумент команды, чтобы он не попадал в shell history. Для необязательной приватной метаинформации можно использовать `--metadata-file <path>`.

### 13. Получить список записей

```bash
./bin/gkeep records list
```

Ожидаемый результат содержит только открытые metadata без приватного payload:

```text
ID                                    TYPE  TITLE    REVISION  UPDATED AT
<record-id>                           text  my note  1         2026-07-08T12:00:00Z
```

### 14. Получить text-запись

```bash
./bin/gkeep records get <record-id>
```

Команда выводит открытую metadata и расшифрованный text payload текущего пользователя.

### 15. Выйти из online-сессии

```bash
./bin/gkeep logout
```

Ожидаемый результат:

```text
logged out
```

Команда удаляет только локальную online-сессию Клиента и не обращается к Серверу.

## Конфигурация Клиента

Поддерживаемые параметры JSON-конфига:

```json
{
  "address": "localhost:8080",
  "ca_cert_file": "",
  "session_file": ""
}
```

- `address` — адрес Сервера в формате `host:port`;
- `ca_cert_file` — путь к дополнительному CA certificate для проверки TLS;
- `session_file` — путь к локальному session-файлу Клиента.

Соответствие источников конфигурации:

| Назначение | JSON | Флаг | Env |
|---|---|---|---|
| Путь к JSON-конфигу | — | `--config`, `-c` | `CONFIG` |
| Адрес Сервера | `address` | `--address`, `-a` | `ADDRESS` |
| Дополнительный CA certificate | `ca_cert_file` | `--ca-cert` | `CA_CERT_FILE` |
| Session-файл | `session_file` | `--session-file` | `SESSION_FILE` |

По умолчанию session-файл хранится как `gopherkeeper/session.json` внутри системного каталога пользовательского кеша. Файл создаётся с правами `0600`, а родительский каталог — с правами `0700`.

## Требования к учётным данным

### Логин

- длина — от 3 до 32 символов;
- допускаются только латинские буквы, цифры и символы `.`, `_`, `-`;
- первый символ должен быть латинской буквой или цифрой;
- пробельные символы в начале и конце удаляются;
- заглавные латинские буквы приводятся к нижнему регистру;
- пробельные символы внутри логина и любые Unicode-символы не допускаются.

Примеры допустимых логинов: `alice`, `bob_42`, `eve.dev`, `king-of-andals`.

### Пароль

- длина — от 3 до 64 символов;
- допускаются печатные ASCII-символы от `!` до `~`;
- пробелы, табуляция, переносы строк, кириллица, emoji и другие Unicode-символы не допускаются;
- пароль чувствителен к регистру и не подвергается обрезке или нормализации.

Пример допустимого пароля: `correct-horse-battery-staple`.

## Команды разработки

Makefile остаётся удобной оболочкой для сборки, тестов и инфраструктуры:

```bash
make build
make test
make test-race
make test-integration
make
make ci
make build-client-cross
```

Клиентские сценарии выше показаны прямыми вызовами `./bin/gkeep`, потому что после добавления JSON-конфига они не требуют длинного набора флагов.
