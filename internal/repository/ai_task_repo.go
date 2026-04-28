package repository

import (
	"LeoAi/internal/model"

	"gorm.io/gorm"
)

/*用户提交生成请求
↓
API 立即返回 TaskID + "pending" 状态
↓
任务进入队列（Redis Stream / RabbitMQ）
↓
后台 Worker 取出任务 → 调用 DeepSeek → 生成文章
↓
更新任务状态为 completed + 保存结果
↓
用户可随时查询任务状态或结果*/

type AITaskRepository struct {
	db *gorm.DB
}

func NewAITaskRepository(db *gorm.DB) *AITaskRepository {
	return &AITaskRepository{db: db}
}

func (r *AITaskRepository) Create(task *model.AITask) error {
	return r.db.Create(task).Error
}

func (r *AITaskRepository) FindByID(id uint) (*model.AITask, error) {
	var task model.AITask
	err := r.db.First(&task, id).Error
	return &task, err
}

func (r *AITaskRepository) FindByUserID(userID uint) ([]model.AITask, error) {
	var tasks []model.AITask
	err := r.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&tasks).Error
	return tasks, err
}

//在我们的 AITask 模型设计中，status 字段一共有 4 种状态：
//	状态,			英文值,					含义,							说明
//	待处理,			pending,		任务已提交，正在等待被处理,		用户提交任务后，立即返回的状态
//	处理中,			processing,		Worker 正在执行 AI 生成,		后台 Worker 取出任务后，修改为这个状态
//	已完成,			completed,		AI 生成成功，已保存结果,		任务正常结束，用户可查看 Output
//	失败,			failed,			AI 生成失败,					出现错误（如网络超时、Token 超限、API 报错等）

func (r *AITaskRepository) UpdateStatus(id uint, status, output, errorMsg string) error {
	updates := map[string]interface{}{
		"status": status,
	}
	if output != "" {
		updates["output"] = output
	}
	if errorMsg != "" {
		updates["error_msg"] = errorMsg
	}
	// 只有当状态为 completed 或 failed 时，才更新 completed_at
	if status == "completed" || status == "failed" {
		updates["completed_at"] = gorm.Expr("NOW()")
	} else {
		// 其他状态（pending、processing）不更新 completed_at
		updates["completed_at"] = nil
	}
	return r.db.Model(&model.AITask{}).Where("id = ?", id).Updates(updates).Error
}
