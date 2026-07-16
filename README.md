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
- `client whoami` — проверка текущего пользователя по сохранённой online-сессии;
- `client sync` / `sync --refresh` — явная синхронизация зашифрованного локального кеша с Сервером;
- `client records create-text` / `update-text` — создание и изменение text-записей;
- `client records create-credentials` / `update-credentials` — создание и изменение credentials-записей;
- `client records create-card` / `update-card` — создание и изменение card-записей;
- `client records create-binary` / `update-binary` — создание и изменение binary-записей;
- `client records list`, `get`, `delete` — online-операции для всех реализованных типов записей;
- `client records list/get --offline --login <login>` — явное read-only чтение ранее синхронизированного зашифрованного кеша.

## Требования

Для локальной разработки нужны Go 1.26, Docker с Compose, `make` и OpenSSL.

## Архитектура

Сервер предоставляет HTTPS API, хранит пользователей и зашифрованные записи в PostgreSQL. Клиент использует тонкие CLI-команды над application use cases, локальную online-сессию и зашифрованный SQLite-кеш. HTTP-контракт описан в [OpenAPI-спецификации](api/openapi.yaml).

## Конфигурация Сервера

| Параметр | Источник | Default |
|---|---|---|
| address | `ADDRESS`, `-a` | `localhost:8080` |
| PostgreSQL DSN | `DATABASE_DSN`, `--database-dsn` | обязательный |
| TLS certificate/key | `TLS_CERT_FILE` / `TLS_KEY_FILE`, `--tls-cert` / `--tls-key` | обязательные |
| JWT secret/TTL | `JWT_SECRET`, `JWT_TTL`, `--jwt-ttl` | secret обязателен, TTL `15m` |
| record master key/key ID | `RECORD_MASTER_KEY`, `RECORD_KEY_ID` | key обязателен, ID `primary` |
| log level | `LOG_LEVEL` | `info` |

## Сборочная информация

Клиент и Сервер выводят версию, дату сборки и commit, если эти значения были подставлены при сборке через ldflags.

## Ограничения данных

- text — 1 МиБ UTF-8;
- binary — 2 МиБ после Base64-декодирования;
- metadata — 64 КиБ UTF-8;
- HTTP request body — 4 МиБ.

## Модель безопасности

Трафик защищен TLS, password hash хранится через bcrypt, доступ авторизуется JWT. Payload записей шифруется на Сервере AES-256-GCM, локальный кеш — ключом из password через Argon2id и AES-256-GCM. Это не end-to-end encryption: Сервер обрабатывает plaintext в памяти; title и технические metadata записей остаются открытыми.

## Известные ограничения MVP

Синхронизация только явная, offline-режим только read-only, автоматического merge конфликтов и истории версий нет. Большие файлы, OTP, KMS и ротация ключей не поддерживаются.

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

Команда создаёт локальный CA certificate и server certificate в каталоге `.local/certs/`. Для локального self-signed TLS Клиенту нужен файл `.local/certs/ca.pem`.

### 4. Настроить JSON-конфиг Клиента

Создать локальный конфиг Клиента:

```bash
cp configs/client.example.json configs/client.json
```

Указать в `configs/client.json` адрес Сервера и путь к локальному CA certificate:

```json
{
  "address": "localhost:8080",
  "ca_cert_file": ".local/certs/ca.pem",
  "session_file": "",
  "cache_dir": ""
}
```

Если `session_file` оставить пустым, Клиент будет хранить online-сессию в системном пользовательском cache-каталоге:

```text
<user-cache-dir>/gopherkeeper/session.json
```

Если `cache_dir` оставить пустым, отдельные кеши аккаунтов будут располагаться в системном пользовательском cache-каталоге:

```text
<user-cache-dir>/gopherkeeper/cache/<account-id>/cache.db
```

`account-id` детерминированно вычисляется из адреса Сервера и канонического login. Исходные значения не используются как части пути.

Для локальной разработки можно явно указать session-файл внутри проекта, например в каталоге `.local/session/`:

```json
{
  "address": "localhost:8080",
  "ca_cert_file": ".local/certs/ca.pem",
  "session_file": ".local/session/session.json",
  "cache_dir": ".local/cache"
}
```

