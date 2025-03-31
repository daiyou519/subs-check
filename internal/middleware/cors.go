package middleware

import (
	"github.com/gin-gonic/gin"
)

// Cors middleware
// @Summary CORS middleware
// @Description Allows all cross-origin requests in debug mode, only same-origin requests in release mode
func Cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if origin == "" {
			c.Next()
			return
		}

		if gin.Mode() == gin.DebugMode {
			c.Header("Access-Control-Allow-Origin", "*")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
			c.Header("Access-Control-Allow-Credentials", "true")

			if c.Request.Method == "OPTIONS" {
				c.AbortWithStatus(204)
				return
			}
		} else {
			host := c.Request.Host
			if origin == "http://"+host || origin == "https://"+host {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
				c.Header("Access-Control-Allow-Credentials", "true")

				if c.Request.Method == "OPTIONS" {
					c.AbortWithStatus(204)
					return
				}
			} else {
				c.AbortWithStatusJSON(403, gin.H{
					"code":    403,
					"message": "Cross-origin requests not allowed",
					"data":    "",
				})
				return
			}
		}

		c.Next()
	}
}
