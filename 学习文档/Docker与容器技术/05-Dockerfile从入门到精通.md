# 05 — Dockerfile 从入门到精通

> Dockerfile 是 Docker 的"灵魂配方"。写好 Dockerfile，你就能随时随地重建完全一致的环境。

---

## 一、什么是 Dockerfile？

### 1.1 一句话定义

**Dockerfile = 一个文本文件，用一条条指令描述如何从零构建一个镜像。**

### 1.2 为什么需要 Dockerfile？

| 方式 | 问题 |
|------|------|
| `docker commit` | 像"手工菜"，做完了别人不知道你怎么做的，不可重现 |
| **Dockerfile** | 像"标准化菜谱"，任何人拿到都能做出一模一样的"菜" |

**Dockerfile 的好处**：
- **可重现**：同一个 Dockerfile 构建出来的镜像一定一样
- **可追溯**：每一步操作都写在文件里，版本可控
- **可自动化**：CI/CD 流水线直接用 Dockerfile 构建
- **可审查**：代码审查时可以看到环境变更

---

## 二、第一个 Dockerfile

### 2.1 最简示例

创建一个文件，名为 `Dockerfile`（没有扩展名）：

```dockerfile
FROM python:3.11-slim

WORKDIR /app

COPY requirements.txt .

RUN pip install --no-cache-dir -r requirements.txt

COPY . .

EXPOSE 8000

CMD ["python", "app.py"]
```

### 2.2 逐行解读

```dockerfile
FROM python:3.11-slim
# 基础镜像，就像盖楼的地基。这里选 Python 3.11 精简版。

WORKDIR /app
# 设置工作目录，后续所有命令都在 /app 下执行。
# 如果目录不存在会自动创建。

COPY requirements.txt .
# 先只复制 requirements.txt（利用缓存，后面会详解）

RUN pip install --no-cache-dir -r requirements.txt
# 安装 Python 依赖

COPY . .
# 把当前目录的所有文件复制到容器的 /app

EXPOSE 8000
# 声明容器监听 8000 端口（只是文档作用，不是真的开端口）

CMD ["python", "app.py"]
# 容器启动时默认执行的命令
```

### 2.3 构建和运行

```bash
# 构建镜像（-t 指定名称和标签，. 表示 Dockerfile 所在目录）
docker build -t my-python-app:v1 .

# 运行
docker run -d -p 8000:8000 my-python-app:v1
```

---

## 三、Dockerfile 指令全解

### 3.1 FROM — 指定基础镜像

```dockerfile
# 基本用法
FROM ubuntu:22.04

# 指定平台
FROM --platform=linux/amd64 python:3.11-slim

# 多阶段构建中的别名
FROM golang:1.22 AS builder

# 空镜像（用于静态编译的二进制文件）
FROM scratch
```

**规则**：每个 Dockerfile 必须以 FROM 开头（ARG 除外）。

### 3.2 RUN — 执行命令

```dockerfile
# Shell 格式（通过 /bin/sh -c 执行）
RUN apt-get update && apt-get install -y curl

# Exec 格式（直接执行，不经过 shell）
RUN ["apt-get", "install", "-y", "curl"]

# 多条命令用 && 连接（减少镜像层数！）
RUN apt-get update && \
    apt-get install -y \
        curl \
        wget \
        vim && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*
```

**关键原则：合并 RUN 命令，减少层数**

```dockerfile
# ❌ 错误：每个 RUN 都会创建一层，浪费空间
RUN apt-get update
RUN apt-get install -y curl
RUN apt-get install -y wget
RUN apt-get clean

# ✅ 正确：合并成一个 RUN，只创建一层
RUN apt-get update && \
    apt-get install -y curl wget && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*
```

### 3.3 COPY vs ADD — 复制文件

```dockerfile
# COPY：简单复制（推荐）
COPY app.py /app/
COPY . /app/
COPY --chown=1000:1000 files/ /app/files/

# ADD：高级复制（有两个额外能力）
ADD https://example.com/file.tar.gz /app/     # 1. 可以下载 URL
ADD archive.tar.gz /app/                       # 2. 自动解压 tar 文件
```

**建议**：除非需要自动解压，否则一律用 COPY。因为 COPY 语义更清晰。

### 3.4 CMD vs ENTRYPOINT — 启动命令

这两个是 Dockerfile 里最容易混淆的指令。

#### CMD — 默认命令（可被覆盖）

