package tui

import (
	"context"
	"errors"
	"fmt"
	"io"

	tea "charm.land/bubbletea/v2"
	"github.com/xhrobj/gopherkeeper/internal/buildinfo"
	"github.com/xhrobj/gopherkeeper/internal/client/config"
)

// Options содержит зависимости и параметры запуска терминального интерфейса.
type Options struct {
	Input          io.Reader
	Output         io.Writer
	Config         config.Config
	ConfigFile     string
	Info           buildinfo.Info
	BackendFactory BackendFactory
}

// Run запускает интерактивный терминальный интерфейс Клиента.
func Run(ctx context.Context, options Options) error {
	if options.BackendFactory == nil {
		return errors.New("TUI backend factory is required")
	}

	model := newModel(ctx, options.Config, options.ConfigFile, options.Info, options.BackendFactory)

	program := tea.NewProgram(
		model,
		tea.WithContext(model.ctx),
		tea.WithInput(options.Input),
		tea.WithOutput(options.Output),
	)

	if _, err := program.Run(); err != nil {
		return fmt.Errorf("run Bubble Tea program: %w", err)
	}

	return nil
}
