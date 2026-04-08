# 🧯 日志、调试、容器排障与应急定位

## 为什么排障不能只会 `kubectl logs`

很多人遇到故障的第一反应是：

```bash
kubectl logs <pod>
```

这当然是对的，但如果你只会这一招，很多问题会查不出来。

因为 Kubernetes 里的故障来源可能有很多层：

- 调度层
- 节点层
- 容器运行时层
- 镜像层
- 配置层
- 网络层
- 存储层
- 应用本身

所以真正有效的排障，通常要把下面几个动作配合起来：

- `get`
- `describe`
- `events`
- `logs`
- `exec`
- `port-forward`
- `cp`
- `debug`

## 一、排障总顺序：先状态，后日志；先外围，后容器内

建议你把下面这条顺序背下来：

1. `get` 看状态
2. `describe` 看事件和容器状态
3. `logs` 看应用输出
4. `exec` 进容器做验证
5. 需要时再 `debug`、`cp`、`port-forward`

为什么这样？

- 如果 Pod 压根没调度成功，日志可能根本没有
- 如果是卷挂载失败、镜像拉取失败，问题往往先体现在事件里
- 如果是网络或配置问题，可能得进容器验证

## 二、日志相关命令

### 1. 查看单个 Pod 日志

```bash
kubectl logs myapp-7d6c7f89d9-abcde -n prod
```

这是最基础的用法，适合快速看应用有没有明显报错。

### 2. 指定容器

```bash
kubectl logs myapp-7d6c7f89d9-abcde -c myapp -n prod
```

如果一个 Pod 里有多个容器，比如：

- 主业务容器
- sidecar
- initContainer

你一定要明确自己在看哪个容器的日志。

### 3. 持续跟日志

```bash
kubectl logs myapp-7d6c7f89d9-abcde -n prod -f
```

适用场景：

- 发版后实时观察启动过程
- 问题复现时盯现场

### 4. 只看最近若干行

```bash
kubectl logs myapp-7d6c7f89d9-abcde -n prod --tail=100
```

这很适合日志很多的时候，避免一上来刷屏。

### 5. 看最近一段时间的日志

```bash
kubectl logs myapp-7d6c7f89d9-abcde -n prod --since=10m
kubectl logs myapp-7d6c7f89d9-abcde -n prod --since=1h
```

当你明确知道故障发生时间段时，这非常有用。

### 6. 看上一个容器实例日志

```bash
kubectl logs myapp-7d6c7f89d9-abcde -n prod --previous
kubectl logs myapp-7d6c7f89d9-abcde -c myapp -n prod --previous
```

这个命令是查 `CrashLoopBackOff` 的关键之一。

因为当前容器可能刚重启，真正导致退出的错误在上一个实例里。

### 7. 按标签批量看日志

```bash
kubectl logs -l app=myapp -n prod --all-containers
kubectl logs -l app=myapp -n prod --tail=50
```

适用场景：

- 一个业务有多个副本
- 你想快速比较多个实例是否都在报同样错误

## 三、`describe`：很多问题是它先告诉你的

### 1. 查看 Pod 详情

```bash
kubectl describe pod myapp-7d6c7f89d9-abcde -n prod
```

重点关注这些地方：

- 容器状态
- 上次退出码
- 重启次数
- 挂载信息
- 探针失败
- 事件

### 2. 看 Deployment / StatefulSet

```bash
kubectl describe deploy myapp -n prod
kubectl describe sts mysql -n prod
```

适用场景：

- Pod 异常可能只是结果，根因可能在控制器层
- 看滚动更新策略、选择器、模板等是否合理

### 3. 看 Service / Ingress

```bash
kubectl describe svc myapp -n prod
kubectl describe ingress myapp -n prod
```

如果问题表现为“服务不通”，这里只要你认真看，经常能迅速发现：

- selector 配错
- endpoint 没选上
- Ingress 后端指向不对

## 四、进入容器验证：`exec`

### 1. 进入交互式 shell

```bash
kubectl exec -it myapp-7d6c7f89d9-abcde -n prod -- sh
kubectl exec -it myapp-7d6c7f89d9-abcde -n prod -- /bin/bash
```

什么时候要进容器？

- 你怀疑环境变量不对
- 你想检查配置文件
- 你想测试 DNS 解析
- 你想 curl 下游服务
- 你想确认挂载目录里到底有没有文件

### 2. 执行单条命令

```bash
kubectl exec myapp-7d6c7f89d9-abcde -n prod -- env
kubectl exec myapp-7d6c7f89d9-abcde -n prod -- ls /app
kubectl exec myapp-7d6c7f89d9-abcde -n prod -- cat /etc/resolv.conf
```

这种方式比直接进入 shell 更适合：

- 自动化脚本
- 快速确认单个结论

### 3. 指定容器

```bash
kubectl exec -it myapp-7d6c7f89d9-abcde -c myapp -n prod -- sh
```

如果 Pod 里有 sidecar，一定要注意容器名，不然你可能进错容器。

## 五、端口转发：当集群内部服务不方便直接暴露时

### 1. 转发 Pod 端口

```bash
kubectl port-forward pod/myapp-7d6c7f89d9-abcde 8080:8080 -n prod
```

### 2. 转发 Service 端口

```bash
kubectl port-forward svc/myapp 8080:80 -n prod
```

### 3. 转发 Deployment

```bash
kubectl port-forward deploy/myapp 8080:8080 -n prod
```

适用场景：

