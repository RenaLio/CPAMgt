# 构建阶段
FROM golang:1.26-alpine AS builder

ARG APP_RELATIVE_PATH=./cmd/server/

# 设置 Go 模块代理镜像源
ARG GOPROXY=https://goproxy.cn,direct
ENV GOPROXY=${GOPROXY}
ENV GOSUMDB=sum.golang.org
ENV CGO_ENABLED=1

WORKDIR /build

# 安装构建依赖 (SQLite 需要 CGO)
RUN set -eux && sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories \
    && apk add --no-cache git gcc musl-dev

# 先复制依赖文件，利用缓存
COPY go.mod go.sum ./
RUN go mod download

# 复制源码并构建 (启用 CGO 以支持 SQLite)
COPY . .
RUN GOOS=linux go build -ldflags="-s -w" -o /build/server ${APP_RELATIVE_PATH}

# 运行阶段
FROM alpine:3.19

# 设置时区和镜像源
RUN set -eux && sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories \
    && apk add --no-cache tzdata ca-certificates \
    && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone \
    && apk del tzdata

# 创建非 root 用户
RUN addgroup -g 1000 appgroup && adduser -u 1000 -G appgroup -D appuser

WORKDIR /app

# 复制构建产物
COPY --from=builder /build/server /app/
COPY --from=builder /build/config /app/config/

# 创建存储目录并设置权限
RUN mkdir -p /app/storage/logs && chown -R appuser:appgroup /app

USER appuser

ARG APP_ENV=prod
ENV APP_ENV=${APP_ENV}

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/v1/ping || exit 1

ENTRYPOINT ["./server"]
CMD ["-conf", "config/config.yaml"]

