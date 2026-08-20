package logger

import (
	"fmt"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
)

func RequestID(c *gin.Context) string {
	if id, exists := c.Get("request_id"); exists {
		return id.(string)
	}
	return ""
}

func RequestMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		clientIP := c.ClientIP()

		logLevel := LevelInfo
		if status >= 500 {
			logLevel = LevelError
		} else if status >= 400 {
			logLevel = LevelWarning
		}

		ctx := map[string]interface{}{
			"status":  status,
			"method":  method,
			"path":    path,
			"ip":      clientIP,
			"latency": fmt.Sprintf("%v", latency),
			"bytes":   c.Writer.Size(),
		}

		msg := fmt.Sprintf("%s %s %d %v", method, path, status, latency)
		writeLog(logLevel, msg, ctx)
	}
}

func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				stack := debug.Stack()
				ctx := map[string]interface{}{
					"panic":  fmt.Sprintf("%v", rec),
					"path":   c.Request.URL.Path,
					"method": c.Request.Method,
					"ip":     c.ClientIP(),
				}
				writeLog(LevelError, fmt.Sprintf("PANIC RECOVERED: %v\n%s", rec, stack), ctx)
				c.AbortWithStatus(500)
			}
		}()
		c.Next()
	}
}
