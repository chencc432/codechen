# 04 — Docker 容器操作完全指南

> 容器是 Docker 的核心产出物——镜像的"活的"形态。学会管理容器，才算真正上手 Docker。

---

## 一、容器的生命周期

### 1.1 先搞懂容器到底是什么

**一句话**：容器就是一个被隔离的进程。

用生活来类比：
- **镜像** = 一个游戏的安装包（`.exe` 安装文件）
- **容器** = 你双击安装包之后，运行起来的那个游戏窗口
- 同一个安装包可以开好几个游戏窗口（同一个镜像创建多个容器）
- 关掉游戏窗口（停止容器），安装包还在（镜像还在）
- 卸载游戏（删除容器），安装包也还在（镜像也还在）

### 1.2 容器的五种状态

```
                docker create
                    │
                    ▼
  ┌──────┐    ┌──────────┐    docker start    ┌─────────┐
  │ 不存在 │───▶│  Created  │───────────────────▶│ Running │
  └──────┘    │  (已创建)  │                    │ (运行中) │
              └──────────┘                    └────┬────┘
                    ▲                              │
                    │                    ┌─────────┼──────────┐
                    │                    │         │          │
                    │              docker stop  docker pause  进程退出
                    │                    │         │          │
                    │                    ▼         ▼          ▼
                    │            ┌─────────┐ ┌─────────┐ ┌────────┐
                    │            │ Stopped  │ │ Paused  │ │ Exited │
                    │            │ (已停止)  │ │ (已暂停) │ │(已退出) │
                    │            └─────────┘ └─────────┘ └────────┘
                    │                    │         │          │
                    │                    └─────────┼──────────┘
                    │                              │
                    │                        docker rm
                    │                              │
                    └──────────────────────────────┘
                                            (回到不存在)

快捷方式：docker run = docker create + docker start
```

**每种状态详解**：

| 状态 | 含义 | 类比 | 触发方式 |
|------|------|------|---------|
| **Created** | 容器已创建但没运行 | 游戏安装好了但没双击启动 | `docker create` |
| **Running** | 容器正在运行 | 游戏正在跑 | `docker start` 或 `docker run` |
| **Paused** | 容器被暂停，进程冻结，内存保留 | 游戏按了暂停键 | `docker pause` |
| **Stopped** | 容器被主动停止 | 你手动关了游戏 | `docker stop` |
| **Exited** | 容器里的主进程自己退出了 | 游戏自己崩溃了或通关结束了 | 主进程结束/崩溃 |

**关键概念——"主进程"**：

每个容器都有一个**主进程**（PID=1），这个进程就是容器的"生命线"：
- 主进程活着 → 容器就是 Running
- 主进程退出 → 容器变成 Exited
- 主进程的退出码（exit code）就是容器的退出码

```bash
# 例子：这个容器的主进程是 echo，执行完就退出了
docker run ubuntu echo "hello"
# 容器一闪就没了，因为 echo 执行完就退出了

# 这个容器的主进程是 nginx，会一直运行
docker run -d nginx
# 容器持续运行，因为 nginx 一直在监听请求
```

---

## 二、docker run 参数超详解

`docker run` 是你用得最多的命令，我把每个参数都掰开了讲。

### 基本语法

```bash
docker run [选项] 镜像名[:标签] [命令] [参数]
#           │      │              │       │
#           │      │              │       └── 传给命令的参数
#           │      │              └── 覆盖镜像默认的启动命令
#           │      └── 用哪个镜像
#           └── 各种配置参数（下面详解）
```

**语法拆解示例**：

```bash
docker run -d --name web -p 8080:80 -e MSG=hi nginx:1.25 nginx -g "daemon off;"
#          │  │          │          │          │          │
#          │  │          │          │          │          └── [命令+参数] 覆盖默认启动命令
#          │  │          │          │          └── 镜像名:标签
#          │  │          │          └── 选项：设置环境变量
#          │  │          └── 选项：端口映射
#          │  └── 选项：指定容器名
#          └── 选项：后台运行
```

---

### 2.1 `-d` — 后台运行（Detach 模式）

**是什么**：让容器在后台运行，不占用你的终端。

**类比**：就像你把音乐播放器最小化到后台，它继续播放，但不挡你的屏幕。

```bash
# 不加 -d：前台运行，日志直接打到你的终端，Ctrl+C 会停止容器
docker run nginx
# 终端被占用了，你不能做别的事

# 加 -d：后台运行，终端立刻返回，你可以继续干别的
docker run -d nginx
# 输出一串容器 ID：a1b2c3d4e5f6...
# 然后你的终端就自由了
```

**什么时候用 -d**：

| 场景 | 用不用 -d | 原因 |
|------|----------|------|
| Web 服务器（nginx、tomcat） | ✅ 用 `-d` | 需要持续运行 |
| 数据库（mysql、redis） | ✅ 用 `-d` | 需要持续运行 |
| 跑个脚本然后退出 | ❌ 不用 | 需要看到输出结果 |
| 进容器里敲命令调试 | ❌ 不用，用 `-it` | 需要交互 |

---

### 2.2 `-it` — 交互模式

`-it` 其实是两个参数的组合：
- **`-i`**（interactive）：保持标准输入打开，让你能输入东西
- **`-t`**（tty）：分配一个伪终端，让输出有颜色、有格式

**类比**：
- 不加 `-it`：就像给别人发微信语音消息，说完就结束
- 加 `-it`：就像打电话，双向实时对话

```bash
# 加 -it：你可以在容器里交互，就像 SSH 到一台远程服务器
docker run -it ubuntu bash
# 现在你进入了 Ubuntu 容器内部
root@a1b2c3d4e5f6:/# ls        ← 你可以在里面执行命令
root@a1b2c3d4e5f6:/# whoami
root
root@a1b2c3d4e5f6:/# exit      ← 退出容器（容器也会停止）

# 不加 -it：没有交互能力
docker run ubuntu bash
# 容器直接退出了，因为 bash 没有输入源，它就结束了
```

**-i 和 -t 分开用的场景**：

```bash
# 只要 -i（有输入，没有终端格式）：适合管道操作
echo "hello world" | docker run -i ubuntu cat
# 输出：hello world
# 把 "hello world" 通过管道传给容器里的 cat 命令

# 只要 -t（有终端格式，没有输入）：几乎不用
# 一般都是 -it 一起用
```

