# 🛠️ kubectl 命令完全手册

## kubectl 简介

`kubectl` 是 Kubernetes 的命令行工具，用于与集群进行交互。

```
kubectl [command] [TYPE] [NAME] [flags]

command: 操作命令 (get, create, delete, apply...)
TYPE:    资源类型 (pods, services, deployments...)
NAME:    资源名称
flags:   可选参数
```

## 命令速查表

### 🔍 查看资源

```bash
# ============ 基础查看 ============
kubectl get pods                           # 查看 Pod
kubectl get pods -o wide                   # 显示更多信息（IP、节点）
kubectl get pods -o yaml                   # YAML 格式输出
kubectl get pods -o json                   # JSON 格式输出
kubectl get pods -o name                   # 只显示名称
kubectl get pods --show-labels             # 显示标签
kubectl get pods -w                        # 持续监听
kubectl get pods --watch                   # 同上

# ============ 常用资源简写 ============
kubectl get po                             # pods
kubectl get svc                            # services
kubectl get deploy                         # deployments
kubectl get rs                             # replicasets
kubectl get ds                             # daemonsets
kubectl get sts                            # statefulsets
kubectl get cm                             # configmaps
kubectl get secret                         # secrets
kubectl get pv                             # persistentvolumes
kubectl get pvc                            # persistentvolumeclaims
kubectl get ns                             # namespaces
kubectl get no                             # nodes
kubectl get ing                            # ingresses
kubectl get ep                             # endpoints
kubectl get hpa                            # horizontalpodautoscalers
kubectl get cj                             # cronjobs
kubectl get sa                             # serviceaccounts

# ============ 多资源查看 ============
kubectl get pods,svc                       # 多种资源
kubectl get all                            # 所有常见资源
kubectl get all -A                         # 所有命名空间

# ============ 命名空间 ============
kubectl get pods -n kube-system            # 指定命名空间
kubectl get pods --all-namespaces          # 所有命名空间
kubectl get pods -A                        # 简写

# ============ 标签筛选 ============
kubectl get pods -l app=nginx              # 单个标签
kubectl get pods -l 'app=nginx,env=prod'   # 多个标签
kubectl get pods -l 'env in (prod,dev)'    # 集合选择
kubectl get pods -l 'app!=nginx'           # 不等于
kubectl get pods -l 'env'                  # 存在标签
kubectl get pods -l '!env'                 # 不存在标签

# ============ 字段筛选 ============
kubectl get pods --field-selector status.phase=Running
kubectl get pods --field-selector metadata.name=nginx
kubectl get pods --field-selector spec.nodeName=node1

# ============ 排序 ============
kubectl get pods --sort-by=.metadata.creationTimestamp
kubectl get pods --sort-by=.status.startTime
kubectl get pods --sort-by='.status.containerStatuses[0].restartCount'

# ============ 自定义输出 ============
kubectl get pods -o custom-columns=NAME:.metadata.name,STATUS:.status.phase
kubectl get pods -o jsonpath='{.items[*].metadata.name}'
kubectl get pods -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}'
```

### 📝 详细信息

```bash
kubectl describe pod nginx                 # Pod 详情
kubectl describe node node1                # 节点详情
kubectl describe svc nginx                 # Service 详情
kubectl describe deploy nginx              # Deployment 详情

# 查看事件
kubectl get events                         # 所有事件
kubectl get events --sort-by=.metadata.creationTimestamp
kubectl get events --field-selector involvedObject.name=nginx
```

### ➕ 创建资源

```bash
# ============ 命令式创建 ============
kubectl run nginx --image=nginx            # 创建 Pod
kubectl create deployment nginx --image=nginx   # 创建 Deployment
kubectl create namespace dev               # 创建命名空间
kubectl create service clusterip nginx --tcp=80:80  # 创建 Service
kubectl create configmap myconfig --from-literal=key=value
kubectl create secret generic mysecret --from-literal=password=secret

# ============ 声明式创建 ============
kubectl apply -f manifest.yaml             # 创建/更新
kubectl apply -f ./directory/              # 目录下所有文件
kubectl apply -f https://example.com/manifest.yaml  # 从 URL

# ============ 生成 YAML ============
kubectl run nginx --image=nginx --dry-run=client -o yaml
kubectl create deployment nginx --image=nginx --dry-run=client -o yaml > deploy.yaml
kubectl expose deployment nginx --port=80 --dry-run=client -o yaml
```

