# gRPC 概述与核心概念

## 1. 什么是 gRPC

gRPC (gRPC Remote Procedure Call) 是 Google 于 2015 年开源的高性能、通用的 RPC 框架。它基于 HTTP/2 协议传输，使用 Protocol Buffers 作为接口定义语言 (IDL) 和序列化格式。

### 1.1 核心定义

```
gRPC = HTTP/2 + Protocol Buffers + IDL + 代码生成
```

- **HTTP/2**: 传输层，提供多路复用、流控、头部压缩
- **Protocol Buffers**: 序列化层，高效二进制编码
- **IDL**: 接口定义语言，跨语言契约
- **代码生成**: 自动生成客户端和服务端桩代码

### 1.2 为什么选择 gRPC

| 特性 | gRPC | REST/JSON | Thrift |
|------|------|-----------|--------|
| 序列化格式 | Protobuf (二进制) | JSON (文本) | Binary/JSON |
| 传输协议 | HTTP/2 | HTTP/1.1 | TCP/HTTP |
| 流式支持 | 原生支持 | 需要额外方案 | 有限支持 |
| 代码生成 | 官方支持 | 需第三方工具 | 官方支持 |
| 性能 | 极高 | 一般 | 高 |
| 浏览器支持 | gRPC-Web | 原生 | 需要代理 |
| 多语言支持 | 11+ 语言 | 任意 | 有限 |

### 1.3 典型应用场景

1. **微服务间通信**: 低延迟、高吞吐的服务间调用
2. **移动端与后端通信**: 二进制协议减少流量消耗
3. **实时流式处理**: 日志采集、股票行情、实时监控
4. **跨语言协作**: 多语言技术栈的统一通信协议
5. **替代 REST API 内部接口**: 内部服务不再需要人类可读的 JSON

---

## 2. gRPC 架构详解

### 2.1 整体架构

```
┌─────────────────────────────────────────────────┐
│                   客户端应用                      │
│  ┌─────────────┐    ┌──────────────────────┐    │
│  │ 业务代码     │───▶│ 生成的客户端桩代码     │    │
│  └─────────────┘    └──────────┬───────────┘    │
│                                │                 │
│                     ┌──────────▼───────────┐    │
│                     │  gRPC 核心库          │    │
│                     │  (序列化/反序列化)     │    │
│                     └──────────┬───────────┘    │
│                                │                 │
│                     ┌──────────▼───────────┐    │
│                     │  HTTP/2 传输层        │    │
│                     └──────────┬───────────┘    │
└────────────────────────────────┼─────────────────┘
                                 │
                    ┌────────────▼────────────┐
                    │      网络 (TCP/TLS)      │
                    └────────────┬────────────┘
                                 │
┌────────────────────────────────┼─────────────────┐
│                   服务端应用    ▼                  │
│                     ┌──────────▼───────────┐    │
│                     │  HTTP/2 传输层        │    │
│                     └──────────┬───────────┘    │
│                                │                 │
│                     ┌──────────▼───────────┐    │
│                     │  gRPC 核心库          │    │
│                     │  (反序列化/序列化)     │    │
│                     └──────────┬───────────┘    │
│                                │                 │
│  ┌─────────────┐    ┌──────────▼───────────┐    │
│  │ 业务实现     │◀───│ 生成的服务端骨架代码   │    │
│  └─────────────┘    └──────────────────────┘    │
└─────────────────────────────────────────────────┘
```

### 2.2 请求处理流程

```
1. 客户端调用桩方法 (Stub Method)
      │
      ▼
2. 拦截器预处理 (Client Interceptor - Before)
      │
      ▼
3. 序列化请求消息 (Serialize Request)
      │
      ▼
4. 通过 HTTP/2 发送帧 (Send HTTP/2 Frames)
      │
      ▼
5. 网络传输 (Network)
      │
      ▼
6. 服务端接收 HTTP/2 帧 (Receive HTTP/2 Frames)
      │
      ▼
7. 反序列化请求消息 (Deserialize Request)
      │
      ▼
8. 拦截器预处理 (Server Interceptor - Before)
      │
      ▼
9. 调用服务实现 (Service Implementation)
      │
      ▼
10. 拦截器后处理 (Server Interceptor - After)
      │
      ▼
11. 序列化响应消息 (Serialize Response)
      │
      ▼
12. 通过 HTTP/2 发送帧 (Send HTTP/2 Frames)
      │
      ▼
13. 客户端接收响应 (Receive Response)
      │
      ▼
14. 拦截器后处理 (Client Interceptor - After)
      │
      ▼
15. 返回结果给调用方 (Return to Caller)
```