**`-it` vs `-d` 怎么选？**

```
你想做什么？
├── 启动一个服务，让它在后台一直跑 → 用 -d
├── 进入容器里面敲命令 → 用 -it
├── 跑个一次性脚本，看输出 → 都不加（前台运行到结束）
└── 启动服务并且看日志 → 用 -d，然后用 docker logs -f 看
```

---

### 2.3 `--rm` — 用完即删

**是什么**：容器退出后自动删除容器，不留垃圾。

**类比**：用一次性纸杯喝完水，杯子自动扔掉。

```bash
# 不加 --rm：容器退出后还在，变成 Exited 状态，占磁盘
docker run ubuntu echo "hello"
docker ps -a
# 你会看到一个 Exited 状态的容器

# 加 --rm：容器退出后自动删除
docker run --rm ubuntu echo "hello"
docker ps -a
# 干干净净，没有残留
```

**什么时候用 --rm**：

| 场景 | 用不用 --rm |
|------|-----------|
| 测试一个命令的效果 | ✅ 用 |
| 跑个一次性脚本 | ✅ 用 |
| 进容器调试看看 | ✅ 用 |
| 运行一个长期服务 | ❌ 不用（重启后容器就没了） |
| 数据库等重要服务 | ❌ 绝对不用 |

**注意**：`--rm` 和 `--restart` 不能同时用（一个说退出就删，一个说退出就重启，矛盾了）。

---

### 2.4 `--name` — 给容器取名

**是什么**：给容器一个有意义的名字，方便后续管理。

**类比**：就像给你的宠物取名字。不取名的话，Docker 会随机生成一个名字（比如 `cranky_einstein`），你记不住。

```bash
# 不指定名字：Docker 随机起名
docker run -d nginx
docker ps
# NAMES 列显示：flamboyant_hopper（随机的，每次不同）

# 指定名字
docker run -d --name my-web-server nginx
docker ps
# NAMES 列显示：my-web-server（清晰明了）

# 后续操作都可以用名字代替容器 ID
docker stop my-web-server     # 比 docker stop a1b2c3d4e5f6 好记多了
docker logs my-web-server
docker exec -it my-web-server bash
```

**命名规则**：
- 只能用字母、数字、下划线 `_`、点 `.`、连字符 `-`
- 必须唯一，不能有两个同名容器
- 如果旧容器占用了名字，要先删掉才能创建同名的

```bash
# 名字被占用了
docker run -d --name web nginx
docker run -d --name web nginx    # 报错！名字已存在

# 解决：先删旧的
docker rm -f web
docker run -d --name web nginx    # 现在可以了
```

---

### 2.5 `-p` — 端口映射（最重要的网络参数）

**是什么**：把容器内部的端口"映射"到宿主机上，让外部能访问容器里的服务。

**为什么需要**：容器默认是隔离的，有自己的网络。就像一个封闭的房间，门都是锁着的。`-p` 就是帮你开一扇窗，让外面的人能和里面通信。

#### 语法详解

```bash
-p 宿主机端口:容器端口
```

**通俗解释**：

```
-p 8080:80
    │     │
    │     └── 容器内部监听的端口（Nginx 默认用 80）
    └── 你的电脑/服务器对外暴露的端口

意思是：当有人访问你的电脑的 8080 端口时，
Docker 会把请求转发到容器内部的 80 端口。
```

#### 图解

```
外部用户浏览器
    │
    │ 访问 http://你的IP:8080
    ▼
┌─────────────────────────────────┐
│  宿主机（你的电脑/服务器）         │
│                                 │
│  端口 8080 ─────┐               │
│                 │ Docker 转发    │
│                 ▼               │
│  ┌──────────────────────┐      │
│  │  容器                 │      │
│  │  Nginx 监听端口 80    │      │
│  │  ← 接收到请求并响应    │      │
│  └──────────────────────┘      │
└─────────────────────────────────┘
```

#### 各种写法

```bash
# 写法 1：最常用 — 宿主机 8080 映射到容器 80
docker run -d -p 8080:80 nginx
# 访问 http://localhost:8080 就能看到 Nginx 页面

# 写法 2：宿主机和容器端口相同
docker run -d -p 80:80 nginx
# 访问 http://localhost:80（即 http://localhost）

# 写法 3：映射多个端口
docker run -d -p 80:80 -p 443:443 nginx
# HTTP 和 HTTPS 都映射了

# 写法 4：指定绑定的 IP — 只允许本机访问
docker run -d -p 127.0.0.1:8080:80 nginx
# 只能通过 localhost:8080 访问，其他电脑访问不了
# 适合不想暴露到公网的服务（比如开发环境的数据库）

# 写法 5：绑定到所有网卡（默认行为）
docker run -d -p 0.0.0.0:8080:80 nginx
# 等价于 -p 8080:80，所有网卡都能访问

# 写法 6：指定协议（默认 TCP）
docker run -d -p 8125:8125/udp my-app
# UDP 端口映射（例如日志收集器 StatsD）

# 写法 7：让 Docker 随机分配宿主机端口
docker run -d -p 80 nginx
# Docker 会随机选一个宿主机端口（比如 32768）
# 用 docker port 查看分配了哪个端口
docker port <容器名或ID>
# 80/tcp -> 0.0.0.0:32768
```

#### `-P`（大写）— 随机映射所有端口

```bash
# 把镜像里 EXPOSE 声明的所有端口都随机映射
docker run -d -P nginx
# Nginx 镜像 EXPOSE 了 80 端口
# Docker 随机分配一个宿主机端口

docker port <容器ID>
# 80/tcp -> 0.0.0.0:32769
```

#### 端口冲突怎么办？

```bash
docker run -d -p 80:80 nginx
docker run -d -p 80:80 nginx    # 报错！宿主机 80 端口已被第一个容器占了

# 报错信息：
# Error: Bind for 0.0.0.0:80 failed: port is already allocated

# 解决：换一个宿主机端口
docker run -d -p 8081:80 nginx    # 用 8081，不冲突了
```

---

### 2.6 `-v` — 挂载卷（数据持久化）

**是什么**：把宿主机的目录"挂载"到容器里，让容器能读写宿主机的文件。

