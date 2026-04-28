package main

import (
	"LeoAi/config"
	"LeoAi/internal/database"
	"LeoAi/internal/handler"
	"LeoAi/internal/model"
	"LeoAi/internal/repository"
	"LeoAi/internal/router"
	"LeoAi/internal/service"
	"context"
	"fmt"
	"log"

	"github.com/redis/go-redis/v9"
)

func main() {
	// 初始化配置文件
	cfg := config.LoadConfig()

	// 初始化数据库连接
	db, err := database.NewMySQLConnection(cfg)
	if err != nil {
		log.Fatal(err)
	}

	// 自动迁移
	db.AutoMigrate(&model.User{}, &model.Document{}, &model.AITask{})

	// 初始化Redis 连接
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword, // 建议把密码也加到 config.yaml
		DB:       0,
	})
	// 测试连接
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		log.Fatal("Redis 连接失败:", err)
	}
	fmt.Println("✅ Redis 连接成功")

	// 初始化Repository层
	userRepo := repository.NewUserRepository(db)
	documentRepo := repository.NewDocumentRepository(db)
	taskRepo := repository.NewAITaskRepository(db)

	// 初始化Service层
	userService := service.NewUserService(userRepo)
	aiService := service.NewAIService(cfg)
	vectorService := service.NewVectorService(cfg.ChromaURL, aiService)
	if err := vectorService.EnsureCollection(); err != nil {
		log.Printf("⚠️ ChromaDB Collection 初始化失败: %v", err)
	}
	documentService := service.NewDocumentService(documentRepo, vectorService)
	taskService := service.NewTaskService(taskRepo, aiService, vectorService, redisClient)

	// 初始化Handler层
	userHandler := handler.NewUserHandler(userService, cfg)
	documentHandler := handler.NewDocumentHandler(documentService)
	taskHandler := handler.NewTaskHandler(taskService)

	// ==================== 初始化消费组（只执行一次） ====================
	if err := taskService.InitConsumerGroup(); err != nil {
		log.Printf("警告: 初始化消费组失败: %v", err)
	}

	// ==================== 启动 Worker ====================
	const workerCount = 2 // 根据需要调整，开发阶段建议 1~2 个

	for i := 1; i <= workerCount; i++ {
		consumerName := fmt.Sprintf("worker-%d", i)
		go taskService.StartWorker(consumerName)
	}


	// 配置路由
	r := router.SetupRouter(userHandler, documentHandler, taskHandler, cfg)

	fmt.Println("🚀 LeoAi 服务启动成功，端口: 8080")
	r.Run(":8080")
}
