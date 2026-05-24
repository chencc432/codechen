# gRPC 生产实践与最佳实践

## 1. 项目结构最佳实践

### 1.1 Go 项目结构

```
project/
├── api/
│   └── proto/
│       └── v1/
│           ├── order.proto
│           └── user.proto
├── gen/
│   └── v1/
│       ├── order.pb.go
│       ├── order_grpc.pb.go
│       ├── user.pb.go
│       └── user_grpc.pb.go
├── internal/
│   ├── service/
│   │   ├── order_service.go
│   │   └── user_service.go
│   ├── repository/
│   │   ├── order_repo.go
│   │   └── user_repo.go
│   ├── interceptor/
│   │   ├── auth.go
│   │   ├── logging.go
│   │   └── recovery.go
│   └── config/
│       └── config.go
├── cmd/
│   ├── server/
│   │   └── main.go
│   └── client/
│       └── main.go
├── deploy/
│   ├── Dockerfile
│   ├── k8s/
│   └── docker-compose.yml
├── docs/
│   └── swagger/
├── buf.yaml
├── buf.gen.yaml
├── Makefile
└── go.mod
```

### 1.2 Protobuf 文件组织

```
api/proto/
├── common/
│   ├── v1/
│   │   ├── types.proto        # 公共类型
│   │   └── errors.proto       # 错误定义
│   └── v2/
│       └── types.proto
├── order/
│   ├── v1/
│   │   ├── order.proto        # 消息定义
│   │   └── order_service.proto # 服务定义
│   └── v2/
│       └── order_service.proto
└── user/
    └── v1/
        └── user_service.proto
```

**版本化最佳实践**:
```protobuf
// v1/order_service.proto
syntax = "proto3";
package order.v1;

// v2/order_service.proto
syntax = "proto3";
package order.v2;

// 不同版本可以独立演进
// 客户端选择使用哪个版本
```

---

## 2. Docker 与 Kubernetes 部署

### 2.1 Dockerfile

```dockerfile
# 多阶段构建
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/server

# 运行阶段
FROM alpine:3.19

RUN apk --no-cache add ca-certificates
COPY --from=builder /server /server
COPY certs/ /certs/

EXPOSE 50051 8080

HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
    CMD grpcurl -plaintext localhost:50051 grpc.health.v1.Health/Check || exit 1

ENTRYPOINT ["/server"]
```

### 2.2 Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: order-service
  labels:
    app: order-service
    version: v1
spec:
  replicas: 3
  selector:
    matchLabels:
      app: order-service
  template:
    metadata:
      labels:
        app: order-service
        version: v1
    spec:
      containers:
        - name: order-service
          image: registry.example.com/order-service:latest
          ports:
            - name: grpc
              containerPort: 50051
            - name: gateway
              containerPort: 8080
          env:
            - name: GRPC_PORT
              value: "50051"
            - name: GATEWAY_PORT
              value: "8080"
            - name: DB_HOST
              valueFrom:
                secretKeyRef:
                  name: db-credentials
                  key: host
          resources:
            requests:
              cpu: 100m
              memory: 128Mi
            limits:
              cpu: 500m
              memory: 512Mi
          readinessProbe:
            exec:
              command: ["grpcurl", "-plaintext", "localhost:50051", "grpc.health.v1.Health/Check"]
            initialDelaySeconds: 5
            periodSeconds: 10
          livenessProbe:
            exec:
              command: ["grpcurl", "-plaintext", "localhost:50051", "grpc.health.v1.Health/Check"]
            initialDelaySeconds: 10
            periodSeconds: 30
          volumeMounts:
            - name: tls-certs
              mountPath: /certs
              readOnly: true
      volumes:
        - name: tls-certs
          secret:
            secretName: order-service-tls
---
apiVersion: v1
kind: Service
metadata:
  name: order-service
spec:
  selector:
    app: order-service
  ports:
    - name: grpc
      port: 50051
      targetPort: grpc
    - name: gateway
      port: 8080
      targetPort: gateway
```

### 2.3 gRPC 健康检查 (Kubernetes)

```go
// 注册健康检查服务
import (
    "google.golang.org/grpc/health"
    "google.golang.org/grpc/health/grpc_health_v1"
)

healthServer := health.NewServer()
grpc_health_v1.RegisterHealthServer(s, healthServer)

