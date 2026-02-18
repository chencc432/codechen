# 🔄 DaemonSet 与 Job/CronJob

## DaemonSet

### 什么是 DaemonSet？

DaemonSet 确保在每个（或指定的）节点上运行一个 Pod 副本。

```
┌─────────────────────────────────────────────────────────────────────┐
│                        DaemonSet                                     │
│                                                                       │
│  ┌─────────────┐   ┌─────────────┐   ┌─────────────┐               │
│  │   Node 1    │   │   Node 2    │   │   Node 3    │               │
│  │  ┌───────┐  │   │  ┌───────┐  │   │  ┌───────┐  │               │
│  │  │ Pod   │  │   │  │ Pod   │  │   │  │ Pod   │  │               │
│  │  │(daemon)│ │   │  │(daemon)│ │   │  │(daemon)│ │               │
│  │  └───────┘  │   │  └───────┘  │   │  └───────┘  │               │
│  └─────────────┘   └─────────────┘   └─────────────┘               │
│                                                                       │
│  典型用例：                                                           │
│  - 日志收集 (Fluentd, Filebeat)                                      │
│  - 监控代理 (Prometheus Node Exporter)                               │
│  - 网络插件 (Calico, Flannel)                                        │
│  - 存储守护进程                                                       │
└─────────────────────────────────────────────────────────────────────┘
```

### DaemonSet YAML

```yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: fluentd
  namespace: kube-system
  labels:
    app: fluentd
spec:
  selector:
    matchLabels:
      app: fluentd
  
  # 更新策略
  updateStrategy:
    type: RollingUpdate          # RollingUpdate 或 OnDelete
    rollingUpdate:
      maxUnavailable: 1
  
  template:
    metadata:
      labels:
        app: fluentd
    spec:
      # 容忍所有污点（可选）
      tolerations:
      - operator: Exists
      
      # 节点选择器（可选）
      nodeSelector:
        kubernetes.io/os: linux
      
      containers:
      - name: fluentd
        image: fluent/fluentd:v1.14
        resources:
          limits:
            memory: 200Mi
          requests:
            cpu: 100m
            memory: 200Mi
        volumeMounts:
        - name: varlog
          mountPath: /var/log
        - name: containers
          mountPath: /var/lib/docker/containers
          readOnly: true
      
      volumes:
      - name: varlog
        hostPath:
          path: /var/log
      - name: containers
        hostPath:
          path: /var/lib/docker/containers
```

### DaemonSet 操作

```bash
# 创建
kubectl apply -f daemonset.yaml

# 查看
kubectl get daemonset -n kube-system
kubectl get ds                           # 简写
kubectl describe ds fluentd -n kube-system

# 查看 Pod（每个节点一个）
kubectl get pods -l app=fluentd -o wide

# 更新
kubectl set image ds/fluentd fluentd=fluent/fluentd:v1.15

# 删除
kubectl delete ds fluentd
```

### 只在特定节点运行

```yaml
spec:
  template:
    spec:
      # 方式 1：节点选择器
      nodeSelector:
        node-type: worker
      
      # 方式 2：节点亲和性
      affinity:
        nodeAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
            - matchExpressions:
              - key: node-role.kubernetes.io/master
                operator: DoesNotExist
```

## Job

### 什么是 Job？

Job 创建一个或多个 Pod，确保指定数量的 Pod 成功终止。

```
┌─────────────────────────────────────────────────────────────────────┐
│                            Job                                       │
│                                                                       │
│  完成数: 5                                                           │
│  并行数: 2                                                           │
│                                                                       │
│  ┌─────────┐ ┌─────────┐                                           │
│  │  Pod 1  │ │  Pod 2  │  ← 并行运行                                │
│  │   ✓     │ │   ✓     │                                           │
│  └─────────┘ └─────────┘                                           │
│                                                                       │
│  ┌─────────┐ ┌─────────┐                                           │
│  │  Pod 3  │ │  Pod 4  │  ← 前面完成后继续                          │
│  │   ✓     │ │   ✓     │                                           │
│  └─────────┘ └─────────┘                                           │
│                                                                       │
│  ┌─────────┐                                                        │
│  │  Pod 5  │             ← 最后一个                                  │
│  │   ✓     │                                                        │
│  └─────────┘                                                        │
│                                                                       │
│  Job 完成！                                                          │
└─────────────────────────────────────────────────────────────────────┘
```

