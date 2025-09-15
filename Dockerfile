# 多阶段构建，减小镜像大小
FROM golang:1.21-alpine AS builder

# 设置工作目录
WORKDIR /app

# 安装必要的包
RUN apk add --no-cache git ca-certificates tzdata

# 复制go mod文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o telemetry main.go

# 运行阶段
FROM alpine:latest

# 安装必要的包
RUN apk --no-cache add ca-certificates tzdata wget curl

# 设置时区
ENV TZ=Asia/Shanghai

# 创建非root用户
RUN addgroup -g 1001 -S telemetry && \
    adduser -u 1001 -S telemetry -G telemetry

# 创建必要的目录
RUN mkdir -p /app/logs /app/data && \
    chown -R telemetry:telemetry /app

# 切换到应用目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/telemetry .
COPY --from=builder /app/production-config-optimized.yaml ./config.yaml

# 设置文件权限
RUN chmod +x telemetry && \
    chown telemetry:telemetry telemetry config.yaml

# 切换到非root用户
USER telemetry

# 暴露端口
EXPOSE 50051 12112 8080

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
    CMD wget --quiet --tries=1 --spider http://localhost:8080/health || exit 1

# 启动应用
CMD ["./telemetry", "-config", "config.yaml"]