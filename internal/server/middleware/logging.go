package middleware

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

type responseWriter struct {
	http.ResponseWriter

	status      int
	size        int
	wroteHeader bool
}

// WriteHeader сохраняет и отправляет HTTP status code один раз.
func (w *responseWriter) WriteHeader(status int) {
	if w.wroteHeader {
		return
	}

	w.status = status
	w.wroteHeader = true
	w.ResponseWriter.WriteHeader(status)
}

// Write отправляет body и учитывает размер ответа.
func (w *responseWriter) Write(body []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}

	size, err := w.ResponseWriter.Write(body)
	w.size += size

	return size, err
}

// Unwrap возвращает исходный ResponseWriter.
func (w *responseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

// WithLogging добавляет логирование HTTP-запросов.
func WithLogging(handler http.Handler, lg *zap.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedAt := time.Now()

		response := &responseWriter{
			ResponseWriter: w,
			status:         http.StatusOK,
		}

		handler.ServeHTTP(response, r)

		fields := []zap.Field{
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Int("status", response.status),
			zap.Int("size", response.size),
			zap.Duration("duration", time.Since(startedAt)),
		}

		if response.status >= http.StatusInternalServerError {
			lg.Error("http request", fields...)
			return
		}

		lg.Debug("http request", fields...)
	})
}