### Job YAML

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: batch-job
spec:
  # 完成数：需要成功完成的 Pod 数量
  completions: 5
  
  # 并行数：同时运行的 Pod 数量
  parallelism: 2
  
  # 重试次数
  backoffLimit: 4
  
  # 超时时间（秒）
  activeDeadlineSeconds: 300
  
  # 完成后保留时间（秒，K8s 1.23+）
  ttlSecondsAfterFinished: 100
  
  template:
    spec:
      restartPolicy: Never        # 或 OnFailure
      containers:
      - name: worker
        image: busybox
        command: ["sh", "-c", "echo Processing item && sleep 30"]
```

### Job 类型

#### 1. 单任务 Job

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: single-job
spec:
  template:
    spec:
      restartPolicy: Never
      containers:
      - name: task
        image: busybox
        command: ["echo", "Hello Job"]
```

#### 2. 并行 Job（固定完成数）

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: parallel-job
spec:
  completions: 10        # 总共需要完成 10 个
  parallelism: 3         # 同时运行 3 个
  template:
    spec:
      restartPolicy: Never
      containers:
      - name: worker
        image: busybox
        command: ["sh", "-c", "echo Task $RANDOM && sleep 5"]
```

#### 3. 工作队列 Job

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: queue-job
spec:
  parallelism: 3         # 只设置并行数
  # 不设置 completions，Pod 自己决定何时完成
  template:
    spec:
      restartPolicy: Never
      containers:
      - name: worker
        image: myapp/queue-processor
```

### Job 操作

```bash
# 创建
kubectl apply -f job.yaml
kubectl create job my-job --image=busybox -- echo "Hello"

# 查看
kubectl get jobs
kubectl describe job batch-job
kubectl get pods -l job-name=batch-job

# 查看日志
kubectl logs job/batch-job

# 删除
kubectl delete job batch-job

# 级联删除 Pod
kubectl delete job batch-job --cascade=foreground
```

## CronJob

### 什么是 CronJob？

CronJob 按照预定时间表创建 Job。

```
┌─────────────────────────────────────────────────────────────────────┐
│                          CronJob                                     │
│                                                                       │
│  schedule: "*/5 * * * *"  (每 5 分钟)                                │
│                                                                       │
│  时间线:                                                              │
│  ─────┬─────┬─────┬─────┬─────┬─────┬─────→                        │
│       │     │     │     │     │     │                               │
│       ▼     ▼     ▼     ▼     ▼     ▼                               │
│     Job 1 Job 2 Job 3 Job 4 Job 5 Job 6                             │
│                                                                       │
│  00:00 00:05 00:10 00:15 00:20 00:25                                │
└─────────────────────────────────────────────────────────────────────┘
```

### Cron 表达式

```
┌───────────── 分钟 (0 - 59)
│ ┌───────────── 小时 (0 - 23)
│ │ ┌───────────── 日 (1 - 31)
│ │ │ ┌───────────── 月 (1 - 12)
│ │ │ │ ┌───────────── 星期 (0 - 6，0 = 周日)
│ │ │ │ │
* * * * *

示例：
*/5 * * * *     # 每 5 分钟
0 * * * *       # 每小时
0 0 * * *       # 每天凌晨
0 0 * * 0       # 每周日凌晨
0 0 1 * *       # 每月 1 日凌晨
0 9 * * 1-5     # 工作日早上 9 点
```

