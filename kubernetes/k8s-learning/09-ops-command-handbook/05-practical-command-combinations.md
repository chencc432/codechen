# 📚 高频运维命令组合与值班速查清单

## 这一篇怎么用

前面几篇已经按主题把命令讲开了。

这一篇不再强调“完整解释每个参数”，而是更偏向真实工作里的使用方式：

- 出现某类问题时先敲什么
- 接着查什么
- 最后怎么验证

你可以把它当成：

- 值班手册
- 故障第一响应清单
- 发版前后检查清单
- 日常巡检清单

## 一、每天上班先做的 5 分钟巡检

### 推荐命令

```bash
kubectl config current-context
kubectl get nodes
kubectl top nodes
kubectl get pods -A
kubectl get events -A --sort-by=.metadata.creationTimestamp
```

### 你想得到什么结论

通过这几条命令，你应该快速回答：

- 当前连的是哪个环境
- 有没有 NotReady 节点
- 有没有明显资源吃紧的节点
- 有没有大面积异常 Pod
- 最近集群是否出现集中报错事件

### 适合谁

- 值班工程师
- 负责日常稳定性的人
- 早上接班的人

## 二、某个业务“看起来挂了”时的第一轮定位

### 推荐命令

```bash
kubectl get pods -n prod
kubectl get svc -n prod
kubectl get ingress -n prod
kubectl get endpoints -n prod
kubectl get events -n prod --sort-by=.metadata.creationTimestamp
```

### 这一步解决什么问题

先判断故障大致落在哪一层：

- Pod 层
- Service 层
- Ingress 层
- 最近是否发生过异常事件

### 下一步

如果发现 Pod 异常，继续：

```bash
kubectl describe pod <pod> -n prod
kubectl logs <pod> -n prod --previous
kubectl logs <pod> -n prod
```

## 三、发版前检查清单

### 推荐命令

```bash
kubectl config current-context
kubectl get deploy myapp -n prod
kubectl rollout history deployment myapp -n prod
kubectl get pods -l app=myapp -n prod -o wide
kubectl diff -f deploy.yaml
```

### 发版前要确认什么

- 环境对不对
- 当前 Deployment 状态是否健康
- 是否有可回滚 revision
- 当前实例是否已经不稳定
- 本次清单到底改了什么

### 为什么重要

因为很多线上事故不是“变更本身太复杂”，而是：

- 在错环境发版
- 带着已有故障继续发版
- 不知道自己改了什么

## 四、发版后观察清单

### 推荐命令

```bash
kubectl rollout status deployment myapp -n prod
kubectl get pods -l app=myapp -n prod -w
kubectl logs -l app=myapp -n prod --tail=50
kubectl get events -n prod --sort-by=.metadata.creationTimestamp
```

### 你要确认什么

- 新副本是否起来
- 是否出现新的重启
- 是否出现探针失败
- 是否有镜像、挂载、调度类异常

### 如果异常怎么办

```bash
kubectl rollout undo deployment myapp -n prod
```

回滚后继续观察：

```bash
kubectl rollout status deployment myapp -n prod
kubectl get pods -l app=myapp -n prod
```

## 五、`CrashLoopBackOff` 处理清单

### 推荐命令

```bash
kubectl get pod <pod> -n prod
kubectl describe pod <pod> -n prod
kubectl logs <pod> -n prod --previous
kubectl logs <pod> -n prod
kubectl get events -n prod --sort-by=.metadata.creationTimestamp
```

### 高概率原因

- 启动参数错了
- 配置没挂进去
- 应用连不上依赖
- 内存不够被杀
- 探针配置过严

### 补充验证

```bash
kubectl exec -it <pod> -n prod -- sh
```

适合确认：

- 配置文件在不在
- 环境变量对不对
- 下游服务能不能通

## 六、`Pending` 处理清单

### 推荐命令

```bash
kubectl describe pod <pod> -n prod
kubectl get nodes
kubectl top nodes
kubectl get pvc -n prod
kubectl get node --show-labels
```

### 重点怀疑方向

