# 🗄️ StatefulSet - 有状态应用

## StatefulSet vs Deployment

| 特性 | Deployment | StatefulSet |
|------|-----------|-------------|
| Pod 名称 | 随机后缀 (nginx-xyz) | 有序编号 (mysql-0, mysql-1) |
| 创建顺序 | 并行创建 | 顺序创建 (0→1→2) |
| 删除顺序 | 并行删除 | 逆序删除 (2→1→0) |
| 存储 | 共享或无状态 | 每个 Pod 独立 PVC |
| 网络标识 | 无稳定标识 | 稳定的 DNS 名称 |
| 适用场景 | 无状态应用 | 数据库、分布式系统 |

## StatefulSet 特性

```
┌─────────────────────────────────────────────────────────────────┐
│                        StatefulSet                               │
│                                                                   │
│   稳定的网络标识                                                   │
│   ┌──────────────────────────────────────────────────────────┐  │
│   │  mysql-0.mysql.default.svc.cluster.local                 │  │
│   │  mysql-1.mysql.default.svc.cluster.local                 │  │
│   │  mysql-2.mysql.default.svc.cluster.local                 │  │
│   └──────────────────────────────────────────────────────────┘  │
│                                                                   │
│   ┌─────────────┐  ┌─────────────┐  ┌─────────────┐            │
│   │  mysql-0    │  │  mysql-1    │  │  mysql-2    │            │
│   │  (Master)   │  │  (Slave)    │  │  (Slave)    │            │
│   │      │      │  │      │      │  │      │      │            │
│   │      ▼      │  │      ▼      │  │      ▼      │            │
│   │  ┌──────┐   │  │  ┌──────┐   │  │  ┌──────┐   │            │
│   │  │PVC-0 │   │  │  │PVC-1 │   │  │  │PVC-2 │   │            │
│   │  └──────┘   │  │  └──────┘   │  │  └──────┘   │            │
│   └─────────────┘  └─────────────┘  └─────────────┘            │
│                                                                   │
│   独立的持久化存储（每个 Pod 有自己的 PVC）                          │
└─────────────────────────────────────────────────────────────────┘
```

## StatefulSet YAML 详解

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: mysql
spec:
  # 关联的 Headless Service
  serviceName: mysql
  
  # 副本数
  replicas: 3
  
  # 选择器
  selector:
    matchLabels:
      app: mysql
  
  # 更新策略
  updateStrategy:
    type: RollingUpdate            # RollingUpdate 或 OnDelete
    rollingUpdate:
      partition: 0                 # 分区更新（只更新序号 >= partition 的 Pod）
  
  # Pod 管理策略
  podManagementPolicy: OrderedReady # OrderedReady 或 Parallel
  
  # Pod 模板
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
          name: mysql
        env:
        - name: MYSQL_ROOT_PASSWORD
          valueFrom:
            secretKeyRef:
              name: mysql-secret
              key: password
        volumeMounts:
        - name: data
          mountPath: /var/lib/mysql
        readinessProbe:
          exec:
            command: ["mysqladmin", "ping"]
          initialDelaySeconds: 10
          periodSeconds: 5
  
  # 卷声明模板 - 为每个 Pod 创建独立 PVC
  volumeClaimTemplates:
  - metadata:
      name: data
    spec:
      accessModes: ["ReadWriteOnce"]
      storageClassName: standard
      resources:
        requests:
          storage: 10Gi
```

## Headless Service

StatefulSet 需要配合 Headless Service 使用。

```yaml
apiVersion: v1
kind: Service
metadata:
  name: mysql
spec:
  clusterIP: None            # Headless Service
  selector:
    app: mysql
  ports:
  - port: 3306
    name: mysql
```

## DNS 解析

```bash
# Headless Service 的 DNS 解析
<pod-name>.<service-name>.<namespace>.svc.cluster.local

# 示例
mysql-0.mysql.default.svc.cluster.local
mysql-1.mysql.default.svc.cluster.local
mysql-2.mysql.default.svc.cluster.local
```

## 更新策略

### RollingUpdate（默认）

```yaml
updateStrategy:
  type: RollingUpdate
  rollingUpdate:
    partition: 0              # 分区号
```

分区更新示例：
```bash
# partition=2 时，只有 mysql-2 及以上会更新
# 用于金丝雀发布

# 逐步降低 partition 值来完成全部更新
partition: 2  # 更新 mysql-2
partition: 1  # 更新 mysql-1, mysql-2
partition: 0  # 更新所有
```

### OnDelete

```yaml
updateStrategy:
  type: OnDelete
```

手动删除 Pod 后才会创建新版本。

## Pod 管理策略

### OrderedReady（默认）

- 按序号 0、1、2 顺序创建
- 按序号 2、1、0 顺序删除
- 前一个 Pod Ready 后才创建下一个

### Parallel

```yaml
podManagementPolicy: Parallel
```

- 并行创建和删除所有 Pod
- 不等待前一个 Pod Ready

## 完整示例：MySQL 主从集群

```yaml
# mysql-secret.yaml
apiVersion: v1
kind: Secret
metadata:
  name: mysql-secret
