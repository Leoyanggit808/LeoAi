package service

/*
1. Redis Stream 是什么？
Redis Stream 是 Redis 5.0 版本引入的一种专门用于做消息队列的数据结构。

你可以把它理解为：
一条只能追加内容的有序日志流（Append-Only Log），就像一条不断向前流动的河流。
生产者（Producer）不断往河里扔消息（追加）
消费者（Consumer）从河的某个位置读取消息
消息有全局唯一编号（ID），按时间顺序排列


2. 为什么我们要在 LeoAi 项目中使用 Redis Stream？
AI 生成任务有以下特点：
耗时长（调用 DeepSeek 可能需要 5~30 秒）
不能让用户等待（用户提交后希望立刻得到响应）
需要可靠处理（不能因为服务重启就丢失任务）*/

/*
 概念,				  含义,					 类比,					 在 LeoAi 中的例子
Stream,				消息队列本身,				一条河流,				ai_tasks（AI任务流）
Message,			一条具体消息,				河里的一条鱼,				一个用户提交的生成任务
Message ID,			每条消息的唯一编号,		鱼身上的编号,				1745567890123-0（时间+序列号）
Consumer Group,		消费组,					捕鱼小组,				ai-workers（我们的 Worker 组）
Consumer,			消费者,					小组里的成员,				worker-1、worker-2 等
ACK,				签收确认,				把鱼捞上来标记“已处理”,		处理完任务后告诉 Redis*/

