// Adapted from https://github.com/go-mods/zerolog-gin

package server

import (
	"time"

	"github.com/smoothdb/smoothdb/database"
	"github.com/smoothdb/smoothdb/logging"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// HTTPLogger is a gin middleware which use zerolog
func HTTPLogger(logger *logging.Logger) gin.HandlerFunc {

	zlog := logger.With().Str("domain", "HTTP").Logger()

	return func(ctx *gin.Context) {

		// return if zerolog is disabled
		if zlog.GetLevel() == zerolog.Disabled {
			ctx.Next()
			return
		}

		// before executing the next handlers
		begin := time.Now()
		path := ctx.Request.URL.Path
		raw := ctx.Request.URL.RawQuery
		if raw != "" {
			path = path + "?" + raw
		}

		// executes the pending handlers
		ctx.Next()

		// after executing the handlers
		statusCode := ctx.Writer.Status()

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
			event.Str("method", ctx.Request.Method)
			event.Str("path", path)
			event.Int("status", statusCode)
			if len(ctx.Errors) > 0 {
				event.Str("err", ctx.Errors[0].Error())
			}

			// post the message
			event.Msg("Request")
		}
	}
}
