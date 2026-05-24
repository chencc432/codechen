# gRPC 拦截器与中间件

## 1. 拦截器概述

拦截器是 gRPC 的中间件机制，允许在 RPC 调用的前后插入自定义逻辑，类似于 HTTP 中间件的概念。

### 1.1 拦截器分类

```
gRPC 拦截器
├── 客户端拦截器 (Client Interceptor)
│   ├── 一元拦截器 (Unary Client Interceptor)
│   └── 流式拦截器 (Stream Client Interceptor)
│
└── 服务端拦截器 (Server Interceptor)
    ├── 一元拦截器 (Unary Server Interceptor)
    └── 流式拦截器 (Stream Server Interceptor)
```

### 1.2 调用链

```
客户端调用
    │
    ▼
[Unary Client Interceptor 1] ─── Before
    │
    ▼
[Unary Client Interceptor 2] ─── Before
    │
    ▼
发送请求到服务端
    │
    ▼
[Unary Server Interceptor 1] ─── Before
    │
    ▼
[Unary Server Interceptor 2] ─── Before
    │
    ▼
服务实现
    │
    ▼
[Unary Server Interceptor 2] ─── After
    │
    ▼
[Unary Server Interceptor 1] ─── After
    │
    ▼
返回响应到客户端
    │
    ▼
[Unary Client Interceptor 2] ─── After
    │
    ▼
[Unary Client Interceptor 1] ─── After
    │
    ▼
客户端收到响应
```

---

## 2. Go 拦截器

### 2.1 一元服务端拦截器

```go
package interceptor

import (
    "context"
    "log"
    "time"

    "google.golang.org/grpc"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/metadata"
    "google.golang.org/grpc/status"
)

// 日志拦截器
func UnaryServerLoggingInterceptor() grpc.UnaryServerInterceptor {
    return func(
        ctx context.Context,
        req interface{},
        info *grpc.UnaryServerInfo,
        handler grpc.UnaryHandler,
    ) (interface{}, error) {
        start := time.Now()

        // 调用前
        log.Printf("[SERVER] %s started", info.FullMethod)

        // 调用实际方法
        resp, err := handler(ctx, req)

        // 调用后
        duration := time.Since(start)
        code := status.Code(err)
        log.Printf("[SERVER] %s finished, code=%v, duration=%v",
            info.FullMethod, code, duration)

        return resp, err
    }
}

// 认证拦截器
func UnaryServerAuthInterceptor() grpc.UnaryServerInterceptor {
    return func(
        ctx context.Context,
        req interface{},
        info *grpc.UnaryServerInfo,
        handler grpc.UnaryHandler,
    ) (interface{}, error) {
        // 跳过不需要认证的方法
        if info.FullMethod == "/grpc.health.v1.Health/Check" {
            return handler(ctx, req)
        }

        // 从 metadata 中获取 token
        md, ok := metadata.FromIncomingContext(ctx)
        if !ok {
            return nil, status.Error(codes.Unauthenticated, "missing metadata")
        }

        tokens := md.Get("authorization")
        if len(tokens) == 0 {
            return nil, status.Error(codes.Unauthenticated, "missing authorization token")
        }

        token := tokens[0]
        if !strings.HasPrefix(token, "Bearer ") {
            return nil, status.Error(codes.Unauthenticated, "invalid token format")
        }

        // 验证 token
        userID, err := validateToken(token[7:])
        if err != nil {
            return nil, status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
        }

        // 将用户信息存入 context
        ctx = context.WithValue(ctx, "userID", userID)

        return handler(ctx, req)
    }
}

// 请求验证拦截器
func UnaryServerValidationInterceptor() grpc.UnaryServerInterceptor {
    return func(
        ctx context.Context,
        req interface{},
        info *grpc.UnaryServerInfo,
        handler grpc.UnaryHandler,
    ) (interface{}, error) {
        // 如果请求实现了 validator 接口，则进行验证
        if validator, ok := req.(interface{ Validate() error }); ok {
            if err := validator.Validate(); err != nil {
                return nil, status.Errorf(codes.InvalidArgument,
                    "validation failed: %v", err)
            }
        }

        return handler(ctx, req)
    }
}

// Panic 恢复拦截器
func UnaryServerRecoveryInterceptor() grpc.UnaryServerInterceptor {
    return func(
        ctx context.Context,
        req interface{},
        info *grpc.UnaryServerInfo,
        handler grpc.UnaryHandler,
    ) (resp interface{}, err error) {
        defer func() {
            if r := recover(); r != nil {
                log.Printf("[SERVER] Panic recovered: %v", r)
                err = status.Errorf(codes.Internal,
                    "internal server error")
            }
        }()

        return handler(ctx, req)
    }
}
```