type: Opaque
stringData:
  root-password: MySecretPassword

---
# mysql-configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: mysql-config
data:
  master.cnf: |
    [mysqld]
    log-bin=mysql-bin
    server-id=1
  slave.cnf: |
    [mysqld]
    super-read-only
    server-id=2

---
# mysql-service.yaml
apiVersion: v1
kind: Service
metadata:
  name: mysql
spec:
  clusterIP: None
  selector:
    app: mysql
  ports:
  - port: 3306

---
# mysql-service-read.yaml  
apiVersion: v1
kind: Service
metadata:
  name: mysql-read
spec:
  selector:
    app: mysql
  ports:
  - port: 3306

---
# mysql-statefulset.yaml
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
      initContainers:
      - name: init-mysql
        image: mysql:8.0
        command:
        - bash
        - "-c"
        - |
          set -ex
          # 根据序号生成 server-id
          [[ `hostname` =~ -([0-9]+)$ ]] || exit 1
          ordinal=${BASH_REMATCH[1]}
          echo [mysqld] > /mnt/conf.d/server-id.cnf
          echo server-id=$((100 + $ordinal)) >> /mnt/conf.d/server-id.cnf
          # 复制对应的配置文件
          if [[ $ordinal -eq 0 ]]; then
            cp /mnt/config-map/master.cnf /mnt/conf.d/
          else
            cp /mnt/config-map/slave.cnf /mnt/conf.d/
          fi
        volumeMounts:
        - name: conf
          mountPath: /mnt/conf.d
        - name: config-map
          mountPath: /mnt/config-map
      
      containers:
      - name: mysql
        image: mysql:8.0
        env:
        - name: MYSQL_ROOT_PASSWORD
          valueFrom:
            secretKeyRef:
              name: mysql-secret
              key: root-password
        ports:
        - containerPort: 3306
        volumeMounts:
        - name: data
          mountPath: /var/lib/mysql
        - name: conf
          mountPath: /etc/mysql/conf.d
        resources:
          requests:
            cpu: 200m
            memory: 256Mi
          limits:
            cpu: 500m
            memory: 512Mi
        livenessProbe:
          exec:
            command: ["mysqladmin", "ping", "-uroot", "-p${MYSQL_ROOT_PASSWORD}"]
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          exec:
            command: ["mysql", "-uroot", "-p${MYSQL_ROOT_PASSWORD}", "-e", "SELECT 1"]
          initialDelaySeconds: 5
          periodSeconds: 5
      
      volumes:
      - name: conf
        emptyDir: {}
      - name: config-map
        configMap:
          name: mysql-config
  
  volumeClaimTemplates:
  - metadata:
      name: data
    spec:
      accessModes: ["ReadWriteOnce"]
      resources:
        requests:
          storage: 10Gi
```

## 常用操作

```bash
# ============ 创建和管理 ============
kubectl apply -f statefulset.yaml
kubectl delete statefulset mysql

# ============ 查看状态 ============
kubectl get statefulset
kubectl get sts                          # 简写
kubectl describe sts mysql

# ============ 查看 Pod（有序命名）============
kubectl get pods -l app=mysql

# ============ 扩缩容 ============
kubectl scale sts mysql --replicas=5

# ============ 更新 ============
kubectl set image sts/mysql mysql=mysql:8.1

# 使用分区更新
kubectl patch sts mysql -p '{"spec":{"updateStrategy":{"rollingUpdate":{"partition":2}}}}'

# ============ 查看 PVC（每个 Pod 独立）============
kubectl get pvc
```

## 实践练习

### 练习：创建简单的 StatefulSet

```bash
# 1. 创建 Headless Service
cat << EOF | kubectl apply -f -
apiVersion: v1
kind: Service
metadata:
  name: nginx
spec:
  clusterIP: None
  selector:
    app: nginx
  ports:
  - port: 80
EOF

# 2. 创建 StatefulSet
cat << EOF | kubectl apply -f -
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: web
spec:
  serviceName: nginx
  replicas: 3
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx
        ports:
        - containerPort: 80
EOF

# 3. 观察有序创建
kubectl get pods -w -l app=nginx

# 4. 测试 DNS
kubectl run test --image=busybox -it --rm -- nslookup nginx
kubectl run test --image=busybox -it --rm -- nslookup web-0.nginx

# 5. 扩容
kubectl scale sts web --replicas=5

# 6. 缩容
kubectl scale sts web --replicas=2

# 7. 清理
kubectl delete sts web
kubectl delete svc nginx
```

## 最佳实践

1. **使用 Headless Service**：必须配合使用
2. **配置持久化存储**：使用 volumeClaimTemplates
3. **合理设置副本数**：根据应用需求
4. **配置健康检查**：确保有序启动
5. **注意数据备份**：删除 StatefulSet 不会删除 PVC

## 下一步

- [DaemonSet 与 Job](./08-daemonset-job.md)



