# 🚀 工作负载发布、变更、回滚与扩缩容

## 为什么运维最怕“直接改”

Kubernetes 的强大之处，在于它允许你非常方便地修改资源：

- 改镜像
- 改副本
- 改环境变量
- 改资源限制
- 改配置

但线上风险也正是从这里开始。

所以这一篇的重点不是“教你怎么改”，而是教你形成一个更像运维的变更思路：

1. 先看当前状态
2. 再决定怎么改
3. 改完立刻验证
4. 有问题要能快速回滚

## 一、先看清楚当前资源

在做任何变更之前，先把现状看清楚。

### 1. 看 Deployment

```bash
kubectl get deploy -n prod
kubectl describe deploy myapp -n prod
kubectl get deploy myapp -n prod -o yaml
```

你至少要确认这些内容：

- 当前副本数
- 当前镜像版本
- 滚动更新策略
- 环境变量和资源限制
- selector 和 pod template

### 2. 看 Pod

```bash
kubectl get pods -l app=myapp -n prod -o wide
kubectl describe pod <pod-name> -n prod
```

这样可以确认：

- 当前 Pod 是否都健康
- 是否已经存在重启较多的实例
- 是否有某个节点上的实例异常

### 3. 看发布历史

```bash
kubectl rollout history deployment myapp -n prod
```

如果这个 Deployment 曾经多次变更，这条命令很重要。

它可以帮助你知道：

- 之前发过哪些 revision
- 如果这次变更失败，可以回到哪个 revision

## 二、声明式变更：`apply` 是最常见的主路径

### 1. 应用 YAML

```bash
kubectl apply -f deploy.yaml
kubectl apply -f k8s/
```

这是最常见、最推荐的方式，因为它更符合基础设施即代码的思路。

适用场景：

- 日常发版
- 通过 Git 管理清单
- 多人协作
- 需要审计和回溯

### 2. 预演而不真正执行

```bash
kubectl apply -f deploy.yaml --dry-run=client
kubectl apply -f deploy.yaml --dry-run=server
```

区别可以这样理解：

- `client`：本地语法和对象构造检查
- `server`：让 API Server 帮你校验是否可接收

线上变更前，`--dry-run=server` 很值得用。

### 3. 查看将被应用的差异

```bash
kubectl diff -f deploy.yaml
```

这条命令的价值非常大，因为它能让你在真正执行前看到：

- 哪些字段会改
- 是加字段、删字段，还是改字段

它能显著降低“以为只是改镜像，结果顺手把别的配置改了”的风险。

## 三、命令式变更：适合紧急处理，但要更谨慎

命令式变更的优点是快，缺点是容易留下配置漂移。

### 1. 改镜像

```bash
kubectl set image deployment/myapp myapp=myrepo/myapp:v2 -n prod
kubectl set image deployment/myapp *=myrepo/myapp:v2 -n prod
```

适用场景：

- 紧急发一个小版本
- 线上快速验证镜像问题

执行后建议立刻配合下面命令观察：

```bash
kubectl rollout status deployment myapp -n prod
kubectl get pods -l app=myapp -n prod -w
```

### 2. 改环境变量

```bash
kubectl set env deployment/myapp LOG_LEVEL=debug -n prod
kubectl set env deployment/myapp JAVA_OPTS='-Xms512m -Xmx512m' -n prod
```

注意：

- 这类改动会触发 Pod 重建
- 改完要确认应用是否真的读到了新配置

### 3. 改资源限制

```bash
kubectl set resources deployment myapp -c myapp --requests=cpu=200m,memory=256Mi --limits=cpu=500m,memory=512Mi -n prod
```

适用场景：

- Pod 因资源不足频繁重启
- 节点资源紧张，需要精细化控制

但要注意：

- requests 改大了，可能导致调度不上
- limits 改太小了，可能导致 OOMKilled

### 4. 打补丁

```bash
kubectl patch deployment myapp -n prod -p '{"spec":{"replicas":5}}'
kubectl patch deployment myapp -n prod --type merge -p '{"spec":{"template":{"metadata":{"labels":{"release":"hotfix"}}}}}'
```

`patch` 很适合快速改一个小字段。

但它也容易让人忽略“变更是不是已经同步回 Git”这件事。

所以在生产里，如果你临时 `patch` 了对象，后续最好补回配置仓库。

## 四、编辑资源：`edit` 能救急，但要控制使用场景

```bash
kubectl edit deployment myapp -n prod
kubectl edit svc myapp -n prod
kubectl edit configmap myapp-config -n prod
```

优点：

- 快
- 适合紧急修复

缺点：

- 不容易审计
- 容易和 Git 中的配置不一致
- 一旦编辑器里改错，可能直接影响线上

所以建议：

- 测试环境可以多用来学习
- 生产环境只在紧急情况下使用
- 用完后要尽快把改动补回 YAML

## 五、扩缩容：看起来简单，其实很容易出问题

### 1. 手动扩缩容

```bash
kubectl scale deployment myapp --replicas=5 -n prod
kubectl scale statefulset mysql --replicas=3 -n prod
```

这条命令适合：

- 临时扛流量
- 验证副本扩张是否正常
- 做压测前准备

但你要先确认：

- 应用是否无状态
- 下游数据库是否扛得住
- Service 是否能正常分发到新副本
- 节点资源是否足够

