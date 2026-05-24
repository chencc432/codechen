# gRPC Go 快速入门

## 1. 环境准备

### 1.1 安装 Go

```bash
# 下载安装 Go 1.21+
# https://go.dev/dl/

# 验证
go version
# go version go1.21.x linux/amd64
```

### 1.2 安装 protoc 与 Go 插件

```bash
# 安装 protoc
# macOS
brew install protobuf

# Ubuntu
sudo apt install -y protobuf-compiler

# 验证
protoc --version

# 安装 Go 插件
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# 确保 PATH 包含 Go bin 目录
export PATH="$PATH:$(go env GOPATH)/bin"
```

### 1.3 安装 grpcurl (调试工具)

```bash
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest
```

---

## 2. 项目结构

### 2.1 推荐目录结构

```
grpc-demo/
├── cmd/
│   ├── server/
│   │   └── main.go           # 服务端入口
│   └── client/
│       └── main.go           # 客户端入口
├── proto/
│   └── demo/
│       └── demo.proto        # Protobuf 定义
├── gen/
│   └── demo/
│       ├── demo.pb.go        # 生成的消息代码
│       └── demo_grpc.pb.go   # 生成的 gRPC 代码
├── internal/
│   └── service/
│       └── demo_service.go   # 服务实现
├── go.mod
├── go.sum
├── Makefile                   # 构建脚本
└── buf.yaml                   # buf 配置 (可选)
```

### 2.2 初始化项目

```bash
mkdir grpc-demo && cd grpc-demo
go mod init github.com/example/grpc-demo
go get google.golang.org/grpc
go get google.golang.org/protobuf
```

---

## 3. 完整示例: 电商订单服务

### 3.1 定义 Protobuf

```protobuf
// proto/order/order.proto
syntax = "proto3";

package order;

option go_package = "github.com/example/grpc-demo/gen/order";

import "google/protobuf/timestamp.proto";
import "google/protobuf/empty.proto";
import "validate/validate.proto";

// 订单服务
service OrderService {
    // 一元 RPC
    rpc CreateOrder(CreateOrderRequest) returns (Order) {}
    rpc GetOrder(GetOrderRequest) returns (Order) {}
    rpc UpdateOrder(UpdateOrderRequest) returns (Order) {}
    rpc DeleteOrder(DeleteOrderRequest) returns (google.protobuf.Empty) {}

    // 服务端流
    rpc ListOrders(ListOrdersRequest) returns (stream Order) {}

    // 客户端流
    rpc BatchCreateOrders(stream CreateOrderRequest) returns (BatchCreateOrdersResponse) {}

    // 双向流
    rpc OrderStream(stream OrderAction) returns (stream OrderEvent) {}
}

// 订单消息
message Order {
    string order_id = 1;
    string user_id = 2;
    repeated OrderItem items = 3;
    double total_amount = 4;
    OrderStatus status = 5;
    string shipping_address = 6;
    google.protobuf.Timestamp created_at = 7;
    google.protobuf.Timestamp updated_at = 8;
}

message OrderItem {
    string product_id = 1;
    string product_name = 2;
    int32 quantity = 3;
    double unit_price = 4;
}

enum OrderStatus {
    ORDER_STATUS_UNSPECIFIED = 0;
    ORDER_STATUS_PENDING = 1;
    ORDER_STATUS_CONFIRMED = 2;
    ORDER_STATUS_SHIPPED = 3;
    ORDER_STATUS_DELIVERED = 4;
    ORDER_STATUS_CANCELLED = 5;
}

// 请求/响应消息
message CreateOrderRequest {
    string user_id = 1 [(validate.rules).string.min_len = 1];
    repeated OrderItem items = 2 [(validate.rules).repeated.min_items = 1];
    string shipping_address = 3 [(validate.rules).string.min_len = 1];
}

message GetOrderRequest {
    string order_id = 1 [(validate.rules).string.min_len = 1];
}

message UpdateOrderRequest {
    string order_id = 1;
    Order order = 2;
}

message DeleteOrderRequest {
    string order_id = 1;
}

message ListOrdersRequest {
    string user_id = 1;
    int32 page_size = 2;
    string page_token = 3;
}

message BatchCreateOrdersResponse {
    repeated Order orders = 1;
    int32 success_count = 2;
    int32 failure_count = 3;
}

message OrderAction {
    oneof action {
        CreateOrderRequest create = 1;
        GetOrderRequest get = 2;
        CancelOrderRequest cancel = 3;
    }
}

message CancelOrderRequest {
    string order_id = 1;
    string reason = 2;
}

message OrderEvent {
    oneof event {
        OrderCreated created = 1;
        OrderStatusChanged status_changed = 2;
        OrderError error = 3;
    }
}

message OrderCreated {
    Order order = 1;
}

message OrderStatusChanged {
    string order_id = 1;
    OrderStatus old_status = 2;
    OrderStatus new_status = 3;
}

message OrderError {
    string order_id = 1;
    string message = 2;
}
```

