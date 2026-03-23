# CPA Management Service

CPA 账户管理服务，提供 CPA 账户管理和定时任务调度功能。支持多数据库（SQLite/MySQL/PostgreSQL），内置 Web UI。

## 使用

### 本地运行

```bash
make server              # 运行服务
make server CONF=config/local.yaml  # 指定配置
```

### 常用命令

```bash
make init      # 安装 wire 工具
make wire      # 生成依赖注入
make build     # 构建二进制
make test      # 运行测试
make clean     # 清理构建产物
```

### Docker 部署

```bash
make docker-build           # 构建镜像
make docker-run             # 运行容器

# 或使用 docker-compose
cd deploy/docker && docker-compose up -d
```

### 配置

配置文件位于 `config/` 目录，默认 `config/config.yaml`：

```yaml
env: dev
http:
  host: 0.0.0.0
  port: 8080
data:
  db:
    user:
      driver: sqlite
      dsn: storage/app.db
log:
  log_level: debug
  mode: both
```

### 环境变量

| 变量名 | 说明 | 默认值 |
|--------|------|--------|
| `HTTP_PROXY` | HTTP 代理地址 | - |
| `HTTPS_PROXY` | HTTPS 代理地址 | - |
| `CODEX_CHECK_POOL_SIZE` | Codex 账户检查任务并发数 | 2 |
| `CPA_POOL_SIZE` | CPA 同步任务并发数 | 3 |

## 架构

```
cmd/
  server/           # HTTP 服务入口
  migration/        # 数据库迁移入口

api/v1/             # API 请求/响应定义

internal/
  config/           # 配置加载
  handler/          # HTTP 处理器
  middleware/       # 中间件 (CORS/日志/认证)
  model/            # 数据模型
  repository/       # 数据访问层
  router/           # 路由注册
  service/          # 业务逻辑层
  task/             # 定时任务
  external/         # 外部服务客户端

pkg/
  app/              # 应用生命周期管理
  server/           # HTTP 服务器封装

web/dist/           # 前端静态资源

deploy/docker/      # Docker 部署文件
```

### 依赖注入

使用 Google Wire，定义在 `cmd/server/wire/wire.go`。修改依赖后运行 `make wire`。

### 任务系统

实现 `task.Task` 接口并注册到 `taskSet`：

```go
type Task interface {
    Name() string
    Run(ctx context.Context) error
    CurrentStats() (any, error)
}
```