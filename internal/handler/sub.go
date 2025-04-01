package handler

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/bestruirui/bestsub/internal/logger"
	"github.com/bestruirui/bestsub/internal/middleware"
	"github.com/bestruirui/bestsub/internal/model"
	"github.com/bestruirui/bestsub/internal/repository"
	"github.com/bestruirui/bestsub/internal/router"
	"github.com/bestruirui/bestsub/internal/service"
	"github.com/bestruirui/bestsub/internal/validator"
	"github.com/gin-gonic/gin"
)

// SubHandler Handles subscription related HTTP requests
type SubHandler struct {
	subRepo    repository.SubRepository
	subFetcher *service.SubFetcher
	config     *model.Config
}

// NewSubHandler Creates a new subscription handler instance
func NewSubHandler(db *sql.DB, config *model.Config) *SubHandler {
	subRepo := repository.NewSubRepository(db)
	subFetcher := service.NewSubFetcher(subRepo)

	return &SubHandler{
		subRepo:    subRepo,
		subFetcher: subFetcher,
		config:     config,
	}
}

// Groups Returns all route group configurations
func (h *SubHandler) Groups() []*router.GroupRouter {
	return []*router.GroupRouter{
		h.SubGroup(),
	}
}

// SubGroup Returns subscription API route group
func (h *SubHandler) SubGroup() *router.GroupRouter {
	// Use chain API to create route group
	return router.NewGroupRouter("/api/sub").
		Use(middleware.JWTAuth(h.config)).
		AddRoute(
			router.NewRoute("/add", router.POST).
				Handle(h.CreateSub).
				WithDescription("Create subscription"),
		).
		AddRoute(
			router.NewRoute("/list", router.GET).
				Handle(h.GetAllSubs).
				WithDescription("Get all subscriptions"),
		).
		AddRoute(
			router.NewRoute("/:id", router.GET).
				Handle(h.GetSub).
				WithDescription("Get subscription details"),
		).
		AddRoute(
			router.NewRoute("/:id/content", router.GET).
				Handle(h.FetchSubContent).
				WithDescription("Fetch subscription content"),
		).
		AddRoute(
			router.NewRoute("/:id", router.PUT).
				Handle(h.UpdateSub).
				WithDescription("Update subscription"),
		).
		AddRoute(
			router.NewRoute("/:id", router.DELETE).
				Handle(h.DeleteSub).
				WithDescription("Delete subscription"),
		)
}

// GetSub godoc
// @Summary 获取订阅详情
// @Description 根据ID获取订阅详情
// @Tags 订阅
// @Accept json
// @Produce json
// @Param id path int true "订阅ID"
// @Success 200 {object} model.SuccessResponse{data=model.Sub} "成功"
// @Failure 400 {object} model.BadRequestResponse{} "无效请求"
// @Failure 401 {object} model.UnauthorizedResponse{} "未授权"
// @Failure 404 {object} model.NotFoundResponse{} "订阅不存在"
// @Failure 500 {object} model.ServerErrorResponse{} "服务器错误"
// @Router /api/sub/{id} [get]
// @Security BearerAuth
func (h *SubHandler) GetSub(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.StandardResponse{
			Code:    http.StatusBadRequest,
			Message: "Invalid subscription ID",
			Data:    nil,
		})
		return
	}

	sub, err := h.subRepo.GetByID(ctx, id)
	if err != nil {
		status := http.StatusInternalServerError
		message := "Failed to retrieve subscription"

		if errors.Is(err, model.ErrSubNotFound) {
			status = http.StatusNotFound
			message = "Subscription not found"
		}

		c.JSON(status, model.StandardResponse{
			Code:    status,
			Message: message,
			Data:    nil,
		})
		logger.Error("Failed to get subscription: %v, SubID: %d", err, id)
		return
	}

	c.JSON(http.StatusOK, model.StandardResponse{
		Code:    http.StatusOK,
		Message: "Success",
		Data:    sub,
	})
}

// CreateSubRequest Request to create a new subscription
type CreateSubRequest struct {
	URL        string `json:"url" binding:"required"`
	Cron       string `json:"cron" binding:"required"`
	AutoUpdate bool   `json:"auto_update" binding:"required"`
}

