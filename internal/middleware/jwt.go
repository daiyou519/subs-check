package middleware

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/bestruirui/bestsub/internal/logger"
	"github.com/bestruirui/bestsub/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
)

// Define JWT related errors
var (
	ErrMissingAuthHeader  = errors.New("missing authentication header")
	ErrInvalidAuthFormat  = errors.New("invalid authentication format")
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrInvalidTokenClaims = errors.New("invalid token claims")
)

// JWTAuth JWT authentication middleware
// Verify the Bearer token in the request header and extract the user ID
func JWTAuth(config *model.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			abortWithError(c, http.StatusUnauthorized, ErrMissingAuthHeader)
			return
		}

		// Parse Bearer token
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			abortWithError(c, http.StatusUnauthorized, ErrInvalidAuthFormat)
			return
		}

		// Extract token string
		tokenString := parts[1]

		// Parse and verify JWT token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Verify signature algorithm
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, ErrInvalidToken
			}
			return []byte(config.JWT.Secret), nil
		})

		// Handle invalid token cases
		if err != nil {
			logger.Debug("JWT parse error: %v", err)
			abortWithError(c, http.StatusUnauthorized, ErrInvalidToken)
			return
		}

		// Check if the token is valid
		if !token.Valid {
			abortWithError(c, http.StatusUnauthorized, ErrInvalidToken)
			return
		}

		// Extract claims
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			abortWithError(c, http.StatusUnauthorized, ErrInvalidTokenClaims)
			return
		}

		// Verify expiration time
		if exp, ok := claims["exp"].(float64); ok {
			expTime := time.Unix(int64(exp), 0)
			if time.Now().After(expTime) {
				abortWithError(c, http.StatusUnauthorized, errors.New("token expired"))
				return
			}
		}

		// Extract user ID and store in context
		userID, ok := claims["user_id"].(float64)
		if !ok {
			abortWithError(c, http.StatusUnauthorized, errors.New("invalid user ID in token"))
			return
		}

		// Set user ID to context
		c.Set("user_id", int64(userID))

		// Continue processing request
		c.Next()
	}
}

// abortWithError Aborts request and returns error response
func abortWithError(c *gin.Context, status int, err error) {
	logger.Warn("JWT authentication failed: %v", err)
	c.AbortWithStatusJSON(status, model.StandardResponse{
		Code:    status,
		Message: err.Error(),
		Data:    nil,
	})
}
