# 09 — nerdctl 完全指南与 containerd 生态

> Docker 不再是唯一的选择。了解 containerd 和 nerdctl，跟上云原生时代的步伐。

---

## 一、从 Docker 到 containerd：行业变革

### 1.1 为什么 Docker 不是终点？

Docker 是伟大的，但它有一些"历史包袱"：

```
Docker 的架构链路：
  docker CLI → dockerd（Docker Daemon）→ containerd → runc → 容器

问题：
  - dockerd 太重了，既管镜像又管网络又管存储
  - Kubernetes 需要的只是"创建和管理容器"，不需要 dockerd 的全部功能
  - Kubernetes 要通过 dockershim 这个"垫片"才能和 Docker 对话
```

### 1.2 Kubernetes 的选择

```
Kubernetes 1.24 之前：
  kubelet → dockershim → dockerd → containerd → runc
  （四层调用链，效率低）

Kubernetes 1.24 之后（2022年）：
  kubelet → CRI → containerd → runc
  （直接对话 containerd，更高效）

结论：K8s 节点上不再需要 Docker，但 containerd 必须有
```

### 1.3 Docker、containerd、nerdctl 的关系

```
┌──────────────────────────────────────────┐
│              用户层（命令行）               │
│                                          │
│  ┌──────────┐  ┌──────────┐  ┌────────┐ │
│  │  docker   │  │  nerdctl  │  │ crictl │ │
│  │  CLI      │  │  CLI      │  │ CLI    │ │
│  └─────┬────┘  └─────┬────┘  └───┬────┘ │
│        │              │           │      │
│        ▼              │           │      │
│  ┌──────────┐         │           │      │
│  │  dockerd  │         │           │      │
│  │(Docker守护)│        │           │      │
│  └─────┬────┘         │           │      │
│        │              │           │      │
│        ▼              ▼           ▼      │
│  ┌───────────────────────────────────┐   │
│  │         containerd                │   │
│  │    （容器运行时，核心组件）          │   │
│  └────────────────┬──────────────────┘   │
│                   │                      │
│                   ▼                      │
│  ┌───────────────────────────────────┐   │
│  │            runc                    │   │
│  │     （OCI 运行时，最底层）          │   │
│  └───────────────────────────────────┘   │
└──────────────────────────────────────────┘
```

| 工具 | 定位 | 说明 |
|------|------|------|
| **docker** | Docker 生态的 CLI | 通过 dockerd → containerd 操作 |
| **nerdctl** | containerd 的 CLI | 直接操作 containerd，绕过 dockerd |
| **crictl** | CRI 调试工具 | 专为 K8s 环境设计 |
| **ctr** | containerd 原生 CLI | 底层工具，命令不友好 |

---

## 二、containerd 深入理解

### 2.1 什么是 containerd？

containerd 是一个**工业级的容器运行时**，负责：
- 拉取和存储镜像
- 创建和管理容器
- 管理容器的网络和存储（通过插件）
- 管理容器的生命周期

**一句话**：containerd 就是 Docker 里真正干活的那个组件，被单独提取出来了。

### 2.2 containerd 架构

```
┌────────────────────────────────────────┐
│              containerd                 │
│                                        │
│  ┌────────────┐  ┌────────────────┐    │
│  │ Content    │  │   Snapshotter  │    │
│  │ Store      │  │ (存储快照管理)   │    │
│  │ (内容存储)  │  │                │    │
│  └────────────┘  └────────────────┘    │
│                                        │
│  ┌────────────┐  ┌────────────────┐    │
│  │ Images     │  │   Containers   │    │
│  │ (镜像管理)  │  │  (容器管理)     │    │
│  └────────────┘  └────────────────┘    │
│                                        │
│  ┌────────────┐  ┌────────────────┐    │
│  │ Tasks      │  │   Namespaces   │    │
│  │ (进程管理)  │  │  (命名空间)     │    │
│  └────────────┘  └────────────────┘    │
│                                        │
│  ┌────────────────────────────────┐    │
│  │        Plugins (CNI, etc.)     │    │
│  └────────────────────────────────┘    │
└────────────────────────────────────────┘
```

### 2.3 containerd 的 Namespace

containerd 有自己的 namespace 概念（不是 Linux Namespace），用来隔离不同工具的资源：

```
┌──────────────────────────────────────┐
│            containerd                 │
│                                      │
│  ┌─────────────┐  ┌──────────────┐  │
│  │ Namespace:   │  │ Namespace:   │  │
│  │ "moby"      │  │ "k8s.io"    │  │
│  │             │  │             │  │
│  │ Docker 的    │  │ Kubernetes  │  │
│  │ 容器和镜像   │  │ 的容器和镜像 │  │
│  └─────────────┘  └──────────────┘  │
│                                      │
│  ┌─────────────┐                    │
│  │ Namespace:   │                    │
│  │ "default"   │                    │
│  │             │                    │
│  │ nerdctl/ctr │                    │
│  │ 的默认空间   │                    │
│  └─────────────┘                    │
└──────────────────────────────────────┘
```

