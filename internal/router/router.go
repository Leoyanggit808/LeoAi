package router

import (
	"LeoAi/config"
	"LeoAi/internal/handler"
	"LeoAi/internal/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRouter(
	userHandler *handler.UserHandler,
	documentHandler *handler.DocumentHandler,
	taskHandler *handler.TaskHandler,
	cfg *config.Config,
) *gin.Engine {

	r := gin.Default()

	// ====================== 前端页面路由 ======================
	// 主工作台（单页应用，包含概览/文档/AI/任务/分类/标签/设置）
	r.GET("/app", func(c *gin.Context) {
		c.File("./static/app.html")
	})
	// 登录/注册页
	r.GET("/auth", func(c *gin.Context) {
		c.File("./static/auth.html")
	})
	// 个人设置页
	r.GET("/profile", func(c *gin.Context) {
		c.File("./static/profile.html")
	})

	// ====================== 公开路由 ======================
	public := r.Group("/api/v1")
	{
		auth := public.Group("/auth")
		{
			auth.POST("/register", userHandler.Register)
			auth.POST("/login", userHandler.Login)
		}
	}

	// ====================== 需要登录的路由 ======================
	protected := r.Group("/api/v1")
	protected.Use(middleware.JWTAuth(cfg))
	{
		// 文档管理
		doc := protected.Group("/documents")
		{
			doc.POST("", documentHandler.Upload)
			doc.GET("", documentHandler.List)
		}

		// AI 任务
		task := protected.Group("/tasks")
		{
			task.POST("", taskHandler.SubmitTask)
			task.GET("/:id", taskHandler.GetTaskStatus)
		}

		// AI 直接生成（临时保留）
		ai := protected.Group("/ai")
		{
			ai.POST("/generate", taskHandler.SubmitTask) // 后续可改为直接调用
		}
	}

	return r
}
