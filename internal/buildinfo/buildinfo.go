package buildinfo

import (
	"fmt"
)

func Print(version, date, commit string) {
	fmt.Printf("Build version: %s\n", value(version))
	fmt.Printf("Build date: %s\n", value(date))
	fmt.Printf("Build commit: %s\n", value(commit))
}

func value(v string) string {
	const notAvailable = "¯\\_(ツ)_/¯"

	if v == "" {
		return notAvailable
	}

	return v
}
