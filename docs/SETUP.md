# Локальная настройка и запуск GophKeeper

Этот документ описывает подготовку локального окружения, конфигурацию Сервера и Клиента, запуск HTTPS API и удобную настройку текущей shell-сессии для команд `gkeep` и TUI.

Пользовательские сценарии отдельных CLI-команд описаны в [руководстве по консольному интерфейсу](CLI.md). Возможности TUI собраны в [основном README](../README.md).

## Требования

Для локального запуска нужны:

- Go 1.26;
- Docker с Docker Compose;
- `make`;
- OpenSSL.

## 1. Клонировать репозиторий

```bash
git clone https://github.com/xhrobj/gopherkeeper.git
cd gopherkeeper
```

Все последующие команды выполняются из корня репозитория.

## 2. Настроить Сервер

Сервер получает конфигурацию из переменных окружения и флагов. Для стандартного локального запуска используется файл `.env`, который Makefile загружает автоматически.

Создать локальный env-файл:

```bash
cp .env.example .env
```

Проверить параметры PostgreSQL и общие настройки:

```dotenv
LOG_LEVEL=info
ADDRESS=localhost:8080

POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_DB=gopherkeeperdb
POSTGRES_USER=gopherkeeper
POSTGRES_PASSWORD=password

JWT_SECRET=
JWT_TTL=30m

RECORD_MASTER_KEY=
# RECORD_KEY_ID=primary
```

`JWT_SECRET` и `RECORD_MASTER_KEY` обязательны. Для локальной разработки их можно сгенерировать готовыми Makefile-командами:

```bash
make gen-jwt-secret
make gen-record-master-key
```

Скопировать первое значение в `JWT_SECRET`, второе — в `RECORD_MASTER_KEY` локального `.env`:

```dotenv
JWT_SECRET=<generated-jwt-secret>
RECORD_MASTER_KEY=<generated-record-master-key>
```

Оба значения являются Base64-представлением случайных 32-байтовых ключей. Не коммитьте заполненный `.env`.

### Параметры Сервера

| Назначение | Env | Флаг | Значение по умолчанию |
|---|---|---|---|
| HTTPS listen address | `ADDRESS` | `-a` | `localhost:8080` |
| PostgreSQL DSN | `DATABASE_DSN` | `--database-dsn` | обязательный |
| TLS certificate | `TLS_CERT_FILE` | `--tls-cert` | обязательный |
| TLS private key | `TLS_KEY_FILE` | `--tls-key` | обязательный |
| JWT secret | `JWT_SECRET` | — | обязательный |
| JWT TTL | `JWT_TTL` | `--jwt-ttl` | `15m` |
| Record master key | `RECORD_MASTER_KEY` | — | обязательный |
| Record key ID | `RECORD_KEY_ID` | — | `primary` |
| Уровень логирования | `LOG_LEVEL` | — | `info` |

Для `make run-server` строка `DATABASE_DSN` автоматически собирается из `POSTGRES_*`, а пути к локальным TLS-файлам передаются готовыми флагами.

## 3. Настроить Клиент

Создать локальный JSON-конфиг:

```bash
cp configs/client.example.json configs/client.json
```

Для стандартного локального HTTPS-запуска указать созданный локальный CA certificate:

```json
{
  "address": "localhost:8080",
  "ca_cert_file": ".local/certs/ca.pem",
  "session_dir": "",
  "cache_dir": ""
}
```

### Параметры JSON-конфига

| Поле | Назначение |
|---|---|
| `address` | адрес Сервера в формате `host:port` |
| `ca_cert_file` | путь к дополнительному CA certificate для проверки TLS |
| `session_dir` | каталог online-сессии; файл всегда называется `session.json` |
| `cache_dir` | базовый каталог локальных зашифрованных кешей |

Если `session_dir` оставить пустым, online-сессия хранится в системном пользовательском cache-каталоге:

```text
<user-cache-dir>/gopherkeeper/session.json
```

Если `cache_dir` оставить пустым, отдельные кеши аккаунтов располагаются там же:

```text
<user-cache-dir>/gopherkeeper/cache/<account-id>/cache.db
```

`account-id` детерминированно вычисляется из адреса Сервера и канонического login. Исходные значения не используются как части пути.