### 2.2 流式服务端拦截器

```go
// 流式日志拦截器
func StreamServerLoggingInterceptor() grpc.StreamServerInterceptor {
    return func(
        srv interface{},
        ss grpc.ServerStream,
        info *grpc.StreamServerInfo,
        handler grpc.StreamHandler,
    ) error {
        start := time.Now()
        log.Printf("[SERVER STREAM] %s started, isClientStream=%v, isServerStream=%v",
            info.FullMethod, info.IsClientStream, info.IsServerStream)

        err := handler(srv, ss)

        duration := time.Since(start)
        code := status.Code(err)
        log.Printf("[SERVER STREAM] %s finished, code=%v, duration=%v",
            info.FullMethod, code, duration)

        return err
    }
}

// 流式认证拦截器
func StreamServerAuthInterceptor() grpc.StreamServerInterceptor {
    return func(
        srv interface{},
        ss grpc.ServerStream,
        info *grpc.StreamServerInfo,
        handler grpc.StreamHandler,
    ) error {
        md, ok := metadata.FromIncomingContext(ss.Context())
        if !ok {
            return status.Error(codes.Unauthenticated, "missing metadata")
        }

        tokens := md.Get("authorization")
        if len(tokens) == 0 {
            return status.Error(codes.Unauthenticated, "missing token")
        }

        return handler(srv, ss)
    }
}
```

### 2.3 一元客户端拦截器

```go
// 客户端日志拦截器
func UnaryClientLoggingInterceptor() grpc.UnaryClientInterceptor {
    return func(
        ctx context.Context,
        method string,
        req, reply interface{},
        cc *grpc.ClientConn,
        invoker grpc.UnaryInvoker,
        opts ...grpc.CallOption,
    ) error {
        start := time.Now()
        log.Printf("[CLIENT] %s started", method)

        err := invoker(ctx, method, req, reply, cc, opts...)

        duration := time.Since(start)
        code := status.Code(err)
        log.Printf("[CLIENT] %s finished, code=%v, duration=%v",
            method, code, duration)

        return err
    }
}

// 客户端注入元数据拦截器
func UnaryClientMetadataInterceptor() grpc.UnaryClientInterceptor {
    return func(
        ctx context.Context,
        method string,
        req, reply interface{},
        cc *grpc.ClientConn,
        invoker grpc.UnaryInvoker,
        opts ...grpc.CallOption,
    ) error {
        // 注入 trace ID
        traceID := uuid.New().String()
        ctx = metadata.AppendToOutgoingContext(ctx,
            "x-trace-id", traceID,
            "x-request-id", uuid.New().String(),
        )

        return invoker(ctx, method, req, reply, cc, opts...)
    }
}

// 客户端重试拦截器
func UnaryClientRetryInterceptor(maxRetries int) grpc.UnaryClientInterceptor {
    return func(
        ctx context.Context,
        method string,
        req, reply interface{},
        cc *grpc.ClientConn,
        invoker grpc.UnaryInvoker,
        opts ...grpc.CallOption,
    ) error {
        var lastErr error

        for attempt := 0; attempt <= maxRetries; attempt++ {
            if attempt > 0 {
                // 指数退避
                backoff := time.Duration(attempt*attempt) * 100 * time.Millisecond
                select {
                case <-ctx.Done():
                    return ctx.Err()
                case <-time.After(backoff):
                }
            }

            lastErr = invoker(ctx, method, req, reply, cc, opts...)
            if lastErr == nil {
                return nil
            }

            // 只重试特定错误码
            code := status.Code(lastErr)
            if code != codes.Unavailable && code != codes.DeadlineExceeded {
                return lastErr
            }

            log.Printf("[CLIENT] %s attempt %d failed: %v, retrying...",
                method, attempt+1, lastErr)
        }

        return lastErr
    }
}
```

### 2.4 流式客户端拦截器

```go
func StreamClientLoggingInterceptor() grpc.StreamClientInterceptor {
    return func(
        ctx context.Context,
        desc *grpc.StreamDesc,
        cc *grpc.ClientConn,
        method string,
        streamer grpc.Streamer,
        opts ...grpc.CallOption,
    ) (grpc.ClientStream, error) {
        log.Printf("[CLIENT STREAM] %s started", method)

        stream, err := streamer(ctx, desc, cc, method, opts...)
        if err != nil {
            log.Printf("[CLIENT STREAM] %s failed to start: %v", method, err)
            return nil, err
        }

        return &loggingClientStream{ClientStream: stream, method: method}, nil
    }
}

type loggingClientStream struct {
    grpc.ClientStream
    method string
}

func (s *loggingClientStream) SendMsg(m interface{}) error {
    log.Printf("[CLIENT STREAM] %s sending message", s.method)
    err := s.ClientStream.SendMsg(m)
    if err != nil {
        log.Printf("[CLIENT STREAM] %s send error: %v", s.method, err)
    }
    return err
}

func (s *loggingClientStream) RecvMsg(m interface{}) error {
    err := s.ClientStream.RecvMsg(m)
    if err == io.EOF {
        log.Printf("[CLIENT STREAM] %s received EOF", s.method)
    } else if err != nil {
        log.Printf("[CLIENT STREAM] %s recv error: %v", s.method, err)
    }
    return err
}
```

### 2.5 注册拦截器

```go
// 服务端
func NewGRPCServer() *grpc.Server {
    s := grpc.NewServer(
        // 一元拦截器链 (按顺序执行)
        grpc.ChainUnaryInterceptor(
            UnaryServerRecoveryInterceptor(),    // 1. panic 恢复 (最外层)
            UnaryServerLoggingInterceptor(),     // 2. 日志
            UnaryServerAuthInterceptor(),        // 3. 认证
            UnaryServerValidationInterceptor(),  // 4. 验证 (最内层)
        ),
        // 流式拦截器链
        grpc.ChainStreamInterceptor(
            StreamServerLoggingInterceptor(),
            StreamServerAuthInterceptor(),
        ),
    )
    return s
}

// 客户端
func NewGRPCClient(target string) (*grpc.ClientConn, error) {
    conn, err := grpc.Dial(target,
        grpc.WithTransportCredentials(insecure.NewCredentials()),
        grpc.ChainUnaryInterceptor(
            UnaryClientMetadataInterceptor(),
            UnaryClientLoggingInterceptor(),
            UnaryClientRetryInterceptor(3),
        ),
        grpc.ChainStreamInterceptor(
            StreamClientLoggingInterceptor(),
        ),
    )
    return conn, err
}
```

---

## 3. Java 拦截器

### 3.1 服务端拦截器

```java
// 日志拦截器
public class ServerLoggingInterceptor implements ServerInterceptor {

    @Override
    public <ReqT, RespT> ServerCall.Listener<ReqT> interceptCall(
            ServerCall<ReqT, RespT> call,
            Metadata headers,
            ServerCallHandler<ReqT, RespT> next) {

        String method = call.getMethodDescriptor().getFullMethodName();
        long startTime = System.currentTimeMillis();

        // 包装 ServerCall 以拦截响应
        ServerCall<ReqT, RespT> wrappedCall = new ForwardingServerCall.SimpleForwardingServerCall<>(call) {
            @Override
            public void close(Status status, Metadata trailers) {
                long duration = System.currentTimeMillis() - startTime;
                System.out.printf("[SERVER] %s finished, status=%s, duration=%dms%n",
                    method, status.getCode(), duration);
                super.close(status, trailers);
            }
        };

        System.out.printf("[SERVER] %s started%n", method);

        return next.startCall(wrappedCall, headers);
    }
}

// 认证拦截器
public class ServerAuthInterceptor implements ServerInterceptor {

    private static final Metadata.Key<String> AUTH_KEY =
        Metadata.Key.of("authorization", Metadata.ASCII_STRING_MARSHALLER);

    @Override
    public <ReqT, RespT> ServerCall.Listener<ReqT> interceptCall(
            ServerCall<ReqT, RespT> call,
            Metadata headers,
            ServerCallHandler<ReqT, RespT> next) {

        String token = headers.get(AUTH_KEY);
        if (token == null || !token.startsWith("Bearer ")) {
            call.close(Status.UNAUTHENTICATED
                .withDescription("Missing or invalid authorization token"),
                new Metadata());
            return new ServerCall.Listener<ReqT>() {};
        }

        // 验证 token
        String userId = validateToken(token.substring(7));

        // 将用户信息存入 Context
        Context ctx = Context.current().withValue(
            Context.key("userId"), userId);

        return Contexts.interceptCall(ctx, call, headers, next);
    }
}
```

### 3.2 客户端拦截器

```java
// 元数据注入拦截器
public class ClientMetadataInterceptor implements ClientInterceptor {

    @Override
    public <ReqT, RespT> ClientCall<ReqT, RespT> interceptCall(
            MethodDescriptor<ReqT, RespT> method,
            CallOptions callOptions,
            Channel next) {

        return new ForwardingClientCall.SimpleForwardingClientCall<ReqT, RespT>(
                next.newCall(method, callOptions)) {
            @Override
            public void start(Listener<RespT> responseListener, Metadata headers) {
                // 注入元数据
                headers.put(
                    Metadata.Key.of("x-trace-id", Metadata.ASCII_STRING_MARSHALLER),
                    UUID.randomUUID().toString()
                );
                headers.put(
                    Metadata.Key.of("authorization", Metadata.ASCII_STRING_MARSHALLER),
                    "Bearer " + getToken()
                );
                super.start(responseListener, headers);
            }
        };
    }
}

// 注册
ManagedChannel channel = ManagedChannelBuilder
    .forAddress("localhost", 50051)
    .intercept(new ClientMetadataInterceptor())
    .usePlaintext()
    .build();
```

---

## 4. Python 拦截器

### 4.1 服务端拦截器

```python
class ServerLoggingInterceptor(grpc.ServerInterceptor):
    """服务端日志拦截器"""

    def intercept_service(self, continuation, handler_call_details):
        method = handler_call_details.method
        start = time.time()
        logger.info(f"[SERVER] {method} called")

        handler = continuation(handler_call_details)

        if handler is None:
            return None

        # 包装处理函数
        def wrapper(request, context):
            try:
                response = handler(request, context)
                duration = time.time() - start
                logger.info(f"[SERVER] {method} completed in {duration:.3f}s")
                return response
            except Exception as e:
                duration = time.time() - start
                logger.error(f"[SERVER] {method} error after {duration:.3f}s: {e}")
                raise

        # 返回新的 handler
        if handler.unary_unary:
            return grpc.unary_unary_rpc_method_handler(
                wrapper,
                request_deserializer=handler.request_deserializer,
                response_serializer=handler.response_serializer,
            )
        return handler


class ServerAuthInterceptor(grpc.ServerInterceptor):
    """服务端认证拦截器"""

    def intercept_service(self, continuation, handler_call_details):
        metadata = dict(handler_call_details.invocation_metadata)
        token = metadata.get("authorization", "")

        if not token.startswith("Bearer "):
            return self._abort_handler(grpc.StatusCode.UNAUTHENTICATED,
                                       "Missing token")

        return continuation(handler_call_details)

    def _abort_handler(self, code, details):
        def abort(request, context):
            context.abort(code, details)

        return grpc.unary_unary_rpc_method_handler(abort)
```

### 4.2 客户端拦截器

```python
class ClientLoggingInterceptor(grpc.UnaryUnaryClientInterceptor):
    """客户端日志拦截器"""

    def intercept_unary_unary(self, continuation, client_call_details, request):
        method = client_call_details.method
        start = time.time()
        logger.info(f"[CLIENT] {method} started")

        response = continuation(client_call_details, request)

        duration = time.time() - start
        logger.info(f"[CLIENT] {method} completed in {duration:.3f}s")
        return response


class ClientMetadataInterceptor(grpc.UnaryUnaryClientInterceptor):
    """客户端元数据拦截器"""

    def intercept_unary_unary(self, continuation, client_call_details, request):
        metadata = list(client_call_details.metadata or [])
        metadata.append(("x-trace-id", str(uuid.uuid4())))
        metadata.append(("authorization", f"Bearer {self.get_token()}"))

        new_details = _ClientCallDetails(
            method=client_call_details.method,
            timeout=client_call_details.timeout,
            metadata=metadata,
            credentials=client_call_details.credentials,
            wait_for_ready=client_call_details.wait_for_ready,
            compression=client_call_details.compression,
        )

        return continuation(new_details, request)
```

---

## 5. 常见拦截器模式

### 5.1 限流拦截器

```go
// Go 限流拦截器
func UnaryServerRateLimitInterceptor(limiter *rate.Limiter) grpc.UnaryServerInterceptor {
    return func(
        ctx context.Context,
        req interface{},
        info *grpc.UnaryServerInfo,
        handler grpc.UnaryHandler,
    ) (interface{}, error) {
        if !limiter.Allow() {
            return nil, status.Error(codes.ResourceExhausted,
                "rate limit exceeded")
        }
        return handler(ctx, req)
    }
}

// 使用
limiter := rate.NewLimiter(rate.Limit(100), 10) // 100/s, burst 10
s := grpc.NewServer(
    grpc.ChainUnaryInterceptor(
        UnaryServerRateLimitInterceptor(limiter),
    ),
)
```

### 5.2 链路追踪拦截器

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/propagation"
    "go.opentelemetry.io/otel/trace"
)

func UnaryServerTracingInterceptor() grpc.UnaryServerInterceptor {
    return func(
        ctx context.Context,
        req interface{},
        info *grpc.UnaryServerInfo,
        handler grpc.UnaryHandler,
    ) (interface{}, error) {
        // 从 metadata 提取 trace context
        propagator := otel.GetTextMapPropagator()
        md, _ := metadata.FromIncomingContext(ctx)
        ctx = propagator.Extract(ctx, &metadataCarrier{md})

        // 创建 span
        tracer := otel.Tracer("grpc-server")
        ctx, span := tracer.Start(ctx, info.FullMethod)
        defer span.End()

        resp, err := handler(ctx, req)
        if err != nil {
            span.RecordError(err)
            span.SetAttributes(
                attribute.String("rpc.status_code", status.Code(err).String()),
            )
        }

        return resp, err
    }
}

// Metadata carrier (实现 propagation.TextMapCarrier)
type metadataCarrier struct {
    md metadata.MD
}

func (c *metadataCarrier) Get(key string) string {
    values := c.md.Get(key)
    if len(values) == 0 {
        return ""
    }
    return values[0]
}