### ✏️ 编辑和更新

```bash
# ============ 编辑 ============
kubectl edit deployment nginx              # 在编辑器中编辑
kubectl edit svc nginx

# ============ 更新镜像 ============
kubectl set image deployment/nginx nginx=nginx:1.21
kubectl set image deployment/nginx *=nginx:1.21   # 所有容器

# ============ 更新资源 ============
kubectl set resources deployment nginx -c nginx --limits=cpu=200m,memory=512Mi
kubectl set env deployment nginx ENV_VAR=value
kubectl set serviceaccount deployment nginx mysa

# ============ 扩缩容 ============
kubectl scale deployment nginx --replicas=5
kubectl autoscale deployment nginx --min=2 --max=10 --cpu-percent=80

# ============ 打补丁 ============
kubectl patch deployment nginx -p '{"spec":{"replicas":3}}'
kubectl patch pod nginx -p '{"metadata":{"labels":{"new-label":"value"}}}'

# ============ 替换 ============
kubectl replace -f manifest.yaml           # 完全替换

# ============ 回滚 ============
kubectl rollout undo deployment nginx
kubectl rollout undo deployment nginx --to-revision=2
kubectl rollout status deployment nginx
kubectl rollout history deployment nginx
kubectl rollout pause deployment nginx
kubectl rollout resume deployment nginx
kubectl rollout restart deployment nginx
```

### 🗑️ 删除资源

```bash
# ============ 基础删除 ============
kubectl delete pod nginx
kubectl delete deployment nginx
kubectl delete svc nginx
kubectl delete -f manifest.yaml

# ============ 批量删除 ============
kubectl delete pods --all                  # 删除所有 Pod
kubectl delete pods -l app=nginx           # 按标签删除
kubectl delete all --all                   # 删除所有资源

# ============ 强制删除 ============
kubectl delete pod nginx --force --grace-period=0
kubectl delete pod nginx --now             # 立即删除

# ============ 级联删除 ============
kubectl delete deployment nginx --cascade=foreground
```

### 🔧 调试和故障排查

```bash
# ============ 日志 ============
kubectl logs nginx                         # 查看日志
kubectl logs nginx -c container            # 指定容器
kubectl logs nginx --previous              # 上一个容器的日志
kubectl logs nginx -f                      # 持续输出
kubectl logs nginx --tail=100              # 最后 100 行
kubectl logs nginx --since=1h              # 最近 1 小时
kubectl logs -l app=nginx                  # 按标签
kubectl logs -l app=nginx --all-containers

# ============ 执行命令 ============
kubectl exec nginx -- ls /                 # 执行命令
kubectl exec nginx -- cat /etc/hostname
kubectl exec -it nginx -- /bin/bash        # 交互式 shell
kubectl exec -it nginx -- sh               # 如果没有 bash
kubectl exec -it nginx -c container -- sh  # 指定容器

# ============ 端口转发 ============
kubectl port-forward pod/nginx 8080:80     # Pod 端口
kubectl port-forward svc/nginx 8080:80     # Service 端口
kubectl port-forward deploy/nginx 8080:80  # Deployment 端口

# ============ 代理 ============
kubectl proxy                              # 启动 API 代理
kubectl proxy --port=8001

# ============ 文件复制 ============
kubectl cp nginx:/etc/nginx/nginx.conf ./nginx.conf
kubectl cp ./config.txt nginx:/tmp/config.txt
kubectl cp nginx:/var/log ./logs -c container

# ============ 调试容器 ============
kubectl debug nginx -it --image=busybox    # 临时调试容器
kubectl debug node/node1 -it --image=busybox  # 节点调试

# ============ 资源使用 ============
kubectl top nodes                          # 节点资源
kubectl top pods                           # Pod 资源
kubectl top pods --containers              # 容器级别
```

### 🏷️ 标签和注解

```bash
# ============ 标签操作 ============
kubectl label pods nginx app=web           # 添加标签
kubectl label pods nginx app=web --overwrite  # 更新标签
kubectl label pods nginx app-               # 删除标签
kubectl label pods --all app=web           # 所有 Pod

# ============ 注解操作 ============
kubectl annotate pods nginx description="my nginx pod"
kubectl annotate pods nginx description-   # 删除注解
```

### 📊 集群信息

