# 03 — Docker 镜像全面详解

> 镜像是 Docker 的灵魂。理解镜像，就理解了 Docker 一半的精髓。

---

## 一、什么是 Docker 镜像？

### 1.1 一句话定义

**镜像 = 一个只读的、分层的文件系统快照 + 运行配置**

它包含了运行一个应用所需的一切：
- 操作系统的基础文件（如 Ubuntu 的工具链）
- 应用运行时（如 Python、Node.js、JDK）
- 应用代码
- 依赖库
- 环境变量、启动命令等配置

### 1.2 生活类比

| 概念 | 类比 |
|------|------|
| 镜像 | 一个"预制菜包"——食材、调料、做法说明都在里面 |
| 容器 | 拆开预制菜包做出来的那道菜 |
| Dockerfile | 预制菜包的生产工艺流程 |
| Registry | 预制菜包的超市（Docker Hub） |

一个镜像可以创建无数个容器，就像一个预制菜包的配方可以批量生产无数份。

---

## 二、镜像的分层存储原理

### 2.1 分层结构

Docker 镜像由多个**只读层（Layer）** 叠加而成：

```
┌─────────────────────────┐
│  Layer 5: COPY app.py   │ ← 你的代码（几 KB）
├─────────────────────────┤
│  Layer 4: RUN pip install│ ← 安装的依赖（几十 MB）
├─────────────────────────┤
│  Layer 3: RUN apt-get   │ ← 安装的系统包
├─────────────────────────┤
│  Layer 2: ENV / WORKDIR │ ← 环境配置（元数据，极小）
├─────────────────────────┤
│  Layer 1: FROM python:3.11 │ ← 基础镜像（几百 MB）
└─────────────────────────┘
```

### 2.2 为什么要分层？

**核心优势：共享与复用**

```
镜像 A（Python Web App）        镜像 B（Python ML App）
┌──────────────────┐            ┌──────────────────┐
│  COPY webapp     │            │  COPY mlapp      │
├──────────────────┤            ├──────────────────┤
│  pip install flask│           │  pip install torch│
├──────────────────┤            ├──────────────────┤
│  python:3.11（共享！）──────────│  python:3.11（共享！）│
└──────────────────┘            └──────────────────┘
```

两个镜像共享同一个 `python:3.11` 基础层，磁盘上只存一份！

**实际效果**：
- 第一次拉 `python:3.11` 基础镜像：下载 1GB
- 第二个基于 `python:3.11` 的镜像：只下载新增的层（几 MB）

### 2.3 查看镜像分层

```bash
# 查看镜像的层信息
docker history nginx:latest

# 输出示例（从上到下是从新到旧）
IMAGE          CREATED       CREATED BY                                      SIZE
a8758716bb6a   2 days ago    CMD ["nginx" "-g" "daemon off;"]                0B
<missing>      2 days ago    EXPOSE map[80/tcp:{}]                           0B
<missing>      2 days ago    STOPSIGNAL SIGQUIT                              0B
<missing>      2 days ago    RUN /bin/sh -c set -x ...                       62.4MB
<missing>      2 days ago    COPY file:xxx in /docker-entrypoint.d           4.62kB
...

# 查看更详细的镜像信息
docker inspect nginx:latest
```

---

## 三、镜像命名规则

### 3.1 完整格式

```
[仓库地址/][命名空间/]镜像名[:标签][@摘要]
```

**拆解说明**：

```
registry.example.com/myteam/myapp:v1.0
│                     │       │     │
│                     │       │     └── 标签（Tag）：版本号
│                     │       └── 镜像名：应用名称
│                     └── 命名空间：通常是用户名或组织名
└── 仓库地址：私有仓库地址（省略则默认 Docker Hub）
```

### 3.2 常见示例