---

## 3. 核心概念详解

### 3.1 服务定义 (Service Definition)

gRPC 通过 `.proto` 文件定义服务接口，这是客户端和服务端之间的契约：

```protobuf
syntax = "proto3";

package helloworld;

// 服务定义
service Greeter {
  // 一元 RPC：最简单的调用模式
  rpc SayHello (HelloRequest) returns (HelloReply) {}

  // 服务端流式 RPC
  rpc SayHelloStream (HelloRequest) returns (stream HelloReply) {}

  // 客户端流式 RPC
  rpc SendGreetings (stream HelloRequest) returns (HelloReply) {}

  // 双向流式 RPC
  rpc BidiHello (stream HelloRequest) returns (stream HelloReply) {}
}

// 消息定义
message HelloRequest {
  string name = 1;
}

message HelloReply {
  string message = 1;
}
```

### 3.2 四种通信模式

#### 一元 RPC (Unary RPC)
```
客户端                          服务端
  │                               │
  │──── 请求消息 ────────────────▶│
  │                               │── 处理请求
  │                               │
  │◀──── 响应消息 ────────────────│
  │                               │
```

#### 服务端流式 RPC (Server Streaming RPC)
```
客户端                          服务端
  │                               │
  │──── 请求消息 ────────────────▶│
  │                               │── 处理并发送
  │◀──── 响应消息 1 ─────────────│
  │◀──── 响应消息 2 ─────────────│
  │◀──── 响应消息 3 ─────────────│
  │◀──── 流结束标志 ─────────────│
  │                               │
```

#### 客户端流式 RPC (Client Streaming RPC)
```
客户端                          服务端
  │                               │
  │──── 请求消息 1 ──────────────▶│
  │──── 请求消息 2 ──────────────▶│── 收集并处理
  │──── 请求消息 3 ──────────────▶│
  │──── 流结束标志 ──────────────▶│
  │                               │
  │◀──── 响应消息 ────────────────│
  │                               │
```

#### 双向流式 RPC (Bidirectional Streaming RPC)
```
客户端                          服务端
  │                               │
  │──── 请求消息 1 ──────────────▶│
  │◀──── 响应消息 1 ─────────────│
  │──── 请求消息 2 ──────────────▶│
  │◀──── 响应消息 2 ─────────────│
  │──── 流结束标志 ──────────────▶│
  │◀──── 流结束标志 ─────────────│
  │                               │
```

### 3.3 Channel (通道)

Channel 是客户端与服务端之间的虚拟连接，代表一个到指定主机和端口的 HTTP/2 连接：

```
┌──────────────────────────────────────┐
│              gRPC Channel             │
│  ┌────────────────────────────────┐  │
│  │     HTTP/2 Connection Pool     │  │
│  │  ┌──────────┐  ┌──────────┐   │  │
│  │  │ Conn 1   │  │ Conn 2   │   │  │
│  │  │ Stream 1 │  │ Stream 1 │   │  │
│  │  │ Stream 2 │  │ Stream 2 │   │  │
│  │  │ Stream 3 │  │ Stream 3 │   │  │
│  │  └──────────┘  └──────────┘   │  │
│  └────────────────────────────────┘  │
│                                      │
│  特性:                               │
│  - 连接复用                           │
│  - 多路复用 (多个 Stream)             │
│  - 自动重连                           │
│  - 负载均衡                           │
└──────────────────────────────────────┘
```

**Channel 状态**:
- `IDLE`: 空闲，没有活跃连接
- `CONNECTING`: 正在建立连接
- `READY`: 连接就绪，可以发送请求
- `TRANSIENT_FAILURE`: 暂时性故障，正在重试
- `SHUTDOWN`: 通道已关闭

### 3.4 Stub (桩)

Stub 是客户端用于调用远程方法的代理对象：

```
┌─────────────────────────────┐
│       Stub 类型              │
├─────────────────────────────┤
│ Blocking Stub               │
│  - 同步阻塞调用              │
│  - 一元 RPC: 阻塞等待响应    │
│  - 流式 RPC: 返回 Iterator   │
│  - 适用于简单场景            │
├─────────────────────────────┤
│ Future Stub                 │
│  - 异步调用，返回 Future     │
│  - 仅支持一元 RPC            │
│  - 可设置回调                │
│  - 适用于需要并发的场景      │
├─────────────────────────────┤
│ Async Stub                  │
│  - 完全异步调用              │
│  - 基于回调/Observer 模式    │
│  - 支持所有 RPC 模式         │
│  - 适用于高性能场景          │
└─────────────────────────────┘
```

