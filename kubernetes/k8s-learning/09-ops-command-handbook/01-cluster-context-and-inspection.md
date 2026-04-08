# 🔎 集群连接、上下文切换与资源巡检

## 为什么运维第一步不是直接查 Pod

很多 Kubernetes 初学者一上来就执行：

```bash
kubectl get pods
```

这条命令当然没有错，但它经常有两个问题：

1. 你不一定连的是正确集群
2. 你不一定在正确 namespace

线上运维最怕的不是“不会敲命令”，而是**在错误环境做了正确动作**。

所以在真正排障前，建议你固定按下面顺序做检查：

1. 先确认上下文
2. 再确认 namespace
3. 再看集群整体状态
4. 最后才下钻到具体工作负载

## 一、先确认你连的是谁

### 1. 查看当前上下文

```bash
kubectl config current-context
```

这条命令解决的是：

- 我现在连接的是测试集群还是生产集群
- 我当前使用的是哪套 kubeconfig 上下文

如果你值班时习惯同时操作多个环境，这条命令建议每次执行变更前都先看一眼。

### 2. 查看所有上下文

```bash
kubectl config get-contexts
```

常见输出里一般会看到：

- context 名称
- cluster 名称
- user 名称
- 当前是否被选中

这相当于你的“环境清单”。

### 3. 切换上下文

```bash
kubectl config use-context prod-cluster
kubectl config use-context test-cluster
```

适用场景：

- 从测试环境切换到生产环境
- 从一个云厂商集群切换到另一个集群
- 同一台机器维护多个客户或多个项目时

### 4. 查看当前最小配置

```bash
kubectl config view --minify
```

这条命令很适合确认当前生效的：

- cluster
- user
- namespace

当你怀疑“我明明切了集群，怎么感觉还是不对”时，这条命令很有帮助。

## 二、确认当前 namespace

### 1. 查看当前上下文绑定的 namespace

```bash
kubectl config view --minify --output 'jsonpath={..namespace}'
```

如果没有输出，通常表示当前默认 namespace 是 `default`。

### 2. 查看所有 namespace

```bash
kubectl get ns
```

这条命令在运维中很常用，因为你首先要知道：

- 业务资源在哪个 namespace
- 是否存在多个环境共用一个集群
- 是否存在系统 namespace，比如 `kube-system`

### 3. 临时指定 namespace

```bash
kubectl get pods -n prod
kubectl describe pod myapp-xxx -n prod
kubectl logs myapp-xxx -n prod
```

这适合你只是临时操作某个命名空间，不想修改当前上下文默认值的场景。

### 4. 修改当前上下文默认 namespace

```bash
kubectl config set-context --current --namespace=prod
```

适合长期在某个 namespace 工作的情况。

但运维上要注意：

- 如果你经常跨 namespace 工作，不建议长期改默认 namespace
- 因为这样容易让你忘记自己当前在哪个业务空间

## 三、集群级别的快速巡检

真正排障前，最好先对集群做一个“总览扫描”。

### 1. 查看集群信息

```bash
kubectl cluster-info
```

它主要用于快速确认：

- API Server 是否可访问
- CoreDNS 等基础服务地址是否可见

如果这一步都失败了，说明问题可能还没到业务层，可能是：

- kubeconfig 配置错了
- 网络不通
- 认证失效
- API Server 异常

### 2. 查看节点状态

```bash
kubectl get nodes
kubectl get nodes -o wide
```

运维中最先看节点，是因为很多业务问题最终都能回到节点层：

- 某节点 NotReady
- 某节点磁盘满了
- 某节点网络异常
- 某节点上容器运行时异常

如果节点整体不健康，先别急着盯 Pod。

### 3. 看所有 namespace 的高层资源

```bash
kubectl get pods -A
kubectl get deploy -A
kubectl get svc -A
kubectl get ingress -A
kubectl get pvc -A
```

这样做的意义是快速回答：

- 这次故障只影响一个 namespace，还是多处同时异常
- 是单个 Pod 问题，还是 Deployment、Service、PVC 也一起有异常

### 4. 查看资源使用情况

```bash
kubectl top nodes
kubectl top pods -A
kubectl top pods -A --containers
```

如果集群安装了 metrics-server，这几条命令非常适合做第一轮容量判断：

- CPU 是否打满
- Memory 是否接近上限
- 是否有某个 Pod 资源突增

注意：

- `kubectl top` 看到的是即时资源使用，不是长期趋势
- 长期趋势仍要看 Prometheus、Grafana 一类监控系统

## 四、资源查询的标准姿势

这一部分是值班时最常用的内容。

### 1. 基础查看

```bash
kubectl get pods
kubectl get pods -o wide
kubectl get pods -A
kubectl get deploy
kubectl get svc
kubectl get ingress
kubectl get pvc
kubectl get events
```

建议你逐渐形成习惯：

- `get` 用来扫全局和看状态
- `describe` 用来下钻看细节
- `logs` 用来看应用日志
- `exec` 用来做容器内验证

### 2. 持续观察

```bash
kubectl get pods -w
kubectl get events --watch
```

适用场景：

- 发版时看 Pod 是否拉起
- 改副本数时看副本变化
- 排障时观察资源是否持续变化

这类命令的价值在于，它能让你看到“状态演进过程”，而不只是一个静态快照。

### 3. 更详细地看资源

```bash
kubectl describe pod myapp-7d6c7f89d9-abcde -n prod
kubectl describe deploy myapp -n prod
kubectl describe svc myapp -n prod
kubectl describe ingress myapp -n prod
kubectl describe node node-1
kubectl describe pvc data-mysql-0 -n prod
```

