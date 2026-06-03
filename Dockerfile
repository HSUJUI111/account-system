# 第一阶段:构建
FROM golang:1.26-alpine AS builder

WORKDIR /app

# 先复制 go.mod 和 go.sum,利用 Docker 层缓存(依赖不变就不重新下载)
COPY go.mod go.sum ./
RUN go mod download

# 复制源码
COPY . .

# 编译,关闭 CGO,静态链接,减小体积
RUN CGO_ENABLED=0 GOOS=linux go build -o account-system ./cmd/main.go

# 第二阶段:运行(用极小的 alpine 镜像)
FROM alpine:3.19

WORKDIR /app

# 装 ca-certificates(HTTPS 调用需要,虽然你目前没用,但是基础设施)
RUN apk --no-cache add ca-certificates

# 从 builder 阶段拷贝编译好的二进制和配置文件
COPY --from=builder /app/account-system .
COPY --from=builder /app/config ./config

EXPOSE 8080

CMD ["./account-system"]