Каталог `.local/session/` предназначен только для локального dev-запуска и не коммитится в репозиторий.

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
```

### 6.1 Запустить Docker

Перед запуском Сервера должен быть запущен Docker.

PostgreSQL поднимается автоматически командой `make run-server`, поэтому отдельно запускать `docker compose` для обычного локального сценария не нужно.

### 6.2. Запустить Сервер

Сервер удобнее запускать через Makefile, предварительно стартовав Docker:

```bash
make run-server
```

Команда:

- поднимет локальный PostgreSQL через Docker Compose;
- сгенерирует локальные TLS certificates, если их ещё нет;
- соберёт бинарник Сервера;
- запустит Сервер с параметрами из `.env`.

После старта Сервер применит миграции и начнёт принимать HTTPS-запросы.

### 6.3. Проверить доступность Сервера

В другом терминале:

```bash
make run-client-health
```

### 7. Подготовить удобный запуск Клиента

Если Сервер удобнее запустить через `make`, то для Клиента удобнее один раз подготовить окружение и запускать напрямую.

1. Добавьте локальный каталог `bin/` в `PATH`, чтобы вместо `./bin/gkeep` можно было вызывать просто `gkeep`:

  ```bash
  export PATH="$PWD/bin:$PATH"
  ```

2. Задайте путь к конфигурации Клиента:

  ```bash
  export CONFIG=configs/client.json
  ```

3. Соберите Клиент:

 ```bash
  make build-client
  ```

После этого команды Клиента можно запускать короче:

```bash
gkeep health
```

Эти переменные действуют только в текущей shell-сессии. Если открыть новый терминал, команды нужно выполнить повторно.

### 8. Проверить доступность Сервера

В другом терминале:

```bash
gkeep health
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
gkeep register -l alice
```

Клиент запросит пароль интерактивно:

```text
Password:
Repeat password:
User alice registered successfully.
```

Пароль не передаётся через аргументы процесса и не попадает в shell history.

### 10. Войти под пользователем

```bash
gkeep login -l alice
```

Ожидаемый результат:

```text
User alice logged in successfully.
```

После успешного входа Клиент сохраняет JWT bearer token в локальный session-файл. Token не выводится в stdout или stderr.

### 11. Проверить текущую online-сессию

```bash
gkeep whoami
```

Если пользователь вошёл:

```text
alice
```

Если локальной online-сессии нет:

```text
not logged in
```

### 12. Создать text-запись

Подготовить файл с приватным текстом:

```bash
mkdir -p .local/tmp
printf 'secret note\n' > .local/tmp/note.txt
```

Создать запись:

```bash
gkeep records create-text --title 'my note' --text-file .local/tmp/note.txt
```

Ожидаемый результат:

```text
Created text record <record-id> with revision 1.
```

Приватный текст передаётся через файл, а не через аргумент команды, чтобы он не попадал в shell history. Для необязательной приватной метаинформации можно использовать `--metadata-file <path>`.

### 13. Создать credentials-запись

В штатном интерактивном режиме передаётся только открытый `title`:

```bash
gkeep records create-credentials --title 'GitHub'
```

Клиент запросит приватные поля отдельно:

```text
Login:
Password:
URL (optional):
```

Password вводится без отображения в терминале. Необязательную приватную метаинформацию можно прочитать из файла через `--metadata-file <path>`.

Ожидаемый результат:

```text
Created credentials record <record-id> with revision 1.
```

### 14. Создать card-запись

В штатном интерактивном режиме передаётся только открытый `title`:

```bash
gkeep records create-card --title "Joel's card"
```

Клиент запросит приватные поля отдельно:

```text
Card number:
Cardholder (optional):
Expiry (MM/YYYY, optional):
CVV (optional):
```

Ожидаемый результат:

```text
Created card record <record-id> with revision 1.
```

### 15. Создать binary-запись

Подготовить файл с приватными бинарными данными:

```bash
printf '\x00\x01\x02\xff' > .local/tmp/backup.bin
```

Создать запись:

```bash
gkeep records create-binary \
  --title 'backup' \
  --binary-file .local/tmp/backup.bin \
  --content-type application/octet-stream