**为什么需要**：容器删了，里面的数据就没了。用 `-v` 把数据存到宿主机上，删容器数据也不会丢。

**类比**：容器就像一个临时工位，每天下班会被清理。`-v` 就像你带了一个自己的抽屉，工位清理了，抽屉里的东西还在。

#### 语法详解

```bash
# 格式 1：命名卷（Docker 管理存储位置）
-v 卷名:容器内路径
docker run -d -v mysql-data:/var/lib/mysql mysql:8.0
#              │           │
#              │           └── 容器内的路径（MySQL 数据存在这里）
#              └── 卷的名字（Docker 自动管理实际存储位置）

# 格式 2：绑定挂载（你指定宿主机路径）
-v 宿主机路径:容器内路径
docker run -d -v /home/user/html:/usr/share/nginx/html nginx
#              │                │
#              │                └── 容器内的路径
#              └── 宿主机的路径（你自己控制）

# 格式 3：只读挂载（容器只能读不能写）
-v 宿主机路径:容器内路径:ro
docker run -d -v /path/to/config:/etc/nginx/nginx.conf:ro nginx
#                                                       │
#                                                       └── ro = read only，只读
```

#### 怎么区分"命名卷"和"绑定挂载"？

```bash
# 规则很简单：看冒号前面的部分
-v my-data:/data          # 冒号前没有 / 开头 → 命名卷
-v /home/user/data:/data  # 冒号前以 / 开头 → 绑定挂载
-v ./data:/data           # 冒号前以 ./ 开头 → 绑定挂载（相对路径）
```

#### 实战示例

```bash
# 示例 1：MySQL 数据持久化（命名卷）
docker run -d --name mysql \
    -v mysql-data:/var/lib/mysql \
    -e MYSQL_ROOT_PASSWORD=123456 \
    mysql:8.0
# 即使删掉容器，mysql-data 卷里的数据还在
# 重新创建容器挂载同一个卷，数据自动恢复

# 示例 2：把你的网站文件挂进 Nginx（绑定挂载）
docker run -d --name web \
    -p 80:80 \
    -v /home/user/my-website:/usr/share/nginx/html \
    nginx
# 你在 /home/user/my-website 里修改文件，容器里立刻生效
# 非常适合开发调试

# 示例 3：挂载配置文件（只读）
docker run -d --name web \
    -p 80:80 \
    -v /home/user/nginx.conf:/etc/nginx/nginx.conf:ro \
    nginx
# 容器读取你自定义的 nginx.conf，但不能修改它

# 示例 4：挂载多个路径
docker run -d --name web \
    -p 80:80 \
    -v /home/user/html:/usr/share/nginx/html:ro \
    -v /home/user/nginx.conf:/etc/nginx/nginx.conf:ro \
    -v nginx-logs:/var/log/nginx \
    nginx
# 网站文件：只读挂载
# 配置文件：只读挂载
# 日志文件：命名卷（持久化保存）
```

#### Windows 路径写法

```powershell
# PowerShell 中用 ${PWD} 获取当前目录
docker run -d -v ${PWD}:/app my-app

# 或者用绝对路径（注意用正斜杠或双引号）
docker run -d -v "C:\Users\me\project":/app my-app
docker run -d -v C:/Users/me/project:/app my-app
```

---

### 2.7 `-e` — 设置环境变量

**是什么**：向容器内部注入环境变量，让应用可以读取配置。

**为什么需要**：很多应用（尤其是数据库、Web 应用）的配置不是写在文件里的，而是通过环境变量传入的。这样同一个镜像可以在不同环境（开发、测试、生产）用不同的配置。

**类比**：就像你去不同的餐厅点"招牌菜"——同一个名字，但每个餐厅做出来的味道不同。环境变量就是决定"味道"的调料配方。

#### 基本用法

```bash
# 设置一个环境变量
docker run -d -e MYSQL_ROOT_PASSWORD=my-secret mysql:8.0
#              │  │                    │
#              │  │                    └── 变量的值
#              │  └── 变量名
#              └── -e 参数

# 设置多个环境变量
docker run -d \
    -e MYSQL_ROOT_PASSWORD=my-secret \
    -e MYSQL_DATABASE=mydb \
    -e MYSQL_USER=admin \
    -e MYSQL_PASSWORD=admin123 \
    mysql:8.0

# 从宿主机环境变量传递（不写=值，自动读取宿主机同名变量）
export MY_SECRET=abc123
docker run -d -e MY_SECRET my-app
# 容器内的 MY_SECRET 值就是 abc123
```

#### 验证环境变量是否生效

```bash
# 方法 1：进入容器查看
docker exec -it my-container env
# 会列出所有环境变量

docker exec -it my-container echo $MYSQL_ROOT_PASSWORD
# 输出：my-secret

# 方法 2：用 docker inspect 查看
docker inspect --format='{{json .Config.Env}}' my-container
# 输出所有环境变量的 JSON 数组
```

#### `--env-file` — 从文件批量读取环境变量

当环境变量很多时，一个个写 `-e` 太麻烦。用文件来管理：

```bash
# 先创建一个 .env 文件
# 格式：每行一个 KEY=VALUE，# 开头是注释
```

`.env` 文件内容：

```
# 数据库配置
DB_HOST=localhost
DB_PORT=3306
DB_USER=admin
DB_PASSWORD=secret123
DB_NAME=mydb

# 应用配置
APP_ENV=production
APP_PORT=8000
DEBUG=false
```

```bash
# 使用 --env-file 加载
docker run -d --env-file .env my-app

# 可以同时用 --env-file 和 -e（-e 优先级更高）
docker run -d --env-file .env -e APP_ENV=development my-app
# APP_ENV 会被覆盖为 development
```

#### 常见镜像需要的环境变量

```bash
# MySQL
-e MYSQL_ROOT_PASSWORD=xxx      # 必须！root 密码
-e MYSQL_DATABASE=xxx           # 自动创建数据库
-e MYSQL_USER=xxx               # 创建用户
-e MYSQL_PASSWORD=xxx           # 用户密码

# PostgreSQL
-e POSTGRES_PASSWORD=xxx        # 必须！
-e POSTGRES_USER=xxx            # 默认 postgres
-e POSTGRES_DB=xxx              # 默认同用户名

# Redis
-e REDIS_PASSWORD=xxx           # 密码（可选）

# Nginx
# Nginx 一般不需要环境变量，用配置文件

# Node.js
-e NODE_ENV=production          # 运行环境
-e PORT=3000                    # 监听端口
```

