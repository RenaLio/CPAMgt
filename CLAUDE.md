# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概述

CPA 管理服务 (cpamgt) - 一个 Go 后端服务，提供 CPA 账户管理和定时任务调度功能。

## 常用命令

```bash
# 运行服务 (默认使用 config/config.yaml)
go run ./cmd/server

# 使用指定配置文件
go run ./cmd/server -conf config/local.yaml

# 运行数据库迁移
go run ./cmd/migration

# 生成 wire 依赖注入代码
go generate ./cmd/server/wire
go generate ./cmd/migration/wire

# 运行测试
go test ./...

# 运行单个测试
go test -v -run TestNestedTransaction ./internal/repository/...
```

## 代码架构

### 分层架构

```
cmd/                # 应用入口点
  server/           # HTTP 服务入口
    main.go
    wire/           # Wire 依赖注入定义
  migration/        # 数据库迁移入口

api/v1/             # API 层 - 请求/响应 DTO 定义

internal/           # 内部包
  config/           # 配置加载 (viper)
  handler/          # HTTP 处理器
  middleware/       # Gin 中间件 (CORS, 日志, 认证)
  model/            # 数据库模型 (GORM)
  repository/       # 数据访问层
  router/           # 路由定义
  scheduler/        # 调度器
  server/           # HTTP 服务器构建
    task/           # 任务服务器
  service/          # 业务逻辑层
  task/             # 定时任务实现
  external/         # 外部服务客户端
    cpa/            # CPA API 客户端
    auth/           # 认证服务
  pkg/              # 内部通用包
    log/            # 日志封装 (slog)
    sloggorm/       # GORM slog 适配器

pkg/                # 外部可引用包
  app/              # 应用生命周期管理
  server/           # 服务器抽象
    http/           # HTTP 服务器封装
  httpclient/       # HTTP 客户端封装

web/                # 前端静态资源 (embed)
  dist/             # 构建产物
```

### 依赖注入 (Wire)

项目使用 Google Wire 进行依赖注入。核心 wire 集合定义在 `cmd/server/wire/wire.go`:

- `repositorySet`: 数据库、Repository、事务管理
- `serviceSet`: 业务服务
- `handlerSet`: HTTP 处理器
- `serverSet`: HTTP 服务器
- `taskSet`: 定时任务

修改依赖后需运行 `go generate` 重新生成 `wire_gen.go`。

### 任务系统

任务定义在 `internal/task/`，实现 `Task` 接口:

```go
type Task interface {
    Name() string
    Run(ctx context.Context) error
    CurrentStats() (any, error)
}
```

任务服务器 (`TaskServer`) 管理:
- 任务注册与配置
- 定时调度 (基于 interval)
- 并发控制 (allowOverlap)
- 运行状态追踪

任务管理 API 在 `/v1/tasks` 路由下。

### 数据库事务

Repository 层支持嵌套事务，通过 context 传递事务:

```go
err := repo.Transaction(ctx, func(ctx context.Context) error {
    db := repo.DB(ctx) // 自动获取当前事务
    // ...
    return nil
})
```

### HTTP API 响应格式

统一响应结构定义在 `api/v1/v1.go`:

```go
type Response struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
    Data    any    `json:"data"`
}
```

使用 `HandleSuccess` / `HandleError` 辅助函数。

## 配置

配置文件使用 YAML 格式，通过 viper 加载。示例 `config/config.yaml`:

```yaml
env: dev
http:
  host: 0.0.0.0
  port: 8080
data:
  db:
    user:
      driver: sqlite  # mysql / postgres / sqlite
      dsn: storage/test.db
log:
  log_level: debug
  mode: both  # console / file / both
```

## 数据库支持

支持三种数据库:
- **SQLite**: `driver: sqlite`, dsn 示例: `storage/test.db?_busy_timeout=5000`
- **MySQL**: `driver: mysql`, dsn 示例: `user:pass@tcp(localhost:3306)/dbname`
- **PostgreSQL**: `driver: postgres`, 使用 pgx 驱动

## 前端集成

前端静态文件嵌入在 `web/dist/`，通过 `gin-contrib/static` 服务。API 路由以 `/v1` 开头，其他路由返回 `index.html` (SPA)。