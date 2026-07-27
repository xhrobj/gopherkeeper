# 🤓 Полное руководство по настройке и запуску GophKeeper

Этот документ подробно описывает подготовку локального окружения, конфигурацию Сервера и Клиента, запуск HTTPS и gRPC API, а также настройку текущей shell-сессии для команд `gkeep` и TUI.

**Если вам нужно просто запустить проект, используйте** [🚀 гайд по быстрому запуску](SETUP_QUICK.md).

* Пользовательские сценарии отдельных CLI-команд описаны в [💻 руководстве по консольному интерфейсу](CLI.md)
* Возможности TUI и основной сценарий работы собраны в [🔐 README проекта](../README.md)
* Архитектурные правила, ограничения данных и модель безопасности описаны в [⚖️ требованиях и ограничениях](PROJECT_REQUIREMENTS.md)

## Требования

Для локального запуска нужны:

* Go 1.26
* Docker с Docker Compose
* `make`
* OpenSSL

## 1. Клонировать проект

```bash
git clone https://github.com/xhrobj/gopherkeeper.git
cd gopherkeeper
```

Все последующие команды выполняются из корня проекта. Стандартная локальная конфигурация использует относительные пути к JSON-конфигу Клиента и TLS-сертификатам.

## 2. Запустить Сервер

### 2.1. Создать локальный `.env`

Сервер получает конфигурацию из переменных окружения и флагов. Для стандартного локального запуска используется файл `.env`, который Makefile загружает автоматически.

Скопируйте пример конфигурации:

```bash
cp .env.example .env
```

Основные параметры локального `.env`:

```dotenv
LOG_LEVEL=info

ADDRESS=localhost:8080
GRPC_ADDRESS=localhost:50051

POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_DB=gopherkeeperdb
POSTGRES_USER=gopherkeeper
POSTGRES_PASSWORD=password

JWT_SECRET=
JWT_TTL=15m

RECORD_MASTER_KEY=
# RECORD_KEY_ID=primary
```

Сервер одновременно запускает HTTPS и gRPC. Адреса транспортов задаются отдельно через `ADDRESS` и `GRPC_ADDRESS`.

Параметры `POSTGRES_*` используются Docker Compose для запуска локального PostgreSQL и Makefile для формирования строки подключения `DATABASE_DSN`.

### 2.2. Сгенерировать серверные секреты

Для запуска Сервера обязательны:

* `JWT_SECRET` — секрет подписи JWT
* `RECORD_MASTER_KEY` — мастер-ключ шифрования приватных payload записей

Сгенерируйте оба значения:

```bash
make gen-jwt-secret
make gen-record-master-key
```

Скопируйте первое полученное значение в `JWT_SECRET`, а второе — в `RECORD_MASTER_KEY` файла `.env`:

```dotenv
JWT_SECRET=<generated-jwt-secret>
RECORD_MASTER_KEY=<generated-record-master-key>
```

Оба значения являются Base64-представлением случайных 32-байтовых ключей.

В исходном `.env.example` эти поля намеренно оставлены пустыми. Заполненный `.env` содержит секреты локального окружения и не должен попадать в репозиторий.

### 2.3. Параметры Сервера

| Назначение          | Env                 | Флаг                   | Стандартное локальное значение |
| ------------------- | ------------------- | ---------------------- | ------------------------------ |
| Адрес HTTPS API     | `ADDRESS`           | `--address`, `-a`      | `localhost:8080`               |
| Адрес gRPC API      | `GRPC_ADDRESS`      | `--grpc-address`, `-g` | `localhost:50051`              |
| PostgreSQL DSN      | `DATABASE_DSN`      | `--database-dsn`       | собирается из `POSTGRES_*`     |
| TLS certificate     | `TLS_CERT_FILE`     | `--tls-cert`           | `.local/certs/server.pem`      |
| TLS private key     | `TLS_KEY_FILE`      | `--tls-key`            | `.local/certs/server-key.pem`  |
| JWT secret          | `JWT_SECRET`        | —                      | обязательный                   |
| JWT TTL             | `JWT_TTL`           | `--jwt-ttl`            | `15m`                          |
| Record master key   | `RECORD_MASTER_KEY` | —                      | обязательный                   |
| Record key ID       | `RECORD_KEY_ID`     | —                      | `primary`                      |
| Уровень логирования | `LOG_LEVEL`         | —                      | `info`                         |

Для `make run-server` строка `DATABASE_DSN` автоматически собирается из `POSTGRES_*`, а пути к локальным TLS-файлам передаются готовыми параметрами.

`RECORD_KEY_ID` является открытым идентификатором используемого мастер-ключа. В текущей конфигурации используется один ключ без ротации, поэтому стандартное значение `primary` менять не требуется.