```bash
# ctr 查看不同 namespace 下的容器
ctr -n moby containers list     # Docker 创建的容器
ctr -n k8s.io containers list   # Kubernetes 创建的容器
ctr -n default containers list  # ctr/nerdctl 默认的容器
```

### 2.4 安装 containerd

#### 独立安装（不通过 Docker）

```bash
# Ubuntu / Debian
sudo apt-get update
sudo apt-get install -y containerd.io

# 生成默认配置
sudo mkdir -p /etc/containerd
containerd config default | sudo tee /etc/containerd/config.toml

# 启动
sudo systemctl enable containerd
sudo systemctl start containerd

# 验证
sudo ctr version
```

#### 已有 Docker 的情况

如果你已经安装了 Docker，containerd 已经在运行了：

```bash
systemctl status containerd
# Active: active (running)
```

---

## 三、nerdctl — containerd 的最佳搭档

### 3.1 什么是 nerdctl？

nerdctl 是 containerd 的命令行客户端，设计目标是**和 docker CLI 完全兼容**。

```
docker run -d -p 80:80 nginx     ← Docker 命令
nerdctl run -d -p 80:80 nginx    ← nerdctl 命令（一模一样！）
```

### 3.2 安装 nerdctl

#### 方式一：nerdctl-full（推荐新手）

nerdctl-full 包含 nerdctl + containerd + CNI 插件 + BuildKit，开箱即用。

```bash
# 下载最新版（以 linux-amd64 为例）
wget https://github.com/containerd/nerdctl/releases/download/v2.0.0/nerdctl-full-2.0.0-linux-amd64.tar.gz

# 解压到系统目录
sudo tar xzf nerdctl-full-2.0.0-linux-amd64.tar.gz -C /usr/local

# 启动相关服务
sudo systemctl enable --now containerd
sudo systemctl enable --now buildkit

# 验证
nerdctl version
```

#### 方式二：只安装 nerdctl（已有 containerd）

```bash
# 只下载 nerdctl 二进制文件
wget https://github.com/containerd/nerdctl/releases/download/v2.0.0/nerdctl-2.0.0-linux-amd64.tar.gz

# 解压
sudo tar xzf nerdctl-2.0.0-linux-amd64.tar.gz -C /usr/local/bin

# 验证
nerdctl version
```

#### 方式三：macOS（通过 Homebrew）

```bash
brew install nerdctl
```

### 3.3 rootless 模式（免 sudo）

nerdctl 原生支持 rootless 模式，更安全：

```bash
# 安装 rootless containerd
containerd-rootless-setuptool.sh install

# 安装 rootless BuildKit
containerd-rootless-setuptool.sh install-buildkit

# 之后就可以不用 sudo 了
nerdctl run -d -p 8080:80 nginx
```

---

## 四、nerdctl 命令全解 — 和 Docker 的对照表

### 4.1 镜像操作

| 操作 | Docker | nerdctl |
|------|--------|---------|
| 拉取镜像 | `docker pull nginx` | `nerdctl pull nginx` |
| 查看镜像 | `docker images` | `nerdctl images` |
| 构建镜像 | `docker build -t app .` | `nerdctl build -t app .` |
| 打标签 | `docker tag a b` | `nerdctl tag a b` |
| 推送镜像 | `docker push img` | `nerdctl push img` |
| 删除镜像 | `docker rmi img` | `nerdctl rmi img` |
| 导出镜像 | `docker save -o f.tar img` | `nerdctl save -o f.tar img` |
| 导入镜像 | `docker load -i f.tar` | `nerdctl load -i f.tar` |

```bash
# 拉取镜像
nerdctl pull nginx:1.25.3

# 查看镜像
nerdctl images

# 构建镜像（需要 BuildKit）
nerdctl build -t my-app:v1 .

# 查看镜像详情
nerdctl image inspect nginx
```

### 4.2 容器操作

| 操作 | Docker | nerdctl |
|------|--------|---------|
| 运行容器 | `docker run -d nginx` | `nerdctl run -d nginx` |
| 查看容器 | `docker ps` | `nerdctl ps` |
| 停止容器 | `docker stop web` | `nerdctl stop web` |
| 启动容器 | `docker start web` | `nerdctl start web` |
| 删除容器 | `docker rm web` | `nerdctl rm web` |
| 进入容器 | `docker exec -it web sh` | `nerdctl exec -it web sh` |
| 查看日志 | `docker logs web` | `nerdctl logs web` |
| 文件复制 | `docker cp web:/f .` | `nerdctl cp web:/f .` |

