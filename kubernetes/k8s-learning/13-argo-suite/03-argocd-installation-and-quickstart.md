# 🛠️ Argo CD 安装与快速上手

> 目标：在 30 分钟内装好 Argo CD，从 Git 部署第一个应用到集群，体验完整的 GitOps 流程。

## 1. 环境准备

最低要求：

- 一个 Kubernetes 集群（Minikube / Kind / 云上托管都行）
- `kubectl` 已配置好并能正常访问集群
- 集群至少 2 vCPU / 4GB 内存
- 能访问 GitHub（用来拉取示例仓库）

## 2. 安装 Argo CD

### 方式一：官方 YAML（推荐学习用）

```bash
# 创建命名空间
kubectl create namespace argocd

# 安装 Argo CD（非 HA 版本，适合学习和小规模）
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml

# 等待所有 Pod Running
kubectl get pods -n argocd -w
```

期望看到：

```text
argocd-application-controller-0       Running
argocd-dex-server-xxx                 Running
argocd-redis-xxx                      Running
argocd-repo-server-xxx                Running
argocd-server-xxx                     Running
```

### 方式二：Helm（适合生产、可自定义 values）

```bash
helm repo add argo https://argoproj.github.io/argo-helm
helm repo update

helm install argocd argo/argo-cd \
  --namespace argocd \
  --create-namespace \
  --set server.service.type=LoadBalancer  # 或 NodePort
```

### HA（高可用）版本

生产环境推荐：

```bash
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/ha/install.yaml
```

HA 版本的 application-controller 和 server 都是多副本 + Leader Election。

## 3. 安装 argocd CLI

```bash
# Linux
curl -sSL -o argocd https://github.com/argoproj/argo-cd/releases/latest/download/argocd-linux-amd64
chmod +x argocd
sudo mv argocd /usr/local/bin/

# macOS
brew install argocd

# Windows（PowerShell）
# 下载 argocd-windows-amd64.exe，重命名为 argocd.exe，放到 PATH 下
# 或者 scoop install argocd

argocd version
```

## 4. 访问 Argo CD UI

### 端口转发（学习用）

```bash
kubectl port-forward svc/argocd-server -n argocd 8080:443
```

浏览器打开 `https://localhost:8080`（会有自签名证书警告，点高级继续即可）。

### 获取初始 admin 密码

```bash
# Argo CD 安装时自动生成的 admin 密码存在 Secret 里
kubectl -n argocd get secret argocd-initial-admin-secret \
  -o jsonpath="{.data.password}" | base64 -d && echo
```

- 用户名：`admin`
- 密码：上面命令的输出

### 登录 CLI

```bash
argocd login localhost:8080 --insecure
# 输入 admin + 密码
```

> 生产环境一定要：① 换成正式域名 + TLS 证书；② 修改初始密码或配置 SSO。

## 5. 部署第一个应用：guestbook

Argo CD 官方提供了示例仓库，我们直接用它来感受 GitOps 流程。

### 通过 CLI 创建 Application

```bash
argocd app create guestbook \
  --repo https://github.com/argoproj/argocd-example-apps.git \
  --path guestbook \
  --dest-server https://kubernetes.default.svc \
  --dest-namespace default
```

这条命令做了什么：

1. 创建了一个名为 `guestbook` 的 Application CR
2. 配置了 Source（Git 仓库 + 路径）
3. 配置了 Destination（本集群 + default 命名空间）

### 查看状态

```bash
argocd app get guestbook
```

输出：

```text
Name:               guestbook
Project:            default
Server:             https://kubernetes.default.svc
Namespace:          default
URL:                https://localhost:8080/applications/guestbook
Repo:               https://github.com/argoproj/argocd-example-apps.git
Target:             HEAD
Path:               guestbook
SyncWindow:         Sync Allowed
Sync Policy:        <none>
Sync Status:        OutOfSync    ← 还没同步
Health Status:      Missing      ← 集群里还没有这些资源
```

### 执行第一次 Sync

```bash
argocd app sync guestbook
```

Argo CD 会：
1. 克隆 Git 仓库
2. 读取 `guestbook/` 目录下的所有 YAML
3. 把它们 `kubectl apply` 到集群的 `default` 命名空间

验证：

```bash
kubectl get deploy,svc -n default
# 应该看到 guestbook-ui deployment 和 service
```

### 也可以通过 YAML 创建（推荐方式）

```yaml
# guestbook-app.yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: guestbook
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/argoproj/argocd-example-apps.git
    targetRevision: HEAD
    path: guestbook
  destination:
    server: https://kubernetes.default.svc
    namespace: default
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
```

```bash
kubectl apply -f guestbook-app.yaml
```

## 6. 在 UI 中操作

打开 `https://localhost:8080`，你会看到：

