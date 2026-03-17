package router

import (
	"net/http"

	"github.com/shashiranjanraj/kashvi/config"
	"github.com/shashiranjanraj/kashvi/pkg/apperror"
	"github.com/shashiranjanraj/kashvi/pkg/logger"
	"github.com/shashiranjanraj/kashvi/pkg/response"
)

// ErrHandlerFunc is an HTTP handler that returns an error.
type ErrHandlerFunc func(http.ResponseWriter, *http.Request) error

// Wrap converts an ErrHandlerFunc into a standard http.HandlerFunc.
// If the handler returns an error, it responds to the client automatically
// using the `response` package based on whether it is an `apperror.Error` or a generic error.
func Wrap(h ErrHandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := h(w, r)
		if err != nil {
			// Convert to an *apperror.Error (defaults to 500 Internal if it's a generic error)
			appErr := apperror.TryConvert(err)
			
			// Log the internal error if there is one
			if appErr.Err != nil {
				log := logger.WithCtx(r.Context())
				log.Error("handler error",
					"error", appErr.Err.Error(),
					"status", appErr.StatusCode,
					"path", r.URL.Path,
					"at", appErr.Location,
				)
			}

			// Reply to the client (include debug info only in development)
			if config.AppEnv() == "local" || config.AppEnv() == "development" || config.AppEnv() == "dev" {
				var cause string
				if appErr.Err != nil {
					cause = appErr.Err.Error()
				}
				response.DebugError(w, appErr.StatusCode, appErr.Message, map[string]any{
					"at":    appErr.Location,
					"cause": cause,
				})
				return
			}
			response.Error(w, appErr.StatusCode, appErr.Message)
		}
	}
}