```bash
# 运行容器
nerdctl run -d --name web -p 80:80 nginx

# 查看容器
nerdctl ps -a

# 进入容器
nerdctl exec -it web bash

# 查看日志
nerdctl logs -f web

# 停止和删除
nerdctl stop web
nerdctl rm web
```

### 4.3 网络操作

```bash
# 创建网络
nerdctl network create my-net

# 查看网络
nerdctl network ls

# 运行时指定网络
nerdctl run -d --name web --network my-net nginx

# 删除网络
nerdctl network rm my-net
```

### 4.4 Volume 操作

```bash
# 创建 Volume
nerdctl volume create my-data

# 查看 Volume
nerdctl volume ls

# 使用 Volume
nerdctl run -d -v my-data:/data alpine

# 删除 Volume
nerdctl volume rm my-data
```

### 4.5 系统命令

```bash
# 查看系统信息
nerdctl system info

# 磁盘使用
nerdctl system df

# 清理未使用资源
nerdctl system prune
nerdctl system prune -a --volumes
```

---

## 五、nerdctl 的独有功能

### 5.1 Docker Compose 兼容

nerdctl 内置了 Compose 支持：

```bash
# 和 docker compose 一样的用法！
nerdctl compose up -d
nerdctl compose down
nerdctl compose ps
nerdctl compose logs -f
```

### 5.2 镜像加密

nerdctl 支持对镜像层进行加密（Docker 不支持）：

```bash
# 使用 JWE 加密推送镜像
nerdctl image encrypt --recipient jwe:mypubkey.pem \
    --platform linux/amd64 \
    my-app:v1 registry.example.com/my-app:v1-enc

nerdctl push registry.example.com/my-app:v1-enc
```

### 5.3 Lazy Pulling（惰性拉取）

传统方式：拉取整个镜像才能启动容器。
Lazy Pulling：只拉取启动需要的部分，边运行边拉。

```bash
# 使用 eStargz 格式的惰性拉取
nerdctl --snapshotter=stargz run -d ghcr.io/stargz-containers/nginx:1.25-esgz
# 容器启动速度可以快几十倍！
```

### 5.4 镜像签名验证

```bash
# 使用 cosign 验证镜像签名
nerdctl pull --verify=cosign \
    --cosign-key cosign.pub \
    registry.example.com/my-app:v1
```

### 5.5 Namespace 管理

```bash
# 指定 namespace
nerdctl --namespace k8s.io ps    # 查看 K8s 的容器
nerdctl --namespace moby ps      # 查看 Docker 的容器

# 简写
nerdctl -n k8s.io images
```

### 5.6 IPFS 集成

nerdctl 支持从 IPFS 网络拉取镜像：

```bash
nerdctl pull ipfs://bafybeicq7...
```

---

## 六、ctr vs crictl vs nerdctl

### 6.1 三兄弟对比

| 特性 | ctr | crictl | nerdctl |
|------|-----|--------|---------|
| 定位 | containerd 原生工具 | CRI 调试工具 | Docker 兼容 CLI |
| 易用性 | ⭐（难用） | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ |
| Docker 兼容 | ❌ | 部分 | ✅ 完全兼容 |
| Compose 支持 | ❌ | ❌ | ✅ |
| Build 支持 | ❌ | ❌ | ✅（BuildKit） |
| 适用场景 | 底层调试 | K8s 节点排查 | 日常使用 |

### 6.2 ctr 常用命令（了解即可）

```bash
# 拉取镜像（注意：需要完整路径）
sudo ctr images pull docker.io/library/nginx:latest

# 查看镜像
sudo ctr images ls

# 运行容器（语法和 docker 差别很大）
sudo ctr run -d docker.io/library/nginx:latest my-nginx

# 查看容器
sudo ctr containers ls

# 查看运行中的任务
sudo ctr tasks ls

# 进入容器
sudo ctr tasks exec --exec-id 0 -t my-nginx sh

# 删除
sudo ctr tasks kill my-nginx
sudo ctr containers rm my-nginx
```

### 6.3 crictl 常用命令（K8s 环境）

```bash
# 查看容器
crictl ps
crictl ps -a

# 查看 Pod
crictl pods

# 查看镜像
crictl images

# 查看日志
crictl logs <container-id>

# 进入容器
crictl exec -it <container-id> sh

# 查看容器详情
crictl inspect <container-id>
```

---

## 七、从 Docker 迁移到 nerdctl

