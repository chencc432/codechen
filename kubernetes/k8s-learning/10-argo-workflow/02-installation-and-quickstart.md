# 🛠️ 安装与快速上手

> 目标：在 30 分钟内装好 Argo Workflow，跑通第一个 Workflow，并能从 UI / CLI 看到执行结果。

## 1. 环境准备

最低要求：

- 一个能用的 Kubernetes 集群（Minikube / Kind / 云上托管都可以）
- `kubectl` 已配置好
- 集群至少 2 vCPU / 4GB 内存（跑 controller + UI + 测试 Pod）

如果你还没集群，参考 [环境搭建指南](../00-setup/environment.md)。

## 2. 安装 Argo Workflow

官方提供两种安装清单：

| 安装清单 | 适用场景 |
|----------|----------|
| `quick-start` | 学习 / 测试，自带 MinIO、PostgreSQL，开箱即用 |
| `install` | 生产部署，依赖你自己提供 S3 / Postgres |

学习用 `quick-start` 最简单：

```bash
# 创建命名空间
kubectl create namespace argo

# 安装（注意把版本号替换为你想要的；下例是写文档时常用的 v3.5.x）
kubectl apply -n argo -f https://github.com/argoproj/argo-workflows/releases/download/v3.5.10/quick-start-postgres.yaml
```

等所有 Pod Running：

```bash
kubectl get pods -n argo -w
```

期望看到（数量随版本不同会变）：

```text
argo-server-xxx                Running
workflow-controller-xxx        Running
minio-xxx                      Running
postgres-xxx                   Running
```

> 如果是国内集群拉镜像慢，可以先把镜像 `docker pull` 到本地或换成镜像源（如阿里云 ACR、华为云 SWR 镜像同步）。

## 3. 安装 argo CLI

CLI 是命令行客户端，用来提交、查看 Workflow。

```bash
# Linux
curl -sLO https://github.com/argoproj/argo-workflows/releases/download/v3.5.10/argo-linux-amd64.gz
gunzip argo-linux-amd64.gz
chmod +x argo-linux-amd64
sudo mv argo-linux-amd64 /usr/local/bin/argo

argo version
```

Windows 在 PowerShell 里下载对应 `argo-windows-amd64` 即可，或者 `scoop install argo`。

## 4. 访问 UI

quick-start 默认开了 `argo-server`，端口转发后即可访问 UI：

```bash
kubectl -n argo port-forward svc/argo-server 2746:2746
```

浏览器打开：

```text
https://localhost:2746
```

> 浏览器会提示证书不安全，因为 quick-start 用的是自签名证书；学习环境点高级 → 继续访问即可。生产环境一定要换成正式证书 + 走 Ingress / LB。

如果你不想加密，也可以改启动参数 `--secure=false`，但**不建议生产用**。

## 5. 第一个 Workflow：hello argo

新建 `hello.yaml`：

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-
spec:
  entrypoint: main
  templates:
    - name: main
      container:
        image: alpine:3.18
        command: ["sh", "-c"]
        args: ["echo hello argo; sleep 5; echo done"]
```

提交：

```bash
argo submit -n argo hello.yaml --watch
```

`--watch` 会实时打印进度，跑完会显示节点状态。

也可以用 `kubectl apply -n argo -f hello.yaml`，效果相同（因为 Workflow 就是 CR）。

## 6. 查看 Workflow

CLI 常用命令：

```bash
# 列出
argo list -n argo

# 看某个的详情
argo get -n argo <workflow-name>

# 看某一步的日志
argo logs -n argo <workflow-name>

# 看某一步具体 Pod 日志（多步骤时）
argo logs -n argo <workflow-name> -c main

# 删除
argo delete -n argo <workflow-name>
```

如果你更习惯 kubectl：

```bash
kubectl get wf -n argo                  # wf 是 Workflow 的简写
kubectl get pods -n argo -l workflows.argoproj.io/workflow=<workflow-name>
kubectl logs -n argo <pod-name>
```

## 7. 第二个 Workflow：DAG（让你看到 Argo 真正的味道）

`dag.yaml`：

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: dag-demo-
spec:
  entrypoint: main
  templates:
    - name: main
      dag:
        tasks:
          - name: A
            template: echo
            arguments:
              parameters: [{name: msg, value: "I am A"}]
          - name: B
            dependencies: [A]
            template: echo
            arguments:
              parameters: [{name: msg, value: "I am B, after A"}]
          - name: C
            dependencies: [A]
            template: echo
            arguments:
              parameters: [{name: msg, value: "I am C, after A"}]
          - name: D
            dependencies: [B, C]
            template: echo
            arguments:
              parameters: [{name: msg, value: "I am D, after B and C"}]
    - name: echo
      inputs:
        parameters:
          - name: msg
      container:
        image: alpine:3.18
        command: [sh, -c]
        args: ["echo {{inputs.parameters.msg}}"]
```

提交：

```bash
argo submit -n argo dag.yaml --watch
```

打开 UI 你能看到一张 DAG 图：A → B/C 并行 → D，每个节点点进去能看日志。

## 8. 常见提交方式

| 方式 | 命令 | 何时用 |
|------|------|--------|
| 单次提交 | `argo submit -n argo wf.yaml` | 临时跑一次 |
| 用模板提交 | `argo submit --from workflowtemplate/build` | 复用模板（见第 06 篇） |
| 带参数提交 | `argo submit wf.yaml -p version=v1.2.0` | 业务参数化 |
| 等待完成 | 加 `--wait` | 脚本里需要拿结果时 |
| 实时 watch | 加 `--watch` | 调试时方便 |

## 9. 卸载

```bash
kubectl delete -n argo -f https://github.com/argoproj/argo-workflows/releases/download/v3.5.10/quick-start-postgres.yaml
kubectl delete namespace argo
```

## 10. 常见装机翻车点

| 现象 | 原因 | 解决 |
|------|------|------|
| Pod 一直 ImagePullBackOff | 网络拉不到镜像 | 换镜像源 / 提前 pull |
| UI 打不开 | port-forward 没起 / TLS 警告未点过 | 重新 port-forward；用 https 不是 http |
| 提交后 Workflow 一直 Pending | controller 没起来 / RBAC 不足 | `kubectl logs deploy/workflow-controller -n argo` 看错误 |
| Step Pod 起不来报 ServiceAccount 没权限 | 业务命名空间没有 `default` SA 的权限 | 给 SA 加 RoleBinding，或者 workflow 里指定 `serviceAccountName` |
| Artifact 上传失败 | MinIO 配置或网络问题 | 看 wait container 日志：`kubectl logs <pod> -c wait` |

> 调试 Argo 时 **wait container 的日志非常关键**——每个步骤 Pod 里除了你自己那个容器，Argo 还注入了一个 `wait` 容器负责处理 artifact 上传/下载和退出收尾，很多奇怪问题都是这里报出来的。

## 下一步

环境跑通后，建议精读 Spec：

- [Workflow Spec 与 Template 类型详解](./03-templates-and-spec.md)