### 3.2 生成代码

```bash
# 方式1: 直接使用 protoc
protoc \
    -I proto \
    -I third_party \
    --go_out=gen --go_opt=paths=source_relative \
    --go-grpc_out=gen --go-grpc_opt=paths=source_relative \
    proto/order/order.proto

# 方式2: 使用 Makefile
```

**Makefile**:
```makefile
.PHONY: proto build run-server run-client clean

PROTO_DIR = proto
GEN_DIR = gen

proto:
	@mkdir -p $(GEN_DIR)/order
	protoc \
		-I $(PROTO_DIR) \
		-I third_party \
		--go_out=$(GEN_DIR) --go_opt=paths=source_relative \
		--go-grpc_out=$(GEN_DIR) --go-grpc_opt=paths=source_relative \
		$(PROTO_DIR)/order/order.proto

build: proto
	go build -o bin/server ./cmd/server
	go build -o bin/client ./cmd/client

run-server: build
	./bin/server

run-client: build
	./bin/client

clean:
	rm -rf $(GEN_DIR) bin
```

### 3.3 服务端实现

```go
// internal/service/order_service.go
package service

import (
    "context"
    "errors"
    "sync"
    "time"

    pb "github.com/example/grpc-demo/gen/order"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
    "google.golang.org/protobuf/types/known/timestamppb"
)

type OrderService struct {
    pb.UnimplementedOrderServiceServer
    mu     sync.RWMutex
    orders map[string]*pb.Order
}

func NewOrderService() *OrderService {
    return &OrderService{
        orders: make(map[string]*pb.Order),
    }
}

// 一元 RPC: 创建订单
func (s *OrderService) CreateOrder(ctx context.Context,
    req *pb.CreateOrderRequest) (*pb.Order, error) {

    // 验证
    if req.UserId == "" {
        return nil, status.Error(codes.InvalidArgument, "user_id is required")
    }
    if len(req.Items) == 0 {
        return nil, status.Error(codes.InvalidArgument, "at least one item is required")
    }

    // 计算总金额
    var totalAmount float64
    for _, item := range req.Items {
        totalAmount += float64(item.Quantity) * item.UnitPrice
    }

    // 创建订单
    orderID := generateOrderID()
    now := timestamppb.Now()
    order := &pb.Order{
        OrderId:         orderID,
        UserId:          req.UserId,
        Items:           req.Items,
        TotalAmount:     totalAmount,
        Status:          pb.OrderStatus_ORDER_STATUS_PENDING,
        ShippingAddress: req.ShippingAddress,
        CreatedAt:       now,
        UpdatedAt:       now,
    }

    s.mu.Lock()
    s.orders[orderID] = order
    s.mu.Unlock()

    return order, nil
}

// 一元 RPC: 获取订单
func (s *OrderService) GetOrder(ctx context.Context,
    req *pb.GetOrderRequest) (*pb.Order, error) {

    s.mu.RLock()
    order, exists := s.orders[req.OrderId]
    s.mu.RUnlock()

    if !exists {
        return nil, status.Errorf(codes.NotFound,
            "order %s not found", req.OrderId)
    }
    return order, nil
}

// 一元 RPC: 更新订单
func (s *OrderService) UpdateOrder(ctx context.Context,
    req *pb.UpdateOrderRequest) (*pb.Order, error) {

    s.mu.Lock()
    defer s.mu.Unlock()

    order, exists := s.orders[req.OrderId]
    if !exists {
        return nil, status.Errorf(codes.NotFound,
            "order %s not found", req.OrderId)
    }

    // 更新字段
    if req.Order.GetShippingAddress() != "" {
        order.ShippingAddress = req.Order.ShippingAddress
    }
    order.UpdatedAt = timestamppb.Now()

    return order, nil
}

// 一元 RPC: 删除订单
func (s *OrderService) DeleteOrder(ctx context.Context,
    req *pb.DeleteOrderRequest) (*emptypb.Empty, error) {

    s.mu.Lock()
    defer s.mu.Unlock()

    if _, exists := s.orders[req.OrderId]; !exists {
        return nil, status.Errorf(codes.NotFound,
            "order %s not found", req.OrderId)
    }

    delete(s.orders, req.OrderId)
    return &emptypb.Empty{}, nil
}

// 服务端流: 列出订单
func (s *OrderService) ListOrders(req *pb.ListOrdersRequest,
    stream pb.OrderService_ListOrdersServer) error {

    s.mu.RLock()
    defer s.mu.RUnlock()

    count := 0
    for _, order := range s.orders {
        // 按用户过滤
        if req.UserId != "" && order.UserId != req.UserId {
            continue
        }

        // 检查客户端是否已断开
        if stream.Context().Err() != nil {
            return stream.Context().Err()
        }

        if err := stream.Send(order); err != nil {
            return err
        }

        count++
        if req.PageSize > 0 && count >= int(req.PageSize) {
            break
        }
    }

    return nil
}

// 客户端流: 批量创建订单
func (s *OrderService) BatchCreateOrders(
    stream pb.OrderService_BatchCreateOrdersServer) error {

    var orders []*pb.Order
    var successCount, failureCount int32

    for {
        req, err := stream.Recv()
        if err == io.EOF {
            return stream.SendAndClose(&pb.BatchCreateOrdersResponse{
                Orders:       orders,
                SuccessCount: successCount,
                FailureCount: failureCount,
            })
        }
        if err != nil {
            return err
        }

        // 创建单个订单
        order, err := s.CreateOrder(stream.Context(), req)
        if err != nil {
            failureCount++
            continue
        }

        orders = append(orders, order)
        successCount++
    }
}

// 双向流: 订单流
func (s *OrderService) OrderStream(
    stream pb.OrderService_OrderStreamServer) error {

    for {
        action, err := stream.Recv()
        if err == io.EOF {
            return nil
        }
        if err != nil {
            return err
        }

        switch a := action.Action.(type) {
        case *pb.OrderAction_Create:
            order, err := s.CreateOrder(stream.Context(), a.Create)
            if err != nil {
                stream.Send(&pb.OrderEvent{
                    Event: &pb.OrderEvent_Error{
                        Error: &pb.OrderError{
                            Message: err.Error(),
                        },
                    },
                })
                continue
            }
            stream.Send(&pb.OrderEvent{
                Event: &pb.OrderEvent_Created{
                    Created: &pb.OrderCreated{
                        Order: order,
                    },
                },
            })

        case *pb.OrderAction_Get:
            order, err := s.GetOrder(stream.Context(), a.Get)
            if err != nil {
                stream.Send(&pb.OrderEvent{
                    Event: &pb.OrderEvent_Error{
                        Error: &pb.OrderError{
                            OrderId: a.Get.OrderId,
                            Message: err.Error(),
                        },
                    },
                })
                continue
            }
            stream.Send(&pb.OrderEvent{
                Event: &pb.OrderEvent_Created{
                    Created: &pb.OrderCreated{
                        Order: order,
                    },
                },
            })

        case *pb.OrderAction_Cancel:
            // 取消订单逻辑
            s.mu.Lock()
            order, exists := s.orders[a.Cancel.OrderId]
            if exists {
                order.Status = pb.OrderStatus_ORDER_STATUS_CANCELLED
                order.UpdatedAt = timestamppb.Now()
            }
            s.mu.Unlock()

            if !exists {
                stream.Send(&pb.OrderEvent{
                    Event: &pb.OrderEvent_Error{
                        Error: &pb.OrderError{
                            OrderId: a.Cancel.OrderId,
                            Message: "order not found",
                        },
                    },
                })
                continue
            }

            stream.Send(&pb.OrderEvent{
                Event: &pb.OrderEvent_StatusChanged{
                    StatusChanged: &pb.OrderStatusChanged{
                        OrderId:   a.Cancel.OrderId,
                        NewStatus: pb.OrderStatus_ORDER_STATUS_CANCELLED,
                    },
                },
            })
        }
    }
}

func generateOrderID() string {
    return fmt.Sprintf("ORD-%d", time.Now().UnixNano())
}
```