// 设置整体健康状态
healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

// 设置特定服务健康状态
healthServer.SetServingStatus("order.v1.OrderService",
    grpc_health_v1.HealthCheckResponse_SERVING)

// 当依赖不可用时
healthServer.SetServingStatus("order.v1.OrderService",
    grpc_health_v1.HealthCheckResponse_NOT_SERVING)
```

---

## 3. 可观测性

### 3.1 日志

```go
// 结构化日志
func UnaryServerLoggingInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
    return func(
        ctx context.Context,
        req interface{},
        info *grpc.UnaryServerInfo,
        handler grpc.UnaryHandler,
    ) (interface{}, error) {
        start := time.Now()
        requestID := ctx.Value("requestID").(string)

        resp, err := handler(ctx, req)

        logger.Info("rpc completed",
            "method", info.FullMethod,
            "request_id", requestID,
            "duration", time.Since(start),
            "status", status.Code(err).String(),
        )

        return resp, err
    }
}
```

### 3.2 链路追踪 (OpenTelemetry)

```go
import (
    "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace"
    "go.opentelemetry.io/otel/sdk/trace"
)

func initTracer() {
    exporter, _ := otlptrace.New(context.Background(),
        otlptrace.WithEndpoint("otel-collector:4317"),
        otlptrace.WithInsecure(),
    )

    tp := trace.NewTracerProvider(
        trace.WithBatcher(exporter),
    )
    otel.SetTracerProvider(tp)
}

// 服务端
s := grpc.NewServer(
    grpc.StatsHandler(otelgrpc.NewServerHandler()),
)

// 客户端
conn, _ := grpc.Dial(target,
    grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
)
```

### 3.3 指标 (Prometheus)

```go
import (
    "github.com/grpc-ecosystem/go-grpc-prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

// 服务端
s := grpc.NewServer(
    grpc.ChainUnaryInterceptor(
        grpc_prometheus.UnaryServerInterceptor,
    ),
    grpc.ChainStreamInterceptor(
        grpc_prometheus.StreamServerInterceptor,
    ),
)

// 注册指标
grpc_prometheus.Register(s)
grpc_prometheus.EnableHandlingTimeHistogram(
    grpc_prometheus.WithHistogramBuckets([]float64{
        0.001, 0.01, 0.05, 0.1, 0.5, 1, 5,
    }),
)

// 暴露 Prometheus 端点
http.Handle("/metrics", promhttp.Handler())
go http.ListenAndServe(":9090", nil)
```

Prometheus 指标:
```
# 请求总数
grpc_server_handled_total{method="GetOrder",status="OK"}

# 请求延迟直方图
grpc_server_handling_seconds_bucket{method="GetOrder",le="0.1"}

# 正在处理的请求数
grpc_server_started_total{method="GetOrder"}

# 消息大小
grpc_server_msg_received_total{method="GetOrder"}
```

### 3.4 Grafana Dashboard 关键指标

```
gRPC 服务监控面板:

1. 请求速率 (QPS)
   - sum(rate(grpc_server_handled_total[5m])) by (method)

2. 错误率
   - sum(rate(grpc_server_handled_total{status!="OK"}[5m]))
     / sum(rate(grpc_server_handled_total[5m]))

3. P50/P95/P99 延迟
   - histogram_quantile(0.95,
     rate(grpc_server_handling_seconds_bucket[5m]))

4. 活跃连接数
   - grpc_server_started_total - grpc_server_handled_total

5. 消息大小
   - histogram_quantile(0.95,
     rate(grpc_server_msg_received_bytes_bucket[5m]))
```

---

## 4. 性能优化

### 4.1 连接复用

```go
// ❌ 错误: 每次 RPC 创建新连接
func bad() {
    for i := 0; i < 100; i++ {
        conn, _ := grpc.Dial("localhost:50051", ...)
        client := pb.NewServiceClient(conn)
        client.DoSomething(ctx, req)
        conn.Close()
    }
}

// ✅ 正确: 复用连接
func good() {
    conn, _ := grpc.Dial("localhost:50051", ...)
    defer conn.Close()
    client := pb.NewServiceClient(conn)

    for i := 0; i < 100; i++ {
        client.DoSomething(ctx, req)
    }
}
```

### 4.2 连接池 (Go 客户端)

```go
// Go 的 grpc.ClientConn 本身就是连接池
// HTTP/2 多路复用，一个连接可以承载多个并发请求
// 通常一个 ClientConn 就够了

// 如果需要多个连接 (极端场景)
type ConnPool struct {
    conns []*grpc.ClientConn
    index uint64
}

func NewConnPool(target string, size int) (*ConnPool, error) {
    pool := &ConnPool{conns: make([]*grpc.ClientConn, size)}
    for i := 0; i < size; i++ {
        conn, err := grpc.Dial(target, ...)
        if err != nil {
            return nil, err
        }
        pool.conns[i] = conn
    }
    return pool, nil
}

func (p *ConnPool) Get() *grpc.ClientConn {
    idx := atomic.AddUint64(&p.index, 1)
    return p.conns[idx%uint64(len(p.conns))]
}
```

### 4.3 消息大小优化

```protobuf
// ❌ 避免: 传输大消息
message Bad {
    bytes file_content = 1;  // 可能几百 MB
}

// ✅ 推荐: 使用流式传输
service FileService {
    rpc Download(DownloadRequest) returns (stream FileChunk) {}
    rpc Upload(stream FileChunk) returns (UploadResponse) {}
}

message FileChunk {
    bytes data = 1;  // 每个 chunk 64KB - 4MB
}
```

```go
// 服务端消息大小配置
s := grpc.NewServer(
    grpc.MaxRecvMsgSize(4*1024*1024),   // 4MB (默认)
    grpc.MaxSendMsgSize(4*1024*1024),   // 4MB
)

// 客户端消息大小配置
conn, _ := grpc.Dial(target,
    grpc.WithDefaultCallOptions(
        grpc.MaxCallRecvMsgSize(4*1024*1024),
    ),
)
```

### 4.4 Keepalive 配置

```go
import "google.golang.org/grpc/keepalive"

// 服务端 Keepalive
s := grpc.NewServer(
    grpc.KeepaliveParams(keepalive.ServerParameters{
        MaxConnectionIdle:     5 * time.Minute,   // 空闲连接最大时间
        MaxConnectionAge:      30 * time.Minute,  // 连接最大存活时间
        MaxConnectionAgeGrace: 10 * time.Second,  // 优雅关闭时间
        Time:                  30 * time.Second,  // Ping 间隔
        Timeout:               10 * time.Second,  // Ping 超时
    }),
    grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
        MinTime:             10 * time.Second,    // 客户端最小 Ping 间隔
        PermitWithoutStream: true,                // 允许无活跃流时 Ping
    }),
)