```bash
# 完整写法
docker.io/library/nginx:1.25      # Docker Hub 官方 nginx 1.25 版

# 简写（省略仓库地址和 library）
nginx:1.25                        # 等价于上面

# 省略标签（默认使用 latest）
nginx                             # 等价于 nginx:latest

# 带命名空间（某个用户/组织的镜像）
bitnami/redis:7.2                 # bitnami 组织的 redis

# 私有仓库
registry.cn-hangzhou.aliyuncs.com/myns/myapp:v2.0
```

### 3.3 Tag 的最佳实践

| Tag 类型 | 示例 | 说明 |
|----------|------|------|
| `latest` | `nginx:latest` | 最新版本，但**不建议生产使用**（内容会变） |
| 语义版本 | `nginx:1.25.3` | 精确版本，**生产环境推荐** |
| 主版本 | `python:3` | 3.x 的最新版 |
| 主次版本 | `python:3.11` | 3.11.x 的最新版 |
| slim | `python:3.11-slim` | 精简版，体积更小 |
| alpine | `python:3.11-alpine` | 基于 Alpine Linux，**最小体积** |

> **避坑**：生产环境永远不要用 `latest`！因为你今天拉的 latest 和明天拉的可能是不同版本，会导致不可预期的问题。

---

## 四、镜像基本操作

### 4.1 搜索镜像

```bash
# 在 Docker Hub 搜索镜像
docker search nginx

# 输出
NAME                    DESCRIPTION                                     STARS     OFFICIAL
nginx                   Official build of Nginx.                        19000     [OK]
bitnami/nginx           Bitnami container image for NGINX               200
linuxserver/nginx       An Nginx container...                           200
...

# 过滤：只显示官方镜像
docker search --filter is-official=true nginx

# 过滤：星标数大于 100
docker search --filter stars=100 nginx
```

### 4.2 拉取镜像

```bash
# 拉取最新版
docker pull nginx
# 等价于 docker pull docker.io/library/nginx:latest

# 拉取指定版本
docker pull nginx:1.25.3

# 拉取指定平台的镜像（适用于 M1/M2 Mac）
docker pull --platform linux/amd64 nginx:1.25.3
docker pull --platform linux/arm64 nginx:1.25.3

# 拉取私有仓库镜像（需要先 docker login）
docker pull registry.example.com/myteam/myapp:v1.0
```

**拉取过程解析**：

```
$ docker pull nginx:1.25.3
1.25.3: Pulling from library/nginx
a2abf6c4d29d: Already exists     ← 这一层本地已有，跳过
a9edb18cadd1: Pull complete       ← 下载新层
589b7251471a: Pull complete
186b1aaa4aa6: Pull complete
b4df32aa5a72: Pull complete
a0bcbecc962e: Pull complete
Digest: sha256:abc123...          ← 镜像的唯一摘要
Status: Downloaded newer image for nginx:1.25.3
```

### 4.3 查看本地镜像

```bash
# 列出所有本地镜像
docker images
# 或
docker image ls

# 输出
REPOSITORY    TAG       IMAGE ID       CREATED        SIZE
nginx         1.25.3    a8758716bb6a   2 days ago     187MB
python        3.11      d4b7e1c27c14   1 week ago     1.01GB
alpine        latest    05455a08881e   2 weeks ago    7.38MB

# 只显示镜像 ID
docker images -q

# 按仓库名过滤
docker images nginx

# 显示所有镜像（包括中间层镜像）
docker images -a

# 格式化输出
docker images --format "{{.Repository}}:{{.Tag}}\t{{.Size}}"

# 按体积排序（大到小）
docker images --format "{{.Size}}\t{{.Repository}}:{{.Tag}}" | sort -rh
```

### 4.4 删除镜像