```text
┌─────────────────────────────────────────────┐
│  Applications                                │
│                                             │
│  ┌─────────────────────────────────┐        │
│  │ 💚 guestbook                    │        │
│  │ Synced / Healthy                │        │
│  │ https://kubernetes.default.svc  │        │
│  └─────────────────────────────────┘        │
│                                             │
└─────────────────────────────────────────────┘
```

点进去能看到：

- **资源拓扑图**：Application → Deployment → ReplicaSet → Pod 的层级关系
- **每个资源的状态**：Healthy / Progressing / Degraded
- **Diff 视图**：Git 定义 vs 集群实际的差异
- **History**：所有 Sync 操作的历史记录

## 7. 体验 GitOps：修改 Git 看自动同步

如果你开了 `automated` 同步，现在做这个实验：

1. Fork 一份 `argocd-example-apps` 仓库到自己账号
2. 把 Application 的 repoURL 改成你 Fork 的仓库
3. 修改 `guestbook/guestbook-ui-deployment.yaml`，比如把 replicas 从 1 改成 3
4. `git push` 到你的仓库

等 3 分钟（默认轮询间隔），Argo CD 会自动检测到变化并 sync：

```bash
# 查看 sync 状态
argocd app get guestbook

# 或者看集群里副本数是否变了
kubectl get deploy guestbook-ui -o jsonpath='{.spec.replicas}'
# 输出：3
```

**这就是 GitOps 的威力**：你没有碰 kubectl，但集群已经更新了。

## 8. CLI 常用命令速查

```bash
# 应用管理
argocd app list                        # 列出所有 Application
argocd app get <app-name>             # 查看详情
argocd app sync <app-name>            # 手动同步
argocd app diff <app-name>            # 看 Git 和集群的差异
argocd app history <app-name>         # 查看同步历史
argocd app rollback <app-name> <id>   # 回滚到某个历史版本

# 资源操作
argocd app resources <app-name>       # 列出 App 管理的所有资源
argocd app logs <app-name> --resource-name <pod>  # 看 Pod 日志

# 仓库管理
argocd repo list                       # 列出已注册的仓库
argocd repo add <url> --username x --password y  # 添加私有仓库

# 集群管理
argocd cluster list                    # 列出已注册的集群
argocd cluster add <context-name>     # 注册新集群

# 账户管理
argocd account update-password         # 改密码
```

## 9. 用 Webhook 加速同步（不等 3 分钟）

默认 Argo CD 每 3 分钟轮询 Git，如果你希望 push 后立刻触发同步，可以配 Webhook：

### GitHub Webhook 配置

1. 在你的 Git 仓库设置 → Webhooks → Add webhook
2. Payload URL：`https://argocd.your-domain.com/api/webhook`
3. Content type：`application/json`
4. Events：选 `push` 事件

这样 push 后 1-2 秒内 Argo CD 就能收到通知并开始同步。

## 10. 常见安装翻车点

| 现象 | 原因 | 解决 |
|------|------|------|
| Pod 一直 ImagePullBackOff | 拉不到 `quay.io/argoproj` 镜像 | 配镜像代理或提前 pull |
| UI 打不开 | port-forward 断了 / 证书问题 | 重新 port-forward；注意用 https 不是 http |
| Login 失败 "invalid credentials" | 密码输错了 | 重新从 Secret 里取密码 |
| Sync 后资源没变化 | Git 的文件没有变化 / path 写错了 | `argocd app diff` 看差异 |
| Sync 失败 "permission denied" | RBAC 问题，Argo CD ServiceAccount 没权限 | 检查目标 namespace 的 RBAC |
| Private repo 拉取失败 | 凭据没配对 | `argocd repo list` 确认仓库状态，重新 add |

## 11. 从 Helm Chart 部署一个应用

很多真实场景不是裸 YAML 而是 Helm Chart：

```bash
argocd app create redis \
  --repo https://charts.bitnami.com/bitnami \
  --helm-chart redis \
  --revision 17.3.14 \
  --dest-server https://kubernetes.default.svc \
  --dest-namespace redis \
  --helm-set replica.replicaCount=3 \
  --sync-option CreateNamespace=true

argocd app sync redis
```

等 Sync 完成后：

```bash
kubectl get pods -n redis
# 看到 redis-master 和 redis-replicas
```

## 12. 清理

```bash
# 删除应用（也会删除集群中 Application 管理的资源）
argocd app delete guestbook --cascade

# 卸载 Argo CD
kubectl delete -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml
kubectl delete namespace argocd
```

> `--cascade` 表示同时删除 Application 管理的集群资源。不加的话只删 Application CR，集群资源保留。

## 下一步

装好之后，建议进阶学习：

- 管理多个应用 / 多环境 / 多集群：[Argo CD 进阶](./04-argocd-advanced.md)
- 想了解渐进发布：[Argo Rollouts](./05-argo-rollouts-core.md)
