package tui

import (
	"errors"
	"os/exec"
	"runtime"
	"strings"

	tea "charm.land/bubbletea/v2"
)

const aboutURL = "https://practicum.yandex.ru/go-advanced/"

type openURLFunc func(string) error

type openURLResultMsg struct {
	err error
}

func openURLCommand(open openURLFunc, value string) tea.Cmd {
	return func() tea.Msg {
		if open == nil {
			return openURLResultMsg{err: errors.New("link opener is unavailable")}
		}
		return openURLResultMsg{err: open(value)}
	}
}

func openExternalURL(value string) error {
	command, err := browserCommand(runtime.GOOS, value)
	if err != nil {
		return err
	}
	return command.Start()
}

func browserCommand(goos, value string) (*exec.Cmd, error) {
	switch goos {
	case "darwin":
		return exec.Command("open", value), nil
	case "linux":
		return exec.Command("xdg-open", value), nil
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", value), nil
	default:
		return nil, errors.New("opening links is not supported on this platform")
	}
}

func cleanOpenURLError(err error) string {
	if err == nil {
		return "Unknown browser error"
	}

	message := strings.TrimSpace(err.Error())
	if message == "" {
		return "Unable to start the system browser"
	}

	return capitalizeFirst(message)
}