- 应用只暴露在集群内
- 你想本地直接访问接口或管理页面
- 你想验证服务本身是否正常，而不是先折腾 Ingress 或公网链路

思路上可以这样理解：

- `port-forward` 是一种低成本、低侵入的临时通路
- 它非常适合排除“到底是应用坏了，还是外部访问链路坏了”

## 六、文件拷贝：`cp`

### 1. 从容器拷文件到本地

```bash
kubectl cp myapp-7d6c7f89d9-abcde:/app/logs/error.log ./error.log -n prod
kubectl cp myapp-7d6c7f89d9-abcde:/tmp/heap.bin ./heap.bin -n prod
```

适用场景：

- 导出日志
- 导出 dump
- 导出配置文件做比对

### 2. 从本地拷文件到容器

```bash
kubectl cp ./debug.conf myapp-7d6c7f89d9-abcde:/tmp/debug.conf -n prod
```

注意：

- 线上环境尽量少做这类动作
- 因为它不可持续，Pod 一重建就没了

它更适合临时调试，而不是正式配置管理。

## 七、临时调试容器：`kubectl debug`

### 1. 给 Pod 加一个临时调试容器

```bash
kubectl debug pod/myapp-7d6c7f89d9-abcde -it -n prod --image=busybox
kubectl debug pod/myapp-7d6c7f89d9-abcde -it -n prod --image=nicolaka/netshoot
```

适用场景：

- 业务镜像太精简，没有 `curl`、`ping`、`nslookup`
- 你需要临时带上网络排障工具

### 2. 调试节点

```bash
kubectl debug node/node-1 -it --image=busybox
```

适用场景：

- 你怀疑是节点网络、磁盘、系统层问题

但这类动作要谨慎，尤其在生产环境里。

## 八、几类最常见故障怎么查

### 场景 1：Pod 处于 `CrashLoopBackOff`

建议命令顺序：

```bash
kubectl get pod myapp-xxx -n prod
kubectl describe pod myapp-xxx -n prod
kubectl logs myapp-xxx -n prod --previous
kubectl logs myapp-xxx -n prod
kubectl get events -n prod --sort-by=.metadata.creationTimestamp
```

常见原因：

- 启动命令错了
- 配置缺失
- 依赖服务不可用
- 资源不够被 OOMKilled
- 探针设计不合理

### 场景 2：Pod 处于 `ImagePullBackOff` / `ErrImagePull`

建议命令顺序：

```bash
kubectl describe pod myapp-xxx -n prod
kubectl get pod myapp-xxx -n prod -o yaml
kubectl get secret -n prod
```

重点看：

- 镜像名是否拼错
- tag 是否存在
- 镜像仓库 secret 是否配置正确
- 节点到镜像仓库网络是否通

### 场景 3：Pod 一直 `Pending`

建议命令顺序：

```bash
kubectl get pod myapp-xxx -n prod
kubectl describe pod myapp-xxx -n prod
kubectl get nodes
kubectl get pvc -n prod
```

常见原因：

- 资源不够，调度不上
- 节点污点不匹配
- 节点选择器不匹配
- PVC 没绑定成功

### 场景 4：用户说服务不通

建议命令顺序：

```bash
kubectl get svc myapp -n prod
kubectl describe svc myapp -n prod
kubectl get endpoints myapp -n prod
kubectl get pods -l app=myapp -n prod -o wide
kubectl port-forward svc/myapp 8080:80 -n prod
```

你要确认：

- Service selector 是否正确
- Endpoints 是否为空
- Pod 是否真的 Ready
- 应用本身在容器内是否正常监听

### 场景 5：怀疑 DNS 异常

```bash
kubectl exec -it myapp-xxx -n prod -- cat /etc/resolv.conf
kubectl exec -it myapp-xxx -n prod -- nslookup kubernetes.default.svc.cluster.local
kubectl get pods -n kube-system
```

如果业务 Pod 里没有 `nslookup`，可以用 `kubectl debug` 带一个调试镜像进去。

## 九、排障时的思维误区

### 1. 只看应用日志，不看事件

很多问题压根还没到应用层，比如：

- 镜像拉取失败
- 卷挂载失败
- 调度失败

这时日志可能根本没意义。

### 2. 一看到异常就重启

重启有时能临时恢复，但也会抹掉现场。

更好的做法通常是：

1. 先 `describe`
2. 先看 `logs --previous`
3. 必要时再重启或删 Pod

### 3. 进容器后没有明确验证目标

不要为了“进去看看”而 `exec`。

你每次进容器，最好都带着明确问题：

- 环境变量对不对
- 配置文件在不在
- 服务能不能连通
- DNS 解析是否正常

## 十、这一篇最值得记住的命令

```bash
kubectl describe pod <pod> -n <ns>
kubectl logs <pod> -n <ns> --previous
kubectl logs <pod> -n <ns> -f
kubectl exec -it <pod> -n <ns> -- sh
kubectl port-forward svc/<svc> 8080:80 -n <ns>
kubectl cp <ns>/<pod>:/path/file ./file
kubectl debug pod/<pod> -it -n <ns> --image=nicolaka/netshoot
kubectl get events -n <ns> --sort-by=.metadata.creationTimestamp
```

## 下一篇

下一篇建议看：

- [节点维护、调度控制与存储相关命令](./04-node-scheduling-and-storage.md)

因为很多线上故障最后都会牵扯到节点、调度、污点、卷、PVC 这些“更底层的运维动作”。
