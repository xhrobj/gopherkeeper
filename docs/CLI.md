# Консольный интерфейс GophKeeper

Отдельные CLI-команды предназначены для автоматизации, диагностики и работы без TUI. Они используют те же application use cases, HTTP-клиент, online-сессию и локальный зашифрованный кеш, что и терминальный интерфейс.

Перед использованием выполните [локальную настройку Сервера и Клиента](SETUP.md). Требования и ограничения собраны в [отдельном документе](PROJECT_REQUIREMENTS.md).

## Доступные команды

- `client`, `client -h`, `client --help`, `client help` — баннер и общая справка;
- `client health --help`, `client help health` — справка команды `health` без баннера;
- `client -v`, `client --version` — баннер и полная информация о сборке;
- `client health` — только результат команды;
- `client register` — регистрация пользователя;
- `client login` — вход пользователя и сохранение локальной online-сессии;
- `client whoami` — проверка текущего пользователя по сохранённой online-сессии;
- `client tui` — запуск интерактивного терминального интерфейса;
- `client sync` — явная синхронизация зашифрованного локального кеша с Сервером;
- `client records create-text` / `update-text` — создание и изменение text-записей;
- `client records create-credentials` / `update-credentials` — создание и изменение credentials-записей;
- `client records create-card` / `update-card` — создание и изменение card-записей;
- `client records create-binary` / `update-binary` — создание и изменение binary-записей;
- `client records list`, `get`, `delete` — online-операции для всех реализованных типов записей;
- `client records list/get --offline --login <login>` — явное read-only чтение ранее синхронизированного зашифрованного кеша.

## Проверить доступность Сервера

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

## Зарегистрировать пользователя

```bash
gkeep register --login alice
```

Клиент запросит пароль интерактивно:

```text
Password:
Repeat password:
User alice registered successfully.
```

Пароль не передаётся через аргументы процесса и не попадает в shell history.

## Войти под пользователем

```bash
gkeep login --login alice
```

Ожидаемый результат:

```text
User alice logged in successfully.
```

После успешного входа Клиент сохраняет JWT bearer token в локальный session-файл. Token не выводится в stdout или stderr.

## Проверить текущую online-сессию

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

## Создать text-запись

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

Приватный текст передаётся через файл, а не через аргумент команды, чтобы он не попадал в shell history. Для необязательной однострочной приватной метаинформации можно использовать `--metadata-file <path>`; файл не должен содержать переносов строк.

## Создать credentials-запись

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

Password вводится без отображения в терминале. Необязательную однострочную приватную метаинформацию можно прочитать из файла через `--metadata-file <path>`; файл не должен содержать переносов строк.

Ожидаемый результат:

```text
Created credentials record <record-id> with revision 1.
```

## Создать card-запись

В штатном интерактивном режиме передаётся только открытый `title`:

```bash
gkeep records create-card --title "Joel's card"
```

Клиент запросит приватные поля отдельно:

```text
Card number (12-20 digits):
Cardholder (optional):
Expiry (MM/YY, optional):
CVV (3 digits, optional):
```

Ожидаемый результат:

```text
Created card record <record-id> with revision 1.
```

## Создать binary-запись

Подготовить файл с приватными бинарными данными:

```bash
printf '\x00\x01\x02\xff' > .local/tmp/backup.bin
```

Создать запись:

```bash
gkeep records create-binary \
  --title 'backup' \
  --binary-file .local/tmp/backup.bin
```

Ожидаемый результат:

```text
Created binary record <record-id> with revision 1.
```

Имя `backup.bin` и необязательная metadata сохраняются внутри зашифрованного payload. Размер бинарных данных после Base64-декодирования не должен превышать 2 МиБ; пустой файл допустим.

## Получить список записей

```bash
gkeep records list
```

Список содержит только открытые системные поля записи и не раскрывает приватный payload:

