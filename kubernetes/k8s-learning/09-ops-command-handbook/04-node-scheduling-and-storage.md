# 🖥️ 节点维护、调度控制与存储相关命令

## 为什么运维迟早会走到节点和存储层

很多 Kubernetes 问题表面看起来像“应用问题”，但最后查下去往往会落到：

- 节点不健康
- 节点资源不足
- 调度约束不匹配
- 污点影响调度
- PVC 没绑定
- 卷挂载失败

所以如果你只会 Deployment、Pod、Service 层面的命令，运维能力会很快碰到天花板。

这一篇就是帮你补齐更偏底层的日常操作能力。

## 一、节点查询：先判断是不是节点本身有问题

### 1. 查看节点

```bash
kubectl get nodes
kubectl get nodes -o wide
```

先看什么：

- `STATUS` 是否是 `Ready`
- 节点角色
- 节点内部 IP
- 版本是否一致

### 2. 查看节点详情

```bash
kubectl describe node node-1
```

这里重点看：

- Conditions
- Taints
- Allocatable
- 已分配资源
- 事件

很多调度失败、磁盘压力、内存压力、网络问题线索，都能在这里看到。

### 3. 查看节点资源使用

```bash
kubectl top nodes
kubectl top node node-1
```

适用场景：

- 某节点 CPU 打满
- 某节点内存过高
- 怀疑某节点负载明显失衡

## 二、节点维护三件套：`cordon`、`drain`、`uncordon`

这是运维中非常高频、也非常容易误操作的一组命令。

### 1. 标记节点不可调度

```bash
kubectl cordon node-1
```

这条命令的意思不是“把节点上的 Pod 赶走”，而是：

- 不再接收新的可调度 Pod

适用场景：

- 你准备维护节点
- 你发现节点有风险，先冻结调度

### 2. 驱逐节点上的工作负载

```bash
kubectl drain node-1 --ignore-daemonsets --delete-emptydir-data
```

这条命令的作用是：

- 尽量安全地把可驱逐 Pod 从当前节点迁走

为什么需要参数：

- `--ignore-daemonsets`：因为 DaemonSet 本来就会驻留在每个节点
- `--delete-emptydir-data`：因为 `emptyDir` 数据跟节点绑定，驱逐时会丢

执行前一定要确认：

- 是否存在单副本业务
- 是否存在 PodDisruptionBudget 限制
- 是否存在本地存储依赖
- 是否存在无法安全迁移的有状态实例

### 3. 维护后恢复调度

```bash
kubectl uncordon node-1
```

执行完节点维护后，不要忘了恢复，不然这个节点会一直不参与调度。

## 三、污点与容忍：控制“谁能上这个节点”

### 1. 给节点打污点

```bash
kubectl taint nodes node-1 dedicated=middleware:NoSchedule
kubectl taint nodes node-1 storage=local:NoExecute
```

这类命令适合：

- 专用节点隔离
- 特殊工作负载隔离
- 故障隔离

### 2. 删除污点

```bash
kubectl taint nodes node-1 dedicated=middleware:NoSchedule-
kubectl taint nodes node-1 storage=local:NoExecute-
```

### 3. 运维上怎么理解污点

可以把污点理解成：

- 节点主动挂出“非请勿入”的标识

只有带了相应 toleration 的 Pod 才允许上来。

### 4. 排障里怎么用

如果 Pod 一直 `Pending`，`describe pod` 里看到和 taint 相关的报错，就要去看：

```bash
kubectl describe node node-1
kubectl get pod myapp-xxx -n prod -o yaml
```

确认：

- 节点上有什么污点
- Pod 有没有对应 toleration

## 四、标签与节点选择：控制“应该去哪类节点”

### 1. 查看节点标签

```bash
kubectl get nodes --show-labels
kubectl get node node-1 --show-labels
```

### 2. 给节点打标签

```bash
kubectl label node node-1 disk=ssd
kubectl label node node-1 zone=az1
kubectl label node node-1 env=prod --overwrite
```

标签常用于：

- SSD / HDD 区分
- 可用区区分
- GPU 节点区分
- 专用业务节点区分

### 3. 删除标签

```bash
kubectl label node node-1 disk-
```

### 4. 为什么运维要会节点标签

因为很多业务调度策略会依赖它，比如：

- `nodeSelector`
- `nodeAffinity`
- topology spread

如果标签错了，业务可能就根本调度不上。

## 五、调度排障：Pod 为什么上不去

当 Pod 处于 `Pending`，你需要重点盯住下面几类命令。

### 1. 看 Pod 详情

```bash
kubectl describe pod myapp-xxx -n prod
```

最关键的是看事件里写的原因：

- `Insufficient cpu`
- `Insufficient memory`
- node affinity 不匹配
- taint 不匹配
- PVC 未绑定

### 2. 看节点总览

