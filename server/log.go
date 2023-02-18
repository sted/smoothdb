package server

import (
	"time"

	"github.com/smoothdb/smoothdb/database"
	"github.com/smoothdb/smoothdb/logging"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// ZeroLogger is a gin middleware which use zerolog
func ZeroLogger(logger *logging.Logger) gin.HandlerFunc {

	//
	// hostname, err := os.Hostname()
	// if err != nil {
	// 	hostname = "unknown"
	// }

	return func(ctx *gin.Context) {
		// get zerolog
		z := logger

		// return if zerolog is disabled
		if z.GetLevel() == zerolog.Disabled {
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

		// // Get payload from request
		// var payload []byte
		// payload, _ = io.ReadAll(ctx.Request.Body)
		// ctx.Request.Body = io.NopCloser(bytes.NewReader(payload))
		// // Get a copy of the body
		// w := &responseBodyWriter{body: &bytes.Buffer{}, ResponseWriter: ctx.Writer}
		// ctx.Writer = w

		// executes the pending handlers
		ctx.Next()

		// after executing the handlers
		duration := time.Since(begin)
		statusCode := ctx.Writer.Status()
		gctx := database.GetSmoothContext(ctx)
		var role string
		if gctx != nil {
			role = gctx.Role
		}

		//
		var event *zerolog.Event

		// set message level
		if statusCode >= 400 && statusCode < 500 {
			event = z.Warn()
		} else if statusCode >= 500 {
			event = z.Error()
		} else {
			event = z.Trace()
		}

		// // Name field
		// event.Str(NameFieldName, opt.Name)
		// // Hostname field
		// event.Str("hostname", hostname)

		// Role field
		event.Str("role", role)
		// ClientIP field
		event.Str("clientIP", ctx.ClientIP())

		// // UserAgent field
		// event.Str(UserAgentFieldName, ctx.Request.UserAgent())
		// Method field
		event.Str("method", ctx.Request.Method)
		// Path field
		event.Str("path", path)
		// // Payload field
		// event.Str(PayloadFieldName, string(payload))
		// // Timestamp field
		// event.Time(TimestampFieldName, begin)
		// Duration field
		event.Dur("elapsed", duration)
		// // Referer field
		// event.Str(RefererFieldName, ctx.Request.Referer())
		// statusCode field
		event.Int("status", statusCode)
		// // DataLength field
		// event.Int(DataLengthFieldName, ctx.Writer.Size())

		// Message
		var message string
		if len(ctx.Errors) > 0 {
			message = ctx.Errors[0].Error()
		} else {
			message = "Request"
		}

		// post the message
		event.Msg(message)
	}
}