// 客户端 Keepalive
conn, _ := grpc.Dial(target,
    grpc.WithKeepaliveParams(keepalive.ClientParameters{
        Time:                30 * time.Second,
        Timeout:             10 * time.Second,
        PermitWithoutStream: true,
    }),
)
```

---

## 5. 优雅关闭

### 5.1 Go 优雅关闭

```go
func main() {
    lis, _ := net.Listen("tcp", ":50051")

    s := grpc.NewServer()
    pb.RegisterOrderServiceServer(s, &orderService{})

    // 健康检查
    healthServer := health.NewServer()
    grpc_health_v1.RegisterHealthServer(s, healthServer)

    // 启动服务
    go s.Serve(lis)
    log.Println("Server started")

    // 等待终止信号
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
    <-sigCh

    log.Println("Shutting down...")

    // 1. 标记为不健康 (停止接收新流量)
    healthServer.SetServingStatus("",
        grpc_health_v1.HealthCheckResponse_NOT_SERVING)

    // 2. 优雅关闭 (等待进行中的 RPC 完成)
    done := make(chan struct{})
    go func() {
        s.GracefulStop()
        close(done)
    }()

    // 3. 超时强制关闭
    select {
    case <-done:
        log.Println("Server stopped gracefully")
    case <-time.After(30 * time.Second):
        log.Println("Force stopping server")
        s.Stop()
    }

    // 4. 关闭数据库连接等资源
    cleanup()
}
```

---

## 6. API 演进

### 6.1 向后兼容变更

```
✅ 安全的变更:
- 新增消息类型
- 新增 optional 字段 (新编号)
- 新增枚举值
- 新增 oneof 字段
- 新增 RPC 方法
- 新增服务
- 重命名字段 (线格式只看编号)
- 修改注释