```bash
kubectl get nodes
kubectl top nodes
kubectl describe node node-1
```

### 3. 看 Pod 调度约束

```bash
kubectl get pod myapp-xxx -n prod -o yaml
```

重点关注：

- `nodeSelector`
- `affinity`
- `tolerations`
- `resources.requests`

## 六、存储查看：PVC、PV、StorageClass

很多有状态应用故障，本质上不是容器问题，而是卷的问题。

### 1. 查看 PVC

```bash
kubectl get pvc -n prod
kubectl describe pvc data-mysql-0 -n prod
```

你要看：

- 状态是不是 `Bound`
- 容量
- 绑定到了哪个 PV
- 事件里有没有 provisioning / mount 相关错误

### 2. 查看 PV

```bash
kubectl get pv
kubectl describe pv pvc-xxxxxxxx
```

重点关注：

- reclaim policy
- access modes
- capacity
- claim 绑定关系

### 3. 查看 StorageClass

```bash
kubectl get storageclass
kubectl describe storageclass standard
```

这对运维很重要，因为它决定了：

- 动态供给怎么做
- 是否允许扩容
- 使用哪个 provisioner

### 4. 快速串起来看

```bash
kubectl get pvc -n prod
kubectl get pv
kubectl get storageclass
```

这一组命令非常适合第一时间判断：

- PVC 有没有绑定
- PV 有没有准备好
- StorageClass 是否存在

## 七、存储相关的常见运维命令

### 1. 看 Pod 挂载情况

```bash
kubectl describe pod mysql-0 -n prod
kubectl get pod mysql-0 -n prod -o yaml
```

你要确认：

- volume 来源
- volumeMount 挂载路径
- PVC 名称

### 2. 检查目录内容

```bash
kubectl exec -it mysql-0 -n prod -- sh
kubectl exec mysql-0 -n prod -- df -h
kubectl exec mysql-0 -n prod -- mount
```

适用场景：

- 怀疑卷没挂上
- 怀疑挂载点不对
- 怀疑空间快满了

### 3. 扩容后查看结果

```bash
kubectl describe pvc data-mysql-0 -n prod
kubectl get pvc data-mysql-0 -n prod
kubectl exec mysql-0 -n prod -- df -h
```

注意：

- PVC 扩容成功不等于文件系统已经在容器里可见
- 需要确认底层存储和文件系统扩容都完成

## 八、节点和存储的高风险动作提醒

### 1. `drain` 前一定要确认影响面

尤其是：

- 单副本业务
- 本地盘业务
- 有状态组件
- 没有 PDB 保护的关键服务

### 2. 删除 PVC 前先确认 `reclaimPolicy`

先看：

```bash
kubectl describe pvc data-mysql-0 -n prod
kubectl describe pv pvc-xxxxxxxx
```

为什么？

- 有些场景删除 PVC 后，后端数据卷也可能随之被回收
- 这不只是“删一个对象”，可能是删数据

### 3. 不要把节点问题当成业务问题

如果同一节点上的多个业务一起异常，优先怀疑：

- 节点资源
- 节点网络
- 容器运行时
- 节点磁盘

## 九、常见工作场景命令顺序

### 场景 1：计划维护一个节点

```bash
# 1. 看节点和负载
kubectl get nodes
kubectl describe node node-1
kubectl get pods -A --field-selector spec.nodeName=node-1

# 2. 冻结调度
kubectl cordon node-1

# 3. 驱逐工作负载
kubectl drain node-1 --ignore-daemonsets --delete-emptydir-data

# 4. 维护结束后恢复
kubectl uncordon node-1
```

### 场景 2：数据库 Pod 一直起不来，怀疑是卷问题

```bash
kubectl get pod mysql-0 -n prod
kubectl describe pod mysql-0 -n prod
kubectl get pvc -n prod
kubectl describe pvc data-mysql-0 -n prod
kubectl get pv
kubectl describe pv <pv-name>
```

### 场景 3：Pod 一直 Pending，怀疑调度约束有问题

```bash
kubectl describe pod myapp-xxx -n prod
kubectl get node --show-labels
kubectl describe node node-1
kubectl get pod myapp-xxx -n prod -o yaml
```

## 十、这一篇最值得记住的命令

```bash
kubectl get nodes -o wide
kubectl describe node <node>
kubectl top nodes
kubectl cordon <node>
kubectl drain <node> --ignore-daemonsets --delete-emptydir-data
kubectl uncordon <node>
kubectl taint nodes <node> key=value:NoSchedule
kubectl label node <node> disk=ssd
kubectl get pvc -n <ns>
kubectl describe pvc <pvc> -n <ns>
kubectl get pv
kubectl describe storageclass <sc>
```

## 下一篇

最后建议看：

- [高频运维命令组合与值班速查清单](./05-practical-command-combinations.md)

那一篇会把前面几篇内容再压缩成更偏实战的“值班速查版”。
