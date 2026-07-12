package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

var errPasswordInputNotTerminal = errors.New("password input is not a terminal")

type passwordReader interface {
	ReadHidden(input io.Reader, output io.Writer, prompt string) (string, error)
	ReadLine(input io.Reader, output io.Writer, prompt string) (string, error)
}

type terminalPasswordReader struct{}

func (terminalPasswordReader) ReadHidden(
	input io.Reader,
	output io.Writer,
	prompt string,
) (string, error) {
	file, ok := input.(*os.File)
	if !ok || !term.IsTerminal(int(file.Fd())) {
		return "", errPasswordInputNotTerminal
	}

	if _, err := fmt.Fprint(output, prompt); err != nil {
		return "", fmt.Errorf("write password prompt: %w", err)
	}

	password, err := term.ReadPassword(int(file.Fd()))
	if err != nil {
		return "", fmt.Errorf("read password: %w", err)
	}

	if _, err := fmt.Fprintln(output); err != nil {
		return "", fmt.Errorf("finish password prompt: %w", err)
	}

	return string(password), nil
}

func (terminalPasswordReader) ReadLine(
	input io.Reader,
	output io.Writer,
	prompt string,
) (string, error) {
	if _, err := fmt.Fprint(output, prompt); err != nil {
		return "", fmt.Errorf("write input prompt: %w", err)
	}

	return readInputLine(input)
}

type streamPasswordReader struct{}

func (streamPasswordReader) ReadHidden(
	input io.Reader,
	_ io.Writer,
	_ string,
) (string, error) {
	return readInputLine(input)
}

func (streamPasswordReader) ReadLine(
	input io.Reader,
	_ io.Writer,
	_ string,
) (string, error) {
	return readInputLine(input)
}

func readInputLine(input io.Reader) (string, error) {
	var line strings.Builder
	var buffer [1]byte

	for {
		read, err := input.Read(buffer[:])
		if read > 0 {
			if buffer[0] == '\n' {
				return strings.TrimSuffix(line.String(), "\r"), nil
			}

			line.WriteByte(buffer[0])
		}

		if err == nil {
			continue
		}
		if !errors.Is(err, io.EOF) {
			return "", fmt.Errorf("read line from stdin: %w", err)
		}
		if line.Len() == 0 {
			return "", errors.New("read line from stdin: no data")
		}

		return strings.TrimSuffix(line.String(), "\r"), nil
	}
}
