// Adapted from https://github.com/go-mods/zerolog-gin

package server

import (
	"context"
	"time"

	"github.com/smoothdb/smoothdb/database"
	"github.com/smoothdb/smoothdb/logging"

	"github.com/rs/zerolog"
)

func HTTPLogger(logger *logging.Logger) Middleware {
	return func(next Handler) Handler {
		zlog := logger.With().Str("domain", "HTTP").Logger()

		return func(ctx context.Context, w ResponseWriter, r *Request) {

			// return if zerolog is disabled
			if zlog.GetLevel() == zerolog.Disabled {
				next(ctx, w, r)
				return
			}

			// before executing the next handlers
			begin := time.Now()
			path := r.URL.Path
			raw := r.URL.RawQuery
			if raw != "" {
				path = path + "?" + raw
			}

			// executes the next handler
			next(ctx, w, r)

			// after executing the handlers
			statusCode := w.Status()

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
				if w.Err() != nil {
					event.Str("err", w.Err().Error())
				}

				// post the message
				event.Msg("Request")
			}
		}
	}
}
