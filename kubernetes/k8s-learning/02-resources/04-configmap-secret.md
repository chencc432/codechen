# ⚙️ ConfigMap 与 Secret - 配置管理

## 概述

ConfigMap 和 Secret 用于将配置数据与应用程序分离，遵循"十二要素应用"原则。

| 类型 | 用途 | 存储方式 | 示例 |
|------|------|---------|------|
| ConfigMap | 非敏感配置 | 明文 | 配置文件、环境变量 |
| Secret | 敏感数据 | Base64 编码 | 密码、Token、证书 |

## ConfigMap

### 创建 ConfigMap

#### 方式 1：从字面值创建

```bash
# 创建键值对
kubectl create configmap app-config \
  --from-literal=DATABASE_HOST=mysql.example.com \
  --from-literal=DATABASE_PORT=3306

# 查看
kubectl get configmap app-config -o yaml
```

#### 方式 2：从文件创建

```bash
# 准备配置文件
cat > app.properties << EOF
database.host=mysql.example.com
database.port=3306
database.name=myapp
EOF

# 从文件创建
kubectl create configmap app-config --from-file=app.properties

# 指定 key 名
kubectl create configmap app-config --from-file=config.properties=app.properties
```

#### 方式 3：从目录创建

```bash
# 创建配置目录
mkdir config
echo "debug=true" > config/debug.conf
echo "log_level=INFO" > config/logging.conf

# 从目录创建（每个文件成为一个 key）
kubectl create configmap app-config --from-dir=config
```

#### 方式 4：YAML 定义

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: app-config
  namespace: default
data:
  # 简单键值
  DATABASE_HOST: mysql.example.com
  DATABASE_PORT: "3306"
  
  # 多行配置文件
  app.properties: |
    database.host=mysql.example.com
    database.port=3306
    database.name=myapp
  
  # JSON 配置
  config.json: |
    {
      "database": {
        "host": "mysql.example.com",
        "port": 3306
      }
    }
```

### 使用 ConfigMap

#### 方式 1：环境变量（单个值）

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: app
spec:
  containers:
  - name: app
    image: myapp
    env:
    - name: DATABASE_HOST
      valueFrom:
        configMapKeyRef:
          name: app-config
          key: DATABASE_HOST
    - name: DATABASE_PORT
      valueFrom:
        configMapKeyRef:
          name: app-config
          key: DATABASE_PORT
```

#### 方式 2：环境变量（全部导入）

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: app
spec:
  containers:
  - name: app
    image: myapp
    envFrom:
    - configMapRef:
        name: app-config
      prefix: CONFIG_             # 可选：添加前缀
```

#### 方式 3：挂载为文件

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: app
spec:
  containers:
  - name: app
    image: myapp
    volumeMounts:
    - name: config-volume
      mountPath: /etc/config
  volumes:
  - name: config-volume
    configMap:
      name: app-config
      # 可选：只挂载特定 key
      items:
      - key: app.properties
        path: application.properties
```

#### 方式 4：挂载为单个文件（不覆盖目录）

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: app
spec:
  containers:
  - name: app
    image: nginx
    volumeMounts:
    - name: config-volume
      mountPath: /etc/nginx/nginx.conf
      subPath: nginx.conf               # 只挂载单个文件
  volumes:
  - name: config-volume
    configMap:
      name: nginx-config
```

### ConfigMap 热更新

```yaml
# 挂载为 Volume 时，ConfigMap 更新会自动同步到 Pod（有延迟）
# 注意：使用 subPath 的不会自动更新

# 手动触发重启以应用新配置
kubectl rollout restart deployment myapp
```

## Secret

### Secret 类型

| 类型 | 用途 | 示例 |
|------|------|------|
| Opaque | 通用类型（默认）| 密码、API Key |
| kubernetes.io/tls | TLS 证书 | HTTPS 证书 |
| kubernetes.io/dockerconfigjson | Docker 仓库认证 | 私有镜像仓库 |
| kubernetes.io/basic-auth | 基本认证 | 用户名密码 |
| kubernetes.io/ssh-auth | SSH 认证 | SSH 私钥 |
| kubernetes.io/service-account-token | SA Token | 服务账户 |

### 创建 Secret

#### 方式 1：通用 Secret（从字面值）

```bash
kubectl create secret generic db-secret \
  --from-literal=username=admin \
  --from-literal=password='S3cret!Pass'
