package httpserver

import (
	"log/slog"
	"net/http"
	"time"
)

func LogRequests(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(
		writer http.ResponseWriter,
		request *http.Request,
	) {
		start := time.Now()
		recorder := &statusRecorder{
			ResponseWriter: writer,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(recorder, request)

		logger.Info(
			"handled request",
			"method", request.Method,
			"path", request.URL.Path,
			"status", recorder.statusCode,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}

type statusRecorder struct {
	http.ResponseWriter

	statusCode int
}

func (r *statusRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}