### 3.4 服务端启动

```go
// cmd/server/main.go
package main

import (
    "log"
    "net"
    "os"
    "os/signal"
    "syscall"

    pb "github.com/example/grpc-demo/gen/order"
    "github.com/example/grpc-demo/internal/service"
    "google.golang.org/grpc"
    "google.golang.org/grpc/reflection"
)

func main() {
    lis, err := net.Listen("tcp", ":50051")
    if err != nil {
        log.Fatalf("failed to listen: %v", err)
    }

    // 创建 gRPC 服务器
    grpcServer := grpc.NewServer(
        grpc.MaxRecvMsgSize(4 * 1024 * 1024),  // 4MB
        grpc.MaxSendMsgSize(4 * 1024 * 1024),  // 4MB
        grpc.MaxConcurrentStreams(1000),
    )

    // 注册服务
    orderService := service.NewOrderService()
    pb.RegisterOrderServiceServer(grpcServer, orderService)

    // 注册反射服务 (grpcurl 需要)
    reflection.Register(grpcServer)

    // 优雅关闭
    go func() {
        sigCh := make(chan os.Signal, 1)
        signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
        <-sigCh
        log.Println("Shutting down gracefully...")
        grpcServer.GracefulStop()
    }()

    log.Printf("Server listening on %s", lis.Addr())
    if err := grpcServer.Serve(lis); err != nil {
        log.Fatalf("failed to serve: %v", err)
    }
}
```

