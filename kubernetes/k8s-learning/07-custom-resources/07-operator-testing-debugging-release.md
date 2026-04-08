# 🧪 Operator 测试、调试与发布

## 为什么这一篇是“从会写到能上线”的分水岭

很多人在学 Operator 时，最开始的兴奋点通常是：

- 我会写 CRD 了
- 我会写 Reconcile 了
- 我能让控制器自动创建 Deployment 了

但到了真正想把它放进团队环境、测试环境、甚至生产环境时，就会发现另一个现实问题：

> 代码能跑，不等于它能稳定工作；本地能跑，不等于它能交付上线。

这时真正决定一个 Operator 是否“像样”的，往往不是 API 定义本身，而是下面这些工程能力：

- 测试
- 调试
- 日志和状态可观测性
- 发布和升级
- 回滚和兼容

所以这一篇的目标，就是把一个 Operator 从“能写”推进到“能验证、能调试、能发布”。

## 1. 为什么 Operator 比普通应用更需要测试

普通 Web 服务的主要逻辑通常是：

- 接请求
- 处理逻辑
- 回响应

而 Operator 的逻辑更像：

- 监听资源变化
- 读当前状态
- 推导目标状态
- 操作集群对象
- 反复重试与收敛

这意味着它天然有几个更容易出问题的点：

- 幂等性
- 状态漂移
- 事件顺序
- 删除逻辑
- 并发与重试
- 与 Kubernetes API 的交互细节

所以 Operator 如果没有测试体系，出问题的概率会远高于普通业务代码。

## 2. 先分清楚几类测试

做 Operator 时，建议把测试拆成 4 层来看：

1. **资源构造与纯逻辑测试**
2. **Reconcile 单元测试**
3. **控制器集成测试**
4. **端到端测试**

## 3. 第一层：资源构造与纯逻辑测试

这是最容易被忽略，但性价比非常高的一层。

例如你通常会写一些函数：

- `desiredDeployment(app)`
- `desiredService(app)`
- `buildConditions(app, deploy)`
- `calculatePhase(...)`

这些函数如果能被拆得足够纯，就很适合做普通单元测试。

### 为什么这一层很重要

因为它能快速验证：

- labels 是否一致
- selector 是否正确
- 副本数是否正确映射
- 端口是否正确
- 状态计算逻辑是否符合预期

这种测试：

- 执行快
- 成本低
- 定位直接

## 4. 第二层：Reconcile 单元测试

这层主要测试：

- 当给定一个 CR 时
- Reconcile 是否会创建正确的下游资源
- 状态是否会被正确更新
- 删除场景是否能正确清理

### 你最想验证的事情

例如：

1. 如果 `AppService` 不存在 Deployment，Reconcile 是否创建 Deployment
2. 如果 `spec.expose=true`，是否创建 Service
3. 如果副本数变化，是否更新 Deployment
4. 如果对象删除，是否处理 Finalizer

### 这一层的意义

它让你验证的不是“某个小函数对不对”，而是：

> 控制器的关键决策路径是否正确。

## 5. 第三层：集成测试

集成测试的重点是：

- 不只是看你函数返回值
- 而是看控制器和 Kubernetes API 行为是否真实协同

在 Operator 开发里，常见做法是使用 `envtest` 这类测试环境。

### 它大概能帮助你做什么

- 启一个轻量测试 API 环境
- 注册 CRD
- 启动控制器
- 提交 CR
- 观察状态与下游资源变化

这样你能更接近真实 Kubernetes 交互，而不是只在内存里“假装”有集群。

## 6. 第四层：端到端测试

端到端测试最接近真实环境。  
它回答的问题是：

> 这个 Operator 在真实集群里，真的能从创建 CR 到完成资源收敛吗？

典型验证路径：

1. 安装 CRD
2. 部署 Operator
3. 提交样例 CR
4. 观察 Deployment / StatefulSet / Service 是否被创建
5. 观察 Ready 状态是否更新
6. 修改 CR，再看控制器是否收敛
7. 删除 CR，再看清理是否完成

这层测试最贵，但价值也最大。

## 7. Operator 测试时最应该覆盖的场景

### 7.1 创建路径

验证：

- 新建 CR 后，是否创建下游资源
- 默认值和补全是否正确

### 7.2 更新路径

验证：

- `spec` 修改后，是否正确收敛
- 是否出现错误更新、遗漏更新、重复更新

### 7.3 删除路径

验证：

- Finalizer 是否生效
- 是否正确清理下游资源或外部资源
- 是否会卡在 `Terminating`

### 7.4 异常路径

验证：

- 下游资源创建失败怎么办
- 状态更新失败怎么办
- 是否会出现无意义重试
- 错误能否被清晰写入状态或日志

## 8. 调试 Operator 的基本思路

Operator 出问题时，不能只盯代码，也不能只盯 CR YAML。  
更有效的调试方式通常是沿着这条链看：

```text
CR -> 控制器日志 -> 下游资源 -> 事件 -> status -> 集群行为
```

## 9. 第一步先看什么：CR 自身状态

优先检查：

```bash
kubectl get <your-resource> -A
kubectl describe <your-resource> <name> -n <ns>
kubectl get <your-resource> <name> -o yaml
```

重点看：

- `spec`
- `status`
- `conditions`
- `observedGeneration`
- metadata 中是否有 finalizer

### 为什么先看它

因为很多问题本质上不是“控制器完全没工作”，而是：

