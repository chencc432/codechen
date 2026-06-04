# 08 — Docker Compose 编排实战

> 一个应用通常需要多个服务（Web + 数据库 + 缓存 + ...），Docker Compose 让你一条命令搞定所有。

---

## 一、为什么需要 Docker Compose？

### 1.1 痛点

假设你的应用需要三个服务：

```bash
# 启动 MySQL
docker run -d --name db -e MYSQL_ROOT_PASSWORD=secret \
    -v db-data:/var/lib/mysql --network app-net mysql:8.0

# 启动 Redis
docker run -d --name cache --network app-net redis:7

# 启动 Web 应用
docker run -d --name web -p 80:80 --network app-net \
    -e DB_HOST=db -e REDIS_HOST=cache my-web-app
```

**问题**：
- 每次启动要敲三条长命令
- 停止也要一个个停
- 命令参数多，容易写错
- 服务之间的依赖关系不直观
- 不方便版本控制

### 1.2 Docker Compose 的解决方案

把所有服务定义在一个 `docker-compose.yml` 文件中：

```yaml
services:
  web:
    image: my-web-app
    ports:
      - "80:80"
    environment:
      - DB_HOST=db
      - REDIS_HOST=cache
    depends_on:
      - db
      - cache

  db:
    image: mysql:8.0
    environment:
      MYSQL_ROOT_PASSWORD: secret
    volumes:
      - db-data:/var/lib/mysql

  cache:
    image: redis:7

volumes:
  db-data:
```

然后一条命令搞定：

```bash
docker compose up -d     # 启动所有服务
docker compose down      # 停止并清理所有服务
```

---

## 二、安装 Docker Compose

### 2.1 新版本（V2，推荐）

Docker Desktop 和新版 Docker Engine 已内置 Compose V2：

```bash
# 验证（注意是 docker compose，中间是空格）
docker compose version
# Docker Compose version v2.x.x
```

### 2.2 旧版本（V1，已废弃）

```bash
# 旧版是独立命令，用连字符
docker-compose version    # 旧版 V1

# 如果你看到教程用 docker-compose，改成 docker compose 即可
```

---

## 三、docker-compose.yml 语法详解

### 3.1 文件结构

```yaml
# 顶层结构
services:    # 服务定义（核心）
  web:
    ...
  db:
    ...

volumes:     # 卷定义（可选）
  db-data:
    ...

networks:    # 网络定义（可选，Compose 会自动创建默认网络）
  app-net:
    ...

configs:     # 配置文件（可选）
secrets:     # 密钥（可选）
```

### 3.2 服务配置详解

#### image — 指定镜像

```yaml
services:
  web:
    image: nginx:1.25.3
```

#### build — 从 Dockerfile 构建

```yaml
services:
  web:
    build: .                          # 当前目录的 Dockerfile

  api:
    build:
      context: ./backend              # 构建上下文目录
      dockerfile: Dockerfile.prod     # 指定 Dockerfile
      args:                           # 构建参数
        - APP_VERSION=2.0
```

#### ports — 端口映射

```yaml
services:
  web:
    ports:
      - "80:80"             # 宿主机80 → 容器80
      - "443:443"
      - "8080:80"           # 宿主机8080 → 容器80
      - "127.0.0.1:3000:3000"  # 只绑定 localhost
```

#### volumes — 数据挂载

```yaml
services:
  db:
    volumes:
      - db-data:/var/lib/mysql              # 命名卷
      - ./init.sql:/docker-entrypoint-initdb.d/init.sql  # Bind Mount
      - /host/path:/container/path:ro       # 只读挂载

volumes:
  db-data:          # 在顶层声明命名卷
```

#### environment — 环境变量

```yaml
services:
  db:
    environment:
      MYSQL_ROOT_PASSWORD: secret       # 直接写值
      MYSQL_DATABASE: mydb
      APP_ENV: ${APP_ENV:-production}   # 从宿主机环境变量读取，默认 production

  # 或者用列表格式
  api:
    environment:
      - DB_HOST=db
      - DB_PORT=3306
```

#### env_file — 从文件读取环境变量

```yaml
services:
  api:
    env_file:
      - .env              # 默认环境变量
      - .env.production   # 生产环境变量（覆盖）
```

`.env` 文件格式：

```
DB_HOST=db
DB_PORT=3306
DB_PASSWORD=secret
```

