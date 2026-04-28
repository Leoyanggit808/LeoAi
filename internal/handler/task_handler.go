package handler

import (
	"LeoAi/internal/service"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type TaskHandler struct {
	taskService *service.TaskService
}

func NewTaskHandler(taskService *service.TaskService) *TaskHandler {
	return &TaskHandler{taskService: taskService}
}

// 提交异步 AI 任务
func (h *TaskHandler) SubmitTask(c *gin.Context) {
	userID := c.GetUint("user_id") //验证用户登录（从 JWT 中取出 user_id）

	var req struct {
		TaskType   string `json:"task_type" binding:"required,oneof=generate improve rag"`
		Input      string `json:"input" binding:"required"`
		DocumentID uint   `json:"document_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	taskID, err := h.taskService.SubmitTask(userID, req.TaskType, req.Input, req.DocumentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{ // 202 Accepted 表示已接受但正在处理
		"message": "任务已提交，正在处理中",
		"task_id": taskID,
	})
}

// / 查询任务状态
func (h *TaskHandler) GetTaskStatus(c *gin.Context) {
	userID := c.GetUint("user_id")
	// 获取路径参数
	taskIDStr := c.Param("id")
	if taskIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "任务ID不能为空"})
		return
	}
	// 转换为 uint
	taskID, err := strconv.ParseUint(taskIDStr, 10, 32)
	//strconv.ParseUint 是 Go 语言标准库 strconv 包中用于将字符串解析为无符号整数的函数。
	//baseint  	进制	（通常填 10，表示十进制）
	//bitSizeint	指定返回结果的位宽（通常填 0 或 64）
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的任务ID"})
		return
	}

	// 使用 Service 提供的方法（正确方式）
	task, err := h.taskService.GetTaskByID(uint(taskID)) //我们这里调用service中的方法获取到单个任务
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}

	// 安全检查：只能查看自己的任务
	if task.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权查看该任务"})
		return
	}

	c.JSON(http.StatusOK, task)
}
