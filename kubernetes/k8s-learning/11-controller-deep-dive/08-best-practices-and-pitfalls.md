# 🚧 控制器最佳实践与常见坑

> 把控制器跑到生产里、扛住各种边界条件，需要把前面 7 篇里的概念全部内化。这一篇按主题归纳"做对的事"和"避免的坑"。

## 1. 设计层面

### ✅ 把控制器写成"对账函数"
- 每次 reconcile 都从最新 spec 出发；不依赖事件本身
- 不要在控制器内存里保留"这个 wf 我已经处理过"的标记

### ✅ 单一职责
- 一个控制器只对一种资源负责（即"主资源"）
- 多种资源协同：用 Owns / Watches，**不要在一个 reconcile 里管两个并列资源的生命周期**

### ✅ 字段语义明确
- spec 表达"我要什么"
- status 表达"实际是什么"
- 永远不要在 status 里写用户输入

### ❌ 不要把控制器逻辑写进 admission webhook
- webhook 只做"拒绝/补全 spec"，不要在里面创建 / 删除资源
- 业务逻辑放控制器里，让它幂等地收敛

### ❌ 不要乱用周期性 RequeueAfter
- 反应应该靠 watch；RequeueAfter 是兜底（外部资源 poll、清理冷启动等）
- RequeueAfter 1 秒会把单 key 打成"实际 1Hz 的 controller-watcher"，很贵

## 2. 性能与缓存

### ✅ 永远从 cache / Lister 读
```go
// good
pod, err := r.cache.Get(ctx, key, &corev1.Pod{})
// bad（除非确实需要立即一致）
pod, err := r.apiReader.Get(ctx, key, &corev1.Pod{})
```

### ✅ 限制 cache 的资源范围
- 不需要全 namespace 时，明确指定 namespace
- 不需要某些大字段时，用 `Cache.TransformByObject` 在入 cache 前移除（如剥掉 Pod 的大 annotations）
- 配置 `LabelSelector` / `FieldSelector` 让 cache 只装相关对象

### ✅ 索引（Indexer）来加速查找
```go
mgr.GetCache().IndexField(ctx, &myv1.Foo{}, "spec.targetRef.name", func(obj client.Object) []string {
    return []string{obj.(*myv1.Foo).Spec.TargetRef.Name}
})
```
之后 `client.MatchingFields{"spec.targetRef.name": "x"}` O(1) 查询。

### ❌ 千万不要在 reconcile 里做 list 全量
- `r.List(ctx, &podList)` 在大集群是性能炸弹
- 改成 `client.MatchingLabels` / `client.InNamespace` / 索引

### ❌ 不要把对象引用塞到 controller 自己的 map 里
- cache 已经有完整一份；自己维护会 OOM、会脏
- 需要二级数据用 IndexField

## 3. 写操作

### ✅ 优先 Patch，少用 Update
- Patch 不会因为别人改了别的字段而失败
- 推荐 `client.MergeFrom(original)` 或 `client.Apply` (Server-Side Apply)

### ✅ Server-Side Apply 适合多控制器协作
```go
return r.Patch(ctx, desired, client.Apply, client.FieldOwner("my-controller"))
```
不同 FieldOwner 编辑不同字段不会冲突，K8s 会自动合并。

### ❌ Update 整个对象（含 status）
```go
// bad: 这会把 status 也覆盖
r.Update(ctx, obj)
```
spec 用主资源 client，status 用 `Status()` 子资源 client。

### ❌ 在循环里频繁写
- 每个变化就写一次会让 informer 反弹回来
- 累积变化在一次 reconcile 末尾统一写

## 4. 错误处理

### ✅ 区分错误类型
| 错误 | 处理 |
|------|------|
| NotFound | return nil（被删了，没事） |
| Conflict | return err（让 workqueue 重试） |
| 其它瞬时（网络/5xx） | return err |
| spec 不合法 | 写 condition + return nil |

### ✅ 失败上限 + 兜底
```go
if queue.NumRequeues(key) > 100 {
    log.Error("give up", "key", key)
    queue.Forget(key)
    return
}
```

### ❌ 不要在 reconcile 里 retry + sleep
- 用返回 err + workqueue 退避
- `time.Sleep` 会占住 worker，阻塞其它对象

### ❌ 不要吞掉错误
- 返回 nil 等于"我已经搞定了"
- 真没搞定却返回 nil → 控制器永远不会再回来 → 状态错了也无人修

## 5. 删除与 finalizer

### ✅ 标准化的 finalizer 名字
`<域名>/<动作>` 例如 `myapp.example.com/cleanup`

### ✅ 清理函数幂等
- "已经不存在"不算错误
- 用 status 里记录的外部 ID 而不是 spec

### ✅ OwnerReference + 受控级联
- 子资源加 OwnerReference，K8s GC 自动级联删
- 跨外部系统资源用 finalizer

### ❌ 永远卡 Terminating
- 部署多副本 + leader election，避免"持有 finalizer 的 controller 不在了"
- 清理函数加超时 + 最大重试，避免永远报错

### ❌ 添加 / 移除 finalizer 时机搞反
- 创建路径：先加 finalizer，再 do business
- 删除路径：先 cleanup，再移 finalizer

## 6. status 与可观测性

### ✅ observedGeneration
status 里加，让用户能判断"控制器已经看见我最新的改动"。

### ✅ Conditions 比 phase 强
- 多维状态用 Conditions
- 旧 condition 留着翻状态，不要删

### ✅ Events 记录历史
```go
r.Recorder.Eventf(obj, corev1.EventTypeNormal, "Provisioned", "...")
```

### ✅ Metrics
- workqueue depth / latency
- reconcile 总数 / 错误数
- 业务级指标（创建外部资源耗时、第三方 API QPS 等）

