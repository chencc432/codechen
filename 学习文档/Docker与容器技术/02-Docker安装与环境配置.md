# 02 — Docker 安装与环境配置

> 工欲善其事，必先利其器。把环境搞好，后面学起来才顺畅。

---

## 一、安装方案选择

不同操作系统，安装方式不同：

| 操作系统 | 推荐方案 | 说明 |
|---------|---------|------|
| **Windows 10/11** | Docker Desktop + WSL2 | 最方便，图形界面 |
| **macOS** | Docker Desktop | 原生支持，一键安装 |
| **Ubuntu/Debian** | 官方 apt 源 | 服务器标准方案 |
| **CentOS/RHEL** | 官方 yum 源 | 生产环境常用 |

---

## 二、Windows 安装（推荐 WSL2 方案）

### 2.1 前置条件

```
✅ Windows 10 版本 2004 及以上，或 Windows 11
✅ 开启 CPU 虚拟化（BIOS 中开启 VT-x / AMD-V）
✅ 至少 4GB 内存（建议 8GB 以上）
```

### 2.2 第一步：安装 WSL2

WSL2（Windows Subsystem for Linux 2）是 Windows 上运行 Linux 的最佳方式。

```powershell
# 以管理员身份运行 PowerShell

# 一键安装 WSL2（会自动安装 Ubuntu）
wsl --install

# 安装完成后重启电脑
# 重启后会自动打开 Ubuntu，设置用户名和密码

# 确认 WSL 版本
wsl --list --verbose
# 输出应该显示 VERSION 为 2
```

### 2.3 第二步：安装 Docker Desktop

1. 下载 Docker Desktop：https://www.docker.com/products/docker-desktop/
2. 双击安装包，一路 Next
3. **关键**：安装时勾选 "Use WSL 2 instead of Hyper-V"
4. 安装完成后重启

### 2.4 第三步：配置验证

```powershell
# 打开终端（PowerShell 或 WSL 终端都可以）

# 查看 Docker 版本
docker --version
# Docker version 27.x.x, build xxxxxx

# 查看详细信息
docker info

# 运行测试容器
docker run hello-world
```

如果看到以下输出，说明安装成功：

```
Hello from Docker!
This message shows that your installation appears to be working correctly.
...
```

---

## 三、macOS 安装

### 3.1 安装 Docker Desktop

```bash
# 方法一：官网下载 dmg 安装
# https://www.docker.com/products/docker-desktop/
# 注意选择 Apple Silicon (M1/M2/M3) 或 Intel 版本

# 方法二：使用 Homebrew
brew install --cask docker
```

### 3.2 启动和验证

1. 打开 Launchpad，点击 Docker 图标
2. 等待 Docker 引擎启动（菜单栏鲸鱼图标变绿）
3. 打开终端验证：

```bash
docker --version
docker run hello-world
```

---

## 四、Linux 安装（Ubuntu 为例）

### 4.1 卸载旧版本

```bash
# 先清除可能存在的旧版本
sudo apt-get remove docker docker-engine docker.io containerd runc
```

### 4.2 安装 Docker Engine

```bash
# 更新包索引
sudo apt-get update

# 安装依赖
sudo apt-get install -y \
    ca-certificates \
    curl \
    gnupg \
    lsb-release

# 添加 Docker 官方 GPG 密钥
sudo install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | \
    sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
sudo chmod a+r /etc/apt/keyrings/docker.gpg

# 添加仓库源
echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] \
  https://download.docker.com/linux/ubuntu \
  $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | \
  sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

# 安装 Docker Engine
sudo apt-get update
sudo apt-get install -y docker-ce docker-ce-cli containerd.io \
    docker-buildx-plugin docker-compose-plugin
```

### 4.3 免 sudo 使用 Docker

```bash
# 将当前用户加入 docker 组（这样就不用每次都 sudo 了）
sudo usermod -aG docker $USER

# 重新登录使其生效（或者执行下面的命令）
newgrp docker

# 验证
docker run hello-world
```

### 4.4 设置开机自启

```bash
sudo systemctl enable docker
sudo systemctl start docker

# 查看 Docker 服务状态
sudo systemctl status docker
```

---

## 五、CentOS / RHEL 安装

```bash
# 卸载旧版本
sudo yum remove docker docker-client docker-client-latest \
    docker-common docker-latest docker-latest-logrotate \
    docker-logrotate docker-engine

# 安装依赖
sudo yum install -y yum-utils

# 添加仓库源
sudo yum-config-manager --add-repo \
    https://download.docker.com/linux/centos/docker-ce.repo

# 安装 Docker
sudo yum install -y docker-ce docker-ce-cli containerd.io \
    docker-buildx-plugin docker-compose-plugin

# 启动并设置开机自启
sudo systemctl start docker
sudo systemctl enable docker

# 加入 docker 组
sudo usermod -aG docker $USER
newgrp docker

# 验证
docker run hello-world
```

---

