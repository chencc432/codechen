# 💾 Volume 与持久化存储

## 存储概述

容器中的文件是临时的，容器重启后数据会丢失。Volume 解决了这个问题。

```
┌─────────────────────────────────────────────────────────────────┐
│                     Kubernetes 存储体系                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│   临时存储                    持久化存储                          │
│   ┌─────────────┐            ┌─────────────────────────────┐    │
│   │  emptyDir   │            │    PersistentVolume (PV)    │    │
│   │  (Pod 生命周期)│           │        ↑                    │    │
│   └─────────────┘            │        │ 绑定                │    │
│                              │        ↓                    │    │
│   配置存储                    │ PersistentVolumeClaim (PVC)│    │
│   ┌─────────────┐            │        ↑                    │    │
│   │ ConfigMap   │            │        │ 使用                │    │
│   │   Secret    │            │        ↓                    │    │
│   └─────────────┘            │       Pod                   │    │
│                              └─────────────────────────────┘    │
│                                                                   │
│   云存储                      本地存储                            │
│   ┌─────────────┐            ┌─────────────┐                    │
│   │    AWS EBS  │            │  hostPath   │                    │
│   │    GCE PD   │            │   local     │                    │
│   │   Azure Disk│            └─────────────┘                    │
│   │     NFS     │                                                │
│   └─────────────┘                                                │
└─────────────────────────────────────────────────────────────────┘
```

## Volume 类型

### 1. emptyDir（临时存储）

Pod 内所有容器共享，Pod 删除时数据丢失。

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: emptydir-demo
spec:
  containers:
  - name: writer
    image: busybox
    command: ["sh", "-c", "echo 'Hello' > /data/hello.txt && sleep 3600"]
    volumeMounts:
    - name: shared-data
      mountPath: /data
  
  - name: reader
    image: busybox
    command: ["sh", "-c", "cat /data/hello.txt && sleep 3600"]
    volumeMounts:
    - name: shared-data
      mountPath: /data
  
  volumes:
  - name: shared-data
    emptyDir: {}
    # 或使用内存
    # emptyDir:
    #   medium: Memory
    #   sizeLimit: 100Mi
```

用途：
- 容器间共享数据
- 缓存数据
- 临时工作空间

### 2. hostPath（节点路径）

挂载节点的文件或目录到 Pod。

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: hostpath-demo
spec:
  containers:
  - name: app
    image: nginx
    volumeMounts:
    - name: host-logs
      mountPath: /var/log/app
  volumes:
  - name: host-logs
    hostPath:
      path: /var/log/app-logs
      type: DirectoryOrCreate     # 类型
```

hostPath type 选项：

| type | 说明 |
|------|------|
| "" | 不检查（默认）|
| DirectoryOrCreate | 目录不存在则创建 |
| Directory | 目录必须存在 |
| FileOrCreate | 文件不存在则创建 |
| File | 文件必须存在 |
| Socket | Unix Socket 必须存在 |

⚠️ **注意**：hostPath 有安全风险，生产环境谨慎使用。

### 3. ConfigMap 和 Secret 作为 Volume

```yaml
# 已在前一章详细介绍
volumes:
- name: config
  configMap:
    name: my-config
- name: secret
  secret:
    secretName: my-secret
```

## PersistentVolume (PV) 和 PersistentVolumeClaim (PVC)

### 概念说明

```
管理员                              用户/开发者
   │                                    │
   ▼                                    ▼
┌──────────────────┐            ┌───────────────────┐
│ PersistentVolume │ ◄────绑定───►│ PersistentVolume  │
│      (PV)        │            │     Claim (PVC)   │
│                  │            │                   │
│ - 存储类型       │            │ - 请求大小        │
│ - 容量大小       │            │ - 访问模式        │
│ - 访问模式       │            │ - 存储类         │
│ - 回收策略       │            └────────┬──────────┘
└──────────────────┘                     │
                                         ▼
                                    ┌─────────┐
                                    │   Pod   │
                                    └─────────┘
```