```dockerfile
# Exec 格式（推荐）
CMD ["python", "app.py"]

# Shell 格式
CMD python app.py

# 仅作为 ENTRYPOINT 的默认参数
CMD ["--port", "8000"]
```

```bash
# CMD 可以被 docker run 的命令覆盖
docker run my-app                    # 执行 CMD 里的 python app.py
docker run my-app python test.py     # CMD 被覆盖，执行 python test.py
```

#### ENTRYPOINT — 入口命令（不可被覆盖）

```dockerfile
# Exec 格式（推荐）
ENTRYPOINT ["python", "app.py"]
```

```bash
# ENTRYPOINT 不会被覆盖，而是追加参数
docker run my-app                    # 执行 python app.py
docker run my-app --debug            # 执行 python app.py --debug
```

#### CMD + ENTRYPOINT 配合使用

```dockerfile
ENTRYPOINT ["python", "app.py"]
CMD ["--port", "8000"]
```

```bash
docker run my-app                    # python app.py --port 8000
docker run my-app --port 9000        # python app.py --port 9000（CMD 被覆盖）
```

**总结表**：

| 场景 | 推荐方式 |
|------|---------|
| 一般应用 | 只用 `CMD` |
| 要固定入口程序、但参数可变 | `ENTRYPOINT` + `CMD`（默认参数） |
| 容器当命令行工具用 | `ENTRYPOINT` 设置工具，`CMD` 设置默认参数 |

### 3.5 ENV — 环境变量

```dockerfile
# 设置环境变量
ENV APP_ENV=production
ENV DB_HOST=localhost DB_PORT=5432

# 后续指令可以引用
RUN echo $APP_ENV
COPY config.$APP_ENV.json /app/config.json
```

```bash
# 运行时可以覆盖
docker run -e APP_ENV=development my-app
```

### 3.6 ARG — 构建参数

```dockerfile
# 定义构建参数（只在构建时有效，运行时不存在！）
ARG PYTHON_VERSION=3.11
FROM python:${PYTHON_VERSION}-slim

ARG APP_VERSION=1.0
RUN echo "Building version: $APP_VERSION"
```

```bash
# 构建时传入参数
docker build --build-arg PYTHON_VERSION=3.12 --build-arg APP_VERSION=2.0 -t my-app .
```

**ARG vs ENV 对比**：

| 特性 | ARG | ENV |
|------|-----|-----|
| 生效范围 | 仅构建阶段 | 构建 + 运行阶段 |
| 运行时可见 | ❌ | ✅ |
| `docker run -e` 可覆盖 | ❌ | ✅ |
| 可以在 FROM 之前使用 | ✅ | ❌ |

### 3.7 EXPOSE — 声明端口

```dockerfile
EXPOSE 80
EXPOSE 443
EXPOSE 8080/tcp
EXPOSE 8125/udp
```

**注意**：EXPOSE 只是"文档声明"，并不会真的打开端口。真正的端口映射需要 `docker run -p`。

### 3.8 VOLUME — 声明匿名卷

```dockerfile
VOLUME /data
VOLUME ["/data", "/logs"]
```

### 3.9 WORKDIR — 工作目录

```dockerfile
WORKDIR /app
# 后续所有 RUN、CMD、COPY 等都在 /app 下执行

WORKDIR sub-dir
# 相对路径，变成 /app/sub-dir

# 不要用 RUN cd，因为每个 RUN 是独立的 shell
# ❌ RUN cd /app && ...
# ✅ WORKDIR /app
```

### 3.10 USER — 运行用户

```dockerfile
# 创建非 root 用户
RUN groupadd -r appuser && useradd -r -g appuser appuser

# 切换到该用户
USER appuser

# 之后的 RUN、CMD、ENTRYPOINT 都以 appuser 身份执行
```

### 3.11 HEALTHCHECK — 健康检查

```dockerfile
HEALTHCHECK --interval=30s --timeout=5s --retries=3 --start-period=10s \
    CMD curl -f http://localhost:8080/health || exit 1
```

| 参数 | 说明 | 默认值 |
|------|------|-------|
| `--interval` | 检查间隔 | 30s |
| `--timeout` | 超时时间 | 30s |
| `--retries` | 连续失败几次判定不健康 | 3 |
| `--start-period` | 启动后的宽限期 | 0s |

```bash
# 查看健康状态
docker ps
# STATUS 列会显示 (healthy) 或 (unhealthy)
```

### 3.12 LABEL — 元数据

```dockerfile
LABEL maintainer="yourname@example.com"
LABEL version="1.0"
LABEL description="My awesome application"
```

---