```

#### 方式 2：从文件创建

```bash
# 准备密钥文件
echo -n 'admin' > ./username
echo -n 'S3cret!Pass' > ./password

kubectl create secret generic db-secret \
  --from-file=username \
  --from-file=password
```

#### 方式 3：TLS Secret

```bash
kubectl create secret tls my-tls-secret \
  --cert=path/to/tls.crt \
  --key=path/to/tls.key
```

#### 方式 4：Docker Registry Secret

```bash
kubectl create secret docker-registry regcred \
  --docker-server=https://index.docker.io/v1/ \
  --docker-username=myuser \
  --docker-password=mypassword \
  --docker-email=myemail@example.com
```

#### 方式 5：YAML 定义

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: db-secret
type: Opaque
data:
  # 值必须是 Base64 编码
  username: YWRtaW4=           # echo -n 'admin' | base64
  password: UzNjcmV0IVBhc3M=   # echo -n 'S3cret!Pass' | base64

---
# 使用 stringData（明文，自动编码）
apiVersion: v1
kind: Secret
metadata:
  name: db-secret-plain
type: Opaque
stringData:
  username: admin
  password: S3cret!Pass
```

### 使用 Secret

#### 方式 1：环境变量

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: app
spec:
  containers:
  - name: app
    image: myapp
    env:
    - name: DB_USERNAME
      valueFrom:
        secretKeyRef:
          name: db-secret
          key: username
    - name: DB_PASSWORD
      valueFrom:
        secretKeyRef:
          name: db-secret
          key: password
```

#### 方式 2：全部导入

```yaml
spec:
  containers:
  - name: app
    image: myapp
    envFrom:
    - secretRef:
        name: db-secret
```

#### 方式 3：挂载为文件

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: app
spec:
  containers:
  - name: app
    image: myapp
    volumeMounts:
    - name: secret-volume
      mountPath: /etc/secrets
      readOnly: true
  volumes:
  - name: secret-volume
    secret:
      secretName: db-secret
      defaultMode: 0400         # 文件权限
```

#### 方式 4：使用 Docker Registry Secret

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: private-image-pod
spec:
  containers:
  - name: app
    image: private-registry.example.com/myapp:latest
  imagePullSecrets:
  - name: regcred
```

### 在 ServiceAccount 中配置镜像拉取

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: myapp-sa
imagePullSecrets:
- name: regcred

---
apiVersion: v1
kind: Pod
metadata:
  name: app
spec:
  serviceAccountName: myapp-sa
  containers:
  - name: app
    image: private-registry.example.com/myapp:latest
```

## 完整示例

### 应用配置完整示例

```yaml
# configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: app-config
data:
  APP_ENV: production
  LOG_LEVEL: INFO
  app.properties: |
    server.port=8080
    server.host=0.0.0.0

---
# secret.yaml
apiVersion: v1
kind: Secret
metadata:
  name: app-secret
type: Opaque
stringData:
  database-url: postgresql://user:password@db:5432/myapp
  api-key: sk-xxxxxxxxxxxx

---
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: myapp
spec:
  replicas: 2
  selector:
    matchLabels:
      app: myapp
  template:
    metadata:
      labels:
        app: myapp
    spec:
      containers:
      - name: myapp
        image: myapp:1.0
        ports:
        - containerPort: 8080
        
        # 环境变量
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: app-secret
              key: database-url
        - name: API_KEY
          valueFrom:
            secretKeyRef:
              name: app-secret
              key: api-key
        
        # 批量导入环境变量
        envFrom:
        - configMapRef:
            name: app-config
        
        # 挂载配置文件
        volumeMounts:
        - name: config-volume
          mountPath: /etc/app
        - name: secret-volume
          mountPath: /etc/secrets
          readOnly: true
      
      volumes:
      - name: config-volume
        configMap:
          name: app-config
          items:
          - key: app.properties
            path: application.properties
      - name: secret-volume
        secret:
          secretName: app-secret
```

