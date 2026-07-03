# 🔐 (^-^)/ GophKeeper. Менеджер паролей и приватных данных

[![(-_-) Go CI](https://github.com/xhrobj/gopherkeeper/actions/workflows/go-ci.yml/badge.svg)](https://github.com/xhrobj/gopherkeeper/actions/workflows/go-ci.yml)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=xhrobj_gophkeeper&metric=alert_status)](https://sonarcloud.io/summary/new_code?id=xhrobj_gophkeeper)
[![Coverage](https://sonarcloud.io/api/project_badges/measure?project=xhrobj_gophkeeper&metric=coverage)](https://sonarcloud.io/summary/new_code?id=xhrobj_gophkeeper)

[Техническое задание](SPECIFICATION.md) · [OpenAPI](api/openapi.yaml)

## Текущее поведение

- `client`, `client -h`, `client --help`, `client help` — баннер и общая справка;
- `client health --help`, `client help health` — справка команды `health` без баннера;
- `client -v`, `client --version` — баннер и полная информация о сборке;
- `client health` — только результат команды.

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

## Регистрация пользователя

Сначала запустить Сервер:

```bash
make run-server
```

В другом терминале выполнить интерактивную регистрацию. Пароль не отображается при вводе и не передаётся через аргументы процесса:

```bash
make run-client-register LOGIN=alice
```

Эквивалентный прямой вызов Клиента:

```bash
./bin/gkeep register --login alice --address localhost:8080 --ca-cert .certs/ca.pem
```

Для CI и скриптов password можно передать одной строкой через stdin. Значение переменной должно поступать из безопасного хранилища секретов, а не записываться буквально в команду:

```bash
printf '%s\n' "$GKEEP_PASSWORD" | ./bin/gkeep register --login alice --password-stdin --address localhost:8080 --ca-cert .certs/ca.pem
```
