package httpserver

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

func (w *responseWriter) WriteHeader(status int) {
	if w.wroteHeader {
		return
	}

	w.status = status
	w.wroteHeader = true
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseWriter) Write(body []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}

	size, err := w.ResponseWriter.Write(body)
	w.size += size

	return size, err
}

func (w *responseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func WithLogging(handler http.Handler, lg *zap.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedAt := time.Now()

		response := &responseWriter{
			ResponseWriter: w,
			status:         http.StatusOK,
		}

		handler.ServeHTTP(response, r)

		lg.Info(
			"http request",
			zap.String("method", r.Method),
			zap.String("uri", r.URL.RequestURI()),
			zap.Int("status", response.status),
			zap.Int("size", response.size),
			zap.Duration("duration", time.Since(startedAt)),
		)
	})
}
