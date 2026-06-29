// Package buildinfo предоставляет вывод информации о сборке приложений.
package buildinfo

import (
	"fmt"
	"io"
)

const notAvailable = "¯\\_(ツ)_/¯"

// Info содержит информацию о сборке приложения.
type Info struct {
	Version string
	Date    string
	Commit  string
}

// Print выводит информацию о сборке в writer.
func Print(w io.Writer, info Info) error {
	if _, err := fmt.Fprintf(w, "Build version: %s\n", value(info.Version)); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "Build date: %s\n", value(info.Date)); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "Build commit: %s\n", value(info.Commit)); err != nil {
		return err
	}

	return nil
}

func value(v string) string {
	if v == "" {
		return notAvailable
	}

	return v
}
