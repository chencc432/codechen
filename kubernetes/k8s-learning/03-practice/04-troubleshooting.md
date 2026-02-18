# 🔍 故障排查与调试

## 故障排查流程

```
┌────────────────────────────────────────────────────────────────────┐
│                        故障排查流程                                  │
├────────────────────────────────────────────────────────────────────┤
│                                                                      │
│   1. 确认问题                                                        │
│      └─→ kubectl get pods/svc/deploy 查看状态                       │
│                                                                      │
│   2. 查看详细信息                                                    │
│      └─→ kubectl describe <resource>                                │
│                                                                      │
│   3. 查看日志                                                        │
│      └─→ kubectl logs <pod>                                         │
│                                                                      │
│   4. 查看事件                                                        │
│      └─→ kubectl get events                                         │
│                                                                      │
│   5. 进入容器调试                                                    │
│      └─→ kubectl exec -it <pod> -- sh                               │
│                                                                      │
│   6. 使用调试工具                                                    │
│      └─→ kubectl debug                                              │
│                                                                      │
└────────────────────────────────────────────────────────────────────┘
```

## Pod 常见问题

### 1. ImagePullBackOff / ErrImagePull

**症状**：Pod 卡在 Pending 或 ImagePullBackOff 状态

**排查步骤**：
```bash
# 查看详细错误
kubectl describe pod <pod-name>

# 检查镜像名称是否正确
kubectl get pod <pod-name> -o jsonpath='{.spec.containers[*].image}'

# 在节点上测试拉取镜像
docker pull <image-name>
```

**常见原因和解决方案**：

| 原因 | 解决方案 |
|------|---------|
| 镜像名称错误 | 检查镜像名称和标签 |
| 私有仓库未认证 | 创建 docker-registry Secret |
| 网络问题 | 检查节点网络，配置镜像加速器 |
| 镜像不存在 | 确认镜像已推送到仓库 |

```bash
# 创建私有仓库认证
kubectl create secret docker-registry regcred \
  --docker-server=<registry-url> \
  --docker-username=<username> \
  --docker-password=<password>

# 在 Pod 中使用
# spec.imagePullSecrets:
# - name: regcred
```

### 2. CrashLoopBackOff

**症状**：Pod 不断重启

**排查步骤**：
```bash
# 查看日志
kubectl logs <pod-name>

# 查看上一次崩溃的日志
kubectl logs <pod-name> --previous

# 查看事件
kubectl describe pod <pod-name>
```

**常见原因和解决方案**：

| 原因 | 解决方案 |
|------|---------|
| 应用启动失败 | 检查应用配置和依赖 |
| 配置错误 | 检查 ConfigMap/Secret |
| 资源不足（OOMKilled）| 增加内存限制 |
| 健康检查失败 | 调整探针配置 |
| 依赖服务未就绪 | 使用 initContainer 等待 |

```bash
# 进入容器调试（如果能短暂运行）
kubectl exec -it <pod-name> -- sh

# 使用调试容器
kubectl debug <pod-name> -it --image=busybox
```

### 3. Pending 状态

**症状**：Pod 一直处于 Pending 状态

**排查步骤**：
```bash
# 查看原因
kubectl describe pod <pod-name>

# 查看事件
kubectl get events --field-selector involvedObject.name=<pod-name>

# 检查节点资源
kubectl describe nodes | grep -A 10 "Allocated resources"
```

**常见原因和解决方案**：

| 原因 | 解决方案 |
|------|---------|
| 资源不足 | 扩容集群或减少资源请求 |
| 节点选择器无匹配 | 检查 nodeSelector/affinity |
| 污点没有容忍 | 添加 tolerations |
| PVC 未绑定 | 检查 PV 和 StorageClass |

### 4. OOMKilled

**症状**：容器因内存不足被杀死

**排查步骤**：
```bash
# 查看终止原因
kubectl describe pod <pod-name> | grep -A 5 "Last State"

# 查看资源使用
kubectl top pod <pod-name>
```

**解决方案**：
```yaml
# 增加内存限制
resources:
  limits:
    memory: "512Mi"  # 根据实际需求调整
  requests:
    memory: "256Mi"
```

### 5. 容器无法启动

**排查步骤**：
```bash
# 检查命令和参数
kubectl get pod <pod-name> -o jsonpath='{.spec.containers[0].command}'
kubectl get pod <pod-name> -o jsonpath='{.spec.containers[0].args}'

# 尝试用不同命令运行
kubectl run debug --image=<image> --command -- sleep 3600
kubectl exec -it debug -- sh
```

## Service 常见问题

### 1. Service 无法访问

**排查步骤**：
```bash
# 1. 检查 Service
kubectl get svc <service-name>
kubectl describe svc <service-name>

# 2. 检查 Endpoints
kubectl get endpoints <service-name>

# 3. 检查 Pod 是否正常
kubectl get pods -l <service-selector>

# 4. 检查 Pod 标签是否匹配
kubectl get pods --show-labels
```

**常见问题**：

```bash
# Endpoints 为空 - 标签不匹配
# 检查 Service selector 和 Pod labels 是否一致
kubectl get svc <service-name> -o jsonpath='{.spec.selector}'
kubectl get pods -l app=myapp --show-labels

# 端口不匹配
kubectl get svc <service-name> -o jsonpath='{.spec.ports}'
kubectl get pod <pod-name> -o jsonpath='{.spec.containers[0].ports}'
```

