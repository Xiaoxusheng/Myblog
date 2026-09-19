# MyBlog 后端镜像：多阶段构建，产出静态二进制（CGO_ENABLED=0）
# 构建上下文 = server/ 目录

# ---------- 构建阶段 ----------
FROM golang:1.27-alpine AS builder

# 国内代理加速依赖下载
ENV GOPROXY=https://goproxy.cn,direct
ENV CGO_ENABLED=0
ENV GOOS=linux

WORKDIR /build

# 先拷贝依赖清单，利用层缓存
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -trimpath -ldflags="-s -w" -o /out/myblog-server .

# ---------- 运行阶段 ----------
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata \
	&& cp -f /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
	&& echo "Asia/Shanghai" > /etc/timezone \
	&& addgroup -S app && adduser -S app -G app

WORKDIR /app
COPY --from=builder /out/myblog-server ./myblog-server

# 上传目录与数据目录（docker-compose 中挂卷持久化）
RUN mkdir -p /app/uploads /app/data && chown -R app:app /app
ENV BLOG_UPLOAD_DIR=/app/uploads
ENV BLOG_DB_PATH=/app/data/blog.db

USER app
EXPOSE 8080

ENTRYPOINT ["./myblog-server"]
