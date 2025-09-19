FROM golang:alpine AS builder

# 安装必要的工具
RUN apk add --no-cache git ca-certificates tzdata

# 设置工作目录
WORKDIR /app

# 复制go.mod和go.sum文件（这些文件变化频率低，放在前面利用缓存）
COPY go.mod go.sum ./

# 下载依赖（只有当go.mod或go.sum变化时才重新执行）
RUN go mod download

# 验证依赖完整性
RUN go mod verify

# 生成
RUN go generate ./...

# 复制源代码（源代码变化频率高，放在依赖下载之后）
COPY . .

# 构建二进制文件 - 支持多平台
# 使用CGO_ENABLED=0生成静态链接的二进制文件
# 使用-ldflags减小二进制文件大小
# 动态设置目标操作系统和架构
RUN go build \
    -o /app/bin/server \
    ./cmd/server

# 运行阶段 - 使用最小化的基础镜像
FROM alpine:3

# 安装运行时必需的包
RUN apk --no-cache add ca-certificates tzdata

# 创建非root用户提高安全性
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/bin/server /app/server

# 创建必要的目录并设置权限
RUN mkdir -p /app/logs && \
    chown -R appuser:appgroup /app

# 切换到非root用户
USER appuser

# 暴露端口
EXPOSE 8080

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# 运行应用
CMD ["./server"]
