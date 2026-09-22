package http

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/MAbduhI/task-management-api/internal/delivery/http/middleware"
	"github.com/MAbduhI/task-management-api/internal/delivery/http/response"
	"github.com/MAbduhI/task-management-api/internal/domain"
	"github.com/MAbduhI/task-management-api/internal/usecase"
)

type TaskHandler struct {
	taskUsecase usecase.TaskUsecase
}

func NewTaskHandler(taskUsecase usecase.TaskUsecase) *TaskHandler {
	return &TaskHandler{taskUsecase: taskUsecase}
}

func (h *TaskHandler) Create(c *gin.Context) {
	idempKey := c.GetHeader("Idempotency-Key")

	var in domain.CreateTaskInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, domain.ErrBadReq("VALIDATION_ERROR", err.Error(), err))
		return
	}

	currentUserID := middleware.GetCurrentUserID(c)
	task, cached, err := h.taskUsecase.Create(c.Request.Context(), currentUserID, idempKey, in)
	if err != nil {
		response.Error(c, err)
		return
	}

	if cached {
		c.Header("X-Idempotency-Cache", "HIT")
	}

	response.Created(c, task)
}

func (h *TaskHandler) List(c *gin.Context) {
	currentUserID := middleware.GetCurrentUserID(c)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	cursor, _ := strconv.ParseInt(c.Query("cursor"), 10, 64)
	status := c.Query("status")
	search := c.Query("search")
	if search == "" {
		search = c.Query("title")
	}

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	query := domain.ListTaskQuery{
		UserID: currentUserID,
		Status: status,
		Search: search,
		Page:   page,
		Limit:  limit,
		Cursor: cursor,
	}

	result, err := h.taskUsecase.List(c.Request.Context(), query)
	if err != nil {
		response.Error(c, err)
		return
	}

	var taskResponses []domain.TaskResponse
	for i := range result.Items {
		item := &result.Items[i]
		var creatorUUID uuid.UUID
		if item.Creator != nil {
			creatorUUID = item.Creator.UUID
		}
		var assigneeUUID *uuid.UUID
		if item.Assignee != nil {
			assigneeUUID = &item.Assignee.UUID
		}

		taskResponses = append(taskResponses, domain.TaskResponse{
			UUID:         item.UUID,
			Title:        item.Title,
			Description:  item.Description,
			Status:       item.Status,
			CreatorUUID:  creatorUUID,
			AssigneeUUID: assigneeUUID,
			CreatedAt:    item.CreatedAt,
			UpdatedAt:    item.UpdatedAt,
		})
	}

	totalPages := 0
	if limit > 0 {
		totalPages = int((result.TotalCount + int64(limit) - 1) / int64(limit))
	}
	hasNext := false
	if cursor > 0 {
		hasNext = len(taskResponses) == limit
	} else {
		hasNext = page < totalPages
	}

	meta := &response.Meta{
		Page:        page,
		Limit:       limit,
		TotalItems:  result.TotalCount,
		TotalPages:  totalPages,
		NextCursor:  result.NextCursor,
		HasNextPage: hasNext,
	}

	response.List(c, taskResponses, meta)
}

func (h *TaskHandler) GetDetail(c *gin.Context) {
	taskUUID, err := uuid.Parse(c.Param("uuid"))
	if err != nil {
		response.Error(c, domain.ErrBadReq("INVALID_UUID", "Invalid task UUID", err))
		return
	}

	currentUserID := middleware.GetCurrentUserID(c)
	task, err := h.taskUsecase.GetByUUID(c.Request.Context(), currentUserID, taskUUID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, task)
}

func (h *TaskHandler) Update(c *gin.Context) {
	taskUUID, err := uuid.Parse(c.Param("uuid"))
	if err != nil {
		response.Error(c, domain.ErrBadReq("INVALID_UUID", "Invalid task UUID", err))
		return
	}

	var in domain.UpdateTaskInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, domain.ErrBadReq("VALIDATION_ERROR", err.Error(), err))
		return
	}

	currentUserID := middleware.GetCurrentUserID(c)
	task, err := h.taskUsecase.Update(c.Request.Context(), currentUserID, taskUUID, in)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, task)
}

func (h *TaskHandler) Delete(c *gin.Context) {
	taskUUID, err := uuid.Parse(c.Param("uuid"))
	if err != nil {
		response.Error(c, domain.ErrBadReq("INVALID_UUID", "Invalid task UUID", err))
		return
	}

	currentUserID := middleware.GetCurrentUserID(c)
	if err := h.taskUsecase.Delete(c.Request.Context(), currentUserID, taskUUID); err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, gin.H{"deleted": true})
}

func (h *TaskHandler) Assign(c *gin.Context) {
	taskUUID, err := uuid.Parse(c.Param("uuid"))
	if err != nil {
		response.Error(c, domain.ErrBadReq("INVALID_UUID", "Invalid task UUID", err))
		return
	}

	var in domain.AssignTaskInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, domain.ErrBadReq("VALIDATION_ERROR", err.Error(), err))
		return
	}

	currentUserID := middleware.GetCurrentUserID(c)
	task, err := h.taskUsecase.Assign(c.Request.Context(), currentUserID, taskUUID, in.AssigneeUUID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, task)
}
