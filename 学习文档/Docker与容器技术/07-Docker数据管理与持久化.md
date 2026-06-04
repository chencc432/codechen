# 07 — Docker 数据管理与持久化

> 容器是"用完就扔"的，但数据不能丢！搞懂持久化，你的数据才安全。

---

## 一、为什么需要数据持久化？

### 1.1 容器的"短暂性"

容器的文件系统本质上是一个**可写层（writable layer）**，叠加在镜像的只读层之上。

```
┌─────────────────────────┐
│  可写层（容器独有）       │ ← 你在容器里写的文件都在这里
├─────────────────────────┤
│  只读层（来自镜像）       │
├─────────────────────────┤
│  只读层（来自镜像）       │
└─────────────────────────┘
```

**问题**：容器一删（`docker rm`），可写层就没了，数据全丢！

```bash
# 演示：数据随容器消失
docker run -it --name test ubuntu bash
echo "important data" > /data.txt
exit

docker rm test
# /data.txt 随之消失，无法找回
```

### 1.2 哪些数据需要持久化？

| 数据类型 | 例子 | 必须持久化？ |
|---------|------|------------|
| 数据库文件 | MySQL 的 `/var/lib/mysql` | ✅ 必须 |
| 上传文件 | 用户头像、附件 | ✅ 必须 |
| 配置文件 | Nginx 配置、应用配置 | ✅ 推荐 |
| 日志文件 | 应用日志 | ⚠️ 看需求 |
| 缓存数据 | Redis 持久化数据 | ⚠️ 看需求 |
| 临时文件 | 编译中间产物 | ❌ 不需要 |

---

## 二、Docker 的三种数据持久化方式

```
┌─────────────────────────────────────────────────────┐
│                      容器                            │
│                                                     │
│  ┌──────────┐  ┌───────────┐  ┌──────────────────┐ │
│  │  Volume   │  │Bind Mount │  │      tmpfs       │ │
│  │  /data    │  │ /app/conf │  │  /tmp（内存）     │ │
│  └─────┬────┘  └─────┬─────┘  └──────────────────┘ │
└────────┼─────────────┼──────────────────────────────┘
         │             │
         ▼             ▼
  Docker 管理的     宿主机的
  存储区域          任意目录
  /var/lib/docker   /home/user
  /volumes/xxx      /my-config
```

| 方式 | 管理者 | 位置 | 适用场景 |
|------|-------|------|---------|
| **Volume** | Docker 管理 | `/var/lib/docker/volumes/` | 数据库、持久化数据 |
| **Bind Mount** | 用户管理 | 宿主机任意路径 | 配置文件、开发模式 |
| **tmpfs** | 内核管理 | 内存中 | 敏感数据、临时缓存 |

---

## 三、Volume（卷）— 推荐方案

### 3.1 什么是 Volume？

Volume 是 Docker 专门管理的存储空间，存在宿主机的 `/var/lib/docker/volumes/` 目录下。

**优点**：
- Docker 负责管理，不用关心底层路径
- 支持 Volume 驱动（可以接云存储、NFS 等）
- 容器删了，Volume 还在
- 多个容器可以共享同一个 Volume

### 3.2 Volume 操作

```bash
# 创建 Volume
docker volume create my-data

# 查看所有 Volume
docker volume ls

# 查看 Volume 详情
docker volume inspect my-data
# 输出会显示实际存储路径：/var/lib/docker/volumes/my-data/_data

# 删除 Volume
docker volume rm my-data

# 删除所有未使用的 Volume（慎用！）
docker volume prune
```

### 3.3 使用 Volume 运行容器

#### 方式一：命名卷（推荐）

```bash
# 使用 -v 语法
docker run -d \
    --name my-mysql \
    -v mysql-data:/var/lib/mysql \
    -e MYSQL_ROOT_PASSWORD=secret \
    mysql:8.0

# 解读：把名为 "mysql-data" 的 Volume 挂载到容器内的 /var/lib/mysql
# 如果 Volume 不存在，Docker 会自动创建
```

#### 方式二：匿名卷

```bash
# 不指定 Volume 名称
docker run -d \
    --name my-mysql \
    -v /var/lib/mysql \
    -e MYSQL_ROOT_PASSWORD=secret \
    mysql:8.0

# Docker 会自动创建一个随机名称的 Volume
# 不推荐：因为名字随机，难以管理
```

