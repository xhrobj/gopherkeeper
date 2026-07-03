package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

type passwordReader interface {
	ReadHidden(input io.Reader, output io.Writer, prompt string) (string, error)
	ReadLine(input io.Reader) (string, error)
}

type terminalPasswordReader struct{}

func (terminalPasswordReader) ReadHidden(
	input io.Reader,
	output io.Writer,
	prompt string,
) (string, error) {
	file, ok := input.(*os.File)
	if !ok || !term.IsTerminal(int(file.Fd())) {
		return "", errors.New("password input is not a terminal; use --password-stdin")
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

func (terminalPasswordReader) ReadLine(input io.Reader) (string, error) {
	password, err := bufio.NewReader(input).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("read password from stdin: %w", err)
	}
	if errors.Is(err, io.EOF) && password == "" {
		return "", errors.New("read password from stdin: no data")
	}

	return strings.TrimSuffix(strings.TrimSuffix(password, "\n"), "\r"), nil
}