### 2. DNS 解析失败

**排查步骤**：
```bash
# 检查 CoreDNS
kubectl get pods -n kube-system -l k8s-app=kube-dns

# 测试 DNS 解析
kubectl run test --image=busybox -it --rm -- nslookup <service-name>
kubectl run test --image=busybox -it --rm -- nslookup kubernetes.default

# 检查 DNS 配置
kubectl get configmap coredns -n kube-system -o yaml
```

## 网络问题

### 1. Pod 之间无法通信

**排查步骤**：
```bash
# 1. 检查 Pod IP
kubectl get pods -o wide

# 2. 从一个 Pod ping 另一个
kubectl exec -it pod1 -- ping <pod2-ip>

# 3. 检查网络策略
kubectl get networkpolicies -A

# 4. 检查 CNI 插件
kubectl get pods -n kube-system | grep -E "calico|flannel|weave"
```

### 2. 外部访问问题

```bash
# NodePort 服务
kubectl get svc <service-name>
curl http://<node-ip>:<node-port>

# LoadBalancer 服务
kubectl get svc <service-name>  # 查看 EXTERNAL-IP
curl http://<external-ip>

# Ingress
kubectl get ingress
kubectl describe ingress <ingress-name>
```

## 存储问题

### 1. PVC 无法绑定

**排查步骤**：
```bash
# 查看 PVC 状态
kubectl get pvc
kubectl describe pvc <pvc-name>

# 查看可用的 PV
kubectl get pv

# 检查 StorageClass
kubectl get storageclass
kubectl describe storageclass <sc-name>
```

**常见原因**：
- 没有可用的 PV
- PV 容量不满足 PVC 请求
- accessModes 不匹配
- StorageClass 不存在

### 2. 挂载失败

```bash
# 查看 Pod 事件
kubectl describe pod <pod-name> | grep -A 10 Events

# 检查节点上的挂载
kubectl debug node/<node-name> -it --image=busybox -- mount | grep <pv-name>
```

## 调试工具

### kubectl debug

```bash
# 为 Pod 添加调试容器
kubectl debug <pod-name> -it --image=busybox

# 调试节点
kubectl debug node/<node-name> -it --image=busybox

# 复制 Pod 并修改命令（调试启动问题）
kubectl debug <pod-name> -it --copy-to=debug-pod --container=app -- sh
```

### 临时容器

```bash
# 向运行中的 Pod 添加临时容器（K8s 1.23+）
kubectl debug -it <pod-name> --image=busybox --target=<container-name>
```

### 常用调试镜像

```bash
# 网络调试
kubectl run debug --image=nicolaka/netshoot -it --rm -- bash

# 通用调试
kubectl run debug --image=busybox -it --rm -- sh

# DNS 调试
kubectl run debug --image=tutum/dnsutils -it --rm -- bash
```

## 性能问题

### 资源瓶颈

```bash
# 查看节点资源
kubectl top nodes
kubectl describe nodes | grep -A 10 "Allocated resources"

# 查看 Pod 资源
kubectl top pods --sort-by=cpu
kubectl top pods --sort-by=memory
kubectl top pods --containers

# 检查资源配额
kubectl describe resourcequota -A
```

### 慢启动问题

```bash
# 检查镜像拉取时间
kubectl describe pod <pod-name> | grep -A 5 "Events"

# 检查探针配置
kubectl get pod <pod-name> -o yaml | grep -A 10 "readinessProbe"
```

## 常用排查命令速查

```bash
# ========== 状态检查 ==========
kubectl get pods -o wide                    # Pod 状态
kubectl get events --sort-by='.metadata.creationTimestamp'  # 事件
kubectl top pods                            # 资源使用

# ========== 详细信息 ==========
kubectl describe pod <pod>                  # Pod 详情
kubectl logs <pod> [--previous]             # 日志
kubectl exec -it <pod> -- sh                # 进入容器

# ========== 网络测试 ==========
kubectl run test --image=busybox -it --rm -- wget -qO- http://<svc>
kubectl run test --image=busybox -it --rm -- nslookup <svc>

# ========== 调试 ==========
kubectl debug <pod> -it --image=busybox     # 添加调试容器
kubectl debug node/<node> -it --image=busybox  # 节点调试
```

## 故障排查清单

### Pod 不启动

- [ ] 检查镜像是否正确
- [ ] 检查镜像拉取权限
- [ ] 检查资源请求是否满足
- [ ] 检查节点选择器和污点
- [ ] 检查 PVC 是否绑定
- [ ] 检查 ConfigMap/Secret 是否存在

### Service 不通

- [ ] 检查 Pod 是否 Running
- [ ] 检查 Endpoints 是否有值
- [ ] 检查标签选择器是否匹配
- [ ] 检查端口配置
- [ ] 检查网络策略

### 应用不正常

- [ ] 检查日志输出
- [ ] 检查健康检查配置
- [ ] 检查环境变量和配置
- [ ] 检查资源使用情况
- [ ] 检查依赖服务

## 下一步

- [Kubernetes 网络模型](../04-advanced/01-networking.md)