---

### 2.8 `-w` — 设置工作目录

**是什么**：设置容器启动后的当前工作目录（当 `cd` 到指定目录）。

```bash
# 不加 -w：工作目录由镜像决定（通常是 / 或镜像 Dockerfile 里设置的）
docker run --rm ubuntu pwd
# 输出：/

# 加 -w：指定工作目录
docker run --rm -w /tmp ubuntu pwd
# 输出：/tmp

# 实际用途：在挂载的目录下执行命令
docker run --rm -v $(pwd):/app -w /app python:3.11-slim python my_script.py
#                                │
#                                └── 先 cd 到 /app，然后执行 python my_script.py
```

---

### 2.9 `-u` — 指定运行用户

**是什么**：指定容器内的进程以哪个用户身份运行。默认大多数容器以 root 运行。

**为什么需要**：安全考虑。root 权限太大了，如果容器被攻击，root 可能影响宿主机。

```bash
# 默认以 root 运行
docker run --rm ubuntu whoami
# 输出：root

# 指定用户 ID 和组 ID
docker run --rm -u 1000:1000 ubuntu id
# 输出：uid=1000 gid=1000 groups=1000

# 指定用户名（前提是容器内存在这个用户）
docker run --rm -u nobody ubuntu whoami
# 输出：nobody

# 进入已运行的容器时指定用户
docker exec -it -u root my-container bash    # 以 root 进入
docker exec -it -u 1000 my-container bash    # 以 uid=1000 的用户进入
```

---

### 2.10 `--restart` — 重启策略

**是什么**：告诉 Docker 在容器退出后要不要自动重启它。

**类比**：
- `no`：手机 App 崩了就崩了，你手动重新打开
- `always`：App 崩了手机自动帮你重新打开
- `unless-stopped`：App 崩了自动重新打开，但如果你手动关的就不开了

#### 四种策略详解

```bash
# 策略 1：no（默认）— 不自动重启
docker run -d --restart=no nginx
# 容器崩了就是崩了，要手动 docker start 重启

# 策略 2：always — 总是重启
docker run -d --restart=always nginx
# 容器退出了？自动重启
# Docker 守护进程重启了？自动重启
# 手动 docker stop 了？Docker 重启后又会自动启动
# 适合：绝对不能停的核心服务（但很"倔"，手动停也会被拉起来）

# 策略 3：unless-stopped — 除非手动停止（推荐！）
docker run -d --restart=unless-stopped nginx
# 容器崩了？自动重启
# Docker 守护进程重启了？自动重启
# 手动 docker stop 了？不会自动重启 ← 和 always 的唯一区别
# 适合：大多数服务的最佳选择

# 策略 4：on-failure[:最大重试次数] — 只在异常退出时重启
docker run -d --restart=on-failure:5 my-app
# 退出码 ≠ 0（即异常退出）时重启，最多重试 5 次
# 退出码 = 0（正常退出）不重启
# 适合：可能偶尔崩溃但不想无限重启的服务
```

#### 怎么选？

```
你的服务是什么？
├── 一次性任务（跑完就结束的） → no（默认）
├── 偶尔可能崩溃的服务 → on-failure:5
├── 需要持续运行的服务 → unless-stopped（推荐）
└── 绝对不能停的核心服务 → always
```

#### 修改已有容器的重启策略

```bash
docker update --restart=unless-stopped my-nginx
# 不用删了重新创建，直接改
```

---

### 2.11 `--network` — 指定网络

**是什么**：指定容器连接到哪个 Docker 网络。

```bash
# 默认：连到 bridge 网络（容器间不能用名字互相访问）
docker run -d nginx

# 创建自定义网络
docker network create my-net

# 连到自定义网络（容器间可以用名字互相访问！）
docker run -d --name web --network my-net nginx
docker run -d --name db --network my-net mysql:8.0
# 现在 web 容器里可以用 "db" 作为主机名连数据库

# 使用宿主机网络（容器不做网络隔离，直接用宿主机的端口）
docker run -d --network host nginx
# 不需要 -p 了，Nginx 直接监听宿主机的 80 端口

# 不要网络（完全隔离）
docker run -d --network none my-app
```

---

### 2.12 `--hostname` 和 `--dns` — 主机名和 DNS

```bash
# 设置容器的主机名（容器里执行 hostname 看到的名字）
docker run -it --hostname my-server ubuntu bash
root@my-server:/# hostname
# 输出：my-server

# 设置 DNS 服务器
docker run -it --dns 8.8.8.8 --dns 114.114.114.114 ubuntu bash
# 容器内解析域名时会使用这些 DNS 服务器
```

---

### 2.13 `--cpus` 和 `-m` — 资源限制

**是什么**：限制容器最多能用多少 CPU 和内存，防止某个容器吃光宿主机资源。

**类比**：自助餐厅里每人限取一盘，防止一个人把菜全拿走，别人没得吃。

#### CPU 限制

```bash
# 限制最多使用 1.5 个 CPU 核心
docker run -d --cpus=1.5 nginx
# 如果你的机器有 4 核，这个容器最多用 1.5 核

# 限制只能使用特定的 CPU 核心
docker run -d --cpuset-cpus="0,2" nginx
# 只能用第 0 号和第 2 号核心（从 0 开始编号）
# 适合：想把不同容器绑到不同核心，避免互相影响

# CPU 份额（相对权重）
docker run -d --cpu-shares=512 --name low-priority nginx
docker run -d --cpu-shares=2048 --name high-priority nginx
# 默认是 1024
# 当 CPU 紧张时，high-priority 分到的 CPU 时间是 low-priority 的 4 倍
# 当 CPU 不紧张时，两个容器都可以用满
# 注意：这不是"硬限制"，是"优先级"
```

**--cpus vs --cpu-shares 的区别**：

```
--cpus=1.5        → 硬限制：不管多忙，最多用 1.5 核
--cpu-shares=512  → 软限制：CPU 不忙时可以超额用，忙时按比例分配
```

#### 内存限制

