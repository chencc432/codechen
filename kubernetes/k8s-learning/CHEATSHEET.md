# 🎯 Kubernetes 命令速查表

## kubectl 基础命令

```bash
# 查看资源
kubectl get pods/svc/deploy/nodes/ns
kubectl get pods -o wide                    # 详细信息
kubectl get pods -A                         # 所有命名空间
kubectl get pods -l app=nginx               # 按标签筛选
kubectl get pods -w                         # 持续监听

# 详细信息
kubectl describe pod <name>
kubectl describe node <name>

# 创建资源
kubectl create deployment nginx --image=nginx
kubectl apply -f manifest.yaml

# 删除资源
kubectl delete pod <name>
kubectl delete -f manifest.yaml

# 日志
kubectl logs <pod>
kubectl logs <pod> -f                       # 持续输出
kubectl logs <pod> --previous               # 上一个容器

# 执行命令
kubectl exec -it <pod> -- sh
kubectl exec <pod> -- ls /

# 端口转发
kubectl port-forward pod/<name> 8080:80
kubectl port-forward svc/<name> 8080:80

# 扩缩容
kubectl scale deployment <name> --replicas=3

# 回滚
kubectl rollout status deployment <name>
kubectl rollout undo deployment <name>
kubectl rollout history deployment <name>
```

## 常用资源简写

| 简写 | 完整名称 |
|------|---------|
| po | pods |
| svc | services |
| deploy | deployments |
| rs | replicasets |
| ds | daemonsets |
| sts | statefulsets |
| cm | configmaps |
| ns | namespaces |
| no | nodes |
| pv | persistentvolumes |
| pvc | persistentvolumeclaims |
| ing | ingresses |
| sa | serviceaccounts |
| hpa | horizontalpodautoscalers |

## 命名空间操作

```bash
kubectl create ns <name>
kubectl get ns
kubectl config set-context --current --namespace=<name>
kubectl get pods -n <namespace>
kubectl delete ns <name>
```

## ConfigMap 和 Secret

```bash
# ConfigMap
kubectl create cm <name> --from-literal=key=value
kubectl create cm <name> --from-file=config.txt
kubectl get cm <name> -o yaml

# Secret
kubectl create secret generic <name> --from-literal=password=xxx
kubectl get secret <name> -o jsonpath='{.data.password}' | base64 -d
```

## 标签操作

```bash
kubectl label pods <name> app=nginx
kubectl label pods <name> app-                # 删除标签
kubectl get pods -l app=nginx
kubectl get pods --show-labels
```

## 调试技巧

```bash
# 查看事件
kubectl get events --sort-by='.metadata.creationTimestamp'

# 进入容器
kubectl exec -it <pod> -- sh

# 临时调试容器
kubectl debug <pod> -it --image=busybox

# 节点调试
kubectl debug node/<node> -it --image=busybox

# 资源使用
kubectl top nodes
kubectl top pods
```

## 生成 YAML

```bash
kubectl run nginx --image=nginx --dry-run=client -o yaml > pod.yaml
kubectl create deploy nginx --image=nginx --dry-run=client -o yaml > deploy.yaml
kubectl expose deploy nginx --port=80 --dry-run=client -o yaml > svc.yaml
```

## 常用 API 资源

```bash
kubectl api-resources           # 所有资源
kubectl api-versions            # API 版本
kubectl explain pod             # 资源文档
kubectl explain pod.spec.containers
```

## 节点操作

```bash
kubectl cordon <node>           # 标记不可调度
kubectl uncordon <node>         # 取消标记
kubectl drain <node>            # 驱逐 Pod
kubectl taint nodes <node> key=value:NoSchedule
```

## 快速别名配置

```bash
# 添加到 ~/.bashrc 或 ~/.zshrc
alias k='kubectl'
alias kgp='kubectl get pods'
alias kgs='kubectl get svc'
alias kgd='kubectl get deploy'
alias kga='kubectl get all'
alias kdp='kubectl describe pod'
alias kl='kubectl logs'
alias ke='kubectl exec -it'

# 自动补全
source <(kubectl completion bash)
complete -o default -F __start_kubectl k
```

---
**祝你 Kubernetes 学习顺利！** 🚀



