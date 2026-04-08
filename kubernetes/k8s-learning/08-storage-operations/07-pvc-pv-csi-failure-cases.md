# 🚨 PVC/PV/CSI 故障案例库与排障剧本

## 为什么这一篇要做成“案例库”

前面的存储章节已经讲了很多原理，也讲了运维方法。  
但真正到了线上排障时，很多人还是会遇到这样的情况：

> 我知道 PVC、PV、StorageClass、CSI 是什么，但我一看到 `Pending`、`ContainerCreating`、卷挂载失败，还是不知道第一步该看哪。

这是因为：

- 原理认知是一回事
- 真正面对具体故障症状又是另一回事

所以这一篇不再只按对象讲，而是按**故障现象**来组织，做成更接近值班和故障手册的形式。

你可以把它当作：

- 线上问题定位手册
- 常见存储故障剧本
- 学习排障路径的案例教材

## 1. 存储排障的总原则

当你面对存储问题时，先不要急着做破坏性操作。  
最重要的不是“赶紧删 Pod 再试”，而是先判断：

1. 数据有没有风险
2. 问题是在声明层、绑定层、挂载层还是后端存储层
3. 当前症状是“资源没创建好”还是“数据已经有损坏风险”

一个总的排障顺序通常是：

```text
症状
  -> Pod
  -> PVC
  -> PV
  -> StorageClass
  -> CSI 控制面/节点侧
  -> 后端存储系统
```

## 2. 故障案例 1：PVC 一直 `Pending`

### 表现

```bash
kubectl get pvc
```

你会看到：

```text
NAME      STATUS    VOLUME   CAPACITY   ACCESS MODES   STORAGECLASS
db-data   Pending
```

### 这类问题的本质

它通常意味着：

- PVC 已经创建
- 但系统还没给它找到或创建合适的卷

### 第一步看什么

```bash
kubectl describe pvc db-data
kubectl get storageclass
kubectl get pv
```

### 最常见原因

- `storageClassName` 不存在
- 没有默认 StorageClass
- 动态供给器异常
- 申请容量过大
- `accessModes` 不匹配

### 排障剧本

1. 看 PVC 事件里是否有 provisioner 报错
2. 看当前集群有没有默认 StorageClass
3. 看指定的 StorageClass 名字有没有拼错
4. 看是否存在可匹配的 PV
5. 如果走动态供给，再看 CSI 控制器是否正常

### 危险动作提醒

这种问题一般还不涉及数据破坏，优先排查，不要急着删 PVC 重建，尤其在生产中。

## 3. 故障案例 2：Pod 卡在 `ContainerCreating`

### 表现

```bash
kubectl get pods
```

看到 Pod 长时间停在：

```text
STATUS = ContainerCreating
```

### 这类问题的常见本质

很多时候不是镜像拉取问题，而是：

- 卷还没挂上
- 节点 attach 失败
- 文件系统 mount 失败

### 第一步看什么

```bash
kubectl describe pod <pod-name>
```

重点看 Events 中是否出现：

- attach volume 失败
- mount volume 失败
- waiting for volumes

### 第二步联动检查

```bash
kubectl get pvc
kubectl describe pvc <pvc-name>
kubectl describe node <node-name>
```

### 排障剧本

1. 确认 PVC 是否 `Bound`
2. 确认 Pod 引用的 PVC 名称是否正确
3. 看节点是否真的拿到了卷
4. 看 CSI Node 侧是否有挂载错误
5. 看底层卷是否已存在但不可用

## 4. 故障案例 3：PVC 已 `Bound`，但应用仍报磁盘不可用

### 表现

- Pod 已 Running
- PVC 已 Bound
- 但应用日志报：
  - 没权限写目录
  - 磁盘不可写
  - 文件不存在

### 这类问题的常见根因

- mountPath 不对
- 文件权限不对
- 容器用户与卷权限不匹配
- 卷以只读方式挂载

### 第一步看什么

```bash
kubectl describe pod <pod-name>
kubectl exec -it <pod-name> -- sh
```

进入容器后检查：

- 挂载点是否存在
- 权限是否正确
- 是否可写

### 排障剧本

1. 检查 `volumeMounts.mountPath`
2. 检查容器用户和文件权限
3. 检查是否配置了 `readOnly: true`
4. 检查应用实际写入目录和你以为的目录是否一致

## 5. 故障案例 4：Pod 重建后数据“丢了”

### 表现

- 重启或重建后应用数据消失
- 但用户以为自己“用了卷”

### 这类问题最常见的根因

- 实际用的是 `emptyDir`
- 实际写入目录不在持久卷挂载路径上
- 新 Pod 挂到了新卷
- 原 PVC / PV 被删除或回收

### 第一步看什么

```bash
kubectl get pod <pod-name> -o yaml
kubectl get pvc
kubectl get pv
```

### 排障剧本

1. 看 Pod 用的到底是什么卷类型
2. 看应用真正写入路径是否挂在持久卷上
3. 看 PVC 是否还存在
4. 看原 PV 是否已经被回收
5. 看回收策略是不是 `Delete`

### 教学重点

这类案例特别适合拿来反复提醒初学者：

> “挂了一个 volume” 不等于 “数据一定持久化”。

## 6. 故障案例 5：卷无法重新附加到新节点

### 表现

- 节点故障后 Pod 被调度到新节点
- 但卷迟迟附加不上

### 可能根因

- 卷有可用区限制
- 后端卷还被旧节点占用
- CSI attach 流程异常
- 本地卷本来就无法跨节点迁移

### 第一步看什么

