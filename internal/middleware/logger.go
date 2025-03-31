package middleware

import (
	"fmt"
	"time"

	"github.com/bestruirui/bestsub/internal/logger"
	"github.com/gin-gonic/gin"
)

// RequestLogger Returns a middleware that logs HTTP request details
// Includes request path, method, status code, IP address, and processing time
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()

		path := c.Request.URL.Path

		query := c.Request.URL.RawQuery
		if query != "" {
			path = path + "?" + query
		}

		clientIP := c.ClientIP()

		c.Next()

		endTime := time.Now()
		latency := endTime.Sub(startTime)

		statusCode := c.Writer.Status()
		method := c.Request.Method

		logMsg := fmt.Sprintf("[HTTP] %-7s| %3d | %10v | %10s | %s",
			method,
			statusCode,
			latency,
			clientIP,
			path,
		)

		if statusCode >= 500 {
			logger.Error(logMsg)
		} else if statusCode >= 400 {
			logger.Warn(logMsg)
		} else {
			logger.Info(logMsg)
		}
	}
}
