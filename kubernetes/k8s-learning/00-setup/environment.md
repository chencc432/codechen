# 🔧 Kubernetes 环境搭建指南

## 本地开发环境选择

| 工具 | 适用场景 | 资源需求 | 推荐指数 |
|------|---------|---------|---------|
| Minikube | 单节点学习 | 2CPU/2GB | ⭐⭐⭐⭐⭐ |
| Kind | 多节点测试 | 4CPU/4GB | ⭐⭐⭐⭐ |
| k3s | 轻量级生产 | 1CPU/512MB | ⭐⭐⭐⭐ |
| Docker Desktop | Mac/Windows | 4CPU/4GB | ⭐⭐⭐ |

## 方案一：Minikube（推荐新手）

### 安装 Minikube

```bash
# Linux
curl -LO https://storage.googleapis.com/minikube/releases/latest/minikube-linux-amd64
sudo install minikube-linux-amd64 /usr/local/bin/minikube

# macOS
brew install minikube

# Windows (PowerShell 管理员)
choco install minikube
```

### 启动集群

```bash
# 启动单节点集群
minikube start

# 指定资源启动
minikube start --cpus=4 --memory=8192 --driver=docker

# 启动多节点集群（进阶）
minikube start --nodes=3

# 查看集群状态
minikube status

# 查看集群信息
kubectl cluster-info
```

### Minikube 常用命令

```bash
# 停止集群
minikube stop

# 删除集群
minikube delete

# SSH 进入节点
minikube ssh

# 打开 Dashboard
minikube dashboard

# 获取服务 URL
minikube service <service-name> --url

# 加载本地镜像到 minikube
minikube image load <image-name>

# 启用插件
minikube addons enable ingress
minikube addons enable metrics-server
minikube addons list
```

## 方案二：Kind（Kubernetes in Docker）

### 安装 Kind

```bash
# Linux/macOS
curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-amd64
chmod +x ./kind
sudo mv ./kind /usr/local/bin/kind

# macOS (Homebrew)
brew install kind

# Windows
choco install kind
```

### 创建集群

```bash
# 创建默认集群
kind create cluster

# 创建指定名称的集群
kind create cluster --name my-cluster

# 使用配置文件创建多节点集群
kind create cluster --config kind-config.yaml
```

### Kind 配置文件示例

```yaml
# kind-config.yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
  - role: control-plane
    extraPortMappings:
      - containerPort: 30000
        hostPort: 30000
        protocol: TCP
  - role: worker
  - role: worker
```

### Kind 常用命令

```bash
# 列出集群
kind get clusters

# 删除集群
kind delete cluster --name my-cluster

# 加载镜像到集群
kind load docker-image <image-name> --name my-cluster

# 获取 kubeconfig
kind get kubeconfig --name my-cluster
```

## 安装 kubectl

kubectl 是 Kubernetes 的命令行工具，必须安装。

```bash
# Linux
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
chmod +x kubectl
sudo mv kubectl /usr/local/bin/

# macOS
brew install kubectl

# Windows
choco install kubernetes-cli

# 验证安装
kubectl version --client
```

## kubectl 自动补全配置

```bash
# Bash
echo 'source <(kubectl completion bash)' >> ~/.bashrc
echo 'alias k=kubectl' >> ~/.bashrc
echo 'complete -o default -F __start_kubectl k' >> ~/.bashrc
source ~/.bashrc

# Zsh
echo 'source <(kubectl completion zsh)' >> ~/.zshrc
echo 'alias k=kubectl' >> ~/.zshrc
source ~/.zshrc
```

## 验证环境

```bash
# 1. 查看集群信息
kubectl cluster-info

# 2. 查看节点
kubectl get nodes

# 3. 查看所有命名空间
kubectl get namespaces

# 4. 查看系统组件
kubectl get pods -n kube-system

# 5. 运行测试 Pod
kubectl run nginx --image=nginx --port=80
kubectl get pods
kubectl delete pod nginx
```

## 常见问题排查

### 问题1：kubectl 无法连接集群

```bash
# 检查 kubeconfig
cat ~/.kube/config

# 检查集群状态
minikube status  # 或 kind get clusters

# 重新启动集群
minikube start
```

### 问题2：镜像拉取失败

```bash
# 使用国内镜像源
minikube start --image-mirror-country=cn

# 或配置 Docker 镜像加速器
# 编辑 /etc/docker/daemon.json
{
  "registry-mirrors": [
    "https://registry.docker-cn.com"
  ]
}
```

### 问题3：资源不足

```bash
# 减少资源配置
minikube start --cpus=2 --memory=2048

# 或使用 k3s 轻量级方案
curl -sfL https://get.k3s.io | sh -
```

## 下一步

环境搭建完成后，继续学习 [Kubernetes 概述与架构](../01-basics/01-overview.md)