```bash
# 限制最多使用 512MB 内存
docker run -d -m 512m nginx
# 或者
docker run -d --memory=512m nginx

# 常见写法
-m 256m     # 256 MB
-m 1g       # 1 GB
-m 2g       # 2 GB

# 限制内存 + swap（交换空间）
docker run -d -m 512m --memory-swap=1g nginx
# 内存上限 512m
# swap 上限 = 1g - 512m = 512m
# 总共可用 1g

# 如果设 --memory-swap 和 --memory 相同，表示禁用 swap
docker run -d -m 512m --memory-swap=512m nginx
# 只能用 512m 内存，不能用 swap

# 超过内存限制会怎样？
# 容器会被 OOM Killer（Out Of Memory Killer）直接杀掉！
# docker inspect 可以看到 OOMKilled: true
```

#### GPU 分配

```bash
# 使用所有 GPU
docker run --gpus all nvidia/cuda:12.0-base nvidia-smi

# 使用 2 个 GPU
docker run --gpus 2 nvidia/cuda:12.0-base nvidia-smi

# 使用指定的 GPU（按 ID）
docker run --gpus '"device=0,1"' nvidia/cuda:12.0-base nvidia-smi
```

#### 查看资源限制是否生效

```bash
# 启动一个有资源限制的容器
docker run -d --name limited --cpus=1.5 -m 512m nginx

# 查看资源使用情况
docker stats limited
# MEM USAGE / LIMIT 会显示 xxx / 512MiB

# 查看具体限制设置
docker inspect --format='CPU: {{.HostConfig.NanoCpus}}, Memory: {{.HostConfig.Memory}}' limited
```

#### 修改已运行容器的资源限制

```bash
docker update --cpus=2 --memory=1g my-nginx
# 不需要停止容器，立即生效
```

---

### 2.14 `--privileged` — 特权模式（危险！）

**是什么**：给容器几乎和宿主机一样的权限，可以访问所有设备、修改内核参数等。

**类比**：普通容器像是住在酒店，只能用自己房间里的东西。`--privileged` 像是拿到了酒店管理员的万能钥匙，所有房间都能进。

```bash
# 普通模式（安全）
docker run --rm ubuntu mount /dev/sda1 /mnt
# 报错！没有权限

# 特权模式（危险但有时必需）
docker run --rm --privileged ubuntu mount /dev/sda1 /mnt
# 可以了，但也意味着容器可以对宿主机做任何事
```

**什么时候不得不用**：
- Docker-in-Docker（容器里运行 Docker）
- 需要访问硬件设备
- 修改网络配置（iptables）
- 修改内核参数

**更安全的替代方案**：用 `--cap-add` 只添加需要的特定权限，而不是给所有权限。

```bash
# 只添加网络管理权限（比 --privileged 安全得多）
docker run --cap-add=NET_ADMIN my-app
```

---

### 2.15 `--tmpfs` — 内存临时文件系统

**是什么**：在容器内创建一个基于内存的临时文件系统，读写速度极快，容器停止后数据消失。

```bash
docker run -d --tmpfs /tmp:size=100m my-app
# /tmp 目录使用内存存储，最大 100MB
# 适合存临时文件、缓存等不需要持久化的数据
```

---

## 三、常见运行场景示例（完整实战）

### 场景 1：运行 Nginx Web 服务器

```bash
# 最简版
docker run -d --name my-nginx -p 80:80 nginx

# 完整生产版
docker run -d \
    --name my-nginx \
    -p 80:80 \
    -p 443:443 \
    -v /path/to/html:/usr/share/nginx/html:ro \
    -v /path/to/nginx.conf:/etc/nginx/nginx.conf:ro \
    -v nginx-logs:/var/log/nginx \
    --restart=unless-stopped \
    --cpus=2 \
    -m 512m \
    nginx:1.25.3
```

**逐行解释**：

```
-d                                → 后台运行
--name my-nginx                   → 容器叫 my-nginx
-p 80:80                          → HTTP 端口映射
-p 443:443                        → HTTPS 端口映射
-v ...html:...html:ro             → 网站文件挂载（只读，容器不能改）
-v ...nginx.conf:...nginx.conf:ro → 配置文件挂载（只读）
-v nginx-logs:/var/log/nginx      → 日志用命名卷持久化
--restart=unless-stopped          → 崩了自动重启
--cpus=2                          → 最多用 2 个 CPU 核心
-m 512m                           → 最多用 512MB 内存
nginx:1.25.3                      → 指定精确版本的镜像
```

### 场景 2：运行 MySQL 数据库

```bash
docker run -d \
    --name my-mysql \
    -p 3306:3306 \
    -e MYSQL_ROOT_PASSWORD=my-secret-pw \
    -e MYSQL_DATABASE=mydb \
    -e MYSQL_USER=admin \
    -e MYSQL_PASSWORD=admin123 \
    -v mysql-data:/var/lib/mysql \
    -v /path/to/my.cnf:/etc/mysql/conf.d/my.cnf:ro \
    --restart=unless-stopped \
    --cpus=4 \
    -m 2g \
    mysql:8.0
```

**逐行解释**：

```
-p 3306:3306                        → MySQL 端口映射
-e MYSQL_ROOT_PASSWORD=my-secret-pw → 设置 root 密码（必须！）
-e MYSQL_DATABASE=mydb              → 启动时自动创建 mydb 数据库
-e MYSQL_USER=admin                 → 创建一个叫 admin 的用户
-e MYSQL_PASSWORD=admin123          → admin 的密码
-v mysql-data:/var/lib/mysql        → 数据持久化！删容器数据不丢
-v .../my.cnf:...my.cnf:ro         → 自定义 MySQL 配置
-m 2g                               → 数据库建议给足够内存
```

### 场景 3：运行一次性任务

```bash
# 用 Python 容器跑个脚本
docker run --rm \
    -v $(pwd):/app \
    -w /app \
    python:3.11-slim \
    python my_script.py
```

**逐行解释**：

```
--rm                → 跑完自动删除容器，不留垃圾
-v $(pwd):/app      → 把当前目录挂载到容器的 /app
-w /app             → 容器启动后 cd 到 /app
python:3.11-slim    → 使用 Python 镜像
python my_script.py → 在容器里执行这个命令
```

**理解 `-v $(pwd):/app -w /app` 这个组合**：