#### depends_on — 依赖关系

```yaml
services:
  web:
    depends_on:
      - db
      - cache

  # 高级写法：等待服务健康才启动
  web:
    depends_on:
      db:
        condition: service_healthy
      cache:
        condition: service_started
```

#### restart — 重启策略

```yaml
services:
  web:
    restart: unless-stopped    # 推荐
  db:
    restart: always
```

#### networks — 网络配置

```yaml
services:
  web:
    networks:
      - frontend
      - backend
  db:
    networks:
      - backend      # db 只在 backend 网络，web 无法直接访问

networks:
  frontend:
  backend:
```

#### healthcheck — 健康检查

```yaml
services:
  web:
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:80"]
      interval: 30s
      timeout: 5s
      retries: 3
      start_period: 10s
```

#### deploy — 资源限制

```yaml
services:
  web:
    deploy:
      resources:
        limits:
          cpus: "2.0"
          memory: 512M
        reservations:
          cpus: "0.5"
          memory: 128M
```

#### command — 覆盖启动命令

```yaml
services:
  api:
    image: python:3.11-slim
    command: python app.py --port 8000

    # 或者列表格式
    command: ["python", "app.py", "--port", "8000"]
```

#### entrypoint — 覆盖入口点

```yaml
services:
  worker:
    image: my-app
    entrypoint: ["python", "worker.py"]
```

---

## 四、Docker Compose 命令详解

### 4.1 核心命令

```bash
# 启动所有服务（前台运行，可以看到日志）
docker compose up

# 后台运行（推荐）
docker compose up -d

# 重新构建镜像并启动
docker compose up -d --build

# 只启动某个服务
docker compose up -d web

# 停止所有服务
docker compose stop

# 停止并删除所有容器、网络
docker compose down

# 停止并删除所有容器、网络、卷（会丢数据！）
docker compose down -v

# 重启服务
docker compose restart
docker compose restart web    # 只重启某个服务
```

### 4.2 查看状态

```bash
# 查看服务状态
docker compose ps

# 查看日志
docker compose logs
docker compose logs -f             # 实时跟踪
docker compose logs -f web         # 只看某个服务
docker compose logs --tail 100 web # 最后 100 行

# 查看资源使用
docker compose top
```

### 4.3 执行命令

```bash
# 进入某个服务的容器
docker compose exec web bash

# 执行一次性命令
docker compose exec db mysql -uroot -psecret

# 运行一次性容器（不连接到服务）
docker compose run --rm web python manage.py migrate
```

### 4.4 构建和扩缩

```bash
# 只构建镜像（不启动）
docker compose build

# 重新构建某个服务
docker compose build web

# 扩缩容（启动 3 个 worker 实例）
docker compose up -d --scale worker=3

# 拉取所有镜像
docker compose pull
```

---

## 五、环境变量与多环境管理

### 5.1 .env 文件（默认环境变量）

在 `docker-compose.yml` 同级目录创建 `.env`：

```
# .env
MYSQL_VERSION=8.0
REDIS_VERSION=7
APP_PORT=80
```

在 `docker-compose.yml` 中引用：

```yaml
services:
  db:
    image: mysql:${MYSQL_VERSION}
  cache:
    image: redis:${REDIS_VERSION}
  web:
    ports:
      - "${APP_PORT}:80"
```

### 5.2 多环境配置

```yaml
# docker-compose.yml（基础配置）
services:
  web:
    image: my-app
    ports:
      - "80:80"
  db:
    image: mysql:8.0
    volumes:
      - db-data:/var/lib/mysql

volumes:
  db-data:
```

```yaml
# docker-compose.override.yml（开发环境，自动加载）
services:
  web:
    build: .
    volumes:
      - .:/app
    environment:
      - DEBUG=true
```

```yaml
# docker-compose.prod.yml（生产环境）
services:
  web:
    restart: always
    environment:
      - DEBUG=false
    deploy:
      resources:
        limits:
          memory: 512M
```

```bash
# 开发环境（自动加载 docker-compose.yml + docker-compose.override.yml）
docker compose up -d

# 生产环境（手动指定）
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d
```

---

## 六、实战案例

### 6.1 WordPress 博客