## 四、.dockerignore 文件

和 `.gitignore` 类似，告诉 Docker 构建时忽略哪些文件。

```
# .dockerignore

# 版本控制
.git
.gitignore

# 依赖目录
node_modules
__pycache__
*.pyc
.venv

# IDE 配置
.vscode
.idea

# Docker 相关
Dockerfile
docker-compose.yml
.dockerignore

# 日志和临时文件
*.log
tmp/
temp/

# 文档（通常不需要放进镜像）
*.md
docs/
LICENSE
```

**为什么需要 .dockerignore**：
1. **加速构建**：不把无关文件发送给 Docker Daemon
2. **减小镜像**：避免把 `node_modules`、`.git` 等打进镜像
3. **安全**：避免把 `.env`、密钥文件打进镜像

---

## 五、多阶段构建（Multi-stage Build）

### 5.1 问题背景

编译型语言（Go、Java、C++ 等）需要编译工具，但运行时不需要。如果不做处理：

```
镜像 = 编译工具（几百 MB）+ 编译产物（几 MB）= 总共很大
```

### 5.2 多阶段构建的思路

```
阶段一：构建环境 → 编译代码 → 产出可执行文件
阶段二：运行环境 → 只拷贝可执行文件 → 最终镜像很小
```

### 5.3 Go 语言示例

```dockerfile
# ========== 阶段一：构建 ==========
FROM golang:1.22 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server .

# ========== 阶段二：运行 ==========
FROM alpine:3.19

RUN apk --no-cache add ca-certificates
WORKDIR /app

COPY --from=builder /app/server .

EXPOSE 8080
CMD ["./server"]
```

**效果对比**：

```
单阶段镜像（包含 Go 编译器）：约 1.2 GB
多阶段镜像（只有 Alpine + 二进制文件）：约 15 MB
体积缩减 98%！
```

### 5.4 Node.js 示例

```dockerfile
# 阶段一：安装依赖和构建
FROM node:20-slim AS builder

WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

# 阶段二：只保留构建产物和生产依赖
FROM node:20-slim

WORKDIR /app
COPY package*.json ./
RUN npm ci --production
COPY --from=builder /app/dist ./dist

EXPOSE 3000
CMD ["node", "dist/index.js"]
```

### 5.5 Python 示例

```dockerfile
# 阶段一：安装编译型依赖
FROM python:3.11-slim AS builder

RUN pip install --user --no-cache-dir numpy pandas scikit-learn

# 阶段二：只拷贝安装好的包
FROM python:3.11-slim

COPY --from=builder /root/.local /root/.local
ENV PATH=/root/.local/bin:$PATH

WORKDIR /app
COPY . .

CMD ["python", "app.py"]
```

---

## 六、构建缓存机制

### 6.1 缓存原理

Docker 构建镜像时，每条指令都会检查缓存：
- 如果这条指令和之前构建时完全一样 → 使用缓存（超快）
- 如果有任何变化 → 从这条指令开始，后续所有层都要重新构建

```
FROM python:3.11-slim        ← 有缓存，秒过
WORKDIR /app                  ← 有缓存，秒过
COPY requirements.txt .       ← requirements.txt 没变？有缓存！
RUN pip install ...           ← 上一步有缓存，这步也有缓存！
COPY . .                      ← 代码改了！缓存失效！
CMD ["python", "app.py"]      ← 上一步失效，这步也要重建
```

### 6.2 缓存优化：先复制依赖文件，再复制代码

```dockerfile
# ✅ 好的写法：充分利用缓存
COPY requirements.txt .            # 依赖文件很少变
RUN pip install -r requirements.txt  # 有缓存，跳过
COPY . .                            # 代码经常变，但不影响上面的缓存

# ❌ 坏的写法：每次改代码都要重装依赖
COPY . .                            # 代码一改，缓存全部失效
RUN pip install -r requirements.txt  # 每次都要重新安装，太慢了
```

### 6.3 其他缓存技巧

```bash
# 构建时禁用缓存（排查问题时用）
docker build --no-cache -t my-app .

# 从特定阶段开始重建
docker build --target builder -t my-app-builder .
```

---

## 七、Dockerfile 最佳实践

### 7.1 镜像瘦身

```dockerfile
# 1. 用 slim 或 alpine 基础镜像
FROM python:3.11-slim     # ✅ 150MB
# FROM python:3.11        # ❌ 1GB

# 2. 合并 RUN 命令，并在同一层清理
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        build-essential && \
    pip install --no-cache-dir -r requirements.txt && \
    apt-get purge -y build-essential && \
    apt-get autoremove -y && \
    rm -rf /var/lib/apt/lists/*

# 3. 使用 .dockerignore 排除不需要的文件

# 4. 多阶段构建，只保留运行时需要的东西
```

