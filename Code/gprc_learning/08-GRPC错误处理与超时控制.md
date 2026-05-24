# gRPC 错误处理与超时控制

## 1. 错误处理概述

gRPC 使用标准化的状态码和错误详情来传递错误信息，比 HTTP 状态码更精细。

### 1.1 错误传播机制

```
服务端错误                    客户端接收
┌──────────────┐             ┌──────────────┐
│ status.Error │─── HTTP/2 ──▶│ RpcError     │
│ code + msg   │   HEADERS   │ code + msg   │
│ + details    │   + DATA    │ + details    │
└──────────────┘             └──────────────┘

HTTP/2 Trailing Headers:
  grpc-status: 5 (NOT_FOUND)
  grpc-message: Order ORD-123 not found
  grpc-status-details-bin: <base64 encoded Details>
```

---

## 2. gRPC 状态码

### 2.1 完整状态码参考

| 状态码 | 数值 | HTTP 映射 | 使用场景 |
|--------|------|-----------|----------|
| OK | 0 | 200 | 成功 |
| CANCELLED | 1 | 499 | 调用被取消 |
| UNKNOWN | 2 | 500 | 未知错误 |
| INVALID_ARGUMENT | 3 | 400 | 参数错误 |
| DEADLINE_EXCEEDED | 4 | 504 | 超时 |
| NOT_FOUND | 5 | 404 | 资源不存在 |
| ALREADY_EXISTS | 6 | 409 | 资源已存在 |
| PERMISSION_DENIED | 7 | 403 | 权限不足 |
| RESOURCE_EXHAUSTED | 8 | 429 | 资源耗尽/限流 |
| FAILED_PRECONDITION | 9 | 400 | 前置条件不满足 |
| ABORTED | 10 | 409 | 操作中止(并发冲突) |
| OUT_OF_RANGE | 11 | 400 | 超出范围 |
| UNIMPLEMENTED | 12 | 501 | 方法未实现 |
| INTERNAL | 13 | 500 | 内部错误 |
| UNAVAILABLE | 14 | 503 | 服务不可用 |
| DATA_LOSS | 15 | 500 | 数据丢失 |
| UNAUTHENTICATED | 16 | 401 | 未认证 |

### 2.2 状态码选择指南

```
收到请求后如何选择状态码:

客户端请求错误 (4xx 类):
├── 参数验证失败 → INVALID_ARGUMENT
├── 认证失败 → UNAUTHENTICATED
├── 权限不足 → PERMISSION_DENIED
├── 资源不存在 → NOT_FOUND
├── 资源已存在 → ALREADY_EXISTS
└── 前置条件不满足 → FAILED_PRECONDITION

服务端问题 (5xx 类):
├── 服务不可用 → UNAVAILABLE
├── 内部错误 → INTERNAL
├── 超时 → DEADLINE_EXCEEDED
├── 限流 → RESOURCE_EXHAUSTED
├── 并发冲突 → ABORTED
└── 未实现 → UNIMPLEMENTED
```

---

## 3. 错误详情 (Error Details)

### 3.1 标准 Error Details

gRPC 提供了 `google.rpc.Status` 和 `google.rpc.*` 错误详情类型：

```protobuf
// google/rpc/status.proto
message Status {
    int32 code = 1;
    string message = 2;
    repeated google.protobuf.Any details = 3;
}

// google/rpc/error_details.proto
message RetryInfo {
    google.protobuf.Duration retry_delay = 1;
}

message DebugInfo {
    repeated string stack_entries = 1;
    string detail = 2;
}

message QuotaFailure {
    message Violation {
        string subject = 1;
        string description = 2;
    }
    repeated Violation violations = 1;
}

message BadRequest {
    message FieldViolation {
        string field = 1;
        string description = 2;
    }
    repeated FieldViolation field_violations = 1;
}

message PreconditionFailure {
    message Violation {
        string type = 1;
        string subject = 2;
        string description = 3;
    }
    repeated Violation violations = 1;
}

message ResourceInfo {
    string resource_type = 1;
    string resource_name = 2;
    string owner = 3;
    string description = 4;
}

message Help {
    message Link {
        string description = 1;
        string url = 2;
    }
    repeated Link links = 1;
}

message LocalizedMessage {
    string locale = 1;
    string message = 2;
}
```

