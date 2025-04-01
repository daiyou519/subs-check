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
	"github.com/bestruirui/bestsub/internal/service"
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
	userSvc  *service.UserService
	config   *model.Config
}

// NewUserHandler Creates new user handler
func NewUserHandler(db *sql.DB, config *model.Config) *UserHandler {
	userRepo := repository.NewUserRepository(db)
	return &UserHandler{
		userRepo: userRepo,
		userSvc:  service.NewUserService(userRepo),
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

// Login godoc
// @Summary 用户登录
// @Description 用户登录并获取JWT令牌
// @Tags 用户
// @Accept json
// @Produce json
// @Param request body LoginRequest true "登录请求参数"
// @Success 200 {object} model.SuccessResponse{data=LoginResponse} "登录成功"
// @Failure 400 {object} model.BadRequestResponse{} "无效的请求参数"
// @Failure 401 {object} model.UnauthorizedResponse{} "用户名或密码错误"
// @Failure 500 {object} model.ServerErrorResponse{} "服务器内部错误"
// @Router /api/user/login [post]
func (h *UserHandler) Login(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), RequestTimeout)
	defer cancel()

	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.BadRequestResponse{
			Code:    http.StatusBadRequest,
			Message: "Invalid request parameters",
			Data:    nil,
		})
		return
	}

	user, err := h.userSvc.Authenticate(ctx, req.Username, req.Password)
	if err != nil {
		status := http.StatusInternalServerError
		message := "Internal server error"

		if errors.Is(err, service.ErrInvalidCredentials) {
			status = http.StatusUnauthorized
			message = "Invalid username or password"
		}

		c.JSON(status, model.ServerErrorResponse{
			Code:    status,
			Message: message,
			Data:    nil,
		})
		logger.Error("Login failed: %v", err)
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
		c.JSON(http.StatusInternalServerError, model.ServerErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Failed to generate token",
			Data:    nil,
		})
		logger.Error("Failed to generate JWT token: %v", err)
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse{
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
// @Summary 用户登出
// @Description 用户登出并使JWT令牌失效
// @Tags 用户
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} model.SuccessResponse{} "登出成功"
// @Failure 401 {object} model.UnauthorizedResponse{} "未授权"
// @Failure 500 {object} model.ServerErrorResponse{} "服务器错误"
// @Router /api/user/logout [post]
func (h *UserHandler) Logout(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, model.UnauthorizedResponse{
			Code:    http.StatusUnauthorized,
			Message: "Unauthorized",
			Data:    nil,
		})
		return
	}

	logger.Info("User logged out: UserID=%d", userID.(int64))

	c.JSON(http.StatusOK, model.SuccessResponse{
		Code:    http.StatusOK,
		Message: "Logout successful",
		Data:    nil,
	})
}

// GetUserInfo godoc
// @Summary 获取用户信息
// @Description 获取当前登录用户的信息
// @Tags 用户
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} model.SuccessResponse{data=model.User} "成功"
// @Failure 401 {object} model.UnauthorizedResponse{} "未授权"
// @Failure 404 {object} model.NotFoundResponse{} "用户不存在"
// @Failure 500 {object} model.ServerErrorResponse{} "服务器错误"
// @Router /api/user/info [get]
func (h *UserHandler) GetUserInfo(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), RequestTimeout)
	defer cancel()

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, model.UnauthorizedResponse{
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

		c.JSON(status, model.ServerErrorResponse{
			Code:    status,
			Message: message,
			Data:    nil,
		})
		logger.Error("Failed to get user info: %v, UserID: %d", err, userID)
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse{
		Code:    http.StatusOK,
		Message: "Success",
		Data:    h.userSvc.SanitizeUser(user),
	})
}

// UpdateUserInfoRequest Update user information request
type UpdateUserInfoRequest struct {
	OldPassword string `json:"old_password" binding:"omitempty"`
	NewPassword string `json:"new_password" binding:"omitempty,min=6"`
	Username    string `json:"username" binding:"omitempty"`
}

// UpdateUserInfo godoc
// @Summary 更新用户信息
// @Description 更新用户信息（用户名、密码）
// @Tags 用户
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body UpdateUserInfoRequest true "更新用户信息请求参数"
// @Success 200 {object} model.SuccessResponse{} "更新成功"
// @Failure 400 {object} model.BadRequestResponse{} "无效的请求参数"
// @Failure 401 {object} model.UnauthorizedResponse{} "未授权或旧密码错误"
// @Failure 404 {object} model.NotFoundResponse{} "用户不存在"
// @Failure 500 {object} model.ServerErrorResponse{} "服务器错误"
// @Router /api/user/info [put]
func (h *UserHandler) UpdateUserInfo(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), RequestTimeout)
	defer cancel()

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, model.UnauthorizedResponse{
			Code:    http.StatusUnauthorized,
			Message: "Unauthorized",
			Data:    nil,
		})
		return
	}

	var req UpdateUserInfoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.BadRequestResponse{
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

		c.JSON(status, model.ServerErrorResponse{
			Code:    status,
			Message: message,
			Data:    nil,
		})
		logger.Error("Failed to get user for update: %v, UserID: %d", err, userID)
		return
	}

	if req.OldPassword != "" && req.NewPassword != "" {
		if err := h.userSvc.ChangePassword(ctx, user.ID, req.OldPassword, req.NewPassword); err != nil {
			status := http.StatusInternalServerError
			message := "Failed to update password"

			if errors.Is(err, service.ErrInvalidCredentials) {
				status = http.StatusUnauthorized
				message = "Invalid old password"
			}

			c.JSON(status, model.ServerErrorResponse{
				Code:    status,
				Message: message,
				Data:    nil,
			})
			logger.Error("Failed to change password: %v", err)
			return
		}
	}

	if req.Username != "" && req.Username != user.Username {
		user.Username = req.Username
		if err := h.userSvc.UpdateUserInfo(ctx, user); err != nil {
			status := http.StatusInternalServerError
			message := "Failed to update username"

			if errors.Is(err, repository.ErrUserExists) {
				status = http.StatusBadRequest
				message = "Username already exists"
			}

			c.JSON(status, model.ServerErrorResponse{
				Code:    status,
				Message: message,
				Data:    nil,
			})
			logger.Error("Failed to update username: %v", err)
			return
		}
	}

	c.JSON(http.StatusOK, model.SuccessResponse{
		Code:    http.StatusOK,
		Message: "User information updated successfully",
		Data:    nil,
	})
}
