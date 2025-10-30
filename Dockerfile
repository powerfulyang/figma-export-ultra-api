FROM golang:alpine AS builder

# 安装必要的工具（含 make 支持）
RUN apk add --no-cache make tzdata

# 设置上海时区
RUN ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone \
    && apk del tzdata

# 设置工作目录
WORKDIR /app

# 复制 go.mod 和 go.sum
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 验证依赖完整性
RUN go mod verify

# 复制源代码
COPY . .

# 生成代码
RUN make gen && make swagger-gen

# 构建二进制文件
RUN make build

# 运行阶段 - 使用最小化的基础镜像
FROM alpine:3 AS runner

# 设置上海时区
RUN apk add --no-cache make tzdata && \
    ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone \
    && apk del tzdata

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