#### 方式三：--mount 语法（更明确）

```bash
docker run -d \
    --name my-mysql \
    --mount type=volume,source=mysql-data,target=/var/lib/mysql \
    -e MYSQL_ROOT_PASSWORD=secret \
    mysql:8.0
```

**-v vs --mount 对比**：

| 特性 | `-v` | `--mount` |
|------|------|-----------|
| 语法 | 简洁 | 冗长但清晰 |
| 目标不存在时 | 自动创建 | 报错 |
| 推荐度 | 日常使用 | 精确控制时使用 |

### 3.4 Volume 数据验证

```bash
# 写入数据
docker run -d --name db1 -v my-data:/data alpine sh -c "echo 'Hello Volume' > /data/test.txt && sleep infinity"

# 删除容器
docker rm -f db1

# 用新容器挂载同一个 Volume，数据还在！
docker run --rm -v my-data:/data alpine cat /data/test.txt
# 输出：Hello Volume
```

### 3.5 多容器共享 Volume

```bash
# 容器 A 写数据
docker run -d --name writer -v shared-data:/data alpine sh -c "while true; do date >> /data/log.txt; sleep 5; done"

# 容器 B 读数据
docker run --rm -v shared-data:/data alpine tail -f /data/log.txt
```

---

## 四、Bind Mount（绑定挂载）

### 4.1 什么是 Bind Mount？

直接把宿主机的某个目录或文件挂载到容器内。

### 4.2 基本用法

```bash
# 语法：-v 宿主机路径:容器路径
docker run -d \
    --name my-nginx \
    -p 80:80 \
    -v /home/user/html:/usr/share/nginx/html \
    -v /home/user/nginx.conf:/etc/nginx/nginx.conf:ro \
    nginx

# 绝对路径 → Bind Mount
# 非绝对路径 → Volume
```

### 4.3 只读挂载

```bash
# 加 :ro 表示容器内只能读，不能写
docker run -d \
    -v /path/to/config:/app/config:ro \
    my-app

# 使用 --mount 语法
docker run -d \
    --mount type=bind,source=/path/to/config,target=/app/config,readonly \
    my-app
```

### 4.4 开发模式中的 Bind Mount

最常用的场景——开发时把代码目录挂载进容器，实时生效：

```bash
# 前端开发：代码改了页面自动刷新
docker run -d \
    --name dev-frontend \
    -p 3000:3000 \
    -v $(pwd)/src:/app/src \
    node:20-slim \
    npx react-scripts start

# Python 开发：代码改了服务自动重启
docker run -d \
    --name dev-api \
    -p 8000:8000 \
    -v $(pwd):/app \
    -w /app \
    python:3.11-slim \
    python -m flask run --host=0.0.0.0 --reload
```

### 4.5 Windows / macOS 路径注意

```bash
# Windows（PowerShell）
docker run -v ${PWD}:/app my-app                    # PowerShell
docker run -v "C:\Users\me\project":/app my-app     # 绝对路径

# macOS / Linux
docker run -v $(pwd):/app my-app
docker run -v /Users/me/project:/app my-app
```

---

## 五、tmpfs 挂载（内存挂载）

```bash
# 数据存在内存中，容器停止就消失
docker run -d \
    --name secure-app \
    --tmpfs /tmp:size=100m \
    my-app

# 使用 --mount 语法
docker run -d \
    --mount type=tmpfs,target=/tmp,tmpfs-size=100m \
    my-app
```

**适用场景**：
- 临时密钥文件（不想写磁盘）
- 高性能临时缓存
- 敏感数据处理

---

## 六、Volume vs Bind Mount 如何选择？

| 场景 | 推荐 | 原因 |
|------|------|------|
| 数据库数据 | **Volume** | Docker 管理，备份方便 |
| 应用持久化数据 | **Volume** | 不依赖宿主机目录结构 |
| 开发时挂载代码 | **Bind Mount** | 需要实时同步本地文件 |
| 挂载配置文件 | **Bind Mount** | 宿主机直接编辑配置 |
| 日志输出 | **Bind Mount** | 方便宿主机上查看和收集 |
| 多容器共享数据 | **Volume** | 更安全，有命名管理 |

