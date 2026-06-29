// Package buildinfo предоставляет вывод информации о сборке приложений.
package buildinfo

import (
	"fmt"
	"io"
)

const notAvailable = "¯\\_(ツ)_/¯"

// Print выводит информацию о сборке в writer.
func Print(w io.Writer, version, date, commit string) error {
	if _, err := fmt.Fprintf(w, "Build version: %s\n", value(version)); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "Build date: %s\n", value(date)); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "Build commit: %s\n", value(commit)); err != nil {
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