```bash
# 按名称删除
docker rmi nginx:1.25.3
# 或
docker image rm nginx:1.25.3

# 按 ID 删除
docker rmi a8758716bb6a

# 强制删除（即使有容器在使用）
docker rmi -f nginx:1.25.3

# 删除所有未使用的镜像（悬空镜像）
docker image prune

# 删除所有未被容器引用的镜像（慎用！）
docker image prune -a

# 批量删除：删除所有 none 标签的镜像
docker rmi $(docker images -f "dangling=true" -q)
```

### 4.5 镜像标签操作

```bash
# 给镜像打标签（不会复制，只是多一个引用）
docker tag nginx:1.25.3 my-nginx:v1
docker tag nginx:1.25.3 registry.example.com/myteam/nginx:v1

# 查看效果
docker images
# 会看到两行，IMAGE ID 相同，说明是同一个镜像的不同标签
```

### 4.6 推送镜像

```bash
# 先登录仓库
docker login
# 输入用户名和密码

# 登录私有仓库
docker login registry.example.com

# 推送镜像（镜像名必须包含仓库地址和命名空间）
docker push myuser/myapp:v1.0
docker push registry.example.com/myteam/myapp:v1.0

# 推送所有标签
docker push myuser/myapp --all-tags

# 登出
docker logout
```

---

## 五、镜像导入导出（离线传输）

在没有网络或无法访问仓库的环境中，可以通过文件传输镜像。

### 5.1 save / load（推荐）

```bash
# 导出镜像到文件（保留完整的层信息和标签）
docker save -o nginx.tar nginx:1.25.3

# 导出多个镜像到同一个文件
docker save -o images.tar nginx:1.25.3 python:3.11 redis:7

# 导出并压缩（推荐，能节省很多空间）
docker save nginx:1.25.3 | gzip > nginx.tar.gz

# 导入镜像
docker load -i nginx.tar
docker load -i nginx.tar.gz    # 自动识别压缩格式

# 从标准输入导入
cat nginx.tar.gz | docker load
```

### 5.2 export / import（不推荐日常使用）

```bash
# 导出容器的文件系统（注意：是容器不是镜像！）
docker export my-container > container.tar

# 从文件系统导入为镜像（会丢失层信息和历史）
docker import container.tar myimage:v1
```

**save/load vs export/import 对比**：

| 特性 | save/load | export/import |
|------|-----------|---------------|
| 操作对象 | 镜像 | 容器 |
| 保留层信息 | ✅ 是 | ❌ 否（压成一层） |
| 保留标签 | ✅ 是 | ❌ 否 |
| 保留历史 | ✅ 是 | ❌ 否 |
| 适用场景 | 离线传输镜像 | 制作基础镜像快照 |

---

## 六、镜像详细信息查看

### 6.1 docker inspect

```bash
# 查看镜像的完整元信息（JSON 格式）
docker inspect nginx:1.25.3

# 查看特定字段
# 查看操作系统
docker inspect --format='{{.Os}}' nginx:1.25.3

# 查看架构
docker inspect --format='{{.Architecture}}' nginx:1.25.3

# 查看暴露的端口
docker inspect --format='{{.Config.ExposedPorts}}' nginx:1.25.3

# 查看环境变量
docker inspect --format='{{json .Config.Env}}' nginx:1.25.3

# 查看默认启动命令
docker inspect --format='{{json .Config.Cmd}}' nginx:1.25.3
```

### 6.2 docker history

```bash
# 查看镜像的构建历史（每一层做了什么）
docker history nginx:1.25.3

# 显示完整命令（不截断）
docker history --no-trunc nginx:1.25.3

# 只显示层大小
docker history --format "{{.Size}}\t{{.CreatedBy}}" nginx:1.25.3
```

---

## 七、常用基础镜像选择指南

选对基础镜像很重要，直接影响安全性和镜像大小。

### 7.1 通用基础镜像

