# 排障、调试与生产清单

前面章节把机制拆开讲了。这一章按「出问题时怎么查」和「上线前要勾什么」收束，当作 client-go 模块的实战检查表。

## 先分清：客户端问题还是集群问题

| 信号 | 更像 |
|------|------|
| 仅你的进程失败，kubectl 正常 | 配置、RBAC、QPS、代码逻辑 |
| kubectl 也慢/超时 | API Server、网络、etcd、PF 限流 |
| 只有 Watch/Informer 异常 | RV、连接、resync、缓存未同步 |
| 只有写失败、读正常 | 冲突、校验、Admission、权限动词 |

本地先用同一份 kubeconfig / SA 跑：

```bash
kubectl auth can-i list pods --as=system:serviceaccount:default:pod-annotator
kubectl get --raw=/readyz?verbose
```

## 日志与可观测性最小集

控制器至少要能回答四个问题：

1. 现在是不是 Leader？
2. Informer 缓存同步了没有？
3. 队列有多深、重试了多少次？
4. 最近一次 Reconcile 失败原因是什么？

建议：

- `UserAgent` 带组件名与版本（见 [02](./02-client-setup.md)）
- 关键路径用 `klog`，失败带 `namespace/name`
- 对用户可见的结果打 Event（见 [07](./07-common-mechanisms.md)）
- 有余力再暴露：`workqueue_depth`、`reconcile_duration`、`reconcile_errors`

## 典型故障剧本

### 1. Lister 总是空 / 数据不全

**症状**：启动后立刻 List 缓存，结果缺失。

**原因**：没 `WaitForCacheSync`，或 Start 之前没注册要的 Informer。

**处理**：

```go
factory.Start(stopCh)
if !cache.WaitForCacheSync(stopCh, podInformer.HasSynced) {
    return fmt.Errorf("cache sync failed")
}
```

只有调用过 `factory.Core().V1().Pods()`（或等价 ForResource）的 Informer 才会被启动。

### 2. 疯狂 Reconcile / CPU 打满

**可能原因**：

- resync 周期过短 + 对象量巨大
- UpdateFunc 未过滤相同 `ResourceVersion`
- 每次 Reconcile 都写回对象，触发新事件，形成热循环
- 错误不可恢复却一直 `AddRateLimited`

**处理**：跳过假更新；写前比较期望与实际，无变化不 Update；Forbidden/Invalid 要 Forget；评估把 factory resync 设为 `0`。

### 3. 409 Conflict 刷屏

**原因**：并发更新同一对象；或拿着过期 `resourceVersion` Update。

**处理**：`retry.RetryOnConflict`；不要改 Lister 返回的原对象；SSA/Patch 缩小冲突面。见 [03](./03-crud-operations.md)。

### 4. Watch 断了又全量 List，API 飙高

Informer 在 `410 Gone` 或长时间断开后会重新 List，大集群上很贵。

**缓解**：缩小 List 范围（Namespace / LabelSelector）；Metadata Informer；避免同一资源建多个 factory；给客户端合理 QPS，但根因是减少全量 List 频率。

### 5. 多副本重复处理

未做 Leader Election，或选主与业务循环生命周期没绑在一起。

**处理**：Lease 选主；失去领导权要停 workers；Reconcile 仍保持幂等（选主不是正确性的唯一保障）。见 [07](./07-common-mechanisms.md)。

### 6. CRD / Dynamic：`no matches for kind`

Discovery 缓存过期，或 CRD 尚未 Established。

**处理**：等 CRD condition；`CachedDiscovery.Invalidate()`；短暂重试 RESTMapper。见 [06](./06-discovery-and-dynamic.md)。

### 7. 删除处理丢了 / panic

`DeleteFunc` 收到 `DeletedFinalStateUnknown`，直接断言 `*Pod` 失败。

**处理**：用 `cache.DeletionHandlingMetaNamespaceKeyFunc`，或手动解 tombstone。见 [04](./04-informer.md)。

## 调试技巧

### 提高日志级别

```bash
./pod-annotator -v=4
```

`klog.V(4)` 一类详细日志只在需要时打开。

### 对照 kubectl 看同一对象

```bash
kubectl get pod x -o yaml
kubectl get events --field-selector involvedObject.name=x
```

若 Event 有、对象无变化：多半是 Update 被拒或逻辑提前 return。  
若对象在变、你的 Handler 无日志：Informer 过滤条件（label/namespace）可能把对象排除了。

### 用 fake client 固定复现

把冲突、NotFound、Forbidden 写成单测，比连真集群稳。见 [07](./07-common-mechanisms.md)。

### 怀疑缓存陈旧时

- 写路径需要“刚写完就读最新”：对**自己写下的对象**再 Get，或信任 Watch 回传，不要假设 Lister 瞬时一致
- 跨对象决策（根据 A 改 B）：接受短暂延迟，靠再次入队收敛

## 上线前清单

### 配置与权限

- [ ] client-go / api / apimachinery 版本对齐
- [ ] UserAgent 可识别
- [ ] QPS/Burst 按对象规模评估
- [ ] RBAC 含 get/list/watch 与所需 write；选主还要 Lease
- [ ] 非 root、只读根文件系统等容器安全基线（按公司规范）

### Informer / 队列

- [ ] 使用 SharedInformerFactory，同资源不重复 Watch
- [ ] Start 前注册 Handler / Indexer
- [ ] WaitForCacheSync 后再 worker
- [ ] Handler 只入队，不重活
- [ ] DeepCopy 后再改
- [ ] 删除路径处理 tombstone
- [ ] 不可恢复错误不会无限重试

### 控制循环

- [ ] Reconcile 幂等
- [ ] 冲突有 RetryOnConflict 或等价逻辑
- [ ] 多副本有 Leader Election（或明确单副本）
- [ ] 优雅退出：处理 SIGTERM，ShutDown queue，ReleaseOnCancel

### 可观测

- [ ] 关键失败有日志 / Event
- [ ] 就绪探针：缓存已同步（且是 Leader，若需要）
- [ ] 知道如何用 `-v` 和 kubectl events 排障

## 机制对照速查

| 你想… | 用 |
|------|----|
| 读内置资源，类型安全 | Clientset |
| 读写 CRD / 任意资源 | Dynamic + RESTMapper |
| 长期跟随变化 | Informer + Lister |
| 可靠处理变化 | WorkQueue + 幂等 Reconcile |
| 多控制器改同一对象 | Patch / SSA |
| 少占内存只关心元数据 | Metadata Informer |
| 多副本高可用 | Leader Election |
| 单测 | Fake client / envtest |

## 本模块结束后

1. 回头扫一遍 [机制全景](./01-introduction.md)，看能否不看笔记画出数据流  
2. 把 [05 实战控制器](./05-controller-demo.md) 加上 Event 与选主  
3. 进入 [自定义资源专题](../07-custom-resources/README.md) 或 [控制器深度专题](../11-controller-deep-dive/README.md)

## 下一步

- [模块总览](./README.md)
- [控制器深度：最佳实践与坑](../11-controller-deep-dive/08-best-practices-and-pitfalls.md)