```
你的电脑                        容器内部
/home/user/project/             /app/
├── my_script.py     ─────▶    ├── my_script.py
├── data.csv         ─────▶    ├── data.csv
└── output/          ─────▶    └── output/

-v $(pwd):/app  → 让容器的 /app 目录和你的项目目录同步
-w /app         → 容器启动后自动 cd 到 /app
效果：容器里执行 python my_script.py 就像在你的项目目录里执行一样
```

### 场景 4：交互式调试

```bash
# 启动一个 Ubuntu 容器，进去敲命令
docker run -it --rm ubuntu:22.04 bash

# 启动一个有网络工具的容器排查网络问题
docker run -it --rm --network host nicolaka/netshoot bash
```

---

## 四、容器管理操作

### 4.1 查看容器 — docker ps

```bash
# 查看正在运行的容器
docker ps

# 查看所有容器（包括已停止的）
docker ps -a

# 只显示容器 ID（配合其他命令使用）
docker ps -q            # 运行中的 ID
docker ps -aq           # 所有容器的 ID

# 查看最近创建的 5 个容器
docker ps -n 5

# 按条件过滤
docker ps -f "status=exited"        # 只看已退出的
docker ps -f "status=running"       # 只看运行中的
docker ps -f "name=nginx"           # 名字包含 nginx 的
docker ps -f "ancestor=mysql:8.0"   # 基于 mysql:8.0 镜像的

# 自定义输出格式
docker ps --format "table {{.ID}}\t{{.Names}}\t{{.Status}}\t{{.Ports}}"

# 显示容器大小
docker ps -s
```

**输出字段详解**：

```
CONTAINER ID   IMAGE     COMMAND                  CREATED         STATUS         PORTS                  NAMES
a1b2c3d4e5f6   nginx     "/docker-entrypoint.…"   5 minutes ago   Up 5 minutes   0.0.0.0:80->80/tcp     my-nginx
│              │         │                        │               │              │                      │
│              │         │                        │               │              │                      └── 容器名
│              │         │                        │               │              └── 端口映射关系
│              │         │                        │               └── 运行状态和运行时长
│              │         │                        └── 创建时间
│              │         └── 容器启动时执行的命令
│              └── 使用的镜像
└── 容器 ID（12 位短格式，完整是 64 位哈希值）
```

**STATUS 常见状态**：

```
Up 5 minutes           → 运行了 5 分钟
Up 2 hours (healthy)   → 运行 2 小时且健康检查通过
Up 1 hour (unhealthy)  → 运行 1 小时但健康检查失败
Exited (0) 3 hours ago → 3 小时前正常退出（退出码 0）
Exited (1) 1 minute ago → 1 分钟前异常退出（退出码 1）
Exited (137) 2 min ago → 2 分钟前被强制杀掉（137 = 128 + 9，即收到 SIGKILL）
Created                → 已创建但未启动
Paused                 → 暂停中
```

**常见退出码含义**：

| 退出码 | 含义 |
|--------|------|
| 0 | 正常退出 |
| 1 | 程序报错退出 |
| 126 | 命令不可执行 |
| 127 | 命令找不到 |
| 137 | 被 SIGKILL 杀掉（`docker kill` 或 OOM） |
| 139 | 段错误（SIGSEGV） |
| 143 | 被 SIGTERM 终止（`docker stop`） |

---

### 4.2 启动 / 停止 / 重启 — 详细解析

#### docker stop — 优雅停止

```bash
docker stop my-nginx
```

**stop 的内部流程**：

```
1. Docker 向容器的主进程发送 SIGTERM 信号
   → "嘿，请你自己优雅地退出吧"
   → 主进程可以做清理工作（关闭数据库连接、保存数据等）

2. 等待 10 秒（默认超时时间）

3. 如果 10 秒内进程没退出，Docker 发送 SIGKILL 信号
   → "不管了，强制杀掉"
   → 进程立即终止，没有清理的机会
```

```bash
# 修改等待时间
docker stop -t 30 my-nginx     # 等 30 秒再强杀
docker stop -t 0 my-nginx      # 立即强杀（等价于 docker kill）
```

#### docker kill — 强制停止

```bash
docker kill my-nginx
```

直接发送 SIGKILL，进程立即终止，没有清理的机会。

```bash
# 发送其他信号
docker kill --signal=SIGHUP my-nginx    # 发送 SIGHUP（有些程序用来重新加载配置）
docker kill --signal=SIGUSR1 my-nginx   # 发送自定义信号
```

**stop vs kill**：

```
docker stop = 先敲门说"请出去"（SIGTERM），等一会，不走就强拖（SIGKILL）
docker kill = 直接破门而入强拖（SIGKILL）

推荐：除非 stop 无效，否则一律用 stop
```

#### docker restart — 重启

```bash
docker restart my-nginx
# 等价于 docker stop + docker start

docker restart -t 5 my-nginx
# 等 5 秒后强杀再启动
```

#### docker pause / unpause — 暂停和恢复

```bash
# 暂停：冻结容器的所有进程，CPU 使用变为 0，但内存保留
docker pause my-nginx

docker ps
# STATUS: Up 10 minutes (Paused)

# 恢复：解冻进程，继续运行
docker unpause my-nginx
```

**pause vs stop 的区别**：

| 特性 | pause | stop |
|------|-------|------|
| 进程状态 | 冻结（还在内存中） | 完全终止 |
| 内存占用 | 保留 | 释放 |
| 恢复速度 | 瞬间恢复 | 需要重新启动进程 |
| 数据库安全 | 安全（进程没变） | 取决于数据库的关闭流程 |
| 适用场景 | 临时释放 CPU | 长期不用 |

---

### 4.3 进入容器 — docker exec 详解

#### 基本用法

```bash
# 进入容器的交互式终端
docker exec -it my-nginx bash

# 解析
# exec     → 在运行中的容器里执行命令
# -i       → 保持输入打开
# -t       → 分配伪终端
# my-nginx → 容器名
# bash     → 要执行的命令（进入 bash shell）
```

#### 容器里没有 bash 怎么办？

```bash
# 有些精简镜像（如 alpine）没有 bash
docker exec -it my-alpine bash
# 报错：OCI runtime exec failed: exec failed: unable to start container process: exec: "bash": executable file not found in $PATH

# 解决：用 sh（几乎所有 Linux 都有 sh）
docker exec -it my-alpine sh
```