### CronJob YAML

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: backup-job
spec:
  # Cron 表达式
  schedule: "0 2 * * *"           # 每天凌晨 2 点
  
  # 时区（K8s 1.27+）
  timeZone: "Asia/Shanghai"
  
  # 并发策略
  concurrencyPolicy: Forbid       # Allow/Forbid/Replace
  
  # 保留历史 Job 数量
  successfulJobsHistoryLimit: 3
  failedJobsHistoryLimit: 1
  
  # 启动截止时间（秒）
  startingDeadlineSeconds: 200
  
  # 挂起
  suspend: false
  
  # Job 模板
  jobTemplate:
    spec:
      template:
        spec:
          restartPolicy: OnFailure
          containers:
          - name: backup
            image: backup-tool:latest
            command: ["/bin/sh", "-c"]
            args:
            - |
              echo "Starting backup at $(date)"
              # 执行备份逻辑
              echo "Backup completed"
```

### 并发策略

| 策略 | 说明 |
|------|------|
| Allow | 允许并发运行（默认）|
| Forbid | 禁止并发，跳过新调度 |
| Replace | 取消当前运行的，启动新的 |

### CronJob 操作

```bash
# 创建
kubectl apply -f cronjob.yaml
kubectl create cronjob my-cron --image=busybox --schedule="*/5 * * * *" -- echo "Hello"

# 查看
kubectl get cronjobs
kubectl get cj                           # 简写
kubectl describe cj backup-job

# 手动触发一次
kubectl create job manual-backup --from=cronjob/backup-job

# 暂停/恢复
kubectl patch cj backup-job -p '{"spec":{"suspend":true}}'
kubectl patch cj backup-job -p '{"spec":{"suspend":false}}'

# 查看生成的 Job
kubectl get jobs

# 删除
kubectl delete cj backup-job
```

## 实践练习

### 练习 1：创建 DaemonSet

```bash
# 创建日志收集 DaemonSet
cat << EOF | kubectl apply -f -
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: log-collector
spec:
  selector:
    matchLabels:
      app: log-collector
  template:
    metadata:
      labels:
        app: log-collector
    spec:
      containers:
      - name: collector
        image: busybox
        command: ["sh", "-c", "while true; do echo 'Collecting logs from \$(hostname)'; sleep 60; done"]
        volumeMounts:
        - name: logs
          mountPath: /var/log
          readOnly: true
      volumes:
      - name: logs
        hostPath:
          path: /var/log
EOF

# 查看（每个节点一个 Pod）
kubectl get ds
kubectl get pods -l app=log-collector -o wide

# 清理
kubectl delete ds log-collector
```

### 练习 2：创建 Job

```bash
# 创建并行 Job
cat << EOF | kubectl apply -f -
apiVersion: batch/v1
kind: Job
metadata:
  name: parallel-job
spec:
  completions: 5
  parallelism: 2
  template:
    spec:
      restartPolicy: Never
      containers:
      - name: worker
        image: busybox
        command: ["sh", "-c", "echo Processing \$HOSTNAME && sleep 10"]
EOF

# 观察执行
kubectl get pods -l job-name=parallel-job -w

# 查看完成状态
kubectl get job parallel-job

# 清理
kubectl delete job parallel-job
```

### 练习 3：创建 CronJob

```bash
# 创建定时任务
cat << EOF | kubectl apply -f -
apiVersion: batch/v1
kind: CronJob
metadata:
  name: hello-cron
spec:
  schedule: "*/1 * * * *"
  jobTemplate:
    spec:
      template:
        spec:
          restartPolicy: OnFailure
          containers:
          - name: hello
            image: busybox
            command: ["sh", "-c", "echo Hello at \$(date)"]
EOF

# 等待 1-2 分钟查看
kubectl get cj
kubectl get jobs
kubectl logs job/<job-name>

# 手动触发
kubectl create job manual-hello --from=cronjob/hello-cron

# 清理
kubectl delete cj hello-cron
kubectl delete job --all
```

## 总结

| 资源类型 | 用途 | 特点 |
|---------|------|------|
| DaemonSet | 每节点运行一个 Pod | 系统守护进程 |
| Job | 一次性任务 | 确保完成 |
| CronJob | 定时任务 | 按计划创建 Job |

## 下一步

- [kubectl 命令完全手册](../03-practice/01-kubectl-commands.md)



