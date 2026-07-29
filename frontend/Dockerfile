FROM node:22-alpine AS builder
WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm ci
COPY . .
RUN npm run build

FROM nginx:alpine
COPY --from=builder /app/build /usr/share/nginx/html
COPY nginx.conf.template /etc/nginx/templates/default.conf.template
# Defaults match the bundled docker-compose.yml (service name "backend", port
# 8989). Override both via environment variables if you rename the backend
# service or change its listen port in config.yml / OP_PORT.
ENV BACKEND_HOST=backend
ENV BACKEND_PORT=8989
EXPOSE 80