//Redis Stream 不像传统队列只有一个队列，它内部实际上维护着多个逻辑队列/列表。
/*==================== Redis Stream 核心队列说明 ====================
// Main Stream：存放所有消息的持久化有序日志流
// Consumer Group：多个 Worker 共同消费的分组机制
// Pending Queue：已投递但未确认（待 ACK）的消息队列
// Dead Letter Queue：处理失败多次后进入的死信队列（需手动实现）*/
import (
	"LeoAi/internal/model"
	"LeoAi/internal/repository"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type TaskService struct {
	taskRepo      *repository.AITaskRepository // 操作数据库任务表
	aiService     *AIService                   // 调用 DeepSeek 生成内容
	vectorService *VectorService               // ← 新增RAG任务
	redis         *redis.Client                // Redis Stream 消息队列
}

func NewTaskService(
	taskRepo *repository.AITaskRepository,
	aiService *AIService,
	vectorService *VectorService, // ← 新增RAG任务
	redis *redis.Client,
) *TaskService {
	return &TaskService{
		taskRepo:      taskRepo,
		aiService:     aiService,
		vectorService: vectorService,
		redis:         redis,
	}
}

// GetTaskByID 获取单个任务（供 Handler 调用）
func (s *TaskService) GetTaskByID(taskID uint) (*model.AITask, error) {
	return s.taskRepo.FindByID(taskID) //这里实际上return的是我们调用taskRepo.FindByID()方法并传入taskID参数返回值，即根据id查询任务返回的是任务
}

// 提交异步任务 （前端调用）
func (s *TaskService) SubmitTask(userID uint, taskType, input string, documentID uint) (uint, error) {
	//在数据库中创建任务记录（立即持久化）

	task := &model.AITask{
		UserID:     userID,
		DocumentID: documentID,
		TaskType:   taskType,
		Input:      input,
		Status:     "pending",
	}
	// 保存到数据库
	if err := s.taskRepo.Create(task); err != nil {
		return 0, err
	} /*任务一旦创建成功，task.ID 就产生了
	返回这个 task.ID 给前端，用户可以立即看到“任务已提交”*/

	// 推送到 Redis Stream（异步队列）
	taskData := map[string]interface{}{
		"task_id":   task.ID, //使用结构体生成的id
		"user_id":   userID,  //使用传入的参数
		"task_type": taskType,
		"input":     input,
	}
	taskJSON, err := json.Marshal(taskData) //先转成json字符串
	if err != nil {
		return 0, err
	}
	//msgID, err := s.redis.XAdd(...).Result()

	//redis.XAdd（）使用 go-redis 客户端向 Redis Stream 添加一条消息
	//redis.XAdd的参数：
	//Stream    Stream 的名称（相当于队列名），这里是 "ai_tasks"
	//Values    要存储的消息内容（键值对）
	//成功时：返回这条消息在 Stream 中的唯一 ID，例如："1745567890123-0"
	//失败时：返回错误（网络问题、Redis 不可用等）

	//我们这里只接收错误信息
	_, err = s.redis.XAdd(
		context.Background(),
		&redis.XAddArgs{
			Stream: "ai_tasks",
			Values: map[string]interface{}{"data": string(taskJSON)}}).Result() //这是一种包装写法：把整个 taskData 作为一个字段 "data" 存进去。
	//最终存到 Redis 中的消息结构类似（举例）：
	/*
		JSON{
		 "data": {
		   "task_id": 123,
		   "user_id": 456,
		   "task_type": "generate",
		   "input": "如何学习Go语言..."
		 }
		}*/
	//优点：结构清晰，方便后续 Worker 解析。

	return task.ID, err
}

// 处理任务（后续会写后台 goroutine 调用）
func (s *TaskService) ProcessTask(taskID uint) error {
	task, err := s.taskRepo.FindByID(taskID)
	if err != nil {
		return err
	}

	// 更新为处理中
	s.taskRepo.UpdateStatus(taskID, "processing", "", "")

	var content string

	switch task.TaskType { //选择分支
	case "rag":
		// 1. 从向量库检索相关片段
		chunks, err := s.vectorService.Search(task.Input, 4)
		if err != nil {
			s.taskRepo.UpdateStatus(taskID, "failed", "", err.Error())
			return err
		}
		if len(chunks) == 0 {
			content = "文档库中未找到与您问题相关的内容，请先上传相关文档。"
		} else {
			// 2. 基于检索结果生成回答
			content, err = s.aiService.RAGGenerate(task.Input, chunks)
			if err != nil {
				s.taskRepo.UpdateStatus(taskID, "failed", "", err.Error())
				return err
			}
		}
	default:
		// 普通任务类型 调用 AI 生成
		content, err = s.aiService.GenerateContent(task.Input)
		if err != nil {
			s.taskRepo.UpdateStatus(taskID, "failed", "", err.Error()) //出现错误调用方法，把task的status设置为failed
			return err
		}
	}

	// 更新为完成
	return s.taskRepo.UpdateStatus(taskID, "completed", content, "") //完成后，设置status为completed并传入生成的结果content覆盖task的output
}

// 处理单条 Redis Stream 消息
func (s *TaskService) processStreamMessage(msg redis.XMessage) {
	/*关于参数：msg
	redis.XMessage 类型结构是什么样的？
	type XMessage struct {
		ID     string                 // ← 这就是 msg.ID
		Values map[string]interface{} // 消息的具体内容（你存的 "data" 字段）
	}*/

	//从 Redis 消息中取出我们之前存的 "data" 字段（是一个 JSON 字符串）。
	dataStr, ok := msg.Values["data"].(string)
	if !ok {
		fmt.Println("❌ 消息格式错误，无法解析 data 字段")
		return
	}

	var taskData map[string]interface{}
	if err := json.Unmarshal([]byte(dataStr), &taskData); err != nil { //把 JSON 字符串解析成 map[string]interface{}，方便后续读取各个字段。
		fmt.Printf("❌ JSON 解析失败: %v\n", err)
		return
	}

	taskIDFloat, ok := taskData["task_id"].(float64) //通过类型断言，从 map 中取出任务 ID
	if !ok {
		fmt.Println("❌ 消息中缺少 task_id")
		return
	}
	taskID := uint(taskIDFloat)

	fmt.Printf("🔄 开始处理任务 ID: %d\n", taskID)

	if err := s.ProcessTask(taskID); err != nil {
		fmt.Printf("❌ 处理任务 %d 失败: %v\n", taskID, err)
	} else {
		fmt.Printf("✅ 任务 %d 处理完成\n", taskID)
	}

	// 5. 【重要】消息确认（ACK），告诉 Redis 这条消息已处理
	s.redis.XAck(context.Background(), "ai_tasks", "ai-workers", msg.ID)
	//传入msg.ID 告诉redis这条消息已经处理过了
	/*	如果你不调用 XACK，会发生以下严重问题：

		这条消息会一直停留在redis的 Pending 队列（已投递但未确认）
		Redis 会认为这条消息“正在处理中”
		即使 Worker 重启，这条消息也不会被其他 Worker 消费
		长期积累会导致 Pending 队列爆炸，占用大量内存
		任务实际上“卡住”了，用户永远看不到结果

		总结：XACK = 告诉 Redis “这条消息我已经处理完了”，是消费者必须履行的义务*/
}

// InitConsumerGroup 创建消费者组		只在程序启动时调用一次
func (s *TaskService) InitConsumerGroup() error {
	err := s.redis.XGroupCreateMkStream(
		context.Background(),
		"ai_tasks",   // Stream 名称
		"ai-workers", // 消费组名称
		"0",          // 从头开始消费
	).Err()
	/*	s.redis.XGroupCreateMkStream(ctx, stream, group, startID)
		这是 go-redis 库提供的一个便捷方法，用于创建 Redis Stream 的消费组。

		方法名称拆解:
		XGroup：操作消费组（Consumer Group）
		Create：创建
		MkStream：Make Stream 的缩写 → 如果 Stream 不存在，就自动创建
		参数：
		ctx			上下文
		stream		Stream 名称（队列名）
		group		消费组名称
		startID		从哪个消息 ID 开始消费*/
	if err != nil {
		if err.Error() == "BUSYGROUP Consumer Group name already exists" {
			fmt.Println("✅ 消费组 ai-workers 已存在，继续使用")
			return nil
		}
		return fmt.Errorf("创建消费组失败: %w", err)
	}
	/*	第一次启动 Worker 时，需要告诉 Redis：“我要创建一个消费组，叫 ai-workers，专门负责处理 ai_tasks 队列。”
		如果 Stream 不存在，MkStream 会自动帮我们创建这个 Stream。
		如果消费组已经存在，Redis 会返回错误 "BUSYGROUP Consumer Group name already exists"，我们的代码会忽略这个错误（防止重复启动报错）。*/
	fmt.Println("✅ 消费组 ai-workers 创建成功")
	return nil

}

// StartWorker 启动后台 Worker
func (s *TaskService) StartWorker(consumerName string) {
	fmt.Printf("🚀 AI Task Worker [%s] 已启动，正在监听 ai_tasks 队列...\n", consumerName)
	for {
		//使用消费组（Consumer Group）的方式，从 Stream 中读取消息。
		//普通 XRead 是单个消费者读取。
		//XReadGroup 是消费组模式，允许多个 Worker 协同工作，且不会重复消费同一条消息。
		streams, err := s.redis.XReadGroup(context.Background(), &redis.XReadGroupArgs{
			Group:    "ai-workers",              // 消费组名称
			Consumer: consumerName,              // 当前 Worker 的名字
			Streams:  []string{"ai_tasks", ">"}, // 要监听的 Stream + ">"
			//">" 表示：只读取当前消费组还没有被任何消费者消费过的消息。 这是消费组模式下最常用的写法。
			Count: 10, // 一次最多读取10条
			Block: 0,  // 阻塞等待时间（0 = 永久阻塞）
		}).Result()

		if err != nil {
			if err == redis.Nil {
				continue
			}
			fmt.Printf("Worker [%s] 读取消息出错: %v\n", consumerName, err)
			time.Sleep(time.Second * 2)
			continue
		}
		/*	XReadGroup 返回的数据结构大致是这样的：
			streams []redis.XStream        ← 第一层：多个 Stream（通常只有一个）
			    |
			    └── XStream {
			           Stream:   "ai_tasks",
			           Messages: []redis.XMessage   ← 第二层：该 Stream 下的多条消息
			        }
			所以我们可以遍历循环处理每一条消息：*/
		for _, stream := range streams {
			/*streams 是 []redis.XStream 类型（一个切片）
			每次循环取出一个 Stream（stream）
			虽然我们只监听了一个 Stream（ai_tasks），但 Redis 返回的数据结构设计成了支持监听多个 Stream 的形式，所以用 range 遍历*/
			for _, msg := range stream.Messages {
				/*stream.Messages 是 []redis.XMessage 类型（一条 Stream 下的消息列表）
				每次循环取出一条具体消息（msg）*/
				s.processStreamMessage(msg)
			}
		}
	}
}
