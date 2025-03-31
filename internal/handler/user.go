package handler

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/bestruirui/bestsub/internal/logger"
	"github.com/bestruirui/bestsub/internal/middleware"
	"github.com/bestruirui/bestsub/internal/model"
	"github.com/bestruirui/bestsub/internal/repository"
	"github.com/bestruirui/bestsub/internal/router"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
)

const (
	// DefaultJWTExpiryHours Default JWT token expiration time (hours)
	DefaultJWTExpiryHours = 24
	// RequestTimeout Request processing timeout
	RequestTimeout = 10 * time.Second
)

// UserHandler User related request handler
type UserHandler struct {
	userRepo repository.UserRepository
	config   *model.Config
}

// NewUserHandler Creates new user handler
func NewUserHandler(db *sql.DB, config *model.Config) *UserHandler {
	return &UserHandler{
		userRepo: repository.NewUserRepository(db),
		config:   config,
	}
}

// Groups Returns all route group configurations
func (h *UserHandler) Groups() []*router.GroupRouter {
	return []*router.GroupRouter{
		router.NewGroupRouter("/api/user").
			AddRoute(
				router.NewRoute("/login", router.POST).
					Handle(h.Login).
					WithDescription("User login"),
			),
		h.UserGroup(),
	}
}

// UserGroup Returns user related API route group
func (h *UserHandler) UserGroup() *router.GroupRouter {
	// Use chain API to create route group
	return router.NewGroupRouter("/api/user").
		Use(middleware.JWTAuth(h.config)).
		AddRoute(
			router.NewRoute("/logout", router.POST).
				Handle(h.Logout).
				WithDescription("User logout"),
		).
		AddRoute(
			router.NewRoute("/info", router.GET).
				Handle(h.GetUserInfo).
				WithDescription("Get user information"),
		).
		AddRoute(
			router.NewRoute("/info", router.PUT).
				Handle(h.UpdateUserInfo).
				WithDescription("Update user information"),
		)
}

// LoginRequest Login request parameters
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse Login response data
type LoginResponse struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Token    string `json:"token"`
	Exp      int64  `json:"exp"`
}

// StandardResponse API standard response structure
type StandardResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// Login godoc
// @Summary User login
// @Description User login and get JWT token
// @Tags User
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login request parameters"
// @Success 200 {object} StandardResponse{data=LoginResponse} "Login successful"
// @Failure 400 {object} StandardResponse{} "Invalid request parameters"
// @Failure 401 {object} StandardResponse{} "Invalid username or password"
// @Failure 500 {object} StandardResponse{} "Internal server error"
// @Router /api/user/login [post]
func (h *UserHandler) Login(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), RequestTimeout)
	defer cancel()

	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, StandardResponse{
			Code:    http.StatusBadRequest,
			Message: "Invalid request parameters",
			Data:    nil,
		})
		return
	}

	user, err := h.userRepo.GetByUsername(ctx, req.Username)
	if err != nil {
		status := http.StatusInternalServerError
		message := "Internal server error"

		if errors.Is(err, repository.ErrUserNotFound) {
			status = http.StatusUnauthorized
			message = "Invalid username or password"
		}

		c.JSON(status, StandardResponse{
			Code:    status,
			Message: message,
			Data:    nil,
		})
		logger.Error("Login failed: %v", err)
		return
	}

	if !user.CheckPassword(req.Password) {
		c.JSON(http.StatusUnauthorized, StandardResponse{
			Code:    http.StatusUnauthorized,
			Message: "Invalid username or password",
			Data:    nil,
		})
		return
	}

	expiresIn := h.config.JWT.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = DefaultJWTExpiryHours
	}
	expTime := time.Now().Add(time.Hour * time.Duration(expiresIn))
	expUnix := expTime.Unix()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"exp":     expUnix,
	})

	tokenString, err := token.SignedString([]byte(h.config.JWT.Secret))
	if err != nil {
		c.JSON(http.StatusInternalServerError, StandardResponse{
			Code:    http.StatusInternalServerError,
			Message: "Failed to generate token",
			Data:    nil,
		})
		logger.Error("Failed to generate JWT token: %v", err)
		return
	}

	c.JSON(http.StatusOK, StandardResponse{
		Code:    http.StatusOK,
		Message: "Login successful",
		Data: LoginResponse{
			ID:       user.ID,
			Username: user.Username,
			Token:    tokenString,
			Exp:      expUnix,
		},
	})
}

// Logout godoc
// @Summary User logout
// @Description User logout and invalidate JWT token
// @Tags User
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} StandardResponse{} "Logout successful"
// @Failure 401 {object} StandardResponse{} "Unauthorized"
// @Router /api/user/logout [post]
func (h *UserHandler) Logout(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, StandardResponse{
			Code:    http.StatusUnauthorized,
			Message: "Unauthorized",
			Data:    nil,
		})
		return
	}

	logger.Info("User logged out: UserID=%d", userID.(int64))

	c.JSON(http.StatusOK, StandardResponse{
		Code:    http.StatusOK,
		Message: "Logout successful",
		Data:    nil,
	})
}