```bash
# ============ 集群状态 ============
kubectl cluster-info                       # 集群信息
kubectl cluster-info dump                  # 详细转储
kubectl get componentstatuses              # 组件状态（已弃用）

# ============ API 资源 ============
kubectl api-resources                      # 所有 API 资源
kubectl api-versions                       # API 版本
kubectl explain pod                        # 资源文档
kubectl explain pod.spec                   # 字段说明
kubectl explain pod.spec.containers
kubectl explain pod --recursive            # 递归显示所有字段

# ============ 节点操作 ============
kubectl get nodes
kubectl describe node node1
kubectl cordon node1                       # 标记不可调度
kubectl uncordon node1                     # 取消标记
kubectl drain node1                        # 驱逐 Pod
kubectl drain node1 --ignore-daemonsets --delete-emptydir-data
kubectl taint nodes node1 key=value:NoSchedule
kubectl taint nodes node1 key:NoSchedule-  # 删除污点
```

### ⚙️ 配置管理

```bash
# ============ kubeconfig ============
kubectl config view                        # 查看配置
kubectl config view --minify               # 当前上下文配置
kubectl config current-context             # 当前上下文
kubectl config get-contexts                # 所有上下文
kubectl config use-context my-context      # 切换上下文
kubectl config set-context --current --namespace=dev
kubectl config set-credentials user --token=xxx

# ============ ConfigMap ============
kubectl create configmap myconfig --from-literal=key=value
kubectl create configmap myconfig --from-file=config.properties
kubectl create configmap myconfig --from-env-file=.env
kubectl get configmap myconfig -o yaml

# ============ Secret ============
kubectl create secret generic mysecret --from-literal=password=secret
kubectl create secret tls tls-secret --cert=cert.crt --key=cert.key
kubectl create secret docker-registry regcred --docker-server=... 
kubectl get secret mysecret -o jsonpath='{.data.password}' | base64 -d
```

## 常用技巧

### 别名设置

```bash
# ~/.bashrc 或 ~/.zshrc
alias k='kubectl'
alias kgp='kubectl get pods'
alias kgs='kubectl get svc'
alias kgd='kubectl get deploy'
alias kga='kubectl get all'
alias kd='kubectl describe'
alias kdp='kubectl describe pod'
alias kds='kubectl describe svc'
alias kl='kubectl logs'
alias klf='kubectl logs -f'
alias ke='kubectl exec -it'
alias ka='kubectl apply -f'
alias kd='kubectl delete'
```

### 自动补全

```bash
# Bash
source <(kubectl completion bash)
echo "source <(kubectl completion bash)" >> ~/.bashrc

# 别名补全
complete -o default -F __start_kubectl k

# Zsh
source <(kubectl completion zsh)
echo "source <(kubectl completion zsh)" >> ~/.zshrc
```

### 常用 JSONPath

```bash
# Pod IP
kubectl get pod nginx -o jsonpath='{.status.podIP}'

# 所有 Pod 名称
kubectl get pods -o jsonpath='{.items[*].metadata.name}'

# 节点 IP
kubectl get nodes -o jsonpath='{.items[*].status.addresses[?(@.type=="InternalIP")].address}'

# 容器镜像
kubectl get pods -o jsonpath='{.items[*].spec.containers[*].image}'

# 格式化输出
kubectl get pods -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.status.phase}{"\n"}{end}'
```

### 自定义列

```bash
kubectl get pods -o custom-columns=\
NAME:.metadata.name,\
STATUS:.status.phase,\
IP:.status.podIP,\
NODE:.spec.nodeName
```

## 实用脚本

### 删除所有失败的 Pod

```bash
kubectl delete pods --field-selector=status.phase=Failed
```

### 获取所有镜像

```bash
kubectl get pods -A -o jsonpath='{range .items[*]}{.spec.containers[*].image}{"\n"}{end}' | sort -u
```

### 导出资源（去除运行时字段）

```bash
kubectl get deployment nginx -o yaml | kubectl neat
# 或手动去除
kubectl get deployment nginx -o yaml | grep -v "creationTimestamp\|uid\|resourceVersion\|selfLink\|status"
```

### 批量操作

```bash
# 重启所有 Deployment
kubectl get deploy -o name | xargs -I {} kubectl rollout restart {}

# 删除所有 Evicted Pod
kubectl get pods -A | grep Evicted | awk '{print $2 " -n " $1}' | xargs -L1 kubectl delete pod
```

## 下一步

- [YAML 编写规范与技巧](./02-yaml-guide.md)