Для более подробного вывода Сервера установите в `.env` уровень логирования `debug`:

```dotenv
LOG_LEVEL=debug
```

### 2.4. Запустить Сервер

Убедитесь, что Docker запущен, и выполните:

```bash
make run-server
```

Команда:

* запустит PostgreSQL через Docker Compose
* дождётся готовности PostgreSQL
* создаст локальные TLS-сертификаты, если их ещё нет
* соберёт `bin/gopherkeeper-server`
* передаст Серверу конфигурацию из `.env`
* применит встроенные миграции
* запустит HTTPS API
* запустит gRPC API

Команды Makefile создают локальные TLS-файлы в каталоге `.local/certs/`:

```text
.local/certs/ca.pem
.local/certs/server.pem
.local/certs/server-key.pem
```

`ca.pem` используется Клиентом для проверки сертификата Сервера. `server-key.pem` является приватным ключом Сервера и Клиенту не передаётся.

Оставьте терминал с Сервером открытым.

Сервер запущен 👍

### 2.5. Остановить или сбросить локальный PostgreSQL

После остановки процесса Сервера контейнер PostgreSQL продолжает работать.

Остановить PostgreSQL без удаления данных:

```bash
make db-down
```

При следующем запуске сохранённые данные будут доступны снова.

Полностью удалить контейнер и локальные данные PostgreSQL:

```bash
make db-erase
```

`make db-erase` необратимо удаляет локальный volume и предназначен только для полного сброса dev-окружения.

## 3. Запустить Клиент

Откройте второй терминал и перейдите в корень проекта.

### 3.1. Создать конфиг Клиента

Скопируйте пример конфигурации:

```bash
cp configs/client.example.json configs/client.json
```

Стандартный локальный конфиг:

```json
{
  "transport": "https",
  "address": "localhost:8080",
  "grpc_address": "localhost:50051",
  "ca_cert_file": ".local/certs/ca.pem",
  "session_dir": "",
  "cache_dir": ""
}
```

Поле `ca_cert_file` содержит путь к локальному CA-сертификату, созданному командой `make run-server`.

### 3.2. Выбрать транспорт

Поле `transport` определяет, через какой транспорт Клиент обращается к Серверу.

| Значение | Используемый адрес |
| -------- | ------------------ |
| `https`  | `address`          |
| `grpc`   | `grpc_address`     |

Оба адреса хранятся в одном конфиге. Клиент использует только адрес выбранного транспорта.

Автоматического переключения на второй транспорт при ошибке соединения нет.

### 3.3. Параметры JSON-конфига

| Поле           | Назначение                                             |
| -------------- | ------------------------------------------------------ |
| `transport`    | выбранный транспорт: `https` или `grpc`                |
| `address`      | адрес HTTPS API в формате `host:port`                  |
| `grpc_address` | адрес gRPC API в формате `host:port`                   |
| `ca_cert_file` | путь к дополнительному CA-сертификату для проверки TLS |
| `session_dir`  | каталог локальной online-сессии                        |
| `cache_dir`    | базовый каталог локальных зашифрованных кешей          |

HTTPS и gRPC используют TLS. Поле `ca_cert_file` применяется для проверки сертификата независимо от выбранного транспорта. В стандартной локальной конфигурации используется `.local/certs/ca.pem`.

### 3.4. Настроить каталог online-сессии

После успешного входа Клиент сохраняет локальную online-сессию в файле `session.json`.

Поле `session_dir` задаёт каталог, а не полный путь к файлу:

```json
"session_dir": ".local/session"
```

Сессия будет сохранена по пути:

```text
.local/session/session.json
```

Если `session_dir` оставить пустым, Клиент использует системный пользовательский cache-каталог:

```text
<user-cache-dir>/gopherkeeper/session.json
```

Session-файл не должен попадать в репозиторий.

### 3.5. Настроить каталог локального кеша

Поле `cache_dir` задаёт базовый каталог зашифрованного локального кеша:

```json
"cache_dir": ".local/cache"
```

Для каждого аккаунта внутри него создаётся отдельная SQLite-база:

```text
.local/cache/<account-id>/cache.db
```

Если `cache_dir` оставить пустым, Клиент использует системный пользовательский cache-каталог:

```text
<user-cache-dir>/gopherkeeper/cache/<account-id>/cache.db
```

Адрес Сервера и выбранный транспорт не участвуют в формировании локального `account-id`. Поэтому для подключения к разным независимым Серверам нужно использовать разные конфиги и разные `session_dir` и `cache_dir`.

### 3.6. Хранить локальные данные внутри проекта

Для полностью локального dev-окружения сессию и кеш можно разместить внутри `.local/`:

```json
{
  "transport": "https",
  "address": "localhost:8080",
  "grpc_address": "localhost:50051",
  "ca_cert_file": ".local/certs/ca.pem",
  "session_dir": ".local/session",
  "cache_dir": ".local/cache"
}
```

Каталог `.local/` не коммитится в репозиторий.

Удалить локальную сессию и кеш такого Клиента:

```bash
rm -rf .local/session .local/cache
```

Серверные записи в PostgreSQL эта команда не затрагивает.

### 3.7. Источники конфигурации Клиента

| Назначение                    | JSON           | Флаг                   | Env            |
| ----------------------------- | -------------- | ---------------------- | -------------- |
| Путь к JSON-конфигу           | —              | `--config`, `-c`       | `CONFIG`       |
| Транспорт                     | `transport`    | `--transport`, `-t`    | `TRANSPORT`    |
| Адрес HTTPS API               | `address`      | `--address`, `-a`      | `ADDRESS`      |
| Адрес gRPC API                | `grpc_address` | `--grpc-address`, `-g` | `GRPC_ADDRESS` |
| Дополнительный CA certificate | `ca_cert_file` | `--ca-cert`            | `CA_CERT_FILE` |
| Каталог online-сессии         | `session_dir`  | `--session-dir`        | `SESSION_DIR`  |
| Каталог локального кеша       | `cache_dir`    | `--cache-dir`          | `CACHE_DIR`    |

Приоритет источников:

```text
flag > env > config file > default
```

Временно переопределить адрес HTTPS можно флагом:

```bash
gkeep --config configs/client.json \
  --address localhost:8443 \
  health
```

Или переменной окружения:

```bash
ADDRESS=localhost:8443 gkeep --config configs/client.json health
```

Если путь к конфигу не задан, Клиент использует env-переменные и значения по умолчанию.

Если путь задан явно, но файл отсутствует, содержит некорректный JSON, неизвестные поля или недопустимое значение `transport`, запуск завершается ошибкой.

При запуске TUI с `--config` или `CONFIG` окно `System → Config...` сохраняет изменения в тот же JSON-файл. Без переданного пути изменения применяются только к текущему процессу и не сохраняются между запусками.

### 3.8. Собрать Клиент

```bash
make build-client
```

Собранный бинарник будет расположен по пути:

```text
bin/gkeep
```

Другие варианты сборки:

```bash
make build
make build-client-cross
```

`make build` собирает Сервер и Клиент, а `make build-client-cross` создаёт клиентские бинарники для поддерживаемых платформ.

### 3.9. Настроить текущую shell-сессию

Добавьте собранный Клиент в `PATH` и укажите путь к конфигу:

```bash
export PATH="$PWD/bin:$PATH"
export CONFIG=configs/client.json
```

После этого вместо:

```bash
./bin/gkeep --config configs/client.json health
```

можно использовать:

```bash
gkeep health
```

Переменные действуют только в текущей shell-сессии.

Без `CONFIG` путь можно передавать явно через флаг `--config`/`-c`:

```bash
gkeep --config configs/client.json health
gkeep -c configs/client.json tui
```

Стандартный конфиг содержит относительные пути, поэтому Клиент запускается из корня проекта. Для запуска из другого каталога используйте абсолютные пути к конфигу, CA-сертификату, каталогу сессии и каталогу кеша.

Клиент настроен и готов к запуску 👍

## 4. Проверить подключение

Проверьте оба транспорта, не изменяя сохранённый конфиг:

```bash
gkeep --transport https health
gkeep -t grpc health
```

Ожидаемый результат обеих команд:

```text
Server status: ok
```

Первая команда использует адрес из поля `address`, вторая — из `grpc_address`.

## 5. Начать работу

### 5.1. Отдельные CLI-команды

Пользовательские сценарии отдельных CLI-команд описаны в [💻 руководстве по консольному интерфейсу](CLI.md).

### 5.2. Терминальный интерфейс

Запустите TUI:

```bash
gkeep tui
```

Основной сценарий работы с TUI описан в [🔐 README проекта](../README.md).

## 6. Полезные Makefile-команды

### Сборка

```bash
make build                 # собрать Сервер и Клиент
make build-server          # собрать только Сервер
make build-client          # собрать только Клиент
make build-client-cross    # собрать Клиент для поддерживаемых платформ
```

### Секреты и TLS

```bash
make gen-jwt-secret          # сгенерировать JWT secret
make gen-record-master-key   # сгенерировать record master key
make gen-tls-certs           # создать локальные TLS-сертификаты
```

### Запуск

```bash
make run-server            # подготовить окружение и запустить Сервер
make run-client-health     # проверить Сервер через configs/client.json
```

### PostgreSQL

```bash
make db-connect            # открыть psql в локальном PostgreSQL
make db-down               # остановить PostgreSQL без удаления данных
make db-erase              # удалить PostgreSQL и его локальные данные
```