#### 执行单条命令（不进入交互模式）

```bash
# 查看文件
docker exec my-nginx cat /etc/nginx/nginx.conf

# 列出目录
docker exec my-nginx ls -la /var/log/nginx/

# 查看进程
docker exec my-nginx ps aux

# 查看网络
docker exec my-nginx ip addr

# 安装工具（调试用）
docker exec my-nginx apt-get update && docker exec my-nginx apt-get install -y curl
docker exec my-nginx curl http://localhost:80
```

#### 以特定用户进入

```bash
# 以 root 用户进入（有些容器默认是非 root）
docker exec -it -u root my-container bash

# 以特定 UID 进入
docker exec -it -u 1000 my-container bash
```

#### 设置环境变量

```bash
# 执行命令时传入临时环境变量
docker exec -e MY_VAR=hello my-container env | grep MY_VAR
# 输出：MY_VAR=hello
```

#### 在特定目录下执行

```bash
# 在指定工作目录执行命令
docker exec -w /var/log my-container ls -la
```

#### exec vs attach 的核心区别

```bash
# ===== exec =====
docker exec -it my-nginx bash
# 在容器里开了一个新进程（新的 bash）
# Ctrl+C 只会终止这个新 bash，容器主进程不受影响
# Ctrl+D 或 exit 退出，容器继续运行
# 推荐！日常使用都用 exec

# ===== attach =====
docker attach my-nginx
# 直接连接到容器的主进程（PID 1）
# 你看到的是主进程的输入输出
# ⚠️ Ctrl+C 会发送 SIGINT 给主进程 → 可能导致容器停止！
# 安全退出方式：按 Ctrl+P 然后 Ctrl+Q（detach 快捷键）
```

| 场景 | 用 exec 还是 attach |
|------|-------------------|
| 进容器调试 | exec |
| 安装工具/查看文件 | exec |
| 看主进程的实时输出 | 用 `docker logs -f` 更安全 |
| 和主进程交互 | attach（小心） |

---

### 4.4 删除容器

```bash
# 删除已停止的容器
docker rm my-nginx

# 删除运行中的容器（需要加 -f）
docker rm -f my-nginx
# 等价于先 docker kill 再 docker rm

# 删除容器同时删除匿名卷
docker rm -v my-nginx

# 删除所有已停止的容器
docker container prune
# 会提示确认，输入 y

# 强制删除所有已停止的容器（不用确认）
docker container prune -f

# 批量删除所有已退出的容器
docker rm $(docker ps -aq -f "status=exited")

# 删除所有容器（包括运行中的，慎用！）
docker rm -f $(docker ps -aq)
```

---

## 五、容器日志与监控

### 5.1 查看日志 — docker logs

```bash
# 查看全部日志（可能很长！）
docker logs my-nginx

# 实时跟踪（像 tail -f）— 最常用的方式
docker logs -f my-nginx
# 按 Ctrl+C 停止跟踪（不会影响容器）

# 只看最后 N 行
docker logs --tail 100 my-nginx     # 最后 100 行
docker logs --tail 0 -f my-nginx   # 只看新产生的日志

# 带时间戳
docker logs -t my-nginx
# 输出：2024-01-15T08:30:00.123456789Z 127.0.0.1 - - [15/Jan/2024...

# 时间范围
docker logs --since 30m my-nginx       # 最近 30 分钟的日志
docker logs --since 2h my-nginx        # 最近 2 小时
docker logs --since "2024-01-15" my-nginx  # 指定日期之后
docker logs --until 10m my-nginx       # 10 分钟之前的日志

# 组合使用（最实用的写法）
docker logs -f --tail 50 -t my-nginx
# 显示最近 50 行 + 带时间戳 + 实时跟踪新日志
```

### 5.2 查看资源使用 — docker stats

```bash
# 实时查看所有容器的资源（每秒刷新）
docker stats

# 只看特定容器
docker stats my-nginx my-mysql

# 看一次就退出（不持续刷新），适合脚本使用
docker stats --no-stream

# 自定义格式
docker stats --format "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.MemPerc}}\t{{.NetIO}}"
```

**输出字段详解**：

```
CONTAINER ID   NAME       CPU %     MEM USAGE / LIMIT     MEM %     NET I/O           BLOCK I/O
a1b2c3d4e5f6   my-nginx   0.05%     4.5MiB / 7.765GiB    0.06%     1.45kB / 0B       0B / 0B
                           │         │         │           │         │                  │
                           │         │         │           │         │                  └── 磁盘读/写
                           │         │         │           │         └── 网络 入/出
                           │         │         │           └── 内存使用百分比
                           │         │         └── 内存上限（-m 设置的，或宿主机总内存）
                           │         └── 当前实际使用的内存
                           └── CPU 使用百分比（100% = 1 核）
```

**看到 CPU 300% 不要慌**：表示用了 3 个核心。如果你的机器有 8 核，最大可以到 800%。

### 5.3 查看容器详细信息 — docker inspect

```bash
# 查看容器完整信息（超长 JSON）
docker inspect my-nginx

# 实用的信息提取（用 --format 过滤）

# 查看 IP 地址
docker inspect --format='{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' my-nginx
# 输出：172.17.0.2

# 查看端口映射
docker inspect --format='{{json .NetworkSettings.Ports}}' my-nginx
# 输出：{"80/tcp":[{"HostIp":"0.0.0.0","HostPort":"8080"}]}

# 查看挂载信息
docker inspect --format='{{json .Mounts}}' my-nginx | python -m json.tool
# 美化 JSON 输出

# 查看启动命令
docker inspect --format='{{json .Config.Cmd}}' my-nginx

# 查看环境变量
docker inspect --format='{{json .Config.Env}}' my-nginx

# 查看重启策略
docker inspect --format='{{.HostConfig.RestartPolicy.Name}}' my-nginx

# 查看容器状态和退出码
docker inspect --format='状态:{{.State.Status}} 退出码:{{.State.ExitCode}}' my-nginx

# 查看容器启动时间
docker inspect --format='{{.State.StartedAt}}' my-nginx

# 查看是否被 OOM 杀掉
docker inspect --format='{{.State.OOMKilled}}' my-nginx
# false 表示没有，true 表示内存不够被杀了
```