`describe` 最重要，因为很多真正关键的信息都不在 `get` 里，而在：

- 事件
- 挂载信息
- 容器状态
- 镜像拉取结果
- 调度失败原因
- 探针失败信息

### 4. 按标签筛选

```bash
kubectl get pods -l app=myapp -n prod
kubectl get pods -l 'app=myapp,env=prod'
kubectl get svc -l app=myapp -n prod
```

这是运维里非常实用的能力，因为一个业务通常会有多个 Pod。

如果你只靠手动找 Pod 名，很容易慢，也容易漏。

### 5. 按字段筛选

```bash
kubectl get pods --field-selector status.phase=Running -n prod
kubectl get pods --field-selector status.phase=Pending -A
kubectl get pods --field-selector spec.nodeName=node-1 -A
```

这个功能在这些场景特别有用：

- 快速筛出所有 Pending Pod
- 看某个节点上跑了哪些 Pod
- 定位某一类状态问题

## 五、输出定制：让命令更像运维工具

很多时候默认输出不够用，定制输出可以让你更快发现问题。

### 1. YAML / JSON 输出

```bash
kubectl get pod myapp-xxx -n prod -o yaml
kubectl get deploy myapp -n prod -o yaml
kubectl get svc myapp -n prod -o json
```

适用场景：

- 想看完整字段
- 想确认 selector、probe、resources、volume 等配置
- 想导出对象做对比

### 2. 自定义列

```bash
kubectl get pods -n prod -o custom-columns=NAME:.metadata.name,PHASE:.status.phase,IP:.status.podIP,NODE:.spec.nodeName
```

优点是：

- 更适合快速扫表
- 只看你真正关心的字段

### 3. JSONPath

```bash
kubectl get pods -n prod -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.status.phase}{"\t"}{.spec.nodeName}{"\n"}{end}'
kubectl get pod myapp-xxx -n prod -o jsonpath='{.status.podIP}'
kubectl get svc myapp -n prod -o jsonpath='{.spec.clusterIP}'
kubectl get nodes -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.status.conditions[?(@.type=="Ready")].status}{"\n"}{end}'
```

JSONPath 一开始会有点绕，但一旦你会了，很多排障动作会明显变快。

最常见的用途有：

- 批量拿名字
- 批量拿 IP
- 批量拿镜像
- 批量拿节点

## 六、事件：排障时不要忽略的关键信号

很多人排障只看日志，不看事件。

这是非常可惜的，因为事件往往直接告诉你为什么调度失败、为什么挂载失败、为什么探针异常。

### 1. 查看事件

```bash
kubectl get events -n prod
kubectl get events -A
kubectl get events --sort-by=.metadata.creationTimestamp -A
```

推荐你值班时优先用下面这条：

```bash
kubectl get events -A --sort-by=.metadata.creationTimestamp
```

因为它能让你快速看到最近集群刚发生了什么。

### 2. 按对象过滤事件

```bash
kubectl get events -n prod --field-selector involvedObject.name=myapp-xxx
```

适用场景：

- 某个 Pod 拉不起来
- 某个 PVC 绑定失败
- 某个节点异常

## 七、常见的巡检顺序

### 场景 1：用户说“服务挂了”

建议先按下面顺序查：

1. `kubectl config current-context`
2. `kubectl get pods -n <ns>`
3. `kubectl get svc -n <ns>`
4. `kubectl get ingress -n <ns>`
5. `kubectl get events -n <ns> --sort-by=.metadata.creationTimestamp`
6. `kubectl describe pod <pod> -n <ns>`
7. `kubectl logs <pod> -n <ns>`

为什么这样查：

- 先确认环境，避免查错集群
- 再看运行状态，确认是否是 Pod 本身异常
- 再看 Service / Ingress，确认是否是流量转发链路问题
- 再看事件和详情，定位更具体原因

### 场景 2：用户说“某个节点上的服务都不正常”

建议先按下面顺序查：

1. `kubectl get nodes`
2. `kubectl describe node <node>`
3. `kubectl get pods -A --field-selector spec.nodeName=<node>`
4. `kubectl top node <node>`
5. `kubectl top pods -A --containers | findstr <关键业务名>`

为什么这样查：

- 这类问题往往不是业务单点问题，而是节点资源、网络、磁盘或运行时问题

## 八、运维避坑提示

### 1. 不要省略 namespace

尤其是生产环境，建议尽量显式带 `-n`，因为这样更安全、也更容易复盘。

### 2. 不要只看 `Running`

Pod 显示 `Running` 并不代表业务可用。

你还要继续确认：

- readiness probe 是否通过
- 日志是否正常
- Service 是否选到了正确 endpoints

### 3. `describe` 和 `events` 非常重要

很多问题不是应用自己报错，而是：

- 调度失败
- 卷挂载失败
- 镜像拉取失败
- 探针失败

这些信息通常优先出现在事件里。

## 九、这一篇你至少要记住什么

如果你现在只想先抓住最重要的东西，请先记住这 6 条：

```bash
kubectl config current-context
kubectl get ns
kubectl get pods -A
kubectl describe pod <pod> -n <ns>
kubectl get events -A --sort-by=.metadata.creationTimestamp
kubectl top pods -A
```

这 6 条命令已经能覆盖大量日常巡检和第一轮故障定位。

## 下一篇

学完这一篇，建议接着看：

- [工作负载发布、变更、回滚与扩缩容](./02-workload-release-and-change.md)

因为运维不只是“看”，还一定会遇到“改”。而改动，是线上风险真正开始出现的地方。
