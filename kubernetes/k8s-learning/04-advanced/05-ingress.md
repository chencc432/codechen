# 🚪 Ingress 与流量管理

## Ingress 概述

Ingress 提供 HTTP(S) 路由到集群内服务的规则。

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Ingress 架构                                  │
├─────────────────────────────────────────────────────────────────────┤
│                                                                       │
│   Internet                                                            │
│       │                                                               │
│       ▼                                                               │
│   ┌─────────────────────────────────────────────────────────────┐  │
│   │              Ingress Controller                              │  │
│   │         (Nginx/Traefik/HAProxy/...)                         │  │
│   └─────────────────────────┬───────────────────────────────────┘  │
│                             │                                        │
│                             ▼                                        │
│   ┌─────────────────────────────────────────────────────────────┐  │
│   │                    Ingress 规则                              │  │
│   │                                                               │  │
│   │   foo.example.com/api   ──→  Service: api     (port: 80)    │  │
│   │   foo.example.com/web   ──→  Service: web     (port: 80)    │  │
│   │   bar.example.com       ──→  Service: bar-app (port: 8080)  │  │
│   │                                                               │  │
│   └─────────────────────────────────────────────────────────────┘  │
│                                                                       │
└─────────────────────────────────────────────────────────────────────┘
```

## 安装 Ingress Controller

### Nginx Ingress Controller

```bash
# 使用 Helm
helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
helm install nginx-ingress ingress-nginx/ingress-nginx

# 或使用 YAML
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/cloud/deploy.yaml

# Minikube
minikube addons enable ingress
```

## 基本 Ingress 配置

### 简单路由

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: simple-ingress
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
spec:
  ingressClassName: nginx
  rules:
  - host: myapp.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: myapp-service
            port:
              number: 80
```

### 多路径路由

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: multi-path-ingress
spec:
  ingressClassName: nginx
  rules:
  - host: app.example.com
    http:
      paths:
      - path: /api
        pathType: Prefix
        backend:
          service:
            name: api-service
            port:
              number: 80
      - path: /web
        pathType: Prefix
        backend:
          service:
            name: web-service
            port:
              number: 80
      - path: /
        pathType: Prefix
        backend:
          service:
            name: default-service
            port:
              number: 80
```

### 多域名路由

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: multi-host-ingress
spec:
  ingressClassName: nginx
  rules:
  - host: api.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: api-service
            port:
              number: 80
  - host: web.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: web-service
            port:
              number: 80
```

## TLS/HTTPS 配置

### 创建 TLS Secret

```bash
# 创建自签名证书（测试用）
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout tls.key -out tls.crt \
  -subj "/CN=*.example.com"

# 创建 Secret
kubectl create secret tls example-tls --cert=tls.crt --key=tls.key
```

### 配置 TLS

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: tls-ingress
  annotations:
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
spec:
  ingressClassName: nginx
  tls:
  - hosts:
    - secure.example.com
    secretName: example-tls
  rules:
  - host: secure.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: secure-service
            port:
              number: 80
```

## 常用注解

### Nginx Ingress 注解

```yaml
metadata:
  annotations:
    # 重写路径
    nginx.ingress.kubernetes.io/rewrite-target: /
    
    # SSL 重定向
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    
    # 限流
    nginx.ingress.kubernetes.io/limit-rps: "10"
    nginx.ingress.kubernetes.io/limit-connections: "5"
    
    # 超时配置
    nginx.ingress.kubernetes.io/proxy-connect-timeout: "30"
    nginx.ingress.kubernetes.io/proxy-read-timeout: "30"
    nginx.ingress.kubernetes.io/proxy-send-timeout: "30"
    
    # 请求体大小
    nginx.ingress.kubernetes.io/proxy-body-size: "10m"
    
    # CORS
    nginx.ingress.kubernetes.io/enable-cors: "true"
    nginx.ingress.kubernetes.io/cors-allow-origin: "*"
    
    # 认证
    nginx.ingress.kubernetes.io/auth-type: basic
    nginx.ingress.kubernetes.io/auth-secret: basic-auth
    
    # WebSocket 支持
    nginx.ingress.kubernetes.io/proxy-read-timeout: "3600"
    nginx.ingress.kubernetes.io/proxy-send-timeout: "3600"
    
    # 会话粘性
    nginx.ingress.kubernetes.io/affinity: "cookie"
    nginx.ingress.kubernetes.io/session-cookie-name: "route"
```

## Path 类型

```yaml
# Exact: 精确匹配
pathType: Exact
path: /foo
# 匹配 /foo，不匹配 /foo/ 或 /foo/bar

# Prefix: 前缀匹配
pathType: Prefix
path: /foo
# 匹配 /foo、/foo/、/foo/bar

# ImplementationSpecific: 由 Ingress Controller 决定
pathType: ImplementationSpecific
```

## 完整示例

```yaml
---
# 后端服务
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
spec:
  replicas: 2
  selector:
    matchLabels:
      app: web
  template:
    metadata:
      labels:
        app: web
    spec:
      containers:
      - name: nginx
        image: nginx
        ports:
        - containerPort: 80

---
apiVersion: v1
kind: Service
metadata:
  name: web-service
spec:
  selector:
    app: web
  ports:
  - port: 80
    targetPort: 80

---
# Ingress
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: web-ingress
  annotations:
    nginx.ingress.kubernetes.io/ssl-redirect: "false"
spec:
  ingressClassName: nginx
  rules:
  - host: web.local
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: web-service
            port:
              number: 80
```

## 操作命令

```bash
# 查看 Ingress
kubectl get ingress
kubectl describe ingress <name>

# 查看 Ingress Controller
kubectl get pods -n ingress-nginx

# 查看 Ingress Controller 日志
kubectl logs -n ingress-nginx -l app.kubernetes.io/name=ingress-nginx

# 测试
curl -H "Host: myapp.example.com" http://<ingress-controller-ip>/
```

## 下一步

- [client-go 入门](../05-client-go/01-introduction.md)