**简单记忆**：
- 需要 Docker 管理生命周期的 → **Volume**
- 需要和宿主机实时同步的 → **Bind Mount**

---

## 七、数据备份与恢复

### 7.1 备份 Volume

```bash
# 创建一个临时容器，挂载要备份的 Volume 和宿主机目录
docker run --rm \
    -v mysql-data:/source:ro \
    -v $(pwd):/backup \
    alpine \
    tar czf /backup/mysql-data-backup.tar.gz -C /source .
```

**流程解读**：
1. 挂载要备份的 `mysql-data` 到容器的 `/source`（只读）
2. 挂载宿主机当前目录到容器的 `/backup`
3. 在容器里把 `/source` 打包到 `/backup` 下

### 7.2 恢复 Volume

```bash
# 创建新 Volume
docker volume create mysql-data-new

# 解压备份到新 Volume
docker run --rm \
    -v mysql-data-new:/target \
    -v $(pwd):/backup:ro \
    alpine \
    tar xzf /backup/mysql-data-backup.tar.gz -C /target
```

### 7.3 迁移 Volume

```bash
# 方法一：备份 → 传输文件 → 恢复
# （在源机器）
docker run --rm -v my-vol:/data -v $(pwd):/backup alpine \
    tar czf /backup/vol.tar.gz -C /data .
scp vol.tar.gz user@target-host:~/

# （在目标机器）
docker volume create my-vol
docker run --rm -v my-vol:/data -v ~/:/backup alpine \
    tar xzf /backup/vol.tar.gz -C /data

# 方法二：直接复制底层目录（需要 root）
sudo cp -r /var/lib/docker/volumes/my-vol /var/lib/docker/volumes/my-vol-copy
```

---

## 八、常用存储场景实战

### 8.1 MySQL 数据持久化

```bash
docker run -d \
    --name mysql \
    -p 3306:3306 \
    -e MYSQL_ROOT_PASSWORD=secret \
    -e MYSQL_DATABASE=mydb \
    -v mysql-data:/var/lib/mysql \
    -v /path/to/my.cnf:/etc/mysql/conf.d/my.cnf:ro \
    --restart=unless-stopped \
    mysql:8.0
```

### 8.2 Redis 数据持久化

```bash
docker run -d \
    --name redis \
    -p 6379:6379 \
    -v redis-data:/data \
    --restart=unless-stopped \
    redis:7 redis-server --appendonly yes
```

### 8.3 Nginx 静态网站

```bash
docker run -d \
    --name nginx \
    -p 80:80 \
    -v $(pwd)/html:/usr/share/nginx/html:ro \
    -v $(pwd)/nginx.conf:/etc/nginx/nginx.conf:ro \
    -v nginx-logs:/var/log/nginx \
    --restart=unless-stopped \
    nginx:1.25
```

---

## 九、本章命令速查表

| 操作 | 命令 |
|------|------|
| 创建 Volume | `docker volume create my-data` |
| 查看所有 Volume | `docker volume ls` |
| 查看 Volume 详情 | `docker volume inspect my-data` |
| 删除 Volume | `docker volume rm my-data` |
| 清理未使用的 Volume | `docker volume prune` |
| 命名卷挂载 | `-v my-data:/container/path` |
| Bind Mount | `-v /host/path:/container/path` |
| 只读挂载 | `-v /path:/path:ro` |
| tmpfs 挂载 | `--tmpfs /tmp:size=100m` |

---

## 十、本章小结

```
✅ 容器删除后，可写层的数据会丢失
✅ 三种持久化方式：Volume、Bind Mount、tmpfs
✅ Volume 是首选方案，Docker 管理，安全可靠
✅ Bind Mount 适合开发模式和配置文件挂载
✅ tmpfs 用于敏感数据和高性能临时存储
✅ 命名卷（-v name:/path）优于匿名卷
✅ 备份 Volume 用 tar 打包，恢复用 tar 解压
✅ 数据库一定要挂 Volume，否则重启就丢数据
```

---

> 下一篇：[08-Docker Compose编排实战](./08-Docker-Compose编排实战.md) — 多个容器一起管理！