### 7.2 安全加固

```dockerfile
# 1. 不要用 root 运行应用
RUN groupadd -r appuser && useradd -r -g appuser -s /sbin/nologin appuser
USER appuser

# 2. 使用固定版本的基础镜像
FROM python:3.11.7-slim    # ✅ 固定版本
# FROM python:latest        # ❌ 版本不可控

# 3. 不要在镜像里放密钥
# ❌ COPY .env /app/.env
# ✅ 运行时通过 -e 或 --env-file 传入

# 4. 最小权限原则：只安装需要的包
RUN apt-get install -y --no-install-recommends curl
```

### 7.3 可维护性

```dockerfile
# 1. 把 ARG 放在合适的位置
ARG PYTHON_VERSION=3.11
FROM python:${PYTHON_VERSION}-slim

# 2. LABEL 标注维护信息
LABEL maintainer="team@example.com"
LABEL version="1.0.0"

# 3. 健康检查
HEALTHCHECK --interval=30s --timeout=5s \
    CMD curl -f http://localhost:8000/health || exit 1
```

---

## 八、docker build 命令详解

```bash
# 基本构建
docker build -t my-app:v1 .

# 指定 Dockerfile 路径
docker build -f Dockerfile.prod -t my-app:prod .

# 传入构建参数
docker build --build-arg VERSION=2.0 -t my-app:v2 .

# 不使用缓存
docker build --no-cache -t my-app:v1 .

# 只构建到某个阶段（多阶段构建）
docker build --target builder -t my-app-builder .

# 构建多平台镜像（需要 buildx）
docker buildx build --platform linux/amd64,linux/arm64 -t my-app:v1 .

# 构建并推送到仓库
docker buildx build --push -t myuser/my-app:v1 .
```

---

## 九、完整实战：Python Flask 应用

### 项目结构

```
my-flask-app/
├── app.py
├── requirements.txt
├── Dockerfile
└── .dockerignore
```

### app.py

```python
from flask import Flask, jsonify

app = Flask(__name__)

@app.route("/")
def hello():
    return jsonify({"message": "Hello from Docker!"})

@app.route("/health")
def health():
    return jsonify({"status": "healthy"})

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8000)
```

### requirements.txt

```
flask==3.0.0
gunicorn==21.2.0
```

### Dockerfile

```dockerfile
FROM python:3.11-slim

RUN groupadd -r appuser && useradd -r -g appuser appuser

WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY . .

RUN chown -R appuser:appuser /app
USER appuser

EXPOSE 8000

HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
    CMD python -c "import urllib.request; urllib.request.urlopen('http://localhost:8000/health')" || exit 1

CMD ["gunicorn", "--bind", "0.0.0.0:8000", "--workers", "4", "app:app"]
```

### 构建和运行

```bash
docker build -t my-flask-app:v1 .
docker run -d -p 8000:8000 --name flask-app my-flask-app:v1

# 测试
curl http://localhost:8000/
# {"message": "Hello from Docker!"}

curl http://localhost:8000/health
# {"status": "healthy"}
```

---

## 十、本章命令速查表

| 操作 | 命令 |
|------|------|
| 构建镜像 | `docker build -t name:tag .` |
| 指定 Dockerfile | `docker build -f Dockerfile.prod .` |
| 传构建参数 | `docker build --build-arg KEY=VAL .` |
| 无缓存构建 | `docker build --no-cache .` |
| 多阶段-只构建某阶段 | `docker build --target stage .` |
| 查看构建历史 | `docker history image:tag` |

---

## 十一、本章小结

```
✅ Dockerfile 是构建镜像的"源代码"，可重现、可审查
✅ FROM 是起点，RUN 执行命令，COPY 复制文件
✅ CMD 是默认命令（可覆盖），ENTRYPOINT 是固定入口
✅ ARG 只在构建时有效，ENV 在运行时也有效
✅ 多阶段构建能大幅减小镜像体积（98%+）
✅ 先 COPY 依赖文件再 COPY 代码，充分利用缓存
✅ 不用 root、不放密钥、用固定版本
✅ .dockerignore 排除不需要进镜像的文件
```

---

> 下一篇：[06-Docker网络详解](./06-Docker网络详解.md) — 容器之间怎么通信？