## 六、镜像加速配置（国内必做！）

### 6.1 为什么需要镜像加速？

Docker 默认从 Docker Hub（美国服务器）拉取镜像，国内直接访问速度很慢，经常超时。**配置镜像加速器是国内用户的必做步骤**。

### 6.2 常见镜像加速源

| 提供商 | 地址 |
|-------|------|
| 阿里云 | `https://<你的ID>.mirror.aliyuncs.com`（需注册获取） |
| 腾讯云 | `https://mirror.ccs.tencentyun.com` |
| 华为云 | `https://mirrors.huaweicloud.com` |
| 网易 | `https://hub-mirror.c.163.com` |
| 中科大 | `https://docker.mirrors.ustc.edu.cn` |

### 6.3 配置方法

#### Linux 配置

```bash
# 创建或编辑 daemon.json
sudo mkdir -p /etc/docker
sudo tee /etc/docker/daemon.json <<-'EOF'
{
    "registry-mirrors": [
        "https://mirror.ccs.tencentyun.com",
        "https://hub-mirror.c.163.com"
    ]
}
EOF

# 重启 Docker 使配置生效
sudo systemctl daemon-reload
sudo systemctl restart docker

# 验证加速器是否生效
docker info | grep -A 5 "Registry Mirrors"
```

#### Docker Desktop 配置（Windows / macOS）

1. 打开 Docker Desktop
2. 点击右上角齿轮图标（Settings）
3. 选择 "Docker Engine"
4. 在 JSON 配置中添加：

```json
{
    "registry-mirrors": [
        "https://mirror.ccs.tencentyun.com",
        "https://hub-mirror.c.163.com"
    ]
}
```

5. 点击 "Apply & Restart"

### 6.4 验证加速效果

```bash
# 拉一个镜像试试速度
time docker pull nginx:latest

# 如果几秒内完成，说明加速器生效了
# 如果还是很慢，检查 daemon.json 配置是否正确
```

---

## 七、Docker Desktop 设置优化

如果你使用 Docker Desktop，建议做以下优化：

### 7.1 资源分配

```
Settings → Resources：
  ├── CPUs：分配 2-4 核（根据你的电脑配置）
  ├── Memory：分配 4-8 GB（建议至少 4GB）
  ├── Swap：1-2 GB
  └── Disk image size：60-100 GB
```

### 7.2 WSL2 集成（Windows）

```
Settings → Resources → WSL Integration：
  ├── ✅ Enable integration with my default WSL distro
  └── ✅ 勾选你要集成的 Linux 发行版（如 Ubuntu）
```

---

## 八、安装后必做的检查清单

运行以下命令，确保一切正常：

```bash
# 1. Docker 版本
docker --version
# 期望：Docker version 27.x.x

# 2. Docker Compose 版本（新版已内置）
docker compose version
# 期望：Docker Compose version v2.x.x

# 3. Docker 详细信息
docker info
# 关注：Server Version、Storage Driver、Registry Mirrors

# 4. 运行测试容器
docker run hello-world
# 期望：Hello from Docker!

# 5. 运行一个交互式容器
docker run -it ubuntu bash
# 进入 Ubuntu 容器的命令行
# 输入 exit 退出

# 6. 运行一个 Web 服务器
docker run -d -p 8080:80 --name test-nginx nginx
# 然后浏览器打开 http://localhost:8080 看到 Nginx 欢迎页

# 7. 清理测试容器
docker rm -f test-nginx
docker system prune -f
```

---

## 九、常见安装问题排查

### 问题 1：Docker Daemon 未启动

```
Cannot connect to the Docker daemon at unix:///var/run/docker.sock.
Is the docker daemon running?
```

**解决**：

```bash
# Linux
sudo systemctl start docker

# Windows / macOS
# 启动 Docker Desktop 应用
```

### 问题 2：权限不足

```
Got permission denied while trying to connect to the Docker daemon socket
```

**解决**：

```bash
sudo usermod -aG docker $USER
# 然后重新登录终端
```

### 问题 3：WSL2 相关错误（Windows）

```
WSL 2 installation is incomplete
```

**解决**：

```powershell
# 更新 WSL 内核
wsl --update
# 然后重启电脑
```

### 问题 4：拉取镜像超时

```
Error response from daemon: Get "https://registry-1.docker.io/v2/": net/http: TLS handshake timeout
```

**解决**：配置镜像加速器（参考第六节），或检查网络连接。

---

## 十、本章小结

```
✅ Windows 推荐：Docker Desktop + WSL2
✅ macOS 推荐：Docker Desktop
✅ Linux 推荐：官方源安装 Docker Engine
✅ 国内必配镜像加速器，否则拉镜像巨慢
✅ docker run hello-world 是验证安装的标准姿势
✅ 遇到权限问题，把用户加到 docker 组
```

---

> 下一篇：[03-Docker镜像全面详解](./03-Docker镜像全面详解.md) — 深入理解镜像的一切！