Для полностью локального dev-окружения каталоги можно разместить внутри `.local/`:

```json
{
  "address": "localhost:8080",
  "ca_cert_file": ".local/certs/ca.pem",
  "session_dir": ".local/session",
  "cache_dir": ".local/cache"
}
```

В `session_dir` указывается именно каталог, а не путь к файлу: имя `session.json` фиксировано. Каталог `.local/` не коммитится в репозиторий.

### Источники конфигурации Клиента

| Назначение | JSON | Флаг | Env |
|---|---|---|---|
| Путь к JSON-конфигу | — | `--config`, `-c` | `CONFIG` |
| Адрес Сервера | `address` | `--address`, `-a` | `ADDRESS` |
| Дополнительный CA certificate | `ca_cert_file` | `--ca-cert` | `CA_CERT_FILE` |
| Каталог session-файла | `session_dir` | `--session-dir` | `SESSION_DIR` |
| Каталог локального кеша | `cache_dir` | `--cache-dir` | `CACHE_DIR` |

Приоритет источников:

```text
flag > env > config file > default
```

Если путь к конфигу не задан, Клиент использует env-переменные и значения по умолчанию. Если путь задан явно, но файл отсутствует, содержит неизвестные поля или некорректный JSON, запуск завершается ошибкой.

При запуске TUI с `--config` или `CONFIG` окно `System → Config...` сохраняет изменения в тот же JSON-файл. Без переданного пути изменения применяются только к текущему процессу и не сохраняются между запусками.

## 4. Запустить Сервер

Перед запуском должен работать Docker.

```bash
make run-server
```

Команда:

- поднимет локальный PostgreSQL через Docker Compose и дождётся его готовности;
- сгенерирует локальные TLS certificates, если их ещё нет;
- соберёт `bin/gopherkeeper-server`;
- передаст Серверу настройки из `.env`;
- применит встроенные миграции при старте;
- запустит HTTPS API.

Терминал с Сервером нужно оставить открытым.

## 5. Подготовить удобный запуск Клиента

В другом терминале собрать Клиент:

```bash
make build-client
```

Добавить локальный `bin/` в `PATH` и один раз задать путь к JSON-конфигу:

```bash
export PATH="$PWD/bin:$PATH"
export CONFIG=configs/client.json
```

После этого вместо длинных вызовов можно использовать короткие команды:

```bash
gkeep health
gkeep tui
```

Эти переменные действуют только в текущей shell-сессии. В новом терминале их нужно экспортировать повторно.

Без `CONFIG` путь можно передавать явно:

```bash
gkeep --config configs/client.json health
gkeep --config configs/client.json tui
```

## 6. Проверить запуск

Проверить HTTPS-соединение Клиента с Сервером:

```bash
gkeep health
```

Ожидаемый результат:

```text
Server status: ok
```

После этого можно запустить TUI:

```bash
gkeep tui
```

Или перейти к [отдельным CLI-командам](CLI.md).

## Полезные Makefile-команды

```bash
make build                 # собрать Сервер и Клиент
make build-server          # собрать только Сервер
make build-client          # собрать только Клиент
make run-client-health     # выполнить health через configs/client.json
make db-connect            # открыть psql в локальном PostgreSQL
make db-down               # остановить PostgreSQL без удаления данных
make db-erase              # удалить PostgreSQL-контейнер и локальные данные
```

`make db-erase` необратимо удаляет локальный volume PostgreSQL и предназначен только для сброса dev-окружения.

## Несколько локальных Клиентов

Для проверки сценария нескольких устройств можно создать два JSON-конфига с одинаковыми `address` и `ca_cert_file`, но разными каталогами сессии и кеша:

```json
{
  "address": "localhost:8080",
  "ca_cert_file": ".local/certs/ca.pem",
  "session_dir": ".local/client-a/session",
  "cache_dir": ".local/client-a/cache"
}
```

```json
{
  "address": "localhost:8080",
  "ca_cert_file": ".local/certs/ca.pem",
  "session_dir": ".local/client-b/session",
  "cache_dir": ".local/client-b/cache"
}
```

Запускать их можно явным флагом:

```bash
gkeep -c configs/client-a.json tui
gkeep -c configs/client-b.json tui
```