### Проверки

```bash
make test                  # запустить unit-тесты
make test-race             # запустить тесты с race detector
make test-integration      # запустить integration-тесты
make                       # запустить полный набор тестов с покрытием
make ci                    # выполнить полный локальный CI
```

`make db-erase` необратимо удаляет локальный PostgreSQL volume.

## 7. Несколько локальных Клиентов

Для проверки сценария нескольких устройств создайте два конфига с одинаковыми адресами Сервера, но разными каталогами сессии и кеша.

`configs/client-a.json`:

```json
{
  "transport": "https",
  "address": "localhost:8080",
  "grpc_address": "localhost:50051",
  "ca_cert_file": ".local/certs/ca.pem",
  "session_dir": ".local/client-a/session",
  "cache_dir": ".local/client-a/cache"
}
```

`configs/client-b.json`:

```json
{
  "transport": "grpc",
  "address": "localhost:8080",
  "grpc_address": "localhost:50051",
  "ca_cert_file": ".local/certs/ca.pem",
  "session_dir": ".local/client-b/session",
  "cache_dir": ".local/client-b/cache"
}
```

Запустите Клиенты с разными конфигами:

```bash
gkeep -c configs/client-a.json tui
gkeep -c configs/client-b.json tui
```

Оба Клиента могут войти под одним login и имитировать разные пользовательские устройства. Разные транспорты при этом работают с одним состоянием на Сервере.

Такой сценарий позволяет проверить:

* появление записей, созданных другим Клиентом
* синхронизацию локальных кешей
* конфликт конкурентного изменения одной ревизии
* работу HTTPS и gRPC с одним аккаунтом

Для подключения к разным независимым Серверам обязательно используйте разные `session_dir` и `cache_dir`.

## 8. Диагностика

### PostgreSQL не становится готовым

Проверьте состояние и логи контейнера:

```bash
docker compose --env-file .env ps
docker compose --env-file .env logs postgres
```

Полностью пересоздать локальную базу:

```bash
make db-erase
make run-server
```

`make db-erase` необратимо удалит локальные данные PostgreSQL.

### Сервер сообщает об отсутствующем секрете

Проверьте заполнение `JWT_SECRET` и `RECORD_MASTER_KEY` в `.env`.

Создать новые значения:

```bash
make gen-jwt-secret
make gen-record-master-key
```

### Не найден CA-сертификат

В стандартной локальной конфигурации CA-сертификат расположен по пути:

```text
.local/certs/ca.pem
```

Проверьте наличие файла:

```bash
ls -l .local/certs/ca.pem
```

Создать локальные TLS-сертификаты отдельно:

```bash
make gen-tls-certs
```

Команда `make run-server` также создаёт их автоматически, если они отсутствуют.

### Ошибка проверки TLS

Проверьте, что:

* `ca_cert_file` указывает на фактическое расположение CA-сертификата; стандартный путь — `.local/certs/ca.pem`
* Клиент запущен из корня проекта либо в конфиге используются абсолютные пути
* Сервер использует сертификат и приватный ключ из того же набора; стандартный каталог — `.local/certs/`
* указан правильный адрес выбранного транспорта

После пересоздания сертификатов перезапустите Сервер.

### Выбранный транспорт недоступен

Проверьте сочетание выбранного транспорта и адреса:

| Транспорт | Поле с адресом |
| --------- | -------------- |
| `https`   | `address`      |
| `grpc`    | `grpc_address` |

Если порт изменяется, одинаковое значение должно быть указано в `.env` Сервера и JSON-конфиге Клиента.

### Клиент не находит JSON-конфиг

Проверьте путь из `CONFIG` и наличие файла:

```bash
echo "$CONFIG"
ls -l configs/client.json
```

Или передайте конфиг явно:

```bash
gkeep -c configs/client.json health
```

### Не работают относительные пути

Стандартный конфиг рассчитан на запуск из корня проекта. Для запуска из другого каталога используйте абсолютные пути в `CONFIG` и `configs/client.json`.

### Сбросить локальную сессию и кеш

Для штатного завершения online-сессии:

```bash
gkeep logout
```

Для полного сброса локального dev-клиента:

```bash
rm -rf .local/session .local/cache
```

После удаления кеша потребуется повторный вход и новая синхронизация.

### Подключиться к разным Серверам

Для независимых Серверов используйте разные конфиги и отдельные каталоги:

```json
{
  "session_dir": ".local/server-a/session",
  "cache_dir": ".local/server-a/cache"
}
```

```json
{
  "session_dir": ".local/server-b/session",
  "cache_dir": ".local/server-b/cache"
}
```

Это предотвращает смешивание локальных сессий и кешей разных Серверов.