### 7.1 最简迁移方案

```bash
# 设置别名，无感切换
echo 'alias docker=nerdctl' >> ~/.bashrc
source ~/.bashrc

# 然后你的所有 docker 命令自动变成 nerdctl
docker run -d -p 80:80 nginx    # 实际执行的是 nerdctl
```

### 7.2 注意事项

| 项目 | Docker | nerdctl | 迁移注意 |
|------|--------|---------|---------|
| 守护进程 | dockerd | containerd | 需确保 containerd 在运行 |
| 镜像构建 | 内置 | 依赖 BuildKit | 需安装 BuildKit |
| 网络 | 内置 | 依赖 CNI 插件 | 需安装 CNI 插件 |
| 存储驱动 | overlay2 | overlayfs | 几乎无差别 |
| Compose | docker compose | nerdctl compose | 完全兼容 |

### 7.3 镜像迁移

Docker 和 nerdctl 使用不同的 containerd namespace，镜像不共享：

```bash
# 方法一：通过文件中转
docker save -o images.tar nginx:1.25
nerdctl load -i images.tar

# 方法二：从仓库重新拉取
nerdctl pull nginx:1.25
```

---

## 八、BuildKit — nerdctl 的构建引擎

### 8.1 什么是 BuildKit？

BuildKit 是下一代镜像构建工具，由 Docker 公司开发，比传统 `docker build` 更强大：

- **并行构建**：多个不相互依赖的步骤同时执行
- **更好的缓存**：智能缓存，支持远程缓存
- **构建密钥**：安全传递密钥，不会留在镜像中
- **多平台构建**：一次构建多个架构的镜像

### 8.2 安装和启动 BuildKit

```bash
# 如果用 nerdctl-full 安装，BuildKit 已包含

# 手动安装
wget https://github.com/moby/buildkit/releases/download/v0.13.0/buildkit-v0.13.0.linux-amd64.tar.gz
sudo tar xzf buildkit-v0.13.0.linux-amd64.tar.gz -C /usr/local

# 启动 BuildKit daemon
sudo systemctl enable --now buildkit
# 或手动启动
sudo buildkitd &
```

### 8.3 使用 BuildKit 构建

```bash
# nerdctl build 自动使用 BuildKit
nerdctl build -t my-app:v1 .

# Docker 中启用 BuildKit
DOCKER_BUILDKIT=1 docker build -t my-app:v1 .
```

### 8.4 BuildKit 高级特性

```dockerfile
# 构建时安全使用密钥（不会留在镜像中！）
RUN --mount=type=secret,id=my_secret cat /run/secrets/my_secret

# 构建缓存挂载（加速依赖安装）
RUN --mount=type=cache,target=/root/.cache/pip \
    pip install -r requirements.txt

# SSH 转发（用于拉取私有仓库）
RUN --mount=type=ssh git clone git@github.com:private/repo.git
```

```bash
# 传入密钥
nerdctl build --secret id=my_secret,src=./secret.txt -t my-app .

# 使用 SSH
nerdctl build --ssh default -t my-app .
```

---

## 九、实际场景：K8s 节点上使用 nerdctl

在 Kubernetes 节点上，通常没有 Docker，但有 containerd。这时 nerdctl 就是你的最佳工具：

```bash
# 查看 K8s 节点上的所有镜像
nerdctl -n k8s.io images

# 查看 K8s 节点上运行的容器
nerdctl -n k8s.io ps

# 进入某个 K8s Pod 的容器
nerdctl -n k8s.io exec -it <container-id> sh

# 查看容器日志
nerdctl -n k8s.io logs <container-id>

# 清理无用镜像
nerdctl -n k8s.io image prune -a

# 手动加载镜像到 K8s 节点（离线部署场景）
nerdctl -n k8s.io load -i my-app.tar
```

---

## 十、本章小结

```
✅ containerd 是 Docker 的核心组件，被独立出来成为标准容器运行时
✅ Kubernetes 1.24+ 不再依赖 Docker，直接使用 containerd
✅ nerdctl 是 containerd 的 CLI，命令和 Docker 完全兼容
✅ nerdctl 独有功能：镜像加密、Lazy Pulling、签名验证、IPFS
✅ 迁移最简方案：alias docker=nerdctl
✅ K8s 节点排查：nerdctl -n k8s.io ps/logs/exec
✅ BuildKit 是下一代构建引擎：并行、缓存、安全密钥
✅ ctr 是底层工具（不友好），crictl 是 CRI 调试工具，nerdctl 是日常首选
```

---

> 下一篇：[10-实战项目与最佳实践](./10-实战项目与最佳实践.md) — 来一个完整的项目实战！