## 常用操作命令

```bash
# ============ ConfigMap ============
# 创建
kubectl create configmap myconfig --from-literal=key=value
kubectl create configmap myconfig --from-file=config.properties

# 查看
kubectl get configmaps
kubectl get cm                               # 简写
kubectl describe configmap myconfig
kubectl get configmap myconfig -o yaml

# 编辑
kubectl edit configmap myconfig

# 删除
kubectl delete configmap myconfig

# ============ Secret ============
# 创建
kubectl create secret generic mysecret --from-literal=password=secret
kubectl create secret tls tls-secret --cert=cert.crt --key=cert.key
kubectl create secret docker-registry regcred --docker-server=... 

# 查看（注意：会显示 Base64 编码的值）
kubectl get secrets
kubectl describe secret mysecret
kubectl get secret mysecret -o yaml

# 解码查看值
kubectl get secret mysecret -o jsonpath='{.data.password}' | base64 -d

# 删除
kubectl delete secret mysecret
```

## 实践练习

### 练习 1：使用 ConfigMap

```bash
# 1. 创建 ConfigMap
kubectl create configmap web-config \
  --from-literal=TITLE="Hello Kubernetes" \
  --from-literal=COLOR="blue"

# 2. 创建使用 ConfigMap 的 Pod
cat << EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: web-pod
spec:
  containers:
  - name: web
    image: busybox
    command: ["sh", "-c", "echo Title: \$TITLE, Color: \$COLOR && sleep 3600"]
    envFrom:
    - configMapRef:
        name: web-config
EOF

# 3. 验证
kubectl logs web-pod

# 4. 清理
kubectl delete pod web-pod
kubectl delete configmap web-config
```

### 练习 2：使用 Secret

```bash
# 1. 创建 Secret
kubectl create secret generic db-credentials \
  --from-literal=username=dbadmin \
  --from-literal=password=secretpassword123

# 2. 创建使用 Secret 的 Pod
cat << EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: db-pod
spec:
  containers:
  - name: app
    image: busybox
    command: ["sh", "-c", "echo User: \$DB_USER, Pass: \$DB_PASS && sleep 3600"]
    env:
    - name: DB_USER
      valueFrom:
        secretKeyRef:
          name: db-credentials
          key: username
    - name: DB_PASS
      valueFrom:
        secretKeyRef:
          name: db-credentials
          key: password
EOF

# 3. 验证
kubectl logs db-pod

# 4. 清理
kubectl delete pod db-pod
kubectl delete secret db-credentials
```

### 练习 3：挂载配置文件

```bash
# 1. 创建带配置文件的 ConfigMap
kubectl create configmap nginx-config --from-literal=nginx.conf='
server {
    listen 80;
    location / {
        return 200 "Hello from ConfigMap!";
    }
}'

# 2. 创建挂载配置的 Pod
cat << EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: nginx-custom
spec:
  containers:
  - name: nginx
    image: nginx
    volumeMounts:
    - name: config
      mountPath: /etc/nginx/conf.d
  volumes:
  - name: config
    configMap:
      name: nginx-config
      items:
      - key: nginx.conf
        path: default.conf
EOF

# 3. 测试
kubectl port-forward nginx-custom 8080:80 &
curl http://localhost:8080

# 4. 清理
kubectl delete pod nginx-custom
kubectl delete configmap nginx-config
```

## 安全注意事项

1. **Secret 不是加密的**：只是 Base64 编码，需要配合 RBAC 限制访问
2. **启用 etcd 加密**：保护存储在 etcd 中的 Secret
3. **使用外部密钥管理**：生产环境考虑使用 Vault、AWS Secrets Manager 等
4. **最小权限原则**：限制对 Secret 的访问权限
5. **不要将 Secret 提交到版本控制**：使用工具如 sealed-secrets 或 SOPS

## 下一步

- [Volume 与持久化存储](./05-volume.md)