### 访问模式 (Access Modes)

| 模式 | 缩写 | 说明 |
|------|------|------|
| ReadWriteOnce | RWO | 单节点读写 |
| ReadOnlyMany | ROX | 多节点只读 |
| ReadWriteMany | RWX | 多节点读写 |
| ReadWriteOncePod | RWOP | 单 Pod 读写（K8s 1.22+）|

### 回收策略 (Reclaim Policy)

| 策略 | 说明 |
|------|------|
| Retain | 保留数据，需手动清理 |
| Delete | 自动删除存储资源 |
| Recycle | 清空数据后重用（已废弃）|

### PV 示例

```yaml
apiVersion: v1
kind: PersistentVolume
metadata:
  name: my-pv
spec:
  capacity:
    storage: 10Gi
  accessModes:
  - ReadWriteOnce
  persistentVolumeReclaimPolicy: Retain
  storageClassName: manual
  
  # NFS 示例
  nfs:
    server: nfs-server.example.com
    path: /exports/data
  
  # hostPath 示例（仅测试用）
  # hostPath:
  #   path: /mnt/data
```

### PVC 示例

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: my-pvc
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 5Gi
  storageClassName: manual
  
  # 可选：指定 PV
  # volumeName: my-pv
```

### Pod 使用 PVC

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: pvc-pod
spec:
  containers:
  - name: app
    image: nginx
    volumeMounts:
    - name: data
      mountPath: /usr/share/nginx/html
  volumes:
  - name: data
    persistentVolumeClaim:
      claimName: my-pvc
```

## StorageClass（存储类）

StorageClass 实现动态卷供应。

### StorageClass 定义

```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: fast-storage
  annotations:
    storageclass.kubernetes.io/is-default-class: "true"
provisioner: kubernetes.io/aws-ebs     # 存储供应商
parameters:
  type: gp3
  fsType: ext4
reclaimPolicy: Delete
volumeBindingMode: WaitForFirstConsumer
allowVolumeExpansion: true
```

### 常见 Provisioner

| 云厂商 | Provisioner |
|--------|-------------|
| AWS EBS | kubernetes.io/aws-ebs |
| GCE PD | kubernetes.io/gce-pd |
| Azure Disk | kubernetes.io/azure-disk |
| 本地存储 | kubernetes.io/no-provisioner |

### 使用 StorageClass

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: dynamic-pvc
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 20Gi
  storageClassName: fast-storage     # 指定 StorageClass
```

## 完整示例：StatefulSet 使用持久化存储

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: mysql
spec:
  serviceName: mysql
  replicas: 3
  selector:
    matchLabels:
      app: mysql
  template:
    metadata:
      labels:
        app: mysql
    spec:
      containers:
      - name: mysql
        image: mysql:8.0
        ports:
        - containerPort: 3306
        env:
        - name: MYSQL_ROOT_PASSWORD
          valueFrom:
            secretKeyRef:
              name: mysql-secret
              key: password
        volumeMounts:
        - name: data
          mountPath: /var/lib/mysql
  
  # 卷声明模板 - 为每个 Pod 创建独立的 PVC
  volumeClaimTemplates:
  - metadata:
      name: data
    spec:
      accessModes: ["ReadWriteOnce"]
      storageClassName: fast-storage
      resources:
        requests:
          storage: 20Gi
```

## PV 生命周期

```
┌─────────────────────────────────────────────────────────────────┐
│                    PV 生命周期                                   │
│                                                                   │
│  ┌───────────┐    ┌───────────┐    ┌───────────┐               │
│  │ Available │───>│   Bound   │───>│ Released  │               │
│  │  (可用)    │    │  (已绑定)  │    │  (已释放)  │               │
│  └───────────┘    └───────────┘    └─────┬─────┘               │
│       ▲                                   │                      │
│       │                                   ▼                      │
│       │              ┌─────────────────────────────┐            │
│       │              │ 根据 Reclaim Policy:        │            │
│       │              │ - Retain: 保持 Released    │            │
│       │              │ - Delete: 删除 PV          │            │
│       │              │ - Recycle: 回到 Available  │            │
│       │              └─────────────────────────────┘            │
│       │                           │                              │
│       └───────────────────────────┘                              │
│                                                                   │
│  Failed 状态: 自动回收失败                                        │
└─────────────────────────────────────────────────────────────────┘
```

