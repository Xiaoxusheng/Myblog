# 门户镜像：分别构建 web/（前台）与 admin/（管理端）的 dist，交给 nginx 承载
# 构建上下文 = 项目根目录（docker compose 中 build.context: .）

# ---------- 前台构建 ----------
FROM node:20-alpine AS web-builder
WORKDIR /src/web
COPY web/package.json web/package-lock.json* ./
RUN npm install --registry=https://registry.npmmirror.com
COPY web/ .
RUN npm run build

# ---------- 管理端构建 ----------
FROM node:20-alpine AS admin-builder
WORKDIR /src/admin
COPY admin/package.json admin/package-lock.json* ./
RUN npm install --registry=https://registry.npmmirror.com
COPY admin/ .
RUN npm run build

# ---------- nginx 运行层 ----------
FROM nginx:alpine

COPY --from=web-builder /src/web/dist /usr/share/nginx/html/web
COPY --from=admin-builder /src/admin/dist /usr/share/nginx/html/admin
COPY deploy/nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80 8081

CMD ["nginx", "-g", "daemon off;"]
