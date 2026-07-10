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
- `client records create-text` / `update-text` — создание и изменение text-записей;
- `client records create-credentials` / `update-credentials` — создание и изменение credentials-записей;
- `client records list`, `get`, `delete` — общие операции для всех реализованных типов записей.

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

Для CI и скриптов password можно передать одной строкой через stdin. Значение переменной должно поступать из безопасного хранилища секретов, а не записываться буквально в команду:

```bash
printf '%s\n' "$GKEEP_PASSWORD" | ./bin/gkeep register -l alice --password-stdin
```

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
mkdir -p .tmp
printf 'secret note\n' > .tmp/note.txt
```

Создать запись:

```bash
gkeep records create-text --title 'my note' --text-file .tmp/note.txt
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

Для CI и скриптов credentials можно передать одним JSON-значением через stdin:

```bash
printf '%s' "$GKEEP_CREDENTIALS_JSON" | \
  gkeep records create-credentials --title 'GitHub' --credentials-stdin
```

`GKEEP_CREDENTIALS_JSON` должен поступать из безопасного хранилища секретов. Не записывайте JSON с password непосредственно в команду или shell script.

Ожидаемый результат:

```text
Created credentials record <record-id> with revision 1.
```

### 14. Получить список записей

```bash
gkeep records list
```

Список содержит только открытые metadata записи и не раскрывает приватный payload:

```text
ID                                    TYPE         TITLE     REVISION  UPDATED AT
<text-record-id>                      text         my note   1         2026-07-08T12:00:00Z
<credentials-record-id>               credentials  GitHub    1         2026-07-10T12:01:00Z
```

### 15. Получить запись

```bash
gkeep records get <record-id>
```

Клиент определяет тип записи и выводит расшифрованный `text` или `credentials` payload владельцу.

Для credentials вывод содержит login, password, URL и metadata. Это секретный вывод: не запускайте команду в общем терминале и не перенаправляйте результат в небезопасные логи или файлы.

### 16. Обновить text-запись

Подготовить новый файл с приватным текстом:

```bash
printf 'updated secret note\n' > .tmp/note-updated.txt
```

Обновить запись, передав ожидаемую текущую ревизию:

```bash
gkeep records update-text <record-id> --revision 1 --title 'updated note' --text-file .tmp/note-updated.txt
```

Ожидаемый результат:

```text
Updated text record <record-id> to revision 2.
```

### 17. Обновить credentials-запись

Интерактивное обновление использует те же безопасные prompts, что и создание:

```bash
gkeep records update-credentials <record-id> --revision 1 --title 'Updated GitHub'
```

gkeep records update-credentials f07ccc28-cf4b-4027-a1e1-5a5de729f65c --revision 1 --title 'Updated GitHub'

Ожидаемый результат:

```text
Updated credentials record <record-id> to revision 2.
```

Для обеих update-команд `--revision` обязателен. Клиент передаёт её Серверу в HTTP-заголовке `If-Match`, чтобы не перетереть изменения с другого устройства. Устаревшая ревизия возвращает:

```text
record revision conflict
```

Тип записи изменить нельзя: text-запись обновляется только через `update-text`, credentials-запись — только через `update-credentials`.

### 18. Удалить запись

Удалить запись любого реализованного типа можно общей командой с актуальной ревизией:

```bash
gkeep records delete <record-id> --revision 2
```

Ожидаемый результат:

```text
Deleted record <record-id>.
```

Если ревизия устарела, Сервер возвращает конфликт и запись не удаляется. После успешного удаления повторный `gkeep records get <record-id>` вернёт `record not found`.

### 19. Выйти из online-сессии

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

Клиентские сценарии выше показаны прямыми вызовами `gkeep` после добавления локального `bin/` в `PATH` и настройки `CONFIG`.
