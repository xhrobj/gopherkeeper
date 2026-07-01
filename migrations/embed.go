// Package migrations содержит SQL-миграции PostgreSQL, встроенные в бинарник Сервера.
package migrations

import "embed"

// Files содержит встроенные SQL-файлы миграций.
//
//go:embed *.sql
var Files embed.FS