### 3.5 客户端实现

```go
// cmd/client/main.go
package main

import (
    "context"
    "io"
    "log"
    "time"

    pb "github.com/example/grpc-demo/gen/order"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
)

func main() {
    conn, err := grpc.Dial("localhost:50051",
        grpc.WithTransportCredentials(insecure.NewCredentials()),
        grpc.WithDefaultCallOptions(
            grpc.MaxCallRecvMsgSize(4*1024*1024),
        ),
    )
    if err != nil {
        log.Fatalf("did not connect: %v", err)
    }
    defer conn.Close()

    client := pb.NewOrderServiceClient(conn)

    // 1. 创建订单
    createOrder(client)

    // 2. 获取订单
    getOrder(client)

    // 3. 列出订单 (服务端流)
    listOrders(client)

    // 4. 批量创建 (客户端流)
    batchCreateOrders(client)
}

func createOrder(client pb.OrderServiceClient) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    resp, err := client.CreateOrder(ctx, &pb.CreateOrderRequest{
        UserId: "user-001",
        Items: []*pb.OrderItem{
            {
                ProductId:   "prod-001",
                ProductName: "Go Programming Book",
                Quantity:    2,
                UnitPrice:   49.99,
            },
            {
                ProductId:   "prod-002",
                ProductName: "gRPC T-Shirt",
                Quantity:    1,
                UnitPrice:   29.99,
            },
        },
        ShippingAddress: "123 Main St, City, Country",
    })
    if err != nil {
        log.Fatalf("CreateOrder failed: %v", err)
    }
    log.Printf("Created order: %s, total: $%.2f", resp.OrderId, resp.TotalAmount)
}

func getOrder(client pb.OrderServiceClient) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    resp, err := client.GetOrder(ctx, &pb.GetOrderRequest{
        OrderId: "ORD-123",  // 替换为实际订单 ID
    })
    if err != nil {
        log.Printf("GetOrder failed: %v", err)
        return
    }
    log.Printf("Got order: %+v", resp)
}

func listOrders(client pb.OrderServiceClient) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    stream, err := client.ListOrders(ctx, &pb.ListOrdersRequest{
        UserId:   "user-001",
        PageSize: 10,
    })
    if err != nil {
        log.Fatalf("ListOrders failed: %v", err)
    }

    for {
        order, err := stream.Recv()
        if err == io.EOF {
            break
        }
        if err != nil {
            log.Fatalf("ListOrders recv error: %v", err)
        }
        log.Printf("Order: %s, status: %v, amount: $%.2f",
            order.OrderId, order.Status, order.TotalAmount)
    }
}

func batchCreateOrders(client pb.OrderServiceClient) {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    stream, err := client.BatchCreateOrders(ctx)
    if err != nil {
        log.Fatalf("BatchCreateOrders failed: %v", err)
    }

    // 发送多个创建请求
    for i := 0; i < 5; i++ {
        err := stream.Send(&pb.CreateOrderRequest{
            UserId: fmt.Sprintf("user-%03d", i),
            Items: []*pb.OrderItem{
                {
                    ProductId:   fmt.Sprintf("prod-%03d", i),
                    ProductName: fmt.Sprintf("Product %d", i),
                    Quantity:    1,
                    UnitPrice:   9.99,
                },
            },
            ShippingAddress: "123 Main St",
        })
        if err != nil {
            log.Fatalf("BatchCreateOrders send error: %v", err)
        }
    }

    resp, err := stream.CloseAndRecv()
    if err != nil {
        log.Fatalf("BatchCreateOrders close error: %v", err)
    }
    log.Printf("Batch created: %d success, %d failure",
        resp.SuccessCount, resp.FailureCount)
}
```