### 2. 自动扩缩容

```bash
kubectl get hpa -n prod
kubectl describe hpa myapp -n prod
```

即使你不直接创建 HPA，运维也要会检查它，因为有时你手动改了副本数，HPA 可能会再把它改回去。

### 3. 重启而不改镜像

```bash
kubectl rollout restart deployment myapp -n prod
kubectl rollout restart statefulset redis -n prod
```

适用场景：

- ConfigMap / Secret 已更新，需要触发重建
- 应用卡死但配置没变
- 临时做一次有控制的重启

注意：

- `rollout restart` 不是“修复问题”的万能键
- 如果根因是镜像、配置、探针、依赖异常，重启只是让问题重新出现

## 六、回滚：线上运维必须熟练

### 1. 查看回滚历史

```bash
kubectl rollout history deployment myapp -n prod
kubectl rollout history deployment myapp --revision=3 -n prod
```

### 2. 查看滚动状态

```bash
kubectl rollout status deployment myapp -n prod
```

这是发版时必须盯的命令之一。

你要确认：

- 新 Pod 是否起来了
- 老 Pod 是否按预期退出
- 滚动是否卡住

### 3. 执行回滚

```bash
kubectl rollout undo deployment myapp -n prod
kubectl rollout undo deployment myapp --to-revision=3 -n prod
```

典型场景：

- 新版本启动失败
- 探针一直不过
- 业务指标明显异常
- 日志出现大量错误

### 4. 暂停和恢复发布

```bash
kubectl rollout pause deployment myapp -n prod
kubectl rollout resume deployment myapp -n prod
```

适合场景：

- 你想分阶段修改多个字段
- 你想暂时冻结滚动过程

但实战中更常见的，是直接观察 `rollout status` 和必要时 `undo`。

## 七、删除：最常见，也最容易造成误伤

### 1. 删除对象

```bash
kubectl delete pod myapp-xxx -n prod
kubectl delete deploy myapp -n prod
kubectl delete svc myapp -n prod
kubectl delete -f deploy.yaml
```

### 2. 为什么删除 Pod 不一定可怕

如果这个 Pod 归属于 Deployment、StatefulSet、DaemonSet，删除 Pod 往往只是让控制器重新拉起一个。

所以很多人会用：

```bash
kubectl delete pod <pod-name> -n prod
```

来做“单实例重建”。

但你必须先确认：

- 它是不是单副本
- 它是不是有状态实例
- 它是不是当前唯一健康实例

### 3. 强制删除

```bash
kubectl delete pod myapp-xxx -n prod --force --grace-period=0
kubectl delete pod myapp-xxx -n prod --now
```

这是高风险动作。

适合场景：

- Pod 卡在 `Terminating`
- 普通删除迟迟不结束

但注意：

- 强制删除只是从 API 视角快速移除对象
- 不一定等于底层容器已经干净退出

生产里不要把它当常规手段。

## 八、发版时的推荐命令顺序

下面给你一套比较实用的 Deployment 变更顺序。

### 场景：把 `myapp` 从 `v1` 发到 `v2`

```bash
# 1. 看现状
kubectl get deploy myapp -n prod
kubectl rollout history deployment myapp -n prod
kubectl get pods -l app=myapp -n prod -o wide

# 2. 预看差异
kubectl diff -f deploy.yaml

# 3. 执行变更
kubectl apply -f deploy.yaml

# 4. 观察滚动过程
kubectl rollout status deployment myapp -n prod
kubectl get pods -l app=myapp -n prod -w

# 5. 如果异常，立即回滚
kubectl rollout undo deployment myapp -n prod
```

这套顺序的核心思想是：

- 先确认现状
- 再执行变更
- 改完立刻验证
- 失败就迅速回退

## 九、StatefulSet 变更要更谨慎

对 StatefulSet，你不能简单套用无状态应用的思路。

常见命令：

```bash
kubectl get sts -n prod
kubectl describe sts mysql -n prod
kubectl rollout status statefulset mysql -n prod
kubectl scale statefulset mysql --replicas=3 -n prod
```

为什么要谨慎：

- Pod 名有序号
- 往往绑定 PVC
- 删除和重建可能涉及数据
- 副本扩缩不只是“多一个进程”，还可能影响集群拓扑

所以对数据库、消息队列、注册中心这类有状态系统，扩缩容前一定要先理解应用本身机制。

## 十、这一篇最值得记住的命令

```bash
kubectl diff -f deploy.yaml
kubectl apply -f deploy.yaml
kubectl set image deployment/myapp myapp=myrepo/myapp:v2 -n prod
kubectl scale deployment myapp --replicas=5 -n prod
kubectl rollout status deployment myapp -n prod
kubectl rollout history deployment myapp -n prod
kubectl rollout undo deployment myapp -n prod
kubectl rollout restart deployment myapp -n prod
```

如果你把这些命令和“先看现状、再变更、立刻验证、异常回滚”这条主线结合起来，你的运维动作会安全很多。

## 下一篇

下一篇建议看：

- [日志、调试、容器排障与应急定位](./03-logs-debug-and-troubleshooting.md)

因为大多数线上问题，最终还是要回到“日志、事件、容器内验证、网络连通性”这几个动作上来。
