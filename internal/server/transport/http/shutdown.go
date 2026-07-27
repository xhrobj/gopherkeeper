package httpserver

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
)

const shutdownTimeout = 30 * time.Second

// ServeTLS запускает HTTPS-сервер и корректно останавливает его при отмене контекста.
func ServeTLS(ctx context.Context, server *http.Server, certFile, keyFile string) error {
	if err := serve(ctx, server, func() error {
		return server.ListenAndServeTLS(certFile, keyFile)
	}); err != nil {
		return fmt.Errorf("serve HTTPS: %w", err)
	}

	return nil
}

func serve(ctx context.Context, server *http.Server, serveFunc func() error) error {
	serveErr := make(chan error, 1)
	go func() {
		serveErr <- serveFunc()
	}()

	select {
	case err := <-serveErr:
		return ignoreServerClosed(err)
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			_ = server.Close()

			return fmt.Errorf("shutdown HTTPS server: %w", err)
		}

		return ignoreServerClosed(<-serveErr)
	}
}

func ignoreServerClosed(err error) error {
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}

	return err
}