| 镜像 | 体积 | 特点 | 适用场景 |
|------|------|------|---------|
| `ubuntu:22.04` | ~77MB | 功能全面，工具丰富 | 开发调试、功能全面的应用 |
| `debian:bookworm-slim` | ~74MB | 稳定，精简版 | 生产环境的通用选择 |
| `alpine:3.19` | ~7MB | **超级轻量**，使用 musl libc | 追求极小体积、安全场景 |
| `busybox` | ~4MB | 最小 Linux 工具集 | 极简场景 |
| `scratch` | 0MB | 空镜像 | Go 等静态编译语言 |

### 7.2 语言运行时镜像

```bash
# Python
python:3.11          # 完整版（约 1GB）
python:3.11-slim     # 精简版（约 150MB）推荐！
python:3.11-alpine   # Alpine 版（约 50MB）可能有兼容问题

# Node.js
node:20              # 完整版（约 1GB）
node:20-slim         # 精简版（约 200MB）推荐！
node:20-alpine       # Alpine 版（约 130MB）

# Java
eclipse-temurin:17-jre    # 只有 JRE（约 270MB）推荐运行时！
eclipse-temurin:17-jdk    # 有 JDK（约 400MB）用于构建

# Go
golang:1.22          # 构建用（约 800MB）
# Go 通常编译成静态二进制，运行时直接用 scratch 或 alpine
```

### 7.3 选择建议

```
生产环境优先级：
  语言-slim > 语言-alpine > debian-slim > 完整版

决策流程：
  能用 slim 就用 slim
  ↓ 太大了想更小
  试试 alpine（但要测试兼容性）
  ↓ Go 等可以静态编译的语言
  直接用 scratch（最小，0MB 基础）
```

---

## 八、镜像空间管理

### 8.1 查看磁盘使用

```bash
# 查看 Docker 整体磁盘使用情况
docker system df

# 输出
TYPE            TOTAL     ACTIVE    SIZE      RECLAIMABLE
Images          15        5         8.324GB   4.215GB (50%)
Containers      10        3         2.134MB   1.567MB (73%)
Local Volumes   8         4         1.234GB   234.5MB (19%)
Build Cache     20        0         3.456GB   3.456GB (100%)

# 详细信息
docker system df -v
```

### 8.2 清理策略

```bash
# 删除所有悬空镜像（没有标签的，<none>:<none>）
docker image prune

# 删除所有未被使用的镜像（慎用）
docker image prune -a

# 一键清理所有未使用的资源（镜像+容器+网络+缓存）
docker system prune

# 加上 volumes（连存储卷也清掉，数据会丢！）
docker system prune --volumes

# 清理超过 24 小时前创建的资源
docker system prune --filter "until=24h"
```

---

## 九、本章命令速查表

| 操作 | 命令 |
|------|------|
| 搜索镜像 | `docker search nginx` |
| 拉取镜像 | `docker pull nginx:1.25.3` |
| 查看本地镜像 | `docker images` |
| 查看镜像详情 | `docker inspect nginx` |
| 查看构建历史 | `docker history nginx` |
| 删除镜像 | `docker rmi nginx:1.25.3` |
| 打标签 | `docker tag nginx:1.25.3 my-nginx:v1` |
| 推送镜像 | `docker push myuser/myapp:v1.0` |
| 导出镜像 | `docker save -o file.tar nginx` |
| 导入镜像 | `docker load -i file.tar` |
| 清理悬空镜像 | `docker image prune` |
| 查看磁盘使用 | `docker system df` |

---

## 十、本章小结

```
✅ 镜像是只读的分层文件系统 + 运行配置
✅ 分层存储的好处：共享基础层，节省空间，加速下载
✅ 镜像命名：[仓库/][命名空间/]镜像名[:标签]
✅ 生产环境不要用 latest 标签
✅ save/load 用于镜像的离线传输
✅ 选基础镜像：slim > alpine > 完整版
✅ 定期 docker system prune 清理磁盘
```

---

> 下一篇：[04-Docker容器操作完全指南](./04-Docker容器操作完全指南.md) — 容器的增删改查，全在这里！