❌ 破坏性变更:
- 删除/重命名字段 (应使用 reserved)
- 复用字段编号
- 修改字段类型 (不兼容的)
- 修改字段编号
- 删除 RPC 方法
- 修改 RPC 请求/响应类型
- 将 repeated 改为 singular
- 将 singular 改为 oneof
```

### 6.2 版本化策略

```protobuf
// 策略1: 包版本化 (推荐)
package order.v1;
package order.v2;

// 策略2: 独立文件
order_v1.proto
order_v2.proto

// 策略3: 同包内演进 (小变更)
// 新增字段不影响旧客户端
// 大变更使用新包
```

---

## 7. 常见陷阱

### 7.1 字段编号冲突

```protobuf
// ❌ 危险: 复用已删除字段的编号
message Bad {
    reserved 2;
    string full_name = 2;  // 编号 2 已被 reserved！
}

// ✅ 安全: 使用新编号
message Good {
    reserved 2;
    string full_name = 3;  // 新编号
}
```

### 7.2 默认值问题

```go
// proto3 中无法区分"未设置"和"默认值"
req := &pb.SearchRequest{PageSize: 0}  // 是未设置还是设置为0？

// 解决方案1: 使用 optional (proto3.15+)
optional int32 page_size = 1;  // Go: *int32

// 解决方案2: 在业务代码中处理
if req.PageSize <= 0 {
    req.PageSize = 10  // 默认值
}

// 解决方案3: 使用 wrapper 类型
google.protobuf.Int32Value page_size = 1;  // Go: *wrapperspb.Int32Value
```

### 7.3 大消息 OOM

```go
// ❌ 危险: 没有限制消息大小
s := grpc.NewServer()  // 默认 4MB

// ✅ 安全: 设置合理的限制
s := grpc.NewServer(
    grpc.MaxRecvMsgSize(4*1024*1024),  // 4MB
)
```

### 7.4 长连接导致的负载不均

```
问题: 客户端与服务端建立长连接，新加入的服务端收不到请求

解决:
1. 使用 round_robin 而非 pick_first
2. 定期重建连接 (MaxConnectionAge)
3. 使用 L7 代理 (Envoy)
4. 使用 Service Mesh
```

---

## 8. 检查清单

### 8.1 上线前检查

```
安全:
□ 启用 TLS 加密
□ 实现认证 (JWT/mTLS)
□ 实现授权 (RBAC)
□ 输入验证
□ 限流配置

可靠性:
□ 超时配置 (客户端 + 服务端)
□ 重试策略 (幂等操作)
□ 健康检查
□ 优雅关闭
□ 错误处理 (使用正确的状态码)

可观测性:
□ 结构化日志
□ 链路追踪 (OpenTelemetry)
□ Prometheus 指标
□ Grafana Dashboard
□ 告警规则

性能:
□ 连接复用
□ Keepalive 配置
□ 消息大小限制
□ 并发控制

部署:
□ Dockerfile (多阶段构建)
□ Kubernetes Deployment + Service
□ 资源限制 (CPU/Memory)
□ 就绪/存活探针
□ 配置管理 (ConfigMap/Secret)
```

### 8.2 gRPC 服务 SLI/SLO 示例

```
可用性 (Availability):
  SLI: 成功请求 / 总请求
  SLO: 99.9% (每月不超过 43 分钟不可用)

延迟 (Latency):
  SLI: P99 请求延迟
  SLO: P99 < 200ms

错误率 (Error Rate):
  SLI: 5xx 错误 / 总请求
  SLO: < 0.1%

吞吐量 (Throughput):
  SLI: 每秒处理的请求数
  SLO: > 10,000 QPS
```

---

## 9. 总结

1. **项目结构**: 清晰分层，proto 文件版本化
2. **部署**: Docker 多阶段构建，K8s 健康检查
3. **可观测性**: 日志 + 追踪 + 指标 三位一体
4. **性能**: 连接复用、Keepalive、消息大小控制
5. **可靠性**: 超时、重试、优雅关闭
6. **API 演进**: 向后兼容，版本化
7. **上线检查**: 安全、可靠性、可观测性、性能、部署

恭喜你完成了 gRPC 完整教程的学习！
