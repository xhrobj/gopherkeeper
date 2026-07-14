package middleware

import "net/http"

// WithNoStore запрещает HTTP-кеширование ответа.
func WithNoStore(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Pragma", "no-cache")

		handler.ServeHTTP(w, r)
	})
}