// GetUserInfo godoc
// @Summary Get user information
// @Description Get information of the currently logged-in user
// @Tags User
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} StandardResponse{data=model.User} "Success"
// @Failure 401 {object} StandardResponse{} "Unauthorized"
// @Failure 404 {object} StandardResponse{} "User not found"
// @Failure 500 {object} StandardResponse{} "Internal server error"
// @Router /api/user/info [get]
func (h *UserHandler) GetUserInfo(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), RequestTimeout)
	defer cancel()

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, StandardResponse{
			Code:    http.StatusUnauthorized,
			Message: "Unauthorized",
			Data:    nil,
		})
		return
	}

	user, err := h.userRepo.GetByID(ctx, userID.(int64))
	if err != nil {
		status := http.StatusInternalServerError
		message := "Internal server error"

		if errors.Is(err, repository.ErrUserNotFound) {
			status = http.StatusNotFound
			message = "User not found"
		}

		c.JSON(status, StandardResponse{
			Code:    status,
			Message: message,
			Data:    nil,
		})
		logger.Error("Failed to get user info: %v, UserID: %d", err, userID)
		return
	}

	c.JSON(http.StatusOK, StandardResponse{
		Code:    http.StatusOK,
		Message: "Success",
		Data:    user.Sanitize(),
	})
}

// UpdateUserInfoRequest Update user information request
type UpdateUserInfoRequest struct {
	OldPassword string `json:"old_password" binding:"omitempty"`
	NewPassword string `json:"new_password" binding:"omitempty,min=6"`
	Username    string `json:"username" binding:"omitempty"`
}

// UpdateUserInfo godoc
// @Summary Update user information
// @Description Update user information (username, password)
// @Tags User
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body UpdateUserInfoRequest true "Update user information request parameters"
// @Success 200 {object} StandardResponse{} "Update successful"
// @Failure 400 {object} StandardResponse{} "Invalid request parameters"
// @Failure 401 {object} StandardResponse{} "Unauthorized or incorrect old password"
// @Failure 404 {object} StandardResponse{} "User not found"
// @Failure 500 {object} StandardResponse{} "Internal server error"
// @Router /api/user/info [put]
func (h *UserHandler) UpdateUserInfo(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), RequestTimeout)
	defer cancel()

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, StandardResponse{
			Code:    http.StatusUnauthorized,
			Message: "Unauthorized",
			Data:    nil,
		})
		return
	}

	var req UpdateUserInfoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, StandardResponse{
			Code:    http.StatusBadRequest,
			Message: "Invalid request parameters",
			Data:    nil,
		})
		return
	}

	user, err := h.userRepo.GetByID(ctx, userID.(int64))
	if err != nil {
		status := http.StatusInternalServerError
		message := "Internal server error"

		if errors.Is(err, repository.ErrUserNotFound) {
			status = http.StatusNotFound
			message = "User not found"
		}

		c.JSON(status, StandardResponse{
			Code:    status,
			Message: message,
			Data:    nil,
		})
		logger.Error("Failed to get user for update: %v, UserID: %d", err, userID)
		return
	}

	if req.OldPassword != "" && req.NewPassword != "" {
		if !user.CheckPassword(req.OldPassword) {
			c.JSON(http.StatusUnauthorized, StandardResponse{
				Code:    http.StatusUnauthorized,
				Message: "Invalid old password",
				Data:    nil,
			})
			return
		}

		if err := user.SetPassword(req.NewPassword); err != nil {
			c.JSON(http.StatusInternalServerError, StandardResponse{
				Code:    http.StatusInternalServerError,
				Message: "Failed to encrypt new password",
				Data:    nil,
			})
			logger.Error("Failed to set new password: %v", err)
			return
		}

		if err := h.userRepo.UpdatePassword(ctx, user.ID, user.Password); err != nil {
			c.JSON(http.StatusInternalServerError, StandardResponse{
				Code:    http.StatusInternalServerError,
				Message: "Failed to update password",
				Data:    nil,
			})
			logger.Error("Failed to update password in DB: %v", err)
			return
		}
	}

	if req.Username != "" && req.Username != user.Username {
		user.Username = req.Username
		if err := h.userRepo.Update(ctx, user); err != nil {
			status := http.StatusInternalServerError
			message := "Failed to update username"

			if errors.Is(err, repository.ErrUserExists) {
				status = http.StatusBadRequest
				message = "Username already exists"
			}

			c.JSON(status, StandardResponse{
				Code:    status,
				Message: message,
				Data:    nil,
			})
			logger.Error("Failed to update username: %v", err)
			return
		}
	}

	c.JSON(http.StatusOK, StandardResponse{
		Code:    http.StatusOK,
		Message: "User information updated successfully",
		Data:    nil,
	})
}
