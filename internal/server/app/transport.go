package app

import (
	"context"
	"errors"
	"fmt"
)

// Transport описывает один серверный transport и функцию его запуска.
type Transport struct {
	// Name задает имя transport'а для сообщений об ошибках.
	Name string

	// Serve запускает transport и блокируется до его остановки.
	Serve func(context.Context) error
}

// ServeTransports параллельно запускает transport'ы и останавливает остальные,
// когда один из них завершается с ошибкой или неожиданно прекращает работу.
func ServeTransports(ctx context.Context, transports ...Transport) error {
	if len(transports) == 0 {
		return errors.New("server transports are required")
	}

	serveCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	results := make(chan transportResult, len(transports))
	for _, transport := range transports {
		transport := transport
		go func() {
			results <- transportResult{
				name: transport.Name,
				err:  transport.Serve(serveCtx),
			}
		}()
	}

	var resultErr error
	for range transports {
		result := <-results
		switch {
		case result.err != nil:
			resultErr = errors.Join(
				resultErr,
				fmt.Errorf("%s transport: %w", result.name, result.err),
			)
			cancel()
		case serveCtx.Err() == nil:
			resultErr = errors.Join(
				resultErr,
				fmt.Errorf("%s transport stopped unexpectedly", result.name),
			)
			cancel()
		}
	}

	return resultErr
}

type transportResult struct {
	name string
	err  error
}