- 用户改了 spec
- 控制器还没处理到
- 或者控制器已经处理，但状态没收敛

## 10. 第二步看控制器日志

这是最核心的一步之一。

优先命令通常是：

```bash
kubectl logs -n <operator-namespace> deploy/<operator-name>
```

如果是本地运行控制器，就直接看本地标准输出。

### 日志里最值得打的内容

一个成熟的 Operator，日志至少应能让你快速看清：

- 当前处理的是哪个对象
- 当前在哪个 Reconcile 阶段
- 创建/更新了哪些下游资源
- 错误发生在哪一步
- 是否会重试

### 常见好日志模式

最好日志里始终带：

- 资源名
- 命名空间
- generation
- 关键动作名称

这样排查时速度会快很多。

## 11. 第三步看下游资源

Operator 的问题很多时候不是 CR 本身错，而是：

- 下游 Deployment 没创建
- Service selector 错了
- StatefulSet 没更新
- PVC 绑定失败

所以要继续看：

```bash
kubectl get deploy,svc,sts,pvc -n <ns>
kubectl describe deploy <name> -n <ns>
kubectl describe svc <name> -n <ns>
```

你要确认：

- 这些资源是否存在
- labels / selector 是否正确
- 副本数是否一致
- 事件里有没有错误

## 12. 第四步看事件

事件对 Operator 排障特别有帮助，因为很多失败不是你代码 panic，而是集群对象创建后运行失败。

例如：

- Pod 拉镜像失败
- PVC Pending
- 探针失败
- 调度失败

可以看：

```bash
kubectl get events -n <ns> --sort-by=.metadata.creationTimestamp
```

这一步能帮你把“控制器逻辑问题”和“下游资源运行问题”区分开。

## 13. `observedGeneration` 是调试利器

当你修改了一个 CR 后，如果发现行为不对，第一时间就该看：

- `metadata.generation`
- `status.observedGeneration`

### 判断方法

如果：

- `generation = 7`
- `observedGeneration = 6`

说明：

- 用户配置已经更新
- 控制器状态还没处理到最新版本

这类信息在判断“到底是控制器没处理，还是已经处理但结果不对”时非常有用。

## 14. 条件（Conditions）为什么不仅是给用户看的

很多人只把 `conditions` 当展示字段，但在调试里它其实非常重要。

例如：

```yaml
conditions:
- type: Ready
  status: "False"
  reason: DeploymentNotAvailable
  message: "waiting for deployment replicas to become available"
```

这会比单纯一个：

```yaml
phase: Failed
```

更有可调试价值。

## 15. Operator 常见故障类型

### 15.1 CRD 没安装或版本不匹配

表现：

- 资源提交失败
- 控制器无法识别资源

### 15.2 RBAC 不足

表现：

- 控制器能启动
- 但创建/更新资源时报权限错误

### 15.3 Reconcile 逻辑不幂等

表现：

- 重复创建
- 资源不断被 patch
- 状态来回抖动

### 15.4 状态更新逻辑有问题

表现：

- `status` 不更新
- `observedGeneration` 不前进
- conditions 一直不对

### 15.5 删除逻辑卡死

表现：

- 资源一直 `Terminating`
- Finalizer 没移除

## 16. 发布前的检查清单

在真正发布一个 Operator 前，建议至少检查下面这些东西。

### API 层

- CRD schema 是否足够严格
- 默认值是否明确
- 打印列是否友好
- 版本命名是否合理

### 控制器层

- Reconcile 是否幂等
- 删除逻辑是否完整
- 状态是否可读
- 错误是否有清晰日志

### 权限层

- RBAC 是否够用
- 是否过度授权

### 工程层

- 镜像是否可构建
- 部署清单是否可安装
- sample CR 是否可用

## 17. 发布流程建议

一个更稳妥的 Operator 发布流程通常是：

1. 本地单元测试通过
2. 集成测试通过
3. 在测试集群部署并验证样例 CR
4. 验证升级和删除路径
5. 再进入预发布或生产

不要跳过“真实集群验证”这一步。

## 18. 升级与回滚要提前设计

Operator 发布不是只发一个镜像。  
很多时候还伴随：

- CRD 变更
- API 版本演进
- 状态字段变化
- 控制逻辑变化

所以必须考虑：

- 先升级 CRD 还是先升级 controller
- 旧 CR 是否兼容
- 升级失败如何回滚

这也是为什么前面的“版本设计”章节很重要。

## 19. 一个最实用的教材式调试顺序

当用户反馈“这个 Operator 不工作”时，推荐按下面顺序排：

1. 看 CR 是否创建成功
2. 看 `spec` 和 `status` 是否合理
3. 看 `observedGeneration`
4. 看 Operator 日志
5. 看下游资源是否创建或更新
6. 看事件
7. 看权限、版本、Finalizer、删除路径

这套顺序非常适合教学，也适合实战。

## 20. 一页总结

- Operator 的价值不只在 API 和 Reconcile，还在测试、调试和发布能力
- 测试建议至少覆盖：纯逻辑、Reconcile、集成、端到端
- 调试时优先顺着 `CR -> status -> logs -> child resources -> events` 这条链排查
- 发布前必须检查 schema、幂等性、RBAC、状态、删除逻辑和升级路径
- 一个“能上线”的 Operator，一定是可测、可调、可发布、可回滚的

## 下一步

- 返回 [自定义资源专题总览](./README.md)