```

Ожидаемый результат:

```text
Created binary record <record-id> with revision 1.
```

Имя `backup.bin` сохраняется внутри зашифрованного payload. Необязательные `content_type` и metadata также хранятся приватно. Размер бинарных данных после Base64-декодирования не должен превышать 2 МиБ; пустой файл допустим.

### 16. Получить список записей

```bash
gkeep records list
```

Список содержит только открытые metadata записи и не раскрывает приватный payload:

```text
ID                                    TYPE         TITLE       REVISION  UPDATED AT
<text-record-id>                      text         my note     1         2026-07-08T12:00:00Z
<credentials-record-id>               credentials  GitHub      1         2026-07-10T12:01:00Z
<card-record-id>                      card         Joel's card 1         2026-07-11T12:02:00Z
<binary-record-id>                    binary       backup      1         2026-07-12T12:03:00Z
```

### 17. Получить запись

Для text, credentials и card используется общая команда:

```bash
gkeep records get <record-id>
```

Клиент определяет тип записи и выводит расшифрованный payload владельцу. Для credentials вывод содержит login, password, URL и metadata. Для card вывод содержит полный номер карты, cardholder, срок действия, CVV и metadata. Это секретный вывод: не запускайте команду в общем терминале и не перенаправляйте результат в небезопасные логи или файлы.

Binary-запись сохраняется только в явно указанный файл:

```bash
gkeep records get <binary-record-id> --output .local/tmp/restored-backup.bin
```

Ожидаемый вывод содержит приватные metadata файла, но не сами бинарные данные:

```text
ID: <binary-record-id>
Type: binary
Title: backup
Revision: 1
Created at: 2026-07-12T12:03:00Z
Updated at: 2026-07-12T12:03:00Z

Filename: backup.bin
Size: 4 bytes
Saved to: .local/tmp/restored-backup.bin
Content type: application/octet-stream
```

Stored filename не используется как локальный путь. Клиент не перезаписывает существующий output-файл: для повторного сохранения нужно удалить его или указать новый путь.

### 18. Обновить text-запись

Подготовить новый файл с приватным текстом:

```bash
printf 'updated secret note\n' > .local/tmp/note-updated.txt
```

Обновить запись, передав ожидаемую текущую ревизию:

```bash
gkeep records update-text <record-id> --revision 1 --title 'updated note' --text-file .local/tmp/note-updated.txt
```

Ожидаемый результат:

```text
Updated text record <record-id> to revision 2.
```

### 19. Обновить credentials-запись

Интерактивное обновление использует те же безопасные prompts, что и создание:

```bash
gkeep records update-credentials <record-id> --revision 1 --title 'Updated GitHub'
```

Ожидаемый результат:

```text
Updated credentials record <record-id> to revision 2.
```

### 20. Обновить card-запись

Интерактивное обновление использует те же безопасные prompts, что и создание:

```bash
gkeep records update-card <record-id> --revision 1 --title "Joel's card updated"
```

Ожидаемый результат:

```text
Updated card record <record-id> to revision 2.
```

### 21. Обновить binary-запись

Подготовить новый файл и передать ожидаемую текущую ревизию:

```bash
printf '\x10\x20\x30\x40' > .local/tmp/backup-updated.bin

gkeep records update-binary <record-id> \
  --revision 1 \
  --title 'updated backup' \
  --binary-file .local/tmp/backup-updated.bin \
  --content-type application/octet-stream
