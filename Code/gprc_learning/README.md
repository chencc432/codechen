# gRPC 完整教程

> 从零到一掌握 gRPC：概念、语法、实战与生产最佳实践

## 教程目录

### 基础篇

| 章节 | 内容 | 难度 |
|------|------|------|
| [01-gRPC 概述与核心概念](01-GRPC概述与核心概念.md) | gRPC 架构、HTTP/2、Channel、Stub、Metadata、状态码 | ⭐ |
| [02-Protocol Buffers 详解](02-Protocol-Buffers详解.md) | proto3 语法、标量类型、枚举、oneof、map、WKT、编码原理 | ⭐⭐ |
| [03-gRPC 四种通信模式](03-GRPC四种通信模式.md) | 一元RPC、服务端流、客户端流、双向流 (Go/Java/Python) | ⭐⭐ |

### 实战篇

| 章节 | 内容 | 难度 |
|------|------|------|
| [04-gRPC Go 快速入门](04-GRPC-Go快速入门.md) | 完整电商订单服务、高级配置、健康检查、反射、测试 | ⭐⭐ |
| [05-gRPC Java 快速入门](05-GRPC-Java快速入门.md) | Maven/Gradle 项目搭建、三种 Stub、Spring Boot 集成 | ⭐⭐ |
| [06-gRPC Python 快速入门](06-GRPC-Python快速入门.md) | 同步/异步服务端、拦截器、TLS、pytest 测试 | ⭐⭐ |

### 进阶篇

| 章节 | 内容 | 难度 |
|------|------|------|
| [07-gRPC 拦截器与中间件](07-GRPC拦截器与中间件.md) | 日志、认证、验证、限流、追踪、指标拦截器 (Go/Java/Python) | ⭐⭐⭐ |
| [08-gRPC 错误处理与超时控制](08-GRPC错误处理与超时控制.md) | 状态码、Error Details、Deadline 传播、取消、重试、Hedging | ⭐⭐⭐ |
| [09-gRPC 安全与认证](09-GRPC安全与认证.md) | TLS/mTLS、JWT、OAuth2、RBAC 授权 | ⭐⭐⭐ |
| [10-gRPC 负载均衡与服务发现](10-GRPC负载均衡与服务发现.md) | 客户端LB、DNS/Consul/Etcd 服务发现、Envoy、Istio | ⭐⭐⭐⭐ |
| [11-gRPC 网关与 REST 互操作](11-GRPC网关与REST互操作.md) | gRPC-Gateway、OpenAPI/Swagger、gRPC-Web | ⭐⭐⭐ |

### 生产篇

| 章节 | 内容 | 难度 |
|------|------|------|
| [12-gRPC 生产实践与最佳实践](12-GRPC生产实践与最佳实践.md) | 项目结构、Docker/K8s 部署、可观测性、性能优化、API 演进 | ⭐⭐⭐⭐ |

---

## 学习路线

```
入门路线:
  01 → 02 → 03 → 04 (或 05/06) → 完成

进阶路线:
  入门路线 → 07 → 08 → 09 → 10 → 11 → 完成

生产路线:
  进阶路线 → 12 → 完成
```

## 前置知识

- 基本的编程经验 (Go / Java / Python 任一)
- 了解 HTTP 协议基础
- 了解基本的网络概念 (TCP、TLS)

## 推荐工具

| 工具 | 用途 | 安装 |
|------|------|------|
| `protoc` | Protobuf 编译器 | `brew install protobuf` |
| `grpcurl` | 命令行 gRPC 客户端 | `go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest` |
| `grpcui` | Web UI gRPC 调试 | `go install github.com/fullstorydev/grpcui/cmd/grpcui@latest` |
| `buf` | 现代 Protobuf 工具链 | `go install github.com/bufbuild/buf/cmd/buf@latest` |
| `evans` | 交互式 gRPC 客户端 | `go install github.com/ktr0731/evans@latest` |

## 参考资源

- [gRPC 官方文档](https://grpc.io/docs/)
- [Protocol Buffers 文档](https://protobuf.dev/)
- [gRPC Go 示例](https://github.com/grpc/grpc-go/tree/master/examples)
- [gRPC Java 示例](https://github.com/grpc/grpc-java/tree/master/examples)
- [gRPC Python 示例](https://github.com/grpc/grpc/tree/master/examples/python)
- [buf 官方文档](https://buf.build/docs)