## 常用操作命令

```bash
# ============ PV 操作 ============
# 查看 PV
kubectl get pv
kubectl get pv -o wide
kubectl describe pv my-pv

# 创建 PV
kubectl apply -f pv.yaml

# 删除 PV
kubectl delete pv my-pv

# ============ PVC 操作 ============
# 查看 PVC
kubectl get pvc
kubectl get pvc -n my-namespace
kubectl describe pvc my-pvc

# 创建 PVC
kubectl apply -f pvc.yaml

# 删除 PVC
kubectl delete pvc my-pvc

# ============ StorageClass 操作 ============
# 查看 StorageClass
kubectl get storageclass
kubectl get sc                           # 简写
kubectl describe sc fast-storage

# 设置默认 StorageClass
kubectl patch storageclass fast-storage -p '{"metadata": {"annotations":{"storageclass.kubernetes.io/is-default-class":"true"}}}'
```

## 实践练习

### 练习 1：使用 emptyDir

```bash
# 创建共享存储的 Pod
cat << EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: emptydir-pod
spec:
  containers:
  - name: producer
    image: busybox
    command: ["sh", "-c", "while true; do date >> /shared/log.txt; sleep 5; done"]
    volumeMounts:
    - name: shared
      mountPath: /shared
  - name: consumer
    image: busybox
    command: ["sh", "-c", "tail -f /shared/log.txt"]
    volumeMounts:
    - name: shared
      mountPath: /shared
  volumes:
  - name: shared
    emptyDir: {}
EOF

# 查看日志
kubectl logs emptydir-pod -c consumer -f

# 清理
kubectl delete pod emptydir-pod
```

### 练习 2：PV 和 PVC

```bash
# 1. 创建 PV
cat << EOF | kubectl apply -f -
apiVersion: v1
kind: PersistentVolume
metadata:
  name: test-pv
spec:
  capacity:
    storage: 1Gi
  accessModes:
  - ReadWriteOnce
  persistentVolumeReclaimPolicy: Retain
  storageClassName: manual
  hostPath:
    path: /tmp/test-pv
EOF

# 2. 创建 PVC
cat << EOF | kubectl apply -f -
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: test-pvc
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 500Mi
  storageClassName: manual
EOF

# 3. 查看绑定状态
kubectl get pv,pvc

# 4. 使用 PVC 的 Pod
cat << EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: pvc-pod
spec:
  containers:
  - name: app
    image: nginx
    volumeMounts:
    - name: data
      mountPath: /usr/share/nginx/html
  volumes:
  - name: data
    persistentVolumeClaim:
      claimName: test-pvc
EOF

# 5. 写入数据
kubectl exec pvc-pod -- sh -c 'echo "Hello PVC" > /usr/share/nginx/html/index.html'

# 6. 验证
kubectl exec pvc-pod -- cat /usr/share/nginx/html/index.html

# 7. 清理
kubectl delete pod pvc-pod
kubectl delete pvc test-pvc
kubectl delete pv test-pv
```

## 最佳实践

1. **使用 StorageClass 动态供应**：避免手动管理 PV
2. **设置资源配额**：限制每个命名空间的存储使用
3. **选择合适的访问模式**：根据应用需求选择 RWO/RWX
4. **配置合理的回收策略**：生产环境通常使用 Retain
5. **监控存储使用**：防止存储耗尽

## 下一步

- [Namespace - 资源隔离](./06-namespace.md)