controller-runtime 默认暴露了一组指标，挂上 ServiceMonitor 即可。

### ❌ status 反弹回路
- 改 status 又触发自己 reconcile，又改 status... 直到 OOM
- 解法：predicate 过滤 status only changes；写 status 前对比是否真变化

## 7. 高可用

### ✅ 多副本 + Leader Election
- 默认开 `LeaderElection: true`
- 部署多副本（生产至少 2，建议 3）

### ✅ 配置 termination grace
- `terminationGracePeriodSeconds: 60`+ 让 leader 优雅释放 lease
- preStop hook 可以加快释放

### ✅ Lease 时长
默认 15s/10s/2s 适合大部分场景。慢节点 / 抖动多的环境可以放宽到 30/20/5。

### ❌ 多副本但忘了开 LeaderElection
立刻翻车：双 Reconcile + 状态来回弹 + 重复创建外部资源。

## 8. RBAC 与安全

### ✅ 最小权限
- 只授予用到的 verbs（get/list/watch/create/update/patch/delete）
- 区分 spec 客户端（用户）和 status 客户端（controller）

### ✅ ServiceAccount 单独
- 控制器自己的 SA
- 业务命名空间不要用 default SA

### ❌ 默认 cluster-admin
不要这样做。

## 9. 测试

### ✅ envtest 跑真 etcd + apiserver
- controller-runtime 自带 envtest
- 快、不需要装 minikube

### ✅ 测试要点
- 提交 spec → 等条件 → 断言资源 / status / events
- 模拟错误（删依赖、改坏 spec）→ 断言 condition + 重试
- 多次 reconcile 应当幂等（资源数不变）

### ❌ 只测 happy path
真实问题大多在异常路径，单测得 cover 失败、超时、被 GC 等。

## 10. 工程交付

### ✅ 用 Kubebuilder / operator-sdk 起项目
- 目录结构标准
- CRD / RBAC / webhook / Dockerfile 都有
- CI 模板齐全

### ✅ 多版本 CRD
- conversion webhook 把老版本对象转成新版
- 详见 [CRD 设计与版本演进](../07-custom-resources/03-crd-design-and-versioning.md)

### ✅ 渐进发布
- 部署前在测试集群跑 envtest + e2e
- 先发到非关键 namespace，再扩大

### ❌ 直接动 spec.schema 的破坏性修改
旧对象会校验失败，老 informer 反序列化失败。要走 conversion + 多版本兼容。

## 11. 常见怪事的速查

| 现象 | 90% 是这些原因 |
|------|---------------|
| reconcile 一直不触发 | informer 没 sync / RBAC 没 watch / leader 没选出来 |
| 拿不到刚创建的对象 | cache 还没同步；下次 reconcile 自然有；要立即一致用 APIReader |
| 同一对象 reconcile 死循环 | 自己写自己（status 变化触发 watch） |
| 频繁 Conflict | 多个 controller / actor 写同字段；改用 SSA |
| 删除卡 Terminating | finalizer 没移除；看 controller 日志为啥 cleanup 失败 |
| 高 CPU / 高内存 | cache 太大 / 入队太频繁 / 没限速 |
| 启动很慢 | 大集群初次 List 全量耗时；考虑限定 cache 范围 |
| Leader 频繁切换 | 网络抖动 / kube-apiserver 慢；调高 LeaseDuration |

## 12. 一份"上线前自查清单"

```text
[ ] reconcile 已经幂等（写过单测验证）
[ ] NotFound / Conflict / 其它错误分别处理
[ ] cache 范围已经按需缩小
[ ] 写操作走 Patch / SSA，不是 Update 整对象
[ ] status 用子资源、observedGeneration、Conditions 三件套
[ ] Finalizer 名字带域名前缀；删除路径正确
[ ] LeaderElection 开启；副本数 >= 2
[ ] RBAC 最小权限
[ ] metrics、healthz、readyz 已暴露
[ ] 监控告警已配（reconcile 错误率、queue depth、leader 切换）
[ ] 文档与 events 让运维能 describe 看懂状态
[ ] envtest + e2e 都过
[ ] CRD 有版本演进策略；存量数据兼容
```

## 13. 一句话送你出门

> 控制器写得好不好，**不是看代码量、看花活，而是看在异常路径上能不能保持自我修复**。
> 如果你的控制器：删依赖能恢复、网络抖能恢复、被踢主能恢复、用户写错 spec 能给出 condition——它就是一个合格的生产控制器。

## 总结：8 篇连起来回头看

读完整个专题，你应该能：

1. 用心智模型解释 Kubernetes 为什么是声明式 + Level-Triggered + 幂等
2. 讲清楚 Informer / Reflector / DeltaFIFO / Indexer / SharedInformer 的职责与协作
3. 用 Workqueue 写出标准的 worker 循环，理解限速器与退避
4. 写出处理 Finalizer、Owner、Status、Conditions 的完整 Reconcile
5. 部署一个高可用的多副本 controller（leader election + lease + RBAC）
6. 在 controller-runtime 与 client-go 之间做正确选型
7. 读懂内置控制器源码、拆解线上控制器的怪问题

带着这些回头去读你曾经写过、看过的控制器代码，会有完全不同的视角。

## 拓展阅读

- Kubernetes 官方文档：[Controllers](https://kubernetes.io/docs/concepts/architecture/controller/)
- controller-runtime：[github.com/kubernetes-sigs/controller-runtime](https://github.com/kubernetes-sigs/controller-runtime)
- Kubebuilder Book：[book.kubebuilder.io](https://book.kubebuilder.io/)
- 内置控制器源码导读：`kubernetes/pkg/controller/replicaset`、`deployment`
