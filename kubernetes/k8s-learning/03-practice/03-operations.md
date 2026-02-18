# 🔧 常见运维操作指南

## 应用部署

### 部署新应用

```bash
# 1. 创建命名空间
kubectl create namespace myapp

# 2. 部署应用
kubectl apply -f deployment.yaml -n myapp

# 3. 创建 Service
kubectl apply -f service.yaml -n myapp

# 4. 验证部署
kubectl get all -n myapp
kubectl rollout status deployment/myapp -n myapp
```

### 更新应用

```bash
# 方式 1：更新镜像
kubectl set image deployment/myapp app=myapp:v2 -n myapp

# 方式 2：应用新的 YAML
kubectl apply -f deployment-v2.yaml -n myapp

# 方式 3：编辑
kubectl edit deployment myapp -n myapp

# 监控更新状态
kubectl rollout status deployment/myapp -n myapp
```

### 回滚

```bash
# 查看历史版本
kubectl rollout history deployment/myapp -n myapp

# 回滚到上一版本
kubectl rollout undo deployment/myapp -n myapp

# 回滚到指定版本
kubectl rollout undo deployment/myapp --to-revision=2 -n myapp
```

## 扩缩容操作

### 手动扩缩容

```bash
# 扩容
kubectl scale deployment myapp --replicas=10 -n myapp

# 缩容
kubectl scale deployment myapp --replicas=3 -n myapp

# 快速缩容到 0（停止服务）
kubectl scale deployment myapp --replicas=0 -n myapp
```

### 自动扩缩容 (HPA)

```bash
# 创建 HPA
kubectl autoscale deployment myapp --min=3 --max=10 --cpu-percent=80 -n myapp

# 查看 HPA 状态
kubectl get hpa -n myapp
kubectl describe hpa myapp -n myapp

# 删除 HPA
kubectl delete hpa myapp -n myapp
```

## 节点运维

### 节点维护

```bash
# 1. 标记节点不可调度
kubectl cordon node1

# 2. 驱逐节点上的 Pod
kubectl drain node1 --ignore-daemonsets --delete-emptydir-data

# 3. 执行维护操作（升级、重启等）
# ...

# 4. 恢复节点调度
kubectl uncordon node1
```

### 节点标签

```bash
# 添加标签
kubectl label nodes node1 disktype=ssd

# 查看节点标签
kubectl get nodes --show-labels

# 删除标签
kubectl label nodes node1 disktype-
```

### 节点污点

```bash
# 添加污点（阻止调度）
kubectl taint nodes node1 key=value:NoSchedule

# 查看污点
kubectl describe node node1 | grep Taints

# 删除污点
kubectl taint nodes node1 key:NoSchedule-
```

## 资源配额管理

### 设置命名空间配额

```yaml
apiVersion: v1
kind: ResourceQuota
metadata:
  name: compute-quota
  namespace: myapp
spec:
  hard:
    requests.cpu: "10"
    requests.memory: 20Gi
    limits.cpu: "20"
    limits.memory: 40Gi
    pods: "50"
```

```bash
# 应用配额
kubectl apply -f quota.yaml

# 查看配额使用情况
kubectl describe quota compute-quota -n myapp
```

### 设置默认限制

```yaml
apiVersion: v1
kind: LimitRange
metadata:
  name: default-limits
  namespace: myapp
spec:
  limits:
  - default:
      cpu: "500m"
      memory: "512Mi"
    defaultRequest:
      cpu: "100m"
      memory: "128Mi"
    type: Container
```

## 日志管理

### 查看日志

```bash
# 基本日志
kubectl logs pod-name -n myapp

# 持续输出
kubectl logs -f pod-name -n myapp

# 指定容器
kubectl logs pod-name -c container-name -n myapp

# 上一个容器的日志（崩溃后）
kubectl logs pod-name --previous -n myapp

# 最近的日志
kubectl logs pod-name --tail=100 -n myapp
kubectl logs pod-name --since=1h -n myapp

# 所有 Pod 的日志
kubectl logs -l app=myapp --all-containers -n myapp
```

### 日志聚合（stern 工具）

```bash
# 安装 stern
brew install stern  # macOS

# 使用 stern 查看多 Pod 日志
stern myapp -n myapp
stern -l app=myapp -n myapp
```

## 监控和诊断

### 资源使用

```bash
# 节点资源
kubectl top nodes

# Pod 资源
kubectl top pods -n myapp
kubectl top pods --containers -n myapp
kubectl top pods --sort-by=memory -n myapp
```

### 事件查看

```bash
# 所有事件
kubectl get events -n myapp

# 按时间排序
kubectl get events --sort-by=.metadata.creationTimestamp -n myapp

# 特定资源的事件
kubectl describe pod pod-name -n myapp | grep -A 20 Events
```

## 配置管理

### ConfigMap 更新

```bash
# 更新 ConfigMap
kubectl edit configmap myconfig -n myapp

# 或替换
kubectl create configmap myconfig --from-file=config.properties -o yaml --dry-run=client | kubectl replace -f -

# 重启 Deployment 应用新配置
kubectl rollout restart deployment myapp -n myapp
```

### Secret 更新

```bash
# 更新 Secret
kubectl create secret generic mysecret --from-literal=password=newpass -o yaml --dry-run=client | kubectl replace -f -

# 重启应用
kubectl rollout restart deployment myapp -n myapp
```

## 备份和恢复

### 导出资源

```bash
# 导出单个资源
kubectl get deployment myapp -n myapp -o yaml > myapp-deploy.yaml

# 导出所有资源
kubectl get all -n myapp -o yaml > myapp-all.yaml

# 导出整个命名空间
kubectl get namespace myapp -o yaml > myapp-ns.yaml
kubectl get all,configmap,secret,pvc -n myapp -o yaml > myapp-backup.yaml
```

### 恢复资源

```bash
# 恢复资源
kubectl apply -f myapp-backup.yaml
```

## 安全操作

### ServiceAccount

```bash
# 创建 ServiceAccount
kubectl create serviceaccount myapp-sa -n myapp

# 为 Deployment 设置 ServiceAccount
kubectl set serviceaccount deployment myapp myapp-sa -n myapp
```

### RBAC

```bash
# 查看角色
kubectl get roles,rolebindings -n myapp
kubectl get clusterroles,clusterrolebindings

# 创建角色绑定
kubectl create rolebinding myapp-admin --role=admin --serviceaccount=myapp:myapp-sa -n myapp
```

## 常用运维脚本

### 清理失败的 Pod

```bash
#!/bin/bash
kubectl get pods -A --field-selector=status.phase=Failed -o name | xargs kubectl delete
```

### 清理 Evicted Pod

```bash
#!/bin/bash
kubectl get pods -A | grep Evicted | awk '{print $2 " -n " $1}' | xargs -L1 kubectl delete pod
```

### 重启所有 Deployment

```bash
#!/bin/bash
NAMESPACE=${1:-default}
kubectl get deployments -n $NAMESPACE -o name | xargs -I {} kubectl rollout restart {} -n $NAMESPACE
```

### 导出所有资源

```bash
#!/bin/bash
NAMESPACE=${1:-default}
BACKUP_DIR="./backup-$(date +%Y%m%d)"
mkdir -p $BACKUP_DIR

for resource in deployment service configmap secret pvc; do
  kubectl get $resource -n $NAMESPACE -o yaml > "$BACKUP_DIR/$resource.yaml"
done
echo "Backup completed to $BACKUP_DIR"
```

## 下一步

- [故障排查与调试](./04-troubleshooting.md)



