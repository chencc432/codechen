# client-go 编程模块

> 目标：不只会调 `Get/List/Create`，而是建立与官方控制器同一套的心智模型——**配置与客户端选型、写路径（Update/Patch/Apply）、读路径（List/Watch/Informer）、WorkQueue 控制循环，以及 Discovery/Dynamic、Event、选主等周边机制**。

## 适合谁

- 要写集群自动化、运维工具、平台控制器
- 会用 kubectl，但不知道程序里该 Watch 还是 Informer
- 准备学 Operator / Kubebuilder，想先吃透底层

## 学习路线

按顺序读，不要只抄 CRUD：

| 序号 | 文档 | 你要带走的东西 |
|------|------|----------------|
| 01 | [入门：机制全景](./01-introduction.md) | 组件地图、何时用哪类客户端 |
| 02 | [客户端配置与连接](./02-client-setup.md) | kubeconfig / In-Cluster、QPS、多集群 |
| 03 | [CRUD 与写路径机制](./03-crud-operations.md) | Update/Patch/Apply、冲突重试、分页 |
| 04 | [Informer 机制详解](./04-informer.md) | Reflector、缓存、Lister、WorkQueue |
| 05 | [实战：自定义控制器](./05-controller-demo.md) | 完整控制循环落地 |
| 06 | [Discovery 与 Dynamic](./06-discovery-and-dynamic.md) | GVR、RESTMapper、CRD/泛型 |
| 07 | [常用机制与工具包](./07-common-mechanisms.md) | Scheme、Event、选主、Fake、retry |
| 08 | [排障与生产清单](./08-debugging-and-pitfalls.md) | 故障剧本、上线检查表 |

## 一张图串起来

```
临时脚本                长期控制器
   │                        │
   ▼                        ▼
Clientset/Dynamic      Informer + Lister
Get/List/Patch         WorkQueue + Reconcile
   │                        │
   └──────────┬─────────────┘
              ▼
         rest.Config
              ▼
         API Server
```

## 学完后去哪

- 先用 [排障与生产清单](./08-debugging-and-pitfalls.md) 自检一遍
- [自定义资源专题](../07-custom-resources/README.md) — CRD 与 Operator
- [控制器深度专题](../11-controller-deep-dive/README.md) — Informer 内部、限速、Finalizer、controller-runtime

## 版本提示

`k8s.io/client-go`、`k8s.io/api`、`k8s.io/apimachinery` 三者 minor 版本应对齐，并与目标集群大版本匹配（例如 K8s 1.29 → `v0.29.x`）。