```

Ожидаемый результат:

```text
Updated binary record <record-id> to revision 2.
```

Для всех update-команд `--revision` обязателен. Клиент передаёт её Серверу в HTTP-заголовке `If-Match`, чтобы не перетереть изменения с другого устройства. Устаревшая ревизия возвращает:

```text
record revision conflict
```

Тип записи изменить нельзя: text-запись обновляется только через `update-text`, credentials-запись — через `update-credentials`, card-запись — через `update-card`, binary-запись — через `update-binary`.

### 22. Удалить запись

Удалить запись любого реализованного типа можно общей командой с актуальной ревизией:

```bash
gkeep records delete <record-id> --revision 2
```

Ожидаемый результат:

```text
Deleted record <record-id>.
```

Если ревизия устарела, Сервер возвращает конфликт и запись не удаляется. После успешного удаления повторный `gkeep records get <record-id>` вернёт `record not found`.

### 23. Синхронизировать локальный кеш

Синхронизация запускается только явной командой:

```bash
gkeep sync
```

Клиент запросит password текущего пользователя без отображения в терминале. Password повторно проверяется Сервером до создания или открытия локального кеша, поэтому опечатка не создаёт кеш с неизвестным ключом.

Ожидаемый отчёт:

```text
Cache synchronization completed.
Added: 2
Updated: 0
Removed: 1
Unchanged: 3
Stale: 1
```

Обычный `sync`:

- загружает новые записи с Сервера;
- удаляет из кеша записи, которых больше нет на Сервере;
- не загружает payload записей с неизменившейся revision;
- оставляет запись с отличающейся revision в прежнем виде и показывает её как `stale`.

Для явной замены всех stale-записей актуальными server-копиями используется:

```bash
gkeep sync --refresh
```

Сервер остаётся источником актуального состояния. Все подготовленные добавления, обновления и удаления применяются к SQLite одной транзакцией. Полные записи, включая title и приватный payload, сохраняются в кеше только в зашифрованном виде; открытыми остаются ID и revision.

Фоновая синхронизация отсутствует. Команды `records create-*`, `list`, `get`, `update-*` и `delete` не изменяют кеш автоматически.

### 24. Прочитать записи из кеша offline

Offline-чтение работает только после хотя бы одной успешной явной синхронизации:

```bash
gkeep sync
```

После этого список ранее синхронизированных записей можно получить без доступного Сервера и без действующей online-сессии:

```bash
gkeep records list --offline --login alice
```

Клиент скрыто запросит password, откроет существующий кеш пары server/login и первым сообщением явно укажет источник:

```text
Source: encrypted local cache (data may be stale).
```

Получение одной записи выполняется так:

```bash
gkeep records get <record-id> --offline --login alice
```

Для binary-записи по-прежнему обязателен безопасный вывод в новый файл:

```bash
gkeep records get <record-id> \
  --offline \
  --login alice \
  --output .local/tmp/restored.bin
```

Правила offline-режима:

- режим выбирается только явно; обычные online-команды не переключаются на кеш при network/TLS/session error;
- `--login` определяет account cache, а password используется только в текущем процессе для его расшифрования;
- отсутствие предварительного `sync`, другой login или неправильный password возвращают ошибку и не создают новый кеш;
- данные могут быть устаревшими, потому что Сервер остаётся источником актуального состояния;
- `records create-*`, `update-*` и `delete` не поддерживают `--offline` и не создают pending/outbox state;
- background-, startup- и автоматическая post-write синхронизация отсутствуют.

Для проверки сценария двух устройств используются два конфига с одинаковыми `address` и `ca_cert_file`, но разными `session_file` и `cache_dir`, например `configs/client-a.json` и `configs/client-b.json`:

1. Оба Клиента входят под Alice и выполняют `sync`.
2. После остановки Сервера Client B читает запись командой `records get --offline`.
3. После восстановления Сервера Client A обновляет revision `1` до `2`.
4. Попытка Client B обновить ту же revision `1` возвращает `record revision conflict`.
5. Обычный `sync` Client B показывает запись как `Stale`, но не заменяет её молча.
6. `gkeep sync --refresh` загружает актуальную server copy, после чего offline get возвращает новую revision.

### 25. Выйти из online-сессии

```bash
gkeep logout
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
  "session_file": "",
  "cache_dir": ""
}
```

- `address` — адрес Сервера в формате `host:port`;
- `ca_cert_file` — путь к дополнительному CA certificate для проверки TLS;
- `session_file` — путь к локальному session-файлу Клиента;
- `cache_dir` — базовый каталог локального зашифрованного кеша.

Соответствие источников конфигурации:

| Назначение | JSON | Флаг | Env |
|---|---|---|---|
| Путь к JSON-конфигу | — | `--config`, `-c` | `CONFIG` |
| Адрес Сервера | `address` | `--address`, `-a` | `ADDRESS` |
| Дополнительный CA certificate | `ca_cert_file` | `--ca-cert` | `CA_CERT_FILE` |
| Session-файл | `session_file` | `--session-file` | `SESSION_FILE` |
| Каталог локального кеша | `cache_dir` | `--cache-dir` | `CACHE_DIR` |

По умолчанию session-файл хранится как `gopherkeeper/session.json` внутри системного каталога пользовательского кеша. Файл создаётся с правами `0600`, а родительский каталог — с правами `0700`.

По умолчанию базовый каталог локального кеша хранится как `gopherkeeper/cache` внутри системного пользовательского cache-каталога. Для каждой пары Сервер + canonical login используется отдельный SHA-256 идентификатор каталога.

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

Клиентские сценарии выше показаны прямыми вызовами `gkeep` после добавления локального `bin/` в `PATH` и настройки `CONFIG`.
