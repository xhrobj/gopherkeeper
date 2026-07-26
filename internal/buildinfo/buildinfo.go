package buildinfo

import (
	"fmt"
	"io"
)

// NotAvailable используется вместо отсутствующего значения информации о сборке.
const NotAvailable = "¯\\_(ツ)_/¯"

// Info содержит информацию о сборке приложения.
type Info struct {
	Version string
	Date    string
	Commit  string
}

// Print выводит информацию о сборке в writer.
func Print(w io.Writer, info Info) error {
	if _, err := fmt.Fprintf(w, "Build version: %s\n", Value(info.Version)); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "Build date: %s\n", Value(info.Date)); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "Build commit: %s\n", Value(info.Commit)); err != nil {
		return err
	}

	return nil
}

// Value возвращает value или стандартное обозначение отсутствующего значения.
func Value(value string) string {
	if value == "" {
		return NotAvailable
	}

	return value
}
