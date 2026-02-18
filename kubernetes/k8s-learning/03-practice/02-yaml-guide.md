# 📄 YAML 编写规范与技巧

## YAML 基础语法

### 数据类型

```yaml
# 字符串
name: nginx
name: "nginx"
name: 'nginx'
description: "This is a \"quoted\" string"

# 多行字符串
# | 保留换行符
script: |
  #!/bin/bash
  echo "Hello"
  echo "World"

# > 折叠换行为空格
description: >
  This is a very long
  description that spans
  multiple lines.

# 数字
replicas: 3
port: 80
cpu: 0.5

# 布尔值
enabled: true
debug: false

# 空值
value: null
value: ~

# 列表
ports:
  - 80
  - 443
  - 8080

# 内联列表
ports: [80, 443, 8080]

# 字典/映射
metadata:
  name: nginx
  namespace: default

# 内联字典
metadata: {name: nginx, namespace: default}
```

### 常见错误

```yaml
# ❌ 缩进错误
spec:
containers:        # 应该缩进
- name: nginx

# ✅ 正确
spec:
  containers:
  - name: nginx

# ❌ Tab 缩进（YAML 只支持空格）
spec:
	containers:    # 使用了 Tab

# ✅ 正确（使用空格）
spec:
  containers:

# ❌ 冒号后缺少空格
name:nginx

# ✅ 正确
name: nginx
```

## Kubernetes YAML 结构

### 必需字段

```yaml
apiVersion: v1          # API 版本
kind: Pod               # 资源类型
metadata:               # 元数据
  name: my-pod          # 资源名称
spec:                   # 规约（期望状态）
  # ...
```

### 查找 API 版本

```bash
# 查看资源对应的 API 版本
kubectl api-resources | grep -i pod
kubectl api-resources | grep -i deployment

# 常见 API 版本
v1                    # 核心 API（Pod, Service, ConfigMap）
apps/v1               # Deployment, StatefulSet, DaemonSet
batch/v1              # Job, CronJob
networking.k8s.io/v1  # Ingress, NetworkPolicy
storage.k8s.io/v1     # StorageClass
rbac.authorization.k8s.io/v1  # Role, RoleBinding
```

### 使用 kubectl explain

```bash
# 查看资源结构
kubectl explain pod
kubectl explain pod.spec
kubectl explain pod.spec.containers
kubectl explain pod.spec.containers.ports

# 递归显示所有字段
kubectl explain pod --recursive
kubectl explain deployment.spec --recursive | head -50
```

## 常用资源模板

### Pod 模板

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: my-pod
  namespace: default
  labels:
    app: myapp
    version: v1
  annotations:
    description: "My application pod"
spec:
  restartPolicy: Always
  containers:
  - name: app
    image: nginx:1.21
    imagePullPolicy: IfNotPresent
    ports:
    - containerPort: 80
      name: http
    env:
    - name: ENV_VAR
      value: "value"
    resources:
      requests:
        cpu: "100m"
        memory: "128Mi"
      limits:
        cpu: "200m"
        memory: "256Mi"
    livenessProbe:
      httpGet:
        path: /healthz
        port: 80
      initialDelaySeconds: 10
      periodSeconds: 10
    readinessProbe:
      httpGet:
        path: /ready
        port: 80
      initialDelaySeconds: 5
      periodSeconds: 5
    volumeMounts:
    - name: config
      mountPath: /etc/config
  volumes:
  - name: config
    configMap:
      name: my-config
```

### Deployment 模板

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-deployment
  labels:
    app: myapp
spec:
  replicas: 3
  selector:
    matchLabels:
      app: myapp
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1
      maxSurge: 1
  template:
    metadata:
      labels:
        app: myapp
    spec:
      containers:
      - name: app
        image: nginx:1.21
        ports:
        - containerPort: 80
        resources:
          requests:
            cpu: "100m"
            memory: "128Mi"
          limits:
            cpu: "200m"
            memory: "256Mi"
        readinessProbe:
          httpGet:
            path: /
            port: 80
          initialDelaySeconds: 5
          periodSeconds: 5
```

### Service 模板

```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-service
  labels:
    app: myapp
spec:
  type: ClusterIP
  selector:
    app: myapp
  ports:
  - name: http
    port: 80
    targetPort: 80
    protocol: TCP
```

### 完整应用模板