```bash
kubectl describe pod <pod-name>
kubectl describe pvc <pvc-name>
kubectl describe pv <pv-name>
kubectl get nodes -o wide
```

### 排障剧本

1. 看 PV 是否有拓扑或节点约束
2. 看旧节点是否仍处于异常但未释放状态
3. 看新节点是否满足卷挂载条件
4. 看 CSI attach 事件是否报错

## 7. 故障案例 6：StorageClass 看起来正常，但卷就是创建不出来

### 表现

- StorageClass 存在
- PVC 也引用了它
- 但一直无法创建卷

### 常见根因

- provisioner 名字和实际驱动不匹配
- CSI Controller 异常
- 后端存储凭证失效
- 参数错误

### 第一步看什么

```bash
kubectl describe storageclass <sc-name>
kubectl describe pvc <pvc-name>
```

### 第二步看什么

需要继续看 CSI 相关控制器或驱动日志。  
核心问题是：

- 请求有没有到驱动
- 驱动为什么拒绝或失败

### 排障剧本

1. 看 provisioner 是否存在
2. 看 StorageClass 参数是否合理
3. 看 CSI Controller 日志
4. 看后端存储系统状态

## 8. 故障案例 7：PVC 扩容后容量没生效

### 表现

- YAML 中容量已经变大
- PVC 也似乎更新了
- 但应用内部看到的空间没变

### 常见根因

- 只改了 PVC，底层没扩容成功
- 卷扩了，但文件系统没扩
- 驱动支持不完整
- 应用视角没刷新

### 第一步看什么

```bash
kubectl describe pvc <pvc-name>
kubectl get pv <pv-name> -o yaml
```

### 排障剧本

1. 看 StorageClass 是否允许扩容
2. 看底层卷是否真的扩容完成
3. 看文件系统扩展是否成功
4. 进容器看实际磁盘空间

## 9. 故障案例 8：卷删除后后端数据也跟着消失

### 表现

- 删除了 PVC
- 结果后端真实数据卷也没了

### 常见根因

- `reclaimPolicy = Delete`

### 第一步看什么

如果事故已经发生，先不要再删更多资源。  
应优先确认：

- 是否有快照
- 是否有备份
- 其他环境是否还有副本

### 教学重点

这是所有存储教学里最应该反复强调的一类事故：

> 删除 PVC 不是“删一个声明对象”那么简单，它可能触发真实卷删除。

## 10. 故障案例 9：CSI 驱动升级后大量卷操作异常

### 表现

- 之前正常的卷创建/挂载突然异常
- 多个业务同时受影响

### 这类问题的特点

这往往已经不是单个业务 YAML 问题，而是平台层故障。

### 第一步看什么

- 是否是最近驱动升级后出现
- 是否多个 StorageClass 同时异常
- 是否多个业务同时受影响

### 排障剧本

1. 先界定故障面
2. 判断是否为平台级存储能力故障
3. 停止高风险变更
4. 查驱动版本、兼容性、日志
5. 评估是否回滚驱动版本

## 11. 故障案例 10：日志或监控系统磁盘快速打满

### 表现

- PVC 使用率急速上升
- 应用开始报磁盘不足
- 写入延迟升高或服务异常

### 典型根因

- 数据保留策略不合理
- 清理任务失效
- 写入流量激增
- 存储容量规划不足

### 排障剧本

1. 先确认是否为可快速清理的数据
2. 确认是否有短期扩容能力
3. 确认保留和归档策略是否失效
4. 评估业务影响范围

这类问题特别常出现在：

- 日志系统
- 监控系统
- 搜索系统

## 12. 一套通用排障命令模板

下面这套命令非常适合当值班排障的第一轮工具箱：

```bash
# 看 Pod
kubectl get pods -A
kubectl describe pod <pod-name>

# 看 PVC / PV
kubectl get pvc -A
kubectl describe pvc <pvc-name>
kubectl get pv
kubectl describe pv <pv-name>

# 看 StorageClass
kubectl get storageclass
kubectl describe storageclass <sc-name>

# 看事件
kubectl get events -A --sort-by=.metadata.creationTimestamp

# 看节点
kubectl get nodes -o wide
kubectl describe node <node-name>
```

## 13. 值班时最重要的行为原则

### 13.1 先确认数据风险，再操作

尤其当问题涉及：

- 删除 PVC
- 重新绑定卷
- 卷迁移
- 故障恢复

### 13.2 先做信息采集，再做破坏性动作

不要一看到 Pod 卡住就立即：

- 删除 Pod
- 删除 PVC
- 重建资源

因为你可能会把可观测信息和恢复机会一起删掉。

### 13.3 区分平台级故障和单业务故障

如果多个业务同时卷异常，优先怀疑：

- CSI
- StorageClass
- 后端存储

而不是每个业务单独 YAML 都出错。

## 14. 教科书式故障判断框架

看到问题时，你可以先用这一套问题快速分类：

1. 是“没绑定”还是“没挂载”
2. 是“卷没准备好”还是“应用不会用”
3. 是“业务单点问题”还是“平台层存储故障”
4. 是“对象状态问题”还是“数据安全问题”

这套判断框架能极大提升排障效率。

## 15. 一页总结

- 存储故障排障一定要按症状和层级联合判断
- 常见链路是：Pod -> PVC -> PV -> StorageClass -> CSI -> 后端存储
- 最危险的不是“卷没创建”，而是误删、误回收和错误恢复
- 面对存储问题，最重要的是先保护数据，再修系统
- 一个成熟团队应该把常见故障沉淀成剧本，而不是只靠经验拍脑袋处理

## 下一步

- 返回 [存储与运维专题](./README.md)