### 3.5 Metadata (元数据)

Metadata 是 gRPC 中的键值对，类似于 HTTP 的 Header：

```
┌──────────────────────────────────┐
│          Metadata 结构            │
│                                  │
│  Initial Metadata (请求头)        │
│  ┌────────────────────────────┐  │
│  │ :method: POST              │  │
│  │ :path: /helloworld/        │  │
│  │ content-type: application/ │  │
│  │   grpc                     │  │
│  │ authorization: Bearer xxx  │  │  ◀── 自定义 Header
│  │ x-request-id: abc-123      │  │  ◀── 追踪 ID
│  │ grpc-timeout: 5S           │  │  ◀── 超时控制
│  └────────────────────────────┘  │
│                                  │
│  Trailing Metadata (尾随头)      │
│  ┌────────────────────────────┐  │
│  │ grpc-status: 0             │  │  ◀── 状态码
│  │ grpc-message: OK           │  │  ◀── 状态消息
│  └────────────────────────────┘  │
└──────────────────────────────────┘
```

### 3.6 Status Code (状态码)

gRPC 定义了一套标准状态码，用于表示 RPC 调用的结果：

| 状态码 | 数值 | 含义 | 说明 |
|--------|------|------|------|
| `OK` | 0 | 成功 | 调用成功完成 |
| `CANCELLED` | 1 | 已取消 | 调用被取消 |
| `UNKNOWN` | 2 | 未知 | 未知错误 |
| `INVALID_ARGUMENT` | 3 | 无效参数 | 客户端指定了无效参数 |
| `DEADLINE_EXCEEDED` | 4 | 超时 | 操作在规定时间内未完成 |
| `NOT_FOUND` | 5 | 未找到 | 请求的资源不存在 |
| `ALREADY_EXISTS` | 6 | 已存在 | 尝试创建的资源已存在 |
| `PERMISSION_DENIED` | 7 | 权限不足 | 权限不足 |
| `RESOURCE_EXHAUSTED` | 8 | 资源耗尽 | 资源配额不足或达到限制 |
| `FAILED_PRECONDITION` | 9 | 前置条件失败 | 系统不在执行操作所需的状态 |
| `ABORTED` | 10 | 已中止 | 操作被中止 |
| `OUT_OF_RANGE` | 11 | 超出范围 | 尝试超出有效范围的操作 |
| `UNIMPLEMENTED` | 12 | 未实现 | 服务未实现该方法 |
| `INTERNAL` | 13 | 内部错误 | 内部错误 |
| `UNAVAILABLE` | 14 | 不可用 | 服务不可用 |
| `DATA_LOSS` | 15 | 数据丢失 | 不可恢复的数据丢失 |
| `UNAUTHENTICATED` | 16 | 未认证 | 认证失败 |

---

## 4. HTTP/2 与 gRPC 的关系

### 4.1 gRPC 如何利用 HTTP/2

```
┌─────────────────────────────────────────┐
│               HTTP/2 Frame              │
├────────┬────────┬───────────────────────┤
│ Length │ Type   │ Flags │ Stream ID │   │
│ 4字节  │ 1字节  │ 1字节  │ 4字节      │   │
├────────┴────────┴───────────────────────┤
│              Payload                    │
└─────────────────────────────────────────┘

gRPC 使用的 Frame 类型:
- HEADERS: 发送请求头/响应头 (含 Metadata)
- DATA:    发送序列化后的消息体
- SETTINGS: 连接参数协商
- WINDOW_UPDATE: 流控
- PING: 保活检测
- GOAWAY: 优雅关闭
```

### 4.2 gRPC 消息在 HTTP/2 上的映射

```
gRPC 请求 (Unary RPC):

客户端 → 服务端:
  HEADERS Frame (Stream ID=1)
    :method = POST
    :path = /package.Service/Method
    :scheme = http
    content-type = application/grpc
    grpc-encoding = gzip
    grpc-timeout = 5S

  DATA Frame (Stream ID=1)
    [Compressed-Flag][Message-Length][Message-Bytes]
    1字节              4字节           变长

服务端 → 客户端:
  HEADERS Frame (Stream ID=1)
    :status = 200
    content-type = application/grpc

  DATA Frame (Stream ID=1)
    [Compressed-Flag][Message-Length][Message-Bytes]

  HEADERS Frame (Stream ID=1)  ← Trailing Headers
    grpc-status = 0
    grpc-message = OK
```

### 4.3 多路复用

HTTP/2 允许在同一个 TCP 连接上同时发送多个请求/响应：

