# 🚀 Kubernetes 完全学习指南

> 从零基础到熟练掌握 Kubernetes，达到工作实战水平

## 📚 课程目录

### 第一部分：基础理论篇
1. [Kubernetes 概述与架构](./01-basics/01-overview.md)
2. [核心组件详解](./01-basics/02-components.md)
3. [核心概念与术语](./01-basics/03-concepts.md)

### 第二部分：核心资源详解
1. [Pod - 最小调度单元](./02-resources/01-pod.md)
2. [Deployment - 无状态应用部署](./02-resources/02-deployment.md)
3. [Service - 服务发现与负载均衡](./02-resources/03-service.md)
4. [ConfigMap 与 Secret](./02-resources/04-configmap-secret.md)
5. [Volume 与持久化存储](./02-resources/05-volume.md)
6. [Namespace - 资源隔离](./02-resources/06-namespace.md)
7. [StatefulSet - 有状态应用](./02-resources/07-statefulset.md)
8. [DaemonSet 与 Job](./02-resources/08-daemonset-job.md)

### 第三部分：实战操作篇
1. [kubectl 命令完全手册](./03-practice/01-kubectl-commands.md)
2. [YAML 编写规范与技巧](./03-practice/02-yaml-guide.md)
3. [常见运维操作指南](./03-practice/03-operations.md)
4. [故障排查与调试](./03-practice/04-troubleshooting.md)

### 第四部分：进阶主题
1. [Kubernetes 网络模型](./04-advanced/01-networking.md)
2. [存储系统详解](./04-advanced/02-storage.md)
3. [调度机制与策略](./04-advanced/03-scheduling.md)
4. [安全与权限控制](./04-advanced/04-security.md)
5. [Ingress 与流量管理](./04-advanced/05-ingress.md)

### 第五部分：client-go 编程
1. [client-go 入门](./05-client-go/01-introduction.md)
2. [客户端配置与连接](./05-client-go/02-client-setup.md)
3. [资源的 CRUD 操作](./05-client-go/03-crud-operations.md)
4. [Informer 机制详解](./05-client-go/04-informer.md)
5. [实战项目：自定义控制器](./05-client-go/05-controller-demo.md)

### 第六部分：实战项目
1. [项目一：部署微服务应用](./06-projects/01-microservice-deploy/)
2. [项目二：日志收集系统](./06-projects/02-logging-system/)
3. [项目三：监控告警系统](./06-projects/03-monitoring/)

## 🎯 学习路径建议

```
Week 1: 基础理论 + Pod/Deployment
Week 2: Service/ConfigMap + kubectl实战
Week 3: 进阶主题（网络/存储/调度）
Week 4: client-go + 实战项目
```

## 💡 学习技巧

1. **理论结合实践**：每个章节都有实战练习，务必动手操作
2. **善用官方文档**：https://kubernetes.io/docs/
3. **多练习 kubectl**：命令行是日常工作的主要工具
4. **理解原理**：不仅要会用，还要理解为什么这样设计

## 🛠️ 环境准备

- 推荐使用 [Minikube](https://minikube.sigs.k8s.io/) 或 [Kind](https://kind.sigs.k8s.io/) 搭建本地环境
- 也可以使用云服务商提供的托管 K8s 服务
- 详见 [环境搭建指南](./00-setup/environment.md)

---

**开始你的 Kubernetes 学习之旅吧！** 🎉