---

## 4. 高级配置

### 4.1 服务端选项

```go
s := grpc.NewServer(
    // 消息大小限制
    grpc.MaxRecvMsgSize(4 * 1024 * 1024),   // 4MB (默认 4MB)
    grpc.MaxSendMsgSize(4 * 1024 * 1024),   // 4MB (默认 math.MaxInt32)

    // 并发控制
    grpc.MaxConcurrentStreams(1000),          // 最大并发流 (默认 250)
    grpc.MaxConnectionIdle(5 * time.Minute),  // 空闲连接超时

    // 连接参数
    grpc.KeepaliveParams(keepalive.ServerParameters{
        MaxConnectionIdle:     5 * time.Minute,
        MaxConnectionAge:      30 * time.Minute,
        MaxConnectionAgeGrace: 10 * time.Second,
        Time:                  30 * time.Second,
        Timeout:               10 * time.Second,
    }),

    // Keepalive 策略
    grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
        MinTime:             10 * time.Second,
        PermitWithoutStream: true,
    }),

    // 拦截器
    grpc.UnaryInterceptor(unaryInterceptor),
    grpc.StreamInterceptor(streamInterceptor),

    // TLS
    grpc.Creds(credentials.NewTLS(tlsConfig)),
)
```

### 4.2 客户端选项