```text
ID                                    TYPE         TITLE       REVISION  UPDATED AT
<text-record-id>                      text         my note     1         2026-07-08T12:00:00Z
<credentials-record-id>               credentials  GitHub      1         2026-07-10T12:01:00Z
<card-record-id>                      card         Joel's card 1         2026-07-11T12:02:00Z
<binary-record-id>                    binary       backup      1         2026-07-12T12:03:00Z
```

## Получить запись

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
```

Stored filename не используется как локальный путь. Клиент не перезаписывает существующий output-файл: для повторного сохранения нужно удалить его или указать новый путь.

## Обновить text-запись

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

## Обновить credentials-запись

Интерактивное обновление использует те же безопасные prompts, что и создание:

```bash
gkeep records update-credentials <record-id> --revision 1 --title 'Updated GitHub'
```

Ожидаемый результат:

```text
Updated credentials record <record-id> to revision 2.
```

## Обновить card-запись

Интерактивное обновление использует те же безопасные prompts, что и создание:

```bash
gkeep records update-card <record-id> --revision 1 --title "Joel's card updated"
```

Ожидаемый результат:

```text
Updated card record <record-id> to revision 2.
```

## Обновить binary-запись

Подготовить новый файл и передать ожидаемую текущую ревизию:

```bash
printf '\x10\x20\x30\x40' > .local/tmp/backup-updated.bin

gkeep records update-binary <record-id> \
  --revision 1 \
  --title 'updated backup' \
  --binary-file .local/tmp/backup-updated.bin
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

## Удалить запись

Удалить запись любого реализованного типа можно общей командой с актуальной ревизией:

```bash
gkeep records delete <record-id> --revision 2
```

Ожидаемый результат:

```text
Deleted record <record-id>.
```

Если ревизия устарела, Сервер возвращает конфликт и запись не удаляется. После успешного удаления повторный `gkeep records get <record-id>` вернёт `record not found`.

## Синхронизировать локальный кеш

Синхронизация запускается только явной командой:

```bash
gkeep sync
```

Клиент запросит password текущего пользователя без отображения в терминале. Password повторно проверяется Сервером до создания или открытия локального кеша, поэтому опечатка не создаёт кеш с неизвестным ключом.

Ожидаемый отчёт:

```text
Cache synchronization completed.
Added: 2
Updated: 1
Removed: 1
Unchanged: 3
```

Команда `sync`:

- загружает новые записи с Сервера;
- заменяет записи с отличающейся revision актуальными server-копиями;
- удаляет из кеша записи, которых больше нет на Сервере;
- не загружает payload записей с неизменившейся revision.

Сервер остаётся источником актуального состояния. После успешной синхронизации кеш соответствует состоянию Сервера на момент операции. Все подготовленные добавления, обновления и удаления применяются к SQLite одной транзакцией; при ошибке прежний согласованный кеш остаётся без изменений. Полные записи, включая title и приватный payload, сохраняются в кеше только в зашифрованном виде; открытыми остаются ID и revision.

Фоновая синхронизация отсутствует. Команды `records create-*`, `list`, `get`, `update-*` и `delete` не изменяют кеш автоматически.

## Прочитать записи из кеша offline

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

Для проверки сценария двух устройств используются два конфига с одинаковыми `address` и `ca_cert_file`, но разными `session_dir` и `cache_dir`, например `configs/client-a.json` и `configs/client-b.json`:

1. Оба Клиента входят под Alice и выполняют `sync`.
2. После остановки Сервера Client B читает запись командой `records get --offline`.
3. После восстановления Сервера Client A обновляет revision `1` до `2`.
4. Попытка Client B обновить ту же revision `1` возвращает `record revision conflict`.
5. До следующей синхронизации offline-кеш Client B продолжает содержать прежнюю revision.
6. Обычный `gkeep sync` загружает актуальную server copy, после чего offline get возвращает новую revision.

## Выйти из online-сессии

```bash
gkeep logout
```

Ожидаемый результат:

```text
logged out
```

Команда удаляет только локальную online-сессию Клиента и не обращается к Серверу.