### 3.2 Go 错误详情

```go
import (
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
    "google.golang.org/genproto/googleapis/rpc/errdetails"
)

// 返回带详情的错误
func (s *server) CreateOrder(ctx context.Context,
    req *pb.CreateOrderRequest) (*pb.Order, error) {

    // 参数验证 - 使用 BadRequest 详情
    if req.UserId == "" {
        st := status.New(codes.InvalidArgument, "validation failed")
        desc := "user_id is required"
        v := &errdetails.BadRequest_FieldViolation{
            Field:       "user_id",
            Description: desc,
        }
        br := &errdetails.BadRequest{}
        br.FieldViolations = append(br.FieldViolations, v)
        st, _ = st.WithDetails(br)
        return nil, st.Err()
    }

    // 限流 - 使用 RetryInfo 详情
    if !s.limiter.Allow() {
        st := status.New(codes.ResourceExhausted, "rate limit exceeded")
        retryInfo := &errdetails.RetryInfo{
            RetryDelay: durationpb.New(1 * time.Second),
        }
        st, _ = st.WithDetails(retryInfo)
        return nil, st.Err()
    }

    // 资源不存在 - 使用 ResourceInfo 详情
    if !s.productExists(req.ProductId) {
        st := status.New(codes.NotFound, "product not found")
        resInfo := &errdetails.ResourceInfo{
            ResourceType: "Product",
            ResourceName: req.ProductId,
            Owner:        "catalog-service",
        }
        st, _ = st.WithDetails(resInfo)
        return nil, st.Err()
    }

    // ...正常处理
}

// 客户端解析错误详情
func handleError(err error) {
    st, ok := status.FromError(err)
    if !ok {
        // 不是 gRPC 错误
        log.Printf("Non-gRPC error: %v", err)
        return
    }

    log.Printf("gRPC error: code=%v, message=%s", st.Code(), st.Message())

    for _, detail := range st.Details() {
        switch d := detail.(type) {
        case *errdetails.BadRequest:
            for _, v := range d.FieldViolations {
                log.Printf("  Field: %s, Error: %s", v.Field, v.Description)
            }
        case *errdetails.RetryInfo:
            log.Printf("  Retry after: %v", d.RetryDelay.AsDuration())
        case *errdetails.ResourceInfo:
            log.Printf("  Resource: type=%s, name=%s",
                d.ResourceType, d.ResourceName)
        case *errdetails.QuotaFailure:
            for _, v := range d.Violations {
                log.Printf("  Quota: subject=%s, desc=%s",
                    v.Subject, v.Description)
            }
        case *errdetails.DebugInfo:
            log.Printf("  Debug: %s", d.Detail)
        case *errdetails.Help:
            for _, l := range d.Links {
                log.Printf("  Help: %s - %s", l.Description, l.Url)
            }
        case *errdetails.LocalizedMessage:
            log.Printf("  Localized [%s]: %s", d.Locale, d.Message)
        }
    }
}
```

### 3.3 Java 错误详情

```java
// 服务端返回带详情的错误
@Override
public void createOrder(CreateOrderRequest request,
                        StreamObserver<Order> responseObserver) {

    if (request.getUserId().isEmpty()) {
        // 使用 StatusRuntimeException
        BadRequest badRequest = BadRequest.newBuilder()
            .addFieldViolations(BadRequest.FieldViolation.newBuilder()
                .setField("user_id")
                .setDescription("user_id is required")
                .build())
            .build();

        StatusRuntimeException exception = Status.INVALID_ARGUMENT
            .withDescription("validation failed")
            .asException(Collections.singletonList(
                Any.pack(badRequest)
            ));

        responseObserver.onError(exception);
        return;
    }

    // 限流
    if (!rateLimiter.tryAcquire()) {
        RetryInfo retryInfo = RetryInfo.newBuilder()
            .setRetryDelay(Duration.newBuilder()
                .setSeconds(1)
                .build())
            .build();

        StatusRuntimeException exception = Status.RESOURCE_EXHAUSTED
            .withDescription("rate limit exceeded")
            .asException(Collections.singletonList(
                Any.pack(retryInfo)
            ));

        responseObserver.onError(exception);
        return;
    }
}

// 客户端解析错误详情
try {
    Order order = blockingStub.createOrder(request);
} catch (StatusRuntimeException e) {
    Status status = Status.fromThrowable(e);

    for (Any detail : e.getTrailers()
            .get(PROTO_STATUS_DETAILS_KEY)) {
        if (detail.is(BadRequest.class)) {
            BadRequest br = detail.unpack(BadRequest.class);
            for (BadRequest.FieldViolation fv : br.getFieldViolationsList()) {
                System.out.printf("Field: %s, Error: %s%n",
                    fv.getField(), fv.getDescription());
            }
        } else if (detail.is(RetryInfo.class)) {
            RetryInfo ri = detail.unpack(RetryInfo.class);
            System.out.printf("Retry after: %s%n", ri.getRetryDelay());
        }
    }
}
```

---

## 4. 超时控制

### 4.1 Deadline 与 Timeout

```
Deadline:  绝对时间点，RPC 必须在此之前完成
Timeout:   相对时间，从 RPC 开始计算的持续时间

客户端设置                    服务端传播
┌──────────────┐             ┌──────────────┐
│ Deadline:    │─── gRPC ───▶│ 检查 ctx     │
│ now + 5s    │   Header    │ deadline     │
│              │             │              │
│ Timeout:    │             │ 如果超时:    │
│ 5 seconds   │             │ 返回         │
└──────────────┘             │ DEADLINE_    │
                             │ EXCEEDED    │
                             └──────────────┘
```

### 4.2 Deadline 传播

```
客户端 (Deadline: T+5s)
    │
    ├── 调用服务A (耗时: 2s)
    │   └── 剩余 Deadline: T+3s 传播到服务A
    │       └── 服务A 调用服务B (剩余: T+3s)
    │           └── 剩余 Deadline: T+1s 传播到服务B
    │
    ├── 调用服务C (剩余 Deadline: T+3s)
    │   └── 如果耗时 > 3s → DEADLINE_EXCEEDED
    │
    └── 总 Deadline: T+5s
        └── 超过 T+5s 后所有子调用自动取消
```

### 4.3 Go 超时控制

```go
// 客户端设置超时
// 方式1: 使用 context.WithTimeout
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

resp, err := client.GetOrder(ctx, &pb.GetOrderRequest{OrderId: "ORD-123"})
if err != nil {
    if status.Code(err) == codes.DeadlineExceeded {
        log.Println("Request timed out")
    }
    log.Fatalf("GetOrder failed: %v", err)
}

// 方式2: 使用 context.WithDeadline
deadline := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)
ctx, cancel := context.WithDeadline(context.Background(), deadline)
defer cancel()

// 方式3: 使用 CallOption
resp, err := client.GetOrder(
    context.Background(),
    &pb.GetOrderRequest{OrderId: "ORD-123"},
    grpc.WaitForReady(true),
)

// 服务端检查超时
func (s *server) GetOrder(ctx context.Context, req *pb.GetOrderRequest) (*pb.Order, error) {
    // 检查是否已经超时
    if ctx.Err() == context.DeadlineExceeded {
        return nil, status.Error(codes.DeadlineExceeded, "request timed out")
    }

    // 检查剩余时间
    if deadline, ok := ctx.Deadline(); ok {
        remaining := time.Until(deadline)
        if remaining < 100*time.Millisecond {
            return nil, status.Error(codes.DeadlineExceeded,
                "insufficient time remaining")
        }
    }

    // 长时间操作中定期检查
    resultCh := make(chan *pb.Order, 1)
    errCh := make(chan error, 1)

    go func() {
        order, err := s.db.GetOrder(req.OrderId)
        if err != nil {
            errCh <- err
            return
        }
        resultCh <- order
    }()

    select {
    case order := <-resultCh:
        return order, nil
    case err := <-errCh:
        return nil, status.Errorf(codes.Internal, "db error: %v", err)
    case <-ctx.Done():
        return nil, status.FromContextError(ctx.Err()).Err()
    }
}
```

### 4.4 Java 超时控制

```java
// 客户端设置超时

// 方式1: withDeadlineAfter
Order order = blockingStub
    .withDeadlineAfter(5, TimeUnit.SECONDS)
    .getOrder(request);

// 方式2: withDeadline
Instant deadline = Instant.now().plusSeconds(5);
Order order = blockingStub
    .withDeadline(deadline)
    .getOrder(request);

// 方式3: Channel 级别默认超时
ManagedChannel channel = ManagedChannelBuilder
    .forAddress("localhost", 50051)
    .defaultLoadBalancingPolicy("round_robin")
    .build();

// 服务端检查超时
@Override
public void getOrder(GetOrderRequest request,
                     StreamObserver<Order> responseObserver) {
    // 检查是否已取消
    if (Context.current().isCancelled()) {
        responseObserver.onError(Status.CANCELLED
            .withDescription("request cancelled by client")
            .asRuntimeException());
        return;
    }

    // 获取剩余时间
    Deadline deadline = Context.current().getDeadline();
    if (deadline != null) {
        long remainingMs = deadline.timeRemaining(TimeUnit.MILLISECONDS);
        if (remainingMs < 100) {
            responseObserver.onError(Status.DEADLINE_EXCEEDED
                .withDescription("insufficient time remaining")
                .asRuntimeException());
            return;
        }
    }
}
```

### 4.5 Python 超时控制

```python
# 客户端设置超时
# 方式1: timeout 参数
order = stub.GetOrder(request, timeout=5)  # 5秒超时

# 方式2: 在创建 channel 时设置
channel = grpc.insecure_channel("localhost:50051")
stub = order_pb2_grpc.OrderServiceStub(channel)

try:
    order = stub.GetOrder(request, timeout=5)
except grpc.RpcError as e:
    if e.code() == grpc.StatusCode.DEADLINE_EXCEEDED:
        print("Request timed out")

# 服务端检查超时
def GetOrder(self, request, context):
    # 检查是否已取消
    if context.is_active():
        # 检查剩余时间
        deadline = context.time_remaining()
        if deadline is not None and deadline < 0.1:
            context.abort(
                grpc.StatusCode.DEADLINE_EXCEEDED,
                "insufficient time remaining",
            )

    # 长时间操作
    try:
        order = self.db.get_order(request.order_id)
    except Exception as e:
        context.abort(grpc.StatusCode.INTERNAL, str(e))
```

---

## 5. 取消机制

### 5.1 客户端取消

```go
// Go: 使用 WithCancel 主动取消
ctx, cancel := context.WithCancel(context.Background())

go func() {
    time.Sleep(2 * time.Second)
    cancel() // 2秒后取消
}()

resp, err := client.GetOrder(ctx, req)
if err != nil {
    if status.Code(err) == codes.Canceled {
        log.Println("Request was cancelled")
    }
}
```

```java
// Java: 使用 Context.cancel
Context.CancellableContext cancellableCtx =
    Context.current().withCancellation();

cancellableCtx.run(() -> {
    try {
        Order order = blockingStub.getOrder(request);
    } catch (StatusRuntimeException e) {
        if (e.getStatus().getCode() == Status.Code.CANCELLED) {
            System.out.println("Request was cancelled");
        }
    }
});

// 取消
cancellableCtx.cancel(new InterruptedException("user cancelled"));
```

### 5.2 服务端处理取消

```go
func (s *server) LongRunningOperation(ctx context.Context,
    req *pb.Request) (*pb.Response, error) {

    // 方式1: 检查 context
    if ctx.Err() == context.Canceled {
        return nil, status.Error(codes.Canceled, "request cancelled")
    }

    // 方式2: 在循环中检查
    for _, item := range items {
        select {
        case <-ctx.Done():
            log.Println("Operation cancelled, cleaning up...")
            // 执行清理操作
            return nil, status.FromContextError(ctx.Err()).Err()
        default:
            // 继续处理
            process(item)
        }
    }

    return &pb.Response{}, nil
}
```

---

## 6. 重试策略

### 6.1 gRPC 内置重试

```go
// Go: 通过 Service Config 配置重试
conn, err := grpc.Dial("localhost:50051",
    grpc.WithDefaultServiceConfig(`{
        "methodConfig": [{
            "name": [{
                "service": "order.OrderService",
                "method": "GetOrder"
            }],
            "retryPolicy": {
                "MaxAttempts": 3,
                "InitialBackoff": "0.1s",
                "MaxBackoff": "1s",
                "BackoffMultiplier": 2,
                "RetryableStatusCodes": ["UNAVAILABLE", "DEADLINE_EXCEEDED"]
            },
            "timeout": "5s"
        }, {
            "name": [{
                "service": "order.OrderService",
                "method": "CreateOrder"
            }],
            "retryPolicy": {
                "MaxAttempts": 1
            },
            "timeout": "10s"
        }]
    }`),
)
```

### 6.2 手动重试

```go
func withRetry(ctx context.Context, fn func(ctx context.Context) error, maxRetries int) error {
    var lastErr error

    for attempt := 0; attempt <= maxRetries; attempt++ {
        if attempt > 0 {
            // 指数退避 + 抖动
            backoff := time.Duration(attempt*attempt) * 100 * time.Millisecond
            jitter := time.Duration(rand.Intn(100)) * time.Millisecond
            select {
            case <-ctx.Done():
                return ctx.Err()
            case <-time.After(backoff + jitter):
            }
        }

        lastErr = fn(ctx)
        if lastErr == nil {
            return nil
        }

        // 不可重试的错误码
        code := status.Code(lastErr)
        if code != codes.Unavailable && code != codes.DeadlineExceeded {
            return lastErr
        }
    }

    return lastErr
}

// 使用
err = withRetry(ctx, func(ctx context.Context) error {
    _, err := client.GetOrder(ctx, req)
    return err
}, 3)
```

### 6.3 Hedging (对冲请求)

```go
// gRPC 支持 Hedging: 同时发送多个请求，使用最先返回的响应
conn, err := grpc.Dial("localhost:50051",
    grpc.WithDefaultServiceConfig(`{
        "methodConfig": [{
            "name": [{
                "service": "order.OrderService"
            }],
            "hedgingPolicy": {
                "MaxAttempts": 3,
                "HedgingDelay": "0.5s",
                "NonFatalStatusCodes": ["UNAVAILABLE"]
            }
        }]
    }`),
)
// 发送第一个请求，0.5s 后没响应则发送第二个，再 0.5s 后发送第三个
// 使用最先到达的响应
```

---

## 7. 错误处理最佳实践

### 7.1 服务端

```
1. 使用精确的状态码 (不要所有错误都返回 INTERNAL)
2. 包含有意义的错误消息 (不要泄露内部实现细节)
3. 使用 Error Details 提供结构化错误信息
4. 区分客户端错误和服务端错误
5. 长时间操作要检查 context 取消
6. 内部错误不要暴露给客户端
7. 记录完整错误日志 (包含堆栈)
```

### 7.2 客户端

```
1. 始终设置合理的 Deadline/Timeout
2. 正确处理所有可能的状态码
3. 对可重试错误实现重试逻辑
4. 解析 Error Details 获取更多信息
5. 不要忽略错误
6. 对关键操作实现幂等性
```

### 7.3 错误映射 (gRPC → HTTP)

如果通过 gRPC-Gateway 提供 REST API:

```
gRPC Status Code        →  HTTP Status Code
OK                      →  200
INVALID_ARGUMENT        →  400
UNAUTHENTICATED         →  401
PERMISSION_DENIED       →  403
NOT_FOUND               →  404
ALREADY_EXISTS          →  409
RESOURCE_EXHAUSTED      →  429
FAILED_PRECONDITION     →  400
ABORTED                 →  409
OUT_OF_RANGE            →  400
UNIMPLEMENTED           →  501
INTERNAL                →  500
UNAVAILABLE             →  503
DEADLINE_EXCEEDED       →  504
```

---

## 8. 总结

1. **状态码**: 使用精确的 gRPC 状态码，不要滥用
2. **错误详情**: 使用 `google.rpc.*` 提供结构化错误信息
3. **超时**: 始终设置 Deadline，它会自动传播到下游
4. **取消**: 使用 Context 取消机制优雅终止
5. **重试**: 对幂等操作配置重试策略，注意退避和抖动
6. **Hedging**: 对延迟敏感的操作可使用对冲请求

下一步: 学习 [gRPC 安全与认证](09-GRPC安全与认证.md)
