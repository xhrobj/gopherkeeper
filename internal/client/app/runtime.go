package app

import (
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/xhrobj/gopherkeeper/internal/client/usecase"
)

// Runtime объединяет online application и lifecycle удалённого transport'а.
type Runtime struct {
	*usecase.Application
	closer    io.Closer
	closeOnce sync.Once
	closeErr  error
}

// Close закрывает принадлежащий runtime удалённый transport.
func (runtime *Runtime) Close() error {
	if runtime == nil {
		return nil
	}

	runtime.closeOnce.Do(func() {
		if runtime.closer == nil {
			return
		}

		if err := runtime.closer.Close(); err != nil {
			runtime.closeErr = fmt.Errorf("close client transport: %w", err)
		}
	})

	return runtime.closeErr
}

func newRuntime(application *usecase.Application, closer io.Closer) (*Runtime, error) {
	if application == nil {
		if closer != nil {
			_ = closer.Close()
		}
		return nil, errors.New("client application is nil")
	}

	return &Runtime{Application: application, closer: closer}, nil
}
