# 🧭 Kubernetes 运维指令大全

## 这个专题是干什么的

如果说 `03-practice/01-kubectl-commands.md` 更像一份 `kubectl` 功能手册，
那这个专题更像一份**面向运维、SRE、DevOps、值班排障场景的命令作战手册**。

这里不只是罗列命令，而是按真实工作方式来组织内容：

- 先告诉你这个场景要解决什么问题
- 再告诉你应该先看什么、后看什么
- 然后给出能直接复制的命令
- 最后补上为什么这样查、容易踩什么坑、线上该注意什么

也就是说，这一套文档的目标不是让你“记住命令”，而是让你逐渐形成：

- 看资源的顺序
- 处理故障的路径
- 做变更时的风险意识
- 面对线上问题时的第一反应

## 适合谁读

这套专题尤其适合下面这些人：

- 刚开始接手 Kubernetes 集群运维的人
- 已经会一些 `kubectl`，但排障顺序经常混乱的人
- 负责发布、扩缩容、节点维护、值班应急的人
- 想把“会执行命令”升级成“知道为什么这样执行”的人

## 和已有文档的关系

仓库里已经有几篇和命令、运维有关的内容：

- [kubectl 命令完全手册](../03-practice/01-kubectl-commands.md)
- [常见运维操作指南](../03-practice/03-operations.md)
- [故障排查与调试](../03-practice/04-troubleshooting.md)
- [Kubernetes 命令速查表](../CHEATSHEET.md)

你可以这样理解它们的分工：

- `CHEATSHEET.md`：偏速查，适合临时翻一下
- `03-practice/01-kubectl-commands.md`：偏命令功能总览
- `03-practice/03-operations.md`：偏常见运维动作
- `03-practice/04-troubleshooting.md`：偏故障处理思路
- `09-ops-command-handbook`：偏**运维值班视角 + 场景化执行手册**

## 这个专题会讲什么

本专题按最常见的运维工作流拆成 5 个部分：

1. [集群连接、上下文切换与资源巡检](./01-cluster-context-and-inspection.md)
2. [工作负载发布、变更、回滚与扩缩容](./02-workload-release-and-change.md)
3. [日志、调试、容器排障与应急定位](./03-logs-debug-and-troubleshooting.md)
4. [节点维护、调度控制与存储相关命令](./04-node-scheduling-and-storage.md)
5. [高频运维命令组合与值班速查清单](./05-practical-command-combinations.md)

## 推荐阅读顺序

如果你是第一次系统看 Kubernetes 运维命令，建议按下面顺序学：

1. 先看 [集群连接、上下文切换与资源巡检](./01-cluster-context-and-inspection.md)
2. 再看 [工作负载发布、变更、回滚与扩缩容](./02-workload-release-and-change.md)
3. 然后看 [日志、调试、容器排障与应急定位](./03-logs-debug-and-troubleshooting.md)
4. 接着看 [节点维护、调度控制与存储相关命令](./04-node-scheduling-and-storage.md)
5. 最后把 [高频运维命令组合与值班速查清单](./05-practical-command-combinations.md) 当作日常手册反复查

## 学这一套时最重要的思路

很多人学命令时，容易掉进两个坑：

1. 只记参数，不理解排障路径
2. 只会执行命令，不知道什么时候不该执行

所以你读这套专题时，请始终带着下面 4 个问题：

1. 这个命令一般在哪个场景下用
2. 它对线上有没有副作用
3. 执行前我需要先确认什么
4. 执行后我应该怎样验证结果

如果你能把这 4 个问题养成习惯，你就会从“会敲命令”逐渐变成“会做运维”。

## 一条非常重要的原则

线上运维不是“命令越多越厉害”，而是：

> 先观察，后操作；先缩小范围，后执行变更；先确认影响面，后做不可逆动作。

比如：

- `kubectl delete pod` 很常见，但你要先确认它是不是单副本业务
- `kubectl drain node` 很常见，但你要先确认它会不会影响本机上的关键工作负载
- `kubectl rollout restart` 很方便，但你要先确认当前镜像、配置和探针是否都健康

这些“先确认”的意识，比背更多命令更重要。

## 学完后你应该具备什么能力

读完这个专题后，你应该能比较熟练地做这些事：

- 快速切换上下文，确认自己连的是哪个集群、哪个 namespace
- 用正确顺序查看 Pod、Deployment、Service、Event、Node、PVC 的状态
- 做常见变更，比如发版、改镜像、改副本数、重启、回滚
- 遇到 `Pending`、`CrashLoopBackOff`、`ImagePullBackOff`、服务不通时能逐层定位
- 处理节点维护、驱逐、污点、标签、存储查看这些日常运维动作
- 把高频命令组合成你自己的值班 SOP

## 建议怎么用

这套文档最适合两种使用方式：

### 第一种：系统学习

你可以从头顺着看，把每个命令都在测试集群实际跑一遍。

建议每学一类命令，就做 3 件事：

1. 自己手敲一遍
2. 观察资源状态变化
3. 用一句话解释“为什么现在要用它”

### 第二种：工作速查

如果你是在值班、排障、发版时临时查命令，可以先看第 5 篇：

- [高频运维命令组合与值班速查清单](./05-practical-command-combinations.md)

里面会把很多高频命令按“先查什么、再查什么”的顺序帮你串起来。

## 下一步

建议先从：

- [集群连接、上下文切换与资源巡检](./01-cluster-context-and-inspection.md)

开始，这一篇会帮你把“我现在到底连的是谁、该先看什么”这件事彻底理顺。