```yaml
services:
  wordpress:
    image: wordpress:latest
    ports:
      - "8080:80"
    environment:
      WORDPRESS_DB_HOST: db
      WORDPRESS_DB_USER: wp_user
      WORDPRESS_DB_PASSWORD: wp_pass
      WORDPRESS_DB_NAME: wordpress
    volumes:
      - wp-content:/var/www/html/wp-content
    depends_on:
      db:
        condition: service_healthy
    restart: unless-stopped

  db:
    image: mysql:8.0
    environment:
      MYSQL_ROOT_PASSWORD: root_secret
      MYSQL_DATABASE: wordpress
      MYSQL_USER: wp_user
      MYSQL_PASSWORD: wp_pass
    volumes:
      - db-data:/var/lib/mysql
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost"]
      interval: 10s
      timeout: 5s
      retries: 5
    restart: unless-stopped

volumes:
  wp-content:
  db-data:
```

### 6.2 Python + Redis + PostgreSQL

```yaml
services:
  api:
    build: .
    ports:
      - "8000:8000"
    environment:
      - DATABASE_URL=postgresql://user:pass@db:5432/mydb
      - REDIS_URL=redis://cache:6379/0
    depends_on:
      db:
        condition: service_healthy
      cache:
        condition: service_started
    restart: unless-stopped

  db:
    image: postgres:16
    environment:
      POSTGRES_USER: user
      POSTGRES_PASSWORD: pass
      POSTGRES_DB: mydb
    volumes:
      - pg-data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U user -d mydb"]
      interval: 10s
      timeout: 5s
      retries: 5
    restart: unless-stopped

  cache:
    image: redis:7-alpine
    volumes:
      - redis-data:/data
    restart: unless-stopped

volumes:
  pg-data:
  redis-data:
```

### 6.3 前后端分离项目

```yaml
services:
  frontend:
    build: ./frontend
    ports:
      - "3000:3000"
    volumes:
      - ./frontend/src:/app/src    # 开发模式热更新
    environment:
      - REACT_APP_API_URL=http://localhost:8000

  backend:
    build: ./backend
    ports:
      - "8000:8000"
    volumes:
      - ./backend:/app
    environment:
      - DATABASE_URL=postgresql://user:pass@db:5432/mydb
    depends_on:
      - db

  db:
    image: postgres:16
    environment:
      POSTGRES_USER: user
      POSTGRES_PASSWORD: pass
      POSTGRES_DB: mydb
    volumes:
      - pg-data:/var/lib/postgresql/data

  adminer:
    image: adminer
    ports:
      - "8888:8080"     # 数据库管理界面

volumes:
  pg-data:
```

---

## 七、Compose 使用技巧

### 7.1 等待依赖就绪

```yaml
services:
  web:
    depends_on:
      db:
        condition: service_healthy    # 等数据库健康才启动

  db:
    healthcheck:
      test: ["CMD", "mysqladmin", "ping"]
      interval: 5s
      retries: 10
```

### 7.2 日志管理

```yaml
services:
  web:
    logging:
      driver: "json-file"
      options:
        max-size: "10m"     # 单文件最大 10MB
        max-file: "3"       # 保留 3 个文件
```

### 7.3 初始化脚本

```yaml
services:
  db:
    image: mysql:8.0
    volumes:
      - ./init:/docker-entrypoint-initdb.d    # 自动执行 .sql 文件
```

---

## 八、本章命令速查表

| 操作 | 命令 |
|------|------|
| 启动（后台） | `docker compose up -d` |
| 启动（含构建） | `docker compose up -d --build` |
| 停止 | `docker compose stop` |
| 停止并清理 | `docker compose down` |
| 停止、清理、删卷 | `docker compose down -v` |
| 查看状态 | `docker compose ps` |
| 查看日志 | `docker compose logs -f` |
| 进入容器 | `docker compose exec web bash` |
| 扩缩容 | `docker compose up -d --scale worker=3` |
| 重新构建 | `docker compose build` |

---

## 九、本章小结

```
✅ Docker Compose 用 YAML 文件定义多服务应用
✅ docker compose up -d 一键启动，docker compose down 一键停止
✅ services 定义服务，volumes 定义存储，networks 定义网络
✅ depends_on + healthcheck 管理启动顺序
✅ .env 文件管理环境变量
✅ 多环境：base + override + prod 组合
✅ Compose 自动创建网络，服务名即主机名
```

---

> 下一篇：[09-nerdctl完全指南与containerd生态](./09-nerdctl完全指南与containerd生态.md) — 新一代容器工具！