```yaml
# 一个文件包含多个资源，用 --- 分隔
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: app-config
data:
  APP_ENV: production
  LOG_LEVEL: INFO

---
apiVersion: v1
kind: Secret
metadata:
  name: app-secret
type: Opaque
stringData:
  database-url: postgresql://user:pass@db:5432/app

---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: app
spec:
  replicas: 3
  selector:
    matchLabels:
      app: myapp
  template:
    metadata:
      labels:
        app: myapp
    spec:
      containers:
      - name: app
        image: myapp:1.0
        ports:
        - containerPort: 8080
        envFrom:
        - configMapRef:
            name: app-config
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: app-secret
              key: database-url

---
apiVersion: v1
kind: Service
metadata:
  name: app
spec:
  selector:
    app: myapp
  ports:
  - port: 80
    targetPort: 8080
```

## YAML 技巧

### 锚点和别名（复用配置）

```yaml
# 定义锚点
defaults: &defaults
  resources:
    requests:
      cpu: "100m"
      memory: "128Mi"
    limits:
      cpu: "200m"
      memory: "256Mi"

spec:
  containers:
  - name: app1
    image: app1:latest
    <<: *defaults          # 引用锚点
  
  - name: app2
    image: app2:latest
    <<: *defaults          # 复用相同配置
```

### 环境变量引用

```yaml
env:
# 直接值
- name: SIMPLE_VAR
  value: "simple value"

# 从 ConfigMap 引用
- name: CONFIG_VAR
  valueFrom:
    configMapKeyRef:
      name: my-config
      key: config-key

# 从 Secret 引用
- name: SECRET_VAR
  valueFrom:
    secretKeyRef:
      name: my-secret
      key: secret-key

# 从 Pod 字段引用
- name: POD_NAME
  valueFrom:
    fieldRef:
      fieldPath: metadata.name
- name: POD_IP
  valueFrom:
    fieldRef:
      fieldPath: status.podIP
- name: NODE_NAME
  valueFrom:
    fieldRef:
      fieldPath: spec.nodeName

# 从容器资源引用
- name: CPU_LIMIT
  valueFrom:
    resourceFieldRef:
      containerName: app
      resource: limits.cpu
```

### 生成 YAML

```bash
# 从命令生成 YAML
kubectl run nginx --image=nginx --dry-run=client -o yaml > pod.yaml

kubectl create deployment nginx --image=nginx \
  --dry-run=client -o yaml > deployment.yaml

kubectl expose deployment nginx --port=80 \
  --dry-run=client -o yaml > service.yaml

# 导出现有资源
kubectl get deployment nginx -o yaml > current-deploy.yaml
```

### 验证 YAML

```bash
# 客户端验证（不发送到服务器）
kubectl apply -f manifest.yaml --dry-run=client

# 服务端验证（发送到服务器但不应用）
kubectl apply -f manifest.yaml --dry-run=server

# 查看差异
kubectl diff -f manifest.yaml

# 使用 kubeval 验证（需要安装）
kubeval manifest.yaml

# 使用 kubeconform 验证（需要安装）
kubeconform manifest.yaml
```

## 常用字段速查

### 容器配置

```yaml
containers:
- name: app
  image: nginx:1.21
  imagePullPolicy: Always/IfNotPresent/Never
  command: ["sh", "-c"]           # 覆盖 ENTRYPOINT
  args: ["echo hello"]            # 覆盖 CMD
  workingDir: /app
  ports:
  - containerPort: 80
    name: http
    protocol: TCP
  env: []
  envFrom: []
  resources: {}
  volumeMounts: []
  livenessProbe: {}
  readinessProbe: {}
  startupProbe: {}
  lifecycle:
    postStart:
      exec:
        command: ["/bin/sh", "-c", "echo started"]
    preStop:
      exec:
        command: ["/bin/sh", "-c", "nginx -s quit"]
  securityContext:
    runAsUser: 1000
    runAsGroup: 3000
    readOnlyRootFilesystem: true
```

### Pod 配置

```yaml
spec:
  restartPolicy: Always/OnFailure/Never
  serviceAccountName: my-sa
  automountServiceAccountToken: false
  nodeName: node1
  nodeSelector:
    disktype: ssd
  affinity: {}
  tolerations: []
  hostNetwork: false
  dnsPolicy: ClusterFirst
  dnsConfig: {}
  securityContext:
    runAsUser: 1000
    fsGroup: 2000
  initContainers: []
  containers: []
  volumes: []
  imagePullSecrets:
  - name: regcred
```

## 最佳实践

1. **使用版本控制**：将 YAML 文件纳入 Git 管理
2. **设置资源限制**：始终配置 resources
3. **使用标签**：便于筛选和管理
4. **配置健康检查**：livenessProbe 和 readinessProbe
5. **分离配置**：使用 ConfigMap 和 Secret
6. **使用命名空间**：隔离不同环境
7. **版本化镜像**：不要使用 `latest` 标签
8. **验证 YAML**：部署前进行验证

## 下一步

- [常见运维操作指南](./03-operations.md)



