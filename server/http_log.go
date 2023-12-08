// Adapted from https://github.com/go-mods/zerolog-gin

package server

import (
	"context"
	"net/http"
	"time"

	"github.com/rs/zerolog"
	"github.com/sted/heligo"
	"github.com/sted/smoothdb/database"
	"github.com/sted/smoothdb/logging"
)

func HTTPLogger(logger *logging.Logger) heligo.Middleware {
	return func(next heligo.Handler) heligo.Handler {
		zlog := logger.With().Str("domain", "HTTP").Logger()

		return func(ctx context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
			// return if zerolog is disabled
			if zlog.GetLevel() == zerolog.Disabled {
				return next(ctx, w, r)
			}

			// before executing the next handlers
			begin := time.Now()
			path := r.URL.Path
			raw := r.URL.RawQuery
			if raw != "" {
				path = path + "?" + raw
			}

			// executes the next handler
			statusCode, err := next(ctx, w, r)

			//
			var event *zerolog.Event

			// set message level
			if statusCode >= 400 && statusCode < 500 {
				event = zlog.Warn()
			} else if statusCode >= 500 {
				event = zlog.Error()
			} else {
				event = zlog.Trace()
			}

			if event.Enabled() {
				duration := time.Since(begin)
				gctx := database.GetSmoothContext(ctx)
				var role string
				if gctx != nil {
					role = gctx.Role
				}

				event.Dur("elapsed", duration)
				event.Str("role", role)
				event.Str("method", r.Method)
				event.Str("path", path)
				event.Int("status", statusCode)
				if err != nil {
					event.Str("err", err.Error())
				}

				// post the message
				event.Msg("Request")
			}
			return statusCode, err
		}
	}
}
