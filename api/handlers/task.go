package handlers

import (
	"github.com/drama-generator/backend/application/services"
	"github.com/drama-generator/backend/pkg/logger"
	"github.com/drama-generator/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type TaskHandler struct {
	taskService *services.TaskService
	log         *logger.Logger
}

func NewTaskHandler(db *gorm.DB, log *logger.Logger) *TaskHandler {
	return &TaskHandler{
		taskService: services.NewTaskService(db, log),
		log:         log,
	}
}

func (h *TaskHandler) GetTaskStatus(c *gin.Context) {
	taskID := c.Param("task_id")

	task, err := h.taskService.GetTask(taskID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			response.NotFound(c, "Task not found")
			return
		}
		h.log.Errorw("Failed to get task", "error", err, "task_id", taskID)
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, task)
}

func (h *TaskHandler) GetResourceTasks(c *gin.Context) {
	resourceID := c.Query("resource_id")
	if resourceID == "" {
		response.BadRequest(c, "Missing resource_id parameter")
		return
	}

	tasks, err := h.taskService.GetTasksByResource(resourceID)
	if err != nil {
		h.log.Errorw("Failed to get resource tasks", "error", err, "resource_id", resourceID)
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, tasks)
}