```go
conn, err := grpc.Dial("localhost:50051",
    // 传输安全
    grpc.WithTransportCredentials(insecure.NewCredentials()),
    // 或 TLS:
    // grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)),

    // 消息大小
    grpc.WithDefaultCallOptions(
        grpc.MaxCallRecvMsgSize(4*1024*1024),
        grpc.MaxCallSendMsgSize(4*1024*1024),
    ),

    // Keepalive
    grpc.WithKeepaliveParams(keepalive.ClientParameters{
        Time:                30 * time.Second,
        Timeout:             10 * time.Second,
        PermitWithoutStream: true,
    }),

    // 拦截器
    grpc.WithUnaryInterceptor(unaryInterceptor),
    grpc.WithStreamInterceptor(streamInterceptor),

    // 重试策略
    grpc.WithDefaultServiceConfig(`{
        "methodConfig": [{
            "name": [{"service": "order.OrderService"}],
            "retryPolicy": {
                "MaxAttempts": 3,
                "InitialBackoff": "0.1s",
                "MaxBackoff": "1s",
                "BackoffMultiplier": 2,
                "RetryableStatusCodes": ["UNAVAILABLE"]
            }
        }]
    }`),

    // 连接池
    grpc.WithMaxMsgSize(4*1024*1024),
)
```

### 4.3 健康检查

```go
import "google.golang.org/grpc/health"
import "google.golang.org/grpc/health/grpc_health_v1"

// 服务端
grpcServer := grpc.NewServer()
healthServer := health.NewServer()
grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)

// 设置服务状态
healthServer.SetServingStatus("order.OrderService", grpc_health_v1.HealthCheckResponse_SERVING)

// 客户端健康检查
healthClient := grpc_health_v1.NewHealthClient(conn)
resp, err := healthClient.Check(ctx, &grpc_health_v1.HealthCheckRequest{
    Service: "order.OrderService",
})
```

### 4.4 服务反射

```go
import "google.golang.org/grpc/reflection"

// 服务端注册反射
grpcServer := grpc.NewServer()
reflection.Register(grpcServer)

// 使用 grpcurl 测试
// grpcurl -plaintext localhost:50051 list
// grpcurl -plaintext localhost:50051 list order.OrderService
// grpcurl -plaintext localhost:50051 describe order.OrderService
// grpcurl -plaintext -d '{"order_id":"ORD-123"}' localhost:50051 order.OrderService/GetOrder
```

---

## 5. 测试

### 5.1 单元测试服务端

```go
// internal/service/order_service_test.go
package service

import (
    "context"
    "testing"

    pb "github.com/example/grpc-demo/gen/order"
)

func TestCreateOrder(t *testing.T) {
    svc := NewOrderService()

    req := &pb.CreateOrderRequest{
        UserId: "user-001",
        Items: []*pb.OrderItem{
            {
                ProductId:   "prod-001",
                ProductName: "Test Product",
                Quantity:    2,
                UnitPrice:   19.99,
            },
        },
        ShippingAddress: "123 Test St",
    }

    order, err := svc.CreateOrder(context.Background(), req)
    if err != nil {
        t.Fatalf("CreateOrder failed: %v", err)
    }

    if order.OrderId == "" {
        t.Error("OrderId should not be empty")
    }
    if order.Status != pb.OrderStatus_ORDER_STATUS_PENDING {
        t.Errorf("expected PENDING, got %v", order.Status)
    }
    if order.TotalAmount != 39.98 {
        t.Errorf("expected 39.98, got %f", order.TotalAmount)
    }
}

func TestGetOrderNotFound(t *testing.T) {
    svc := NewOrderService()

    _, err := svc.GetOrder(context.Background(), &pb.GetOrderRequest{
        OrderId: "nonexistent",
    })
    if err == nil {
        t.Error("expected error for nonexistent order")
    }

    st, ok := status.FromError(err)
    if !ok {
        t.Fatal("expected gRPC status error")
    }
    if st.Code() != codes.NotFound {
        t.Errorf("expected NOT_FOUND, got %v", st.Code())
    }
}
```

