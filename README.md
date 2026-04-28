# LeoAi

> 一个基于 AI 的个人内容生成平台，支持文档管理、智能任务处理与向量检索。

## 技术栈

| 层级 | 技术 |
|------|------|
| 后端框架 | Go + Gin |
| ORM | Gorm |
| 数据库 | MySQL |
| 缓存 | Redis |
| AI 接口 | DeepSeek API / SiliconFlow API |
| 向量数据库 | ChromaDB |
| 认证 | JWT |

## 项目结构

```
LeoAi/
├── cmd/
│   └── main.go            # 程序入口
├── config/
│   ├── config.go          # 配置加载
│   └── config.yaml        # 配置文件
├── internal/
│   ├── database/          # 数据库连接
│   ├── handler/           # HTTP 处理器
│   ├── middleware/        # JWT 中间件
│   ├── model/             # 数据模型
│   ├── repository/        # 数据访问层
│   ├── router/            # 路由配置
│   ├── service/           # 业务逻辑层
│   └── util/              # 工具函数
├── static/                # 前端页面
├── uploads/               # 用户上传文件
└── .gitignore
```

## 功能模块

- **用户认证**：注册 / 登录 / JWT 鉴权
- **文档管理**：文件上传、列表查看
- **AI 任务**：异步任务提交与状态查询
- **向量检索**：基于 ChromaDB 的文档 embedding 与相似度搜索

## 快速开始

### 环境要求

- Go 1.21+
- MySQL
- Redis
- ChromaDB（可选，用于向量检索）

### 1. 克隆项目

```bash
git clone https://github.com/Leoyanggit808/LeoAi.git
cd LeoAi
```

### 2. 配置

编辑 `config/config.yaml`，填入你的密钥和数据库信息：

```yaml
mysql:
  dsn: "root:你的密码@tcp(127.0.0.1:3306)/leoai?charset=utf8mb4&parseTime=True&loc=Local"

redis:
  addr: "127.0.0.1:6379"
  password: "你的Redis密码"

deepseek:
  api_key: "你的DeepSeek密钥"

jwt:
  secret: "自定义JWT密钥"

chroma:
  url: "http://localhost:8000"

siliconflow:
  api_key: "你的SiliconFlow密钥"
```

### 3. 启动依赖服务

```bash
# MySQL 创建数据库
mysql -u root -p -e "CREATE DATABASE IF NOT EXISTS leoai CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"

# Redis
redis-server

# ChromaDB（可选）
pip install chromadb
chroma run --host localhost --port 8000
```

### 4. 运行

```bash
go run cmd/main.go
```

服务启动后访问 `http://localhost:8080`

## API 路由

### 公开接口

| 方法 | 路径 | 描述 |
|------|------|------|
| POST | `/api/v1/auth/register` | 用户注册 |
| POST | `/api/v1/auth/login` | 用户登录 |
| GET | `/app` | 主工作台页面 |
| GET | `/auth` | 登录/注册页面 |
| GET | `/profile` | 个人设置页面 |

### 需认证接口（携带 JWT）

| 方法 | 路径 | 描述 |
|------|------|------|
| POST | `/api/v1/documents` | 上传文档 |
| GET | `/api/v1/documents` | 获取文档列表 |
| POST | `/api/v1/tasks` | 提交 AI 任务 |
| GET | `/api/v1/tasks/:id` | 查询任务状态 |
| POST | `/api/v1/ai/generate` | 直接 AI 生成 |

## License

MIT