func (c *metadataCarrier) Set(key, value string) {
    c.md.Set(key, value)
}

func (c *metadataCarrier) Keys() []string {
    keys := make([]string, 0, len(c.md))
    for k := range c.md {
        keys = append(keys, k)
    }
    return keys
}
```

### 5.3 指标收集拦截器

```go
import "github.com/prometheus/client_golang/prometheus"

var (
    grpcRequestsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "grpc_server_requests_total",
            Help: "Total number of gRPC requests",
        },
        []string{"method", "status"},
    )

    grpcRequestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "grpc_server_request_duration_seconds",
            Help:    "gRPC request duration in seconds",
            Buckets: prometheus.DefBuckets,
        },
        []string{"method"},
    )
)

func UnaryServerMetricsInterceptor() grpc.UnaryServerInterceptor {
    return func(
        ctx context.Context,
        req interface{},
        info *grpc.UnaryServerInfo,
        handler grpc.UnaryHandler,
    ) (interface{}, error) {
        timer := prometheus.NewTimer(
            grpcRequestDuration.WithLabelValues(info.FullMethod),
        )
        defer timer.ObserveDuration()

        resp, err := handler(ctx, req)

        statusCode := status.Code(err).String()
        grpcRequestsTotal.WithLabelValues(info.FullMethod, statusCode).Inc()

        return resp, err
    }
}
```

### 5.4 请求ID注入拦截器

```go
func UnaryServerRequestIDInterceptor() grpc.UnaryServerInterceptor {
    return func(
        ctx context.Context,
        req interface{},
        info *grpc.UnaryServerInfo,
        handler grpc.UnaryHandler,
    ) (interface{}, error) {
        // 从 metadata 获取或生成 request ID
        md, _ := metadata.FromIncomingContext(ctx)
        requestIDs := md.Get("x-request-id")

        var requestID string
        if len(requestIDs) > 0 {
            requestID = requestIDs[0]
        } else {
            requestID = uuid.New().String()
        }

        // 存入 context
        ctx = context.WithValue(ctx, "requestID", requestID)

        // 添加到响应 header
        header := metadata.Pairs("x-request-id", requestID)
        grpc.SendHeader(ctx, header)

        return handler(ctx, req)
    }
}
```

---

## 6. 拦截器最佳实践

### 6.1 拦截器顺序

```go
// 推荐的拦截器顺序 (从外到内):
grpc.ChainUnaryInterceptor(
    RecoveryInterceptor(),     // 1. panic 恢复 (最外层，兜底)
    RequestIDInterceptor(),    // 2. 请求 ID (尽早生成)
    LoggingInterceptor(),      // 3. 日志 (记录所有请求)
    TracingInterceptor(),      // 4. 链路追踪
    MetricsInterceptor(),      // 5. 指标收集
    RateLimitInterceptor(),    // 6. 限流 (在认证前，保护系统)
    AuthInterceptor(),         // 7. 认证 (限流后)
    ValidationInterceptor(),   // 8. 验证 (认证后，最内层)
)
```

### 6.2 注意事项

1. **拦截器应该是无状态的**: 不要在拦截器中存储请求相关的状态
2. **错误处理要完善**: 确保拦截器中的错误不会导致 panic
3. **Context 传递**: 必须将修改后的 context 传递给 handler
4. **性能考虑**: 拦截器在每次 RPC 调用时都会执行，避免重操作
5. **链式调用**: 使用 ChainInterceptor 而不是单一 Interceptor

---

## 7. 总结

1. **四种拦截器**: 客户端一元/流式、服务端一元/流式
2. **执行顺序**: 外层拦截器先进入后退出，类似洋葱模型
3. **常见模式**: 日志、认证、验证、限流、追踪、指标、恢复
4. **顺序很重要**: 恢复→请求ID→日志→追踪→指标→限流→认证→验证

下一步: 学习 [gRPC 错误处理与超时控制](08-GRPC错误处理与超时控制.md)
