# syntax=docker/dockerfile:1

# 统一 Go 版本，与 go.mod 保持一致（使用最新版）
ARG GO_VERSION=1.25
ARG TARGETPLATFORM

# 构建阶段
FROM --platform=$TARGETPLATFORM golang:${GO_VERSION}-alpine AS build
WORKDIR /app

# 安装必要的工具
RUN apk add --no-cache git ca-certificates && update-ca-certificates

# 优化依赖缓存：先复制依赖文件，单独下载依赖
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# 复制源代码
COPY . .

# 设置构建环境
ENV CGO_ENABLED=0

# 生成 ent 代码
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go generate ./ent

# 构建应用程序，使用缓存挂载优化编译速度
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

# 运行时阶段：使用更小的基础镜像
FROM --platform=$TARGETPLATFORM alpine:3.20
RUN addgroup -S app && adduser -S app -G app && \
    apk add --no-cache ca-certificates tzdata && \
    update-ca-certificates

# 设置工作目录
WORKDIR /app

# 复制构建产物
COPY --from=build /out/server /usr/local/bin/server

# 使用非特权用户
USER app:app

# 设置环境变量
ENV APP_ENV=prod \
    SERVER_ADDR=:8080 \
    TZ=UTC

# 暴露端口
EXPOSE 8080

# 设置健康检查（可选）
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# 启动应用
ENTRYPOINT ["/usr/local/bin/server"]