// CreateSub godoc
// @Summary 创建新订阅
// @Description 使用提供的URL创建新订阅
// @Tags 订阅
// @Accept json
// @Produce json
// @Param sub body CreateSubRequest true "订阅数据"
// @Success 201 {object} model.SuccessResponse{data=model.Sub} "订阅创建成功"
// @Failure 400 {object} model.BadRequestResponse{} "无效请求"
// @Failure 401 {object} model.UnauthorizedResponse{} "未授权"
// @Failure 409 {object} model.ConflictResponse{} "订阅已存在"
// @Failure 500 {object} model.ServerErrorResponse{} "服务器错误"
// @Router /api/sub/add [post]
// @Security BearerAuth
func (h *SubHandler) CreateSub(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	var req CreateSubRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.BadRequestResponse{
			Code:    http.StatusBadRequest,
			Message: "Invalid request data",
			Data:    nil,
		})
		return
	}

	// 验证cron表达式
	if err := validator.ValidateCron(req.Cron); err != nil {
		c.JSON(http.StatusBadRequest, model.BadRequestResponse{
			Code:    http.StatusBadRequest,
			Message: "Invalid cron expression: " + err.Error(),
			Data:    nil,
		})
		return
	}

	sub := &model.Sub{
		URL:        req.URL,
		TotalNodes: 0,
		AliveNodes: 0,
		Cron:       req.Cron,
		AutoUpdate: req.AutoUpdate,
	}

	if err := h.subRepo.Create(ctx, sub); err != nil {
		status := http.StatusInternalServerError
		message := "Failed to create subscription"

		if errors.Is(err, model.ErrSubExists) {
			status = http.StatusConflict
			message = "Subscription URL already exists"
		}

		c.JSON(status, model.ServerErrorResponse{
			Code:    status,
			Message: message,
			Data:    nil,
		})
		logger.Error("Failed to create subscription: %v", err)
		return
	}

	c.JSON(http.StatusCreated, model.SuccessResponse{
		Code:    http.StatusCreated,
		Message: "Subscription created successfully",
		Data:    sub,
	})
}

// UpdateSubRequest Request to update a subscription
type UpdateSubRequest struct {
	URL        string `json:"url"`
	Cron       string `json:"cron"`
	AutoUpdate *bool  `json:"auto_update"`
}

// UpdateSub godoc
// @Summary 更新订阅
// @Description 更新订阅URL
// @Tags 订阅
// @Accept json
// @Produce json
// @Param id path int true "订阅ID"
// @Param sub body UpdateSubRequest true "更新的订阅数据"
// @Success 200 {object} model.SuccessResponse{data=model.Sub} "订阅已更新"
// @Failure 400 {object} model.BadRequestResponse{} "无效请求"
// @Failure 401 {object} model.UnauthorizedResponse{} "未授权"
// @Failure 404 {object} model.NotFoundResponse{} "订阅不存在"
// @Failure 500 {object} model.ServerErrorResponse{} "服务器错误"
// @Router /api/sub/{id} [put]
// @Security BearerAuth
func (h *SubHandler) UpdateSub(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.StandardResponse{
			Code:    http.StatusBadRequest,
			Message: "Invalid subscription ID",
			Data:    nil,
		})
		return
	}

	sub, err := h.subRepo.GetByID(ctx, id)
	if err != nil {
		status := http.StatusInternalServerError
		message := "Failed to retrieve subscription"

		if errors.Is(err, model.ErrSubNotFound) {
			status = http.StatusNotFound
			message = "Subscription not found"
		}

		c.JSON(status, model.StandardResponse{
			Code:    status,
			Message: message,
			Data:    nil,
		})
		logger.Error("Failed to get subscription for update: %v, SubID: %d", err, id)
		return
	}

	var req UpdateSubRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.BadRequestResponse{
			Code:    http.StatusBadRequest,
			Message: "Invalid request data",
			Data:    nil,
		})
		return
	}

	if req.URL != "" {
		sub.URL = req.URL
	}
	if req.Cron != "" {
		// 验证cron表达式
		if err := validator.ValidateCron(req.Cron); err != nil {
			c.JSON(http.StatusBadRequest, model.BadRequestResponse{
				Code:    http.StatusBadRequest,
				Message: "Invalid cron expression: " + err.Error(),
				Data:    nil,
			})
			return
		}
		sub.Cron = req.Cron
	}
	if req.AutoUpdate != nil {
		sub.AutoUpdate = *req.AutoUpdate
	}

	if err := h.subRepo.Update(ctx, sub); err != nil {
		c.JSON(http.StatusInternalServerError, model.ServerErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Failed to update subscription",
			Data:    nil,
		})
		logger.Error("Failed to update subscription: %v, SubID: %d", err, id)
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse{
		Code:    http.StatusOK,
		Message: "Subscription updated successfully",
		Data:    sub,
	})
}

// DeleteSub godoc
// @Summary 删除订阅
// @Description 根据ID删除订阅
// @Tags 订阅
// @Accept json
// @Produce json
// @Param id path int true "订阅ID"
// @Success 200 {object} model.SuccessResponse{} "订阅已删除"
// @Failure 400 {object} model.BadRequestResponse{} "无效请求"
// @Failure 401 {object} model.UnauthorizedResponse{} "未授权"
// @Failure 404 {object} model.NotFoundResponse{} "订阅不存在"
// @Failure 500 {object} model.ServerErrorResponse{} "服务器错误"
// @Router /api/sub/{id} [delete]
// @Security BearerAuth
func (h *SubHandler) DeleteSub(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.BadRequestResponse{
			Code:    http.StatusBadRequest,
			Message: "Invalid subscription ID",
			Data:    nil,
		})
		return
	}

	if err := h.subRepo.Delete(ctx, id); err != nil {
		status := http.StatusInternalServerError
		message := "Failed to delete subscription"

		if errors.Is(err, model.ErrSubNotFound) {
			status = http.StatusNotFound
			message = "Subscription not found"
		}

		c.JSON(status, model.ServerErrorResponse{
			Code:    status,
			Message: message,
			Data:    nil,
		})
		logger.Error("Failed to delete subscription: %v, SubID: %d", err, id)
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse{
		Code:    http.StatusOK,
		Message: "Subscription deleted successfully",
		Data:    nil,
	})
}

// UpdateStatsRequest Request to update subscription stats
type UpdateStatsRequest struct {
	TotalNodes int `json:"total_nodes" binding:"required,min=0"`
	AliveNodes int `json:"alive_nodes" binding:"required,min=0"`
}

// GetAllSubs godoc
// @Summary 获取所有订阅
// @Description 获取所有订阅的列表
// @Tags 订阅
// @Accept json
// @Produce json
// @Success 200 {object} model.SuccessResponse{data=[]model.Sub} "成功"
// @Failure 500 {object} model.ServerErrorResponse{} "服务器错误"
// @Failure 401 {object} model.UnauthorizedResponse{} "未授权"
// @Router /api/sub/list [get]
// @Security BearerAuth
func (h *SubHandler) GetAllSubs(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	subs, err := h.subRepo.GetAll(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ServerErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Failed to retrieve subscriptions",
			Data:    nil,
		})
		logger.Error("Failed to get all subscriptions: %v", err)
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse{
		Code:    http.StatusOK,
		Message: "Success",
		Data:    subs,
	})
}

// FetchSubContent godoc
// @Summary 获取订阅内容
// @Description 从订阅URL中获取内容并存储到内存中
// @Tags 订阅
// @Accept json
// @Produce json
// @Param id path int true "订阅ID"
// @Success 200 {object} model.SuccessResponse{data=model.Sub} "成功"
// @Failure 400 {object} model.BadRequestResponse{} "无效请求"
// @Failure 404 {object} model.ServerErrorResponse{} "订阅不存在"
// @Failure 500 {object} model.ServerErrorResponse{} "服务器错误"
// @Router /api/sub/{id}/content [get]
// @Security BearerAuth
func (h *SubHandler) FetchSubContent(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.BadRequestResponse{
			Code:    http.StatusBadRequest,
			Message: "Invalid subscription ID",
			Data:    nil,
		})
		return
	}

	// 获取订阅内容
	sub, err := h.subFetcher.FetchSub(ctx, id)
	if err != nil {
		status := http.StatusInternalServerError
		message := "Failed to fetch subscription content"

		if errors.Is(err, model.ErrSubNotFound) {
			status = http.StatusNotFound
			message = "Subscription not found"
		} else if errors.Is(err, model.ErrInvalidSubURL) {
			status = http.StatusBadRequest
			message = "Invalid subscription URL"
		} else if errors.Is(err, model.ErrFetchFailed) {
			status = http.StatusServiceUnavailable
			message = "Failed to fetch subscription data"
		}

		c.JSON(status, model.ServerErrorResponse{
			Code:    status,
			Message: message,
			Data:    nil,
		})
		logger.Error("Failed to fetch subscription content: %v, SubID: %d", err, id)
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse{
		Code:    http.StatusOK,
		Message: "Success",
		Data:    sub,
	})
}