- 节点资源不足
- 污点不匹配
- 节点标签 / 亲和性不匹配
- PVC 没准备好

### 如果你怀疑是节点约束

```bash
kubectl get pod <pod> -n prod -o yaml
kubectl describe node <node>
```

## 七、服务不通处理清单

### 推荐命令

```bash
kubectl get svc myapp -n prod
kubectl describe svc myapp -n prod
kubectl get endpoints myapp -n prod
kubectl get pods -l app=myapp -n prod -o wide
kubectl port-forward svc/myapp 8080:80 -n prod
```

### 你想回答的问题

- Service selector 对不对
- Service 有没有选到后端 Pod
- Pod 是不是 Ready
- 应用本身是不是正常监听
- 是应用问题，还是外部网络链路问题

## 八、节点异常处理清单

### 推荐命令

```bash
kubectl get nodes
kubectl describe node node-1
kubectl top node node-1
kubectl get pods -A --field-selector spec.nodeName=node-1
kubectl get events -A --sort-by=.metadata.creationTimestamp
```

### 重点关注

- Ready 状态
- 内存、磁盘、网络相关 condition
- 节点上的关键 Pod 是否集中异常
- 最近是否出现节点级事件

### 如果需要下线维护

```bash
kubectl cordon node-1
kubectl drain node-1 --ignore-daemonsets --delete-emptydir-data
kubectl uncordon node-1
```

## 九、PVC / 存储异常处理清单

### 推荐命令

```bash
kubectl get pvc -n prod
kubectl describe pvc data-mysql-0 -n prod
kubectl get pv
kubectl describe pv <pv-name>
kubectl get storageclass
kubectl describe storageclass <sc-name>
```

### 你想确认什么

- PVC 是否 `Bound`
- 绑定的是哪个 PV
- PV 回收策略是什么
- StorageClass 是否支持扩容
- 失败是发生在 provision、bind、attach 还是 mount 阶段

## 十、快速拿到“可操作信息”的命令

### 1. 某 namespace 里所有 Pod 状态

```bash
kubectl get pods -n prod -o wide
```

### 2. 批量查看名称、状态、节点

```bash
kubectl get pods -n prod -o custom-columns=NAME:.metadata.name,PHASE:.status.phase,NODE:.spec.nodeName,IP:.status.podIP
```

### 3. 查看所有 Pod 使用的镜像

```bash
kubectl get pods -A -o jsonpath='{range .items[*]}{.metadata.namespace}{"/"}{.metadata.name}{"\t"}{.spec.containers[*].image}{"\n"}{end}'
```

### 4. 查看某节点上的 Pod

```bash
kubectl get pods -A --field-selector spec.nodeName=node-1
```

### 5. 查看最近事件

```bash
kubectl get events -A --sort-by=.metadata.creationTimestamp
```

## 十一、值班时最该背下来的 20 条命令

```bash
kubectl config current-context
kubectl get ns
kubectl get nodes -o wide
kubectl top nodes
kubectl get pods -A
kubectl get pods -n <ns> -o wide
kubectl describe pod <pod> -n <ns>
kubectl logs <pod> -n <ns>
kubectl logs <pod> -n <ns> --previous
kubectl exec -it <pod> -n <ns> -- sh
kubectl get events -A --sort-by=.metadata.creationTimestamp
kubectl get svc -n <ns>
kubectl describe svc <svc> -n <ns>
kubectl get endpoints <svc> -n <ns>
kubectl rollout status deployment <name> -n <ns>
kubectl rollout history deployment <name> -n <ns>
kubectl rollout undo deployment <name> -n <ns>
kubectl cordon <node>
kubectl drain <node> --ignore-daemonsets --delete-emptydir-data
kubectl get pvc -n <ns>
```

## 十二、最后一条建议

真正值班时，最有价值的不是“背出更多命令”，而是形成固定节奏：

1. 先确认环境
2. 先看全局影响面
3. 先看事件和状态
4. 再看日志和容器内验证
5. 最后才做变更或重启

如果你把这条节奏养成肌肉记忆，很多线上问题都会稳很多。
