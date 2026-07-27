# 🔑 Ручная генерация серверных секретов и TLS-сертификатов

Это руководство описывает ручной вариант подготовки серверных секретов и локальных TLS-сертификатов GophKeeper через OpenSSL. Для обычного локального запуска используйте команды Makefile из [🤓 полного руководства по настройке и запуску](SETUP.md):

```bash
make gen-jwt-secret
make gen-record-master-key
make gen-tls-certs
```

Ручная генерация пригодится, если нужно выполнить те же операции без Makefile, проверить отдельные команды OpenSSL, восстановить удалённые файлы или подготовить собственный набор сертификатов для другого окружения — например, удаленного тестового стенда.

## Требования

Нужен OpenSSL, доступный в текущей shell-сессии:

```bash
openssl version
```

> ⚠️ На Windows выполняйте команды в **Git Bash**. Перед командами OpenSSL отключите автоматическое преобразование Unix-подобных аргументов в Windows-пути:

```bash
export MSYS2_ARG_CONV_EXCL='*'
```

Эта переменная нужна только для текущей Git Bash-сессии. Скрипт `scripts/generate-tls-certs.sh` устанавливает её самостоятельно.

Все последующие команды выполняются из корня проекта.

## 1. Сгенерировать серверные секреты

### 1.1. JWT secret

```bash
openssl rand -base64 32
```

Скопируйте полученное значение в `JWT_SECRET` файла `.env`:

```dotenv
JWT_SECRET=<generated-jwt-secret>
```

### 1.2. Record master key

Сгенерируйте отдельное значение той же длины:

```bash
openssl rand -base64 32
```

Скопируйте его в `RECORD_MASTER_KEY` файла `.env`:

```dotenv
RECORD_MASTER_KEY=<generated-record-master-key>
```

Для `JWT_SECRET` и `RECORD_MASTER_KEY` должны использоваться **разные** случайные значения.

## 2. Подготовить каталог сертификатов

```bash
umask 077
rm -rf .local/certs
mkdir -p .local/certs
```

Создайте файл расширений серверного сертификата:

```bash
cat > .local/certs/server.ext <<'EOF'
[server_cert]
basicConstraints = critical, CA:FALSE
keyUsage = critical, digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth
subjectAltName = DNS:localhost,DNS:server,IP:127.0.0.1,IP:::1
EOF
```

Имена и IP-адреса в `subjectAltName` соответствуют стандартному локальному запуску GophKeeper. При использовании другого hostname добавьте его до подписания сертификата.

## 3. Создать локальный центр сертификации

Сгенерируйте приватный ключ CA:

```bash
openssl genrsa -out .local/certs/ca-key.pem 2048
```

Создайте самоподписанный сертификат CA:

```bash
openssl req \
  -x509 \
  -new \
  -sha256 \
  -key .local/certs/ca-key.pem \
  -days 3650 \
  -subj "/CN=GophKeeper Development CA" \
  -addext "basicConstraints = critical, CA:TRUE" \
  -addext "keyUsage = critical, keyCertSign, cRLSign" \
  -addext "subjectKeyIdentifier = hash" \
  -out .local/certs/ca.pem
```

## 4. Создать и подписать сертификат Сервера

Сгенерируйте приватный ключ Сервера:

```bash
openssl genrsa -out .local/certs/server-key.pem 2048
```

Создайте запрос на сертификат:

```bash
openssl req \
  -new \
  -sha256 \
  -key .local/certs/server-key.pem \
  -subj "/CN=server" \
  -out .local/certs/server.csr
```

Подпишите серверный сертификат локальным CA:

```bash
openssl x509 \
  -req \
  -sha256 \
  -in .local/certs/server.csr \
  -CA .local/certs/ca.pem \
  -CAkey .local/certs/ca-key.pem \
  -set_serial 1 \
  -days 3650 \
  -extfile .local/certs/server.ext \
  -extensions server_cert \
  -out .local/certs/server.pem
```

Удалите временные файлы:

```bash
rm -f .local/certs/server.csr .local/certs/server.ext
```

На Linux и macOS дополнительно установите права доступа:

```bash
chmod 600 .local/certs/ca-key.pem .local/certs/server-key.pem
chmod 644 .local/certs/ca.pem .local/certs/server.pem
```

В Git Bash команды `chmod` можно выполнить, но итоговый доступ к файлам определяется ACL Windows.

## 5. Проверить сертификат

```bash
openssl verify \
  -x509_strict \
  -CAfile .local/certs/ca.pem \
  .local/certs/server.pem
```

Ожидаемый результат:

```text
.local/certs/server.pem: OK
```

Итоговый каталог должен содержать:

```text
.local/certs/ca.pem
.local/certs/ca-key.pem
.local/certs/server.pem
.local/certs/server-key.pem
```

Назначение файлов:

- `ca.pem` — доверенный CA-сертификат для Клиента
- `ca-key.pem` — приватный ключ локального CA
- `server.pem` — сертификат Сервера
- `server-key.pem` — приватный ключ Сервера

Приватные ключи `ca-key.pem` и `server-key.pem` нельзя передавать Клиенту или публиковать.

## 6. Подключить файлы к GophKeeper

Стандартный локальный запуск через Makefile использует эти файлы автоматически. При прямом запуске Сервера передайте их флагами:

```bash
./bin/gopherkeeper-server \
  --tls-cert .local/certs/server.pem \
  --tls-key .local/certs/server-key.pem
```

В конфиге Клиента укажите CA-сертификат:

```json
{
  "ca_cert_file": ".local/certs/ca.pem"
}
```

После замены набора сертификатов перезапустите Сервер. Все Клиенты должны использовать `ca.pem` из того же набора, которым подписан `server.pem`.
