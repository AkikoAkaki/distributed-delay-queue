# Stage 1: Builder
FROM golang:1.25-alpine AS builder

# 安装构建必要的工具
RUN apk add --no-cache git make

WORKDIR /app

# 利用 Docker Layer 缓存依赖下载
COPY go.mod ./
RUN go mod download

# 复制源代码
COPY . .

# 编译两个二进制文件
# CGO_ENABLED=0: 静态链接，确保在 Alpine 这种精简系统里也能跑
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/worker ./cmd/worker

# Stage 2: Runner (生产环境镜像)
FROM alpine:latest

# 安装基础证书 (防止 HTTPS 请求失败) 和 时区数据
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# 从 Builder 阶段复制二进制文件
COPY --from=builder /bin/server /app/server
COPY --from=builder /bin/worker /app/worker
# 复制配置文件模板 (运行时可通过 Volume 覆盖)
COPY --from=builder /app/config/config.example.yaml /app/config.yaml

# 暴露端口 (Server 用)
EXPOSE 8080 9090

# 默认启动 Server，也可以通过 CMD ["./worker"] 覆盖
CMD ["./server"]