```
TCP 连接
├── Stream 1: /service.A/MethodX (请求/响应)
├── Stream 3: /service.B/MethodY (请求/响应)
├── Stream 5: /service.C/MethodZ (服务端流)
└── Stream 7: /service.D/MethodW (客户端流)

每个 Stream 独立:
- 独立的流控窗口
- 独立的优先级
- 独立的错误处理
- 互不阻塞
```

---

## 5. gRPC 生态

### 5.1 官方支持的语言

| 语言 | 传输层 | Protobuf 运行时 | 代码生成 | 成熟度 |
|------|--------|----------------|----------|--------|
| C/C++ | grpc-core | protobuf | protoc-gen-grpc | 稳定 |
| C# | grpc-core | protobuf | protoc-gen-grpc | 稳定 |
| Dart | grpc-dart | protobuf | protoc-gen-dart | 稳定 |
| Go | grpc-go | protobuf | protoc-gen-go-grpc | 稳定 |
| Java | grpc-java | protobuf | protoc-gen-java | 稳定 |
| Kotlin | grpc-kotlin | protobuf | protoc-gen-kotlin | 稳定 |
| Node.js | grpc-node | protobuf | protoc-gen-node | 稳定 |
| Objective-C | grpc-core | protobuf | protoc-gen-objc | 稳定 |
| PHP | grpc-php | protobuf | protoc-gen-php | 稳定 |
| Python | grpc-python | protobuf | protoc-gen-python | 稳定 |
| Ruby | grpc-ruby | protobuf | protoc-gen-ruby | 稳定 |
| Rust | grpc-rs / tonic | protobuf | 自定义 | 稳定 |
| Swift | grpc-swift | protobuf | protoc-gen-swift | 稳定 |

### 5.2 扩展生态

```
gRPC 生态
├── gRPC-Gateway       # REST/JSON ↔ gRPC 转换
├── gRPC-Web           # 浏览器端 gRPC
├── grpc-cli           # 命令行工具
├── grpcurl            # 类 curl 的 gRPC 工具
├── grpcui             # Web UI 调试工具
├── protoc-gen-doc     # 文档生成
├── protoc-gen-validate # 校验规则生成
├── grpc-health        # 健康检查协议
├── grpc-reflection    # 服务反射协议
├── grpc-lb            # 负载均衡
└── opencensus-grpc    # 链路追踪 & 指标
```

---

## 6. gRPC vs REST 深度对比

### 6.1 性能对比

```
序列化大小对比 (同一数据结构):
┌──────────┬──────────┬───────────┐
│ 格式      │ 大小     │ 相对比例   │
├──────────┼──────────┼───────────┤
│ Protobuf │ 42 bytes │ 1x (基准) │
│ JSON     │ 98 bytes │ 2.3x      │
│ XML      │ 165 bytes│ 3.9x      │
└──────────┴──────────┴───────────┘

吞吐量对比 (同一硬件):
┌──────────┬──────────────┬──────────────┐
│ 框架      │ QPS          │ 平均延迟      │
├──────────┼──────────────┼──────────────┤
│ gRPC     │ ~100,000     │ ~1.2ms       │
│ REST/JSON│ ~35,000      │ ~3.5ms       │
│ REST/XML │ ~15,000      │ ~8.0ms       │
└──────────┴──────────────┴──────────────┘
```

### 6.2 功能对比

| 特性 | gRPC | REST |
|------|------|------|
| 契约优先 | .proto 文件 | OpenAPI/Swagger |
| 代码生成 | 官方自动生成 | 需第三方工具 |
| 流式通信 | 四种模式 | 需 WebSocket/SSE |
| 双向通信 | 原生支持 | 需要 WebSocket |
| 浏览器支持 | gRPC-Web (有限) | 原生支持 |
| 调试工具 | grpcurl, grpcui | curl, Postman |
| 人类可读 | 否 (二进制) | 是 (文本) |
| 防火墙友好 | 一般 | 是 |
| 学习曲线 | 较陡 | 平缓 |

### 6.3 选择建议

```
选择 gRPC 的场景:
✅ 微服务间内部通信
✅ 对性能有极致要求
✅ 需要流式通信
✅ 多语言技术栈
✅ 移动端与后端通信

选择 REST 的场景:
✅ 面向外部开发者的 API
✅ 需要浏览器直接调用
✅ 简单的 CRUD 操作
✅ 团队对 gRPC 不熟悉
✅ 需要人类可读的请求/响应
```

---

## 7. 开发环境搭建

### 7.1 安装 Protocol Buffers 编译器

```bash
# macOS
brew install protobuf

# Ubuntu/Debian
sudo apt install -y protobuf-compiler

# Windows (使用 Chocolatey)
choco install protobuf

# 或手动下载
# https://github.com/protocolbuffers/protobuf/releases
```

验证安装:
```bash
protoc --version
# 期望输出: libprotoc 25.x 或更高版本
```

### 7.2 安装 Go gRPC 插件

```bash
# 安装 Go 的 protoc 插件
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# 确保 $GOPATH/bin 在 PATH 中
export PATH="$PATH:$(go env GOPATH)/bin"
```

### 7.3 安装 Java gRPC 插件

```xml
<!-- Maven pom.xml -->
<extensions>
    <extension>
        <groupId>kr.motd.maven</groupId>
        <artifactId>os-maven-plugin</artifactId>
        <version>1.7.1</version>
    </extension>
</extensions>

<plugins>
    <plugin>
        <groupId>org.xolstice.maven.plugins</groupId>
        <artifactId>protobuf-maven-plugin</artifactId>
        <version>0.6.1</version>
        <configuration>
            <protocArtifact>
                com.google.protobuf:protoc:3.25.1:exe:${os.detected.classifier}
            </protocArtifact>
            <pluginId>grpc-java</pluginId>
            <pluginArtifact>
                io.grpc:protoc-gen-grpc-java:1.60.0:exe:${os.detected.classifier}
            </pluginArtifact>
        </configuration>
        <executions>
            <execution>
                <goals>
                    <goal>compile</goal>
                    <goal>compile-custom</goal>
                </goals>
            </execution>
        </executions>
    </plugin>
</plugins>
```

### 7.4 安装 Python gRPC 工具

```bash
pip install grpcio grpcio-tools

# 生成代码
python -m grpc_tools.protoc \
    -I. \
    --python_out=. \
    --grpc_python_out=. \
    your_service.proto
```

### 7.5 推荐开发工具

| 工具 | 用途 | 安装方式 |
|------|------|----------|
| `grpcurl` | 命令行调用 gRPC | `go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest` |
| `grpcui` | Web UI 调试 gRPC | `go install github.com/fullstorydev/grpcui/cmd/grpcui@latest` |
| `protoc-gen-doc` | 生成 API 文档 | `go install github.com/pseudomuto/protoc-gen-doc/cmd/protoc-gen-doc@latest` |
| `buf` | 现代 Protobuf 工具链 | `go install github.com/bufbuild/buf/cmd/buf@latest` |
| `evans` | 交互式 gRPC 客户端 | `go install github.com/ktr0731/evans@latest` |

---

## 8. 第一个 gRPC 程序 (概念预览)

### 8.1 定义服务 (.proto)

```protobuf
syntax = "proto3";

package helloworld;

option go_package = "helloworld/helloworld";

service Greeter {
  rpc SayHello (HelloRequest) returns (HelloReply) {}
}

message HelloRequest {
  string name = 1;
}

message HelloReply {
  string message = 1;
}
```

### 8.2 生成代码

```bash
# Go
protoc --go_out=. --go-grpc_out=. helloworld.proto

# Java (通过 Maven 自动生成)

# Python
python -m grpc_tools.protoc -I. --python_out=. --grpc_python_out=. helloworld.proto
```

### 8.3 实现服务端 (Go 示例)

```go
type server struct {
    pb.UnimplementedGreeterServer
}

func (s *server) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
    return &pb.HelloReply{Message: "Hello " + req.GetName()}, nil
}

func main() {
    lis, _ := net.Listen("tcp", ":50051")
    s := grpc.NewServer()
    pb.RegisterGreeterServer(s, &server{})
    s.Serve(lis)
}
```

### 8.4 实现客户端 (Go 示例)

```go
func main() {
    conn, _ := grpc.Dial("localhost:50051", grpc.WithInsecure())
    defer conn.Close()

    client := pb.NewGreeterClient(conn)
    resp, _ := client.SayHello(context.Background(), &pb.HelloRequest{Name: "World"})
    fmt.Println(resp.GetMessage())
}
```

---

## 9. 总结

gRPC 的核心优势:
1. **高性能**: 基于 HTTP/2 + Protobuf，比 REST/JSON 快 3-7 倍
2. **强类型契约**: .proto 文件确保客户端和服务端一致
3. **流式通信**: 四种通信模式满足各种场景
4. **多语言支持**: 11+ 语言官方支持，代码自动生成
5. **生态丰富**: 健康检查、反射、网关、追踪等开箱即用

下一步: 学习 [Protocol Buffers 详解](02-Protocol-Buffers详解.md)