### 5.4 查看容器进程

```bash
docker top my-nginx

# 输出类似 Linux 的 ps 命令
UID    PID    PPID   C   STIME   TTY   TIME       CMD
root   1234   1233   0   08:30   ?     00:00:00   nginx: master process
nginx  1235   1234   0   08:30   ?     00:00:02   nginx: worker process
```

### 5.5 查看容器文件变更

```bash
docker diff my-nginx

# 输出示例
A /var/log/nginx/access.log     # A = Added（新增）
A /var/log/nginx/error.log
C /var                           # C = Changed（修改）
C /var/log
C /var/log/nginx
C /run                           # D = Deleted（删除，这里没有）
```

---

## 六、容器与宿主机之间的文件传输

```bash
# ===== 从容器复制到宿主机 =====
docker cp my-nginx:/etc/nginx/nginx.conf ./nginx.conf
# 把容器里的 nginx.conf 复制到你当前目录

docker cp my-nginx:/var/log/nginx/ ./nginx-logs/
# 把容器里的 nginx 日志目录整个复制出来

# ===== 从宿主机复制到容器 =====
docker cp ./index.html my-nginx:/usr/share/nginx/html/
# 把你的 index.html 放进容器

docker cp ./config/ my-nginx:/app/config/
# 整个目录复制进去
```

**注意**：
- `docker cp` 不需要容器在运行中，已停止的容器也能复制
- 目标路径如果不存在，Docker 会自动创建
- 如果你要频繁同步文件，用 `-v` 挂载比 `docker cp` 方便得多

---

## 七、容器资源限制实战

### 7.1 验证内存限制

```bash
# 启动一个限制 100MB 内存的容器
docker run -d --name mem-test -m 100m python:3.11-slim sleep infinity

# 查看限制是否生效
docker stats mem-test --no-stream
# MEM USAGE / LIMIT 会显示 xxx / 100MiB

# 在容器里尝试吃内存
docker exec -it mem-test python -c "
data = []
try:
    while True:
        data.append('x' * 1024 * 1024)  # 每次吃 1MB
except MemoryError:
    print(f'分配了 {len(data)} MB 后被限制了')
"

# 如果超过限制，容器可能被 OOM 杀掉
docker inspect --format='{{.State.OOMKilled}}' mem-test
```

### 7.2 验证 CPU 限制

```bash
# 限制只能用 0.5 个 CPU 核心
docker run -d --name cpu-test --cpus=0.5 ubuntu:22.04 sh -c "while true; do :; done"

# 观察 CPU 使用率
docker stats cpu-test --no-stream
# CPU % 应该稳定在 50% 左右（因为只能用 0.5 核）

# 清理
docker rm -f cpu-test
```

---

## 八、容器导出为镜像

```bash
# 将容器当前状态保存为新镜像
docker commit my-nginx my-custom-nginx:v1

# 附带提交信息和作者
docker commit \
    -m "添加了自定义配置和静态文件" \
    -a "yourname" \
    my-nginx my-custom-nginx:v1

# 查看新镜像
docker images my-custom-nginx
```

> **重要提醒**：`docker commit` 能用但**不推荐用于生产**。原因：
> - 不可追溯：别人不知道你在容器里改了什么
> - 不可重现：换台机器就重现不了
> - 镜像膨胀：每次 commit 都增加一层
> - 正确做法：把所有变更写进 Dockerfile，用 `docker build` 构建

---

## 九、完整命令速查表

### 容器生命周期

| 操作 | 命令 | 说明 |
|------|------|------|
| 创建+运行 | `docker run -d --name web nginx` | 最常用 |
| 仅创建 | `docker create --name web nginx` | 创建但不启动 |
| 启动 | `docker start web` | 启动已停止的容器 |
| 停止 | `docker stop web` | 优雅停止（SIGTERM → SIGKILL） |
| 强杀 | `docker kill web` | 直接 SIGKILL |
| 重启 | `docker restart web` | stop + start |
| 暂停 | `docker pause web` | 冻结进程 |
| 恢复 | `docker unpause web` | 解冻进程 |
| 删除 | `docker rm web` | 删除已停止的容器 |
| 强制删除 | `docker rm -f web` | 可删运行中的 |
| 批量清理 | `docker container prune` | 删所有已停止的 |

### 容器信息

| 操作 | 命令 |
|------|------|
| 查看运行中 | `docker ps` |
| 查看全部 | `docker ps -a` |
| 查看日志 | `docker logs -f --tail 100 web` |
| 查看资源 | `docker stats` |
| 查看详情 | `docker inspect web` |
| 查看进程 | `docker top web` |
| 查看变更 | `docker diff web` |
| 查看端口 | `docker port web` |

### 容器操作

| 操作 | 命令 |
|------|------|
| 进入容器 | `docker exec -it web bash` |
| 执行命令 | `docker exec web ls /app` |
| 文件复制 | `docker cp web:/path ./local` |
| 改资源限制 | `docker update --cpus=2 -m 1g web` |
| 改重启策略 | `docker update --restart=always web` |
| 导出为镜像 | `docker commit web myimage:v1` |

---

## 十、本章小结

```
✅ 容器 = 一个被隔离的进程，主进程退出 = 容器退出
✅ -d 后台运行，-it 交互运行，--rm 用完即删
✅ --name 给容器取名，方便后续管理
✅ -p 8080:80 把宿主机 8080 映射到容器 80，让外部能访问
✅ -v 挂载数据，命名卷（my-data:/path）或绑定挂载（/host:/container）
✅ -e 传环境变量，--env-file 从文件批量读取
✅ -w 设工作目录，-u 设运行用户
✅ --restart=unless-stopped 是大多数服务的最佳重启策略
✅ --cpus 限 CPU（硬限制），-m 限内存（超了被 OOM 杀掉）
✅ docker exec -it 进容器，比 attach 安全得多
✅ docker logs -f --tail 50 实时看日志
✅ docker stats 实时看资源使用
✅ docker inspect 查看一切详细信息
✅ docker stop 优雅停止，docker kill 强制杀掉
✅ 不推荐 docker commit，应该用 Dockerfile
```

---

> 下一篇：[05-Dockerfile从入门到精通](./05-Dockerfile从入门到精通.md) — 学会自己构建镜像！
