# 🔐 (^-^)/ GophKeeper. Менеджер паролей и приватных данных

[![(-_-) Go CI](https://github.com/xhrobj/gopherkeeper/actions/workflows/go-ci.yml/badge.svg)](https://github.com/xhrobj/gopherkeeper/actions/workflows/go-ci.yml)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=xhrobj_gophkeeper&metric=alert_status)](https://sonarcloud.io/summary/new_code?id=xhrobj_gophkeeper)
[![Coverage](https://sonarcloud.io/api/project_badges/measure?project=xhrobj_gophkeeper&metric=coverage)](https://sonarcloud.io/summary/new_code?id=xhrobj_gophkeeper)

GophKeeper — клиент-серверный менеджер паролей и приватных данных. Сервер хранит зашифрованные записи в PostgreSQL, а Клиент предоставляет полнофункциональный TUI поверх защищённого HTTPS API.

[Техническое задание](SPECIFICATION.md) · [Требования и ограничения](docs/PROJECT_REQUIREMENTS.md) · [Настройка и запуск](docs/SETUP.md) · [Консольный интерфейс](docs/CLI.md) · [OpenAPI](api/openapi.yaml)

## Возможности TUI

Текущая TUI-версия поддерживает:

- настройку подключения к Серверу;
- регистрацию, вход и завершение online-сессии;
- проверку состояния Сервера и текущего пользователя;
- отдельный просмотр серверных записей через меню `Records`;
- read-only просмотр ранее синхронизированного локального кеша через `Cache` / `Browse...`;
- просмотр, создание, редактирование и удаление серверных записей типов `credentials`, `card`, `text` и `binary`;
- безопасное скрытие чувствительных полей;
- сохранение binary-записей в файл;
- защиту от молчаливой перезаписи через `revision` и `If-Match`;
- явную синхронизацию зашифрованного локального кеша через `Cache` / `Sync...`.

Описание отдельных CLI-команд и консольного offline read-only находится в [руководстве по консольному интерфейсу](docs/CLI.md).

## Запуск TUI

Полная подготовка `.env`, JSON-конфига Клиента, локальных TLS-сертификатов и удобной shell-сессии описана в [руководстве по локальной настройке и запуску](docs/SETUP.md).

После запуска Сервера и подготовки Клиента TUI открывается командой:

```bash
gkeep tui
```

Путь к конфигу можно передать явно:

```bash
gkeep --config configs/client.json tui
```

Встроенный file/folder picker намеренно ограничен каталогом, из которого запущен Клиент, и его подкаталогами. Перейти выше каталога запуска нельзя. Запускайте `gkeep tui` из каталога, содержащего доступные для выбора файлы и подкаталоги. Также Picker рассчитан на работу с локальными файлами/папками и не выполняет дополнительную проверку символических ссылок и специальных объектов файловой системы.