### 5.2 集成测试

```go
// integration_test.go
package integration

import (
    "context"
    "net"
    "testing"
    "time"

    pb "github.com/example/grpc-demo/gen/order"
    "github.com/example/grpc-demo/internal/service"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
)

func TestOrderServiceIntegration(t *testing.T) {
    // 启动测试服务
    lis, err := net.Listen("tcp", "localhost:0") // 随机端口
    if err != nil {
        t.Fatalf("failed to listen: %v", err)
    }

    server := grpc.NewServer()
    pb.RegisterOrderServiceServer(server, service.NewOrderService())
    go server.Serve(lis)
    defer server.Stop()

    // 创建客户端
    conn, err := grpc.Dial(lis.Addr().String(),
        grpc.WithTransportCredentials(insecure.NewCredentials()),
    )
    if err != nil {
        t.Fatalf("failed to dial: %v", err)
    }
    defer conn.Close()

    client := pb.NewOrderServiceClient(conn)
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    // 测试创建订单
    order, err := client.CreateOrder(ctx, &pb.CreateOrderRequest{
        UserId: "test-user",
        Items: []*pb.OrderItem{
            {ProductId: "p1", Quantity: 1, UnitPrice: 10.0},
        },
        ShippingAddress: "Test Address",
    })
    if err != nil {
        t.Fatalf("CreateOrder failed: %v", err)
    }

    // 测试获取订单
    got, err := client.GetOrder(ctx, &pb.GetOrderRequest{
        OrderId: order.OrderId,
    })
    if err != nil {
        t.Fatalf("GetOrder failed: %v", err)
    }
    if got.OrderId != order.OrderId {
        t.Errorf("expected %s, got %s", order.OrderId, got.OrderId)
    }
}
```

---

## 6. 调试技巧

### 6.1 使用 grpcurl

```bash
# 列出服务
grpcurl -plaintext localhost:50051 list

# 查看服务描述
grpcurl -plaintext localhost:50051 describe order.OrderService

# 查看消息描述
grpcurl -plaintext localhost:50051 describe order.CreateOrderRequest

# 调用方法
grpcurl -plaintext \
    -d '{"user_id":"user-001","items":[{"product_id":"p1","quantity":1,"unit_price":10.0}],"shipping_address":"Test"}' \
    localhost:50051 \
    order.OrderService/CreateOrder

# 使用 proto 文件
grpcurl -plaintext -import-path proto -proto order/order.proto \
    -d '{"order_id":"ORD-123"}' \
    localhost:50051 \
    order.OrderService/GetOrder
```

### 6.2 使用 grpcui

```bash
# 启动 Web UI
grpcui -plaintext localhost:50051

# 输出: gRPC Web UI available at http://127.0.0.1:xxxxx
```

### 6.3 启用日志

```go
import "google.golang.org/grpc/grpclog"

// 设置日志级别
grpclog.SetLoggerV2(grpclog.NewLoggerV2(os.Stdout, os.Stderr, os.Stderr))

// 环境变量方式
// GRPC_GO_LOG_VERBOSITY_LEVEL=99
// GRPC_GO_LOG_SEVERITY_LEVEL=info
```

---

## 7. 总结

Go gRPC 开发核心要点:

1. **项目结构**: proto / gen / internal/service / cmd 分层
2. **代码生成**: protoc + protoc-gen-go + protoc-gen-go-grpc
3. **四种模式**: 一元、服务端流、客户端流、双向流
4. **服务端配置**: 消息大小、并发、Keepalive、拦截器
5. **客户端配置**: TLS、重试、Keepalive、连接池
6. **测试**: 单元测试 + 集成测试
7. **调试**: grpcurl、grpcui、反射服务

下一步: 学习 [gRPC Java 快速入门](05-GRPC-Java快速入门.md)
