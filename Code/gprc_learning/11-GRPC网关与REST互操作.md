# gRPC 网关与 REST 互操作

## 1. 为什么需要 REST 互操作

```
问题场景:
┌──────────┐                    ┌──────────┐
│ 浏览器    │──── JSON/HTTP ────▶│ gRPC服务  │  ❌ 浏览器不支持 gRPC
└──────────┘                    └──────────┘

┌──────────┐                    ┌──────────┐
│ 移动端    │──── JSON/HTTP ────▶│ gRPC服务  │  ❌ 某些环境不支持
└──────────┘                    └──────────┘

┌──────────┐                    ┌──────────┐
│ 第三方    │──── REST API ────▶│ gRPC服务  │  ❌ 外部开发者期望 REST
└──────────┘                    └──────────┘

解决方案:
┌──────────┐    ┌──────────┐    ┌──────────┐
│ 浏览器    │───▶│gRPC-Gateway│──▶│ gRPC服务  │  ✅ 网关转换协议
└──────────┘    │ REST→gRPC │    └──────────┘
                └──────────┘
```

---

## 2. gRPC-Gateway

### 2.1 工作原理

```
                    gRPC-Gateway
┌────────┐    ┌───────────────────────┐    ┌──────────┐
│ REST   │    │                       │    │          │
│ Client │───▶│ HTTP Handler          │    │ gRPC     │
│        │JSON│  ├── 路由匹配         │gRPC│ Server   │
│        │───▶│  ├── JSON → Protobuf  │───▶│          │
│        │    │  ├── 调用 gRPC 客户端  │    │          │
│        │◀───│  ├── Protobuf → JSON  │◀───│          │
│        │JSON│  └── 返回 HTTP 响应   │gRPC│          │
└────────┘    └───────────────────────┘    └──────────┘
```

### 2.2 安装

```bash
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
```

### 2.3 Proto 定义 (添加 HTTP 注解)

```protobuf
// proto/order/order.proto
syntax = "proto3";

package order;

option go_package = "github.com/example/grpc-demo/gen/order";

import "google/api/annotations.proto";
import "google/protobuf/empty.proto";

service OrderService {
    // 创建订单
    rpc CreateOrder(CreateOrderRequest) returns (Order) {
        option (google.api.http) = {
            post: "/v1/orders"
            body: "*"
        };
    }

    // 获取订单
    rpc GetOrder(GetOrderRequest) returns (Order) {
        option (google.api.http) = {
            get: "/v1/orders/{order_id}"
        };
    }

    // 更新订单
    rpc UpdateOrder(UpdateOrderRequest) returns (Order) {
        option (google.api.http) = {
            put: "/v1/orders/{order_id}"
            body: "*"
        };
    }

    // 删除订单
    rpc DeleteOrder(DeleteOrderRequest) returns (google.protobuf.Empty) {
        option (google.api.http) = {
            delete: "/v1/orders/{order_id}"
        };
    }

    // 列出订单
    rpc ListOrders(ListOrdersRequest) returns (OrderList) {
        option (google.api.http) = {
            get: "/v1/orders"
        };
    }
}

message Order {
    string order_id = 1;
    string user_id = 2;
    repeated OrderItem items = 3;
    double total_amount = 4;
    string status = 5;
}

message OrderItem {
    string product_id = 1;
    string product_name = 2;
    int32 quantity = 3;
    double unit_price = 4;
}

message CreateOrderRequest {
    string user_id = 1;
    repeated OrderItem items = 2;
    string shipping_address = 3;
}

message GetOrderRequest {
    string order_id = 1;
}

message UpdateOrderRequest {
    string order_id = 1;
    string user_id = 2;
    repeated OrderItem items = 3;
}

message DeleteOrderRequest {
    string order_id = 1;
}

message ListOrdersRequest {
    string user_id = 1;
    int32 page_size = 2;
    string page_token = 3;
}

message OrderList {
    repeated Order orders = 1;
    string next_page_token = 2;
}
```

### 2.4 生成代码

```bash
# 需要下载 google/api/annotations.proto 和 http.proto
mkdir -p third_party/google/api
curl -o third_party/google/api/annotations.proto \
    https://raw.githubusercontent.com/googleapis/googleapis/master/google/api/annotations.proto
curl -o third_party/google/api/http.proto \
    https://raw.githubusercontent.com/googleapis/googleapis/master/google/api/http.proto

# 生成代码
protoc \
    -I proto \
    -I third_party \
    --go_out=gen --go_opt=paths=source_relative \
    --go-grpc_out=gen --go-grpc_opt=paths=source_relative \
    --grpc-gateway_out=gen --grpc-gateway_opt=paths=source_relative \
    proto/order/order.proto
```

### 2.5 启动 Gateway

```go
package main

import (
    "context"
    "log"
    "net"
    "net/http"

    "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
    pb "github.com/example/grpc-demo/gen/order"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
)

func main() {
    // 1. 启动 gRPC 服务
    go startGRPCServer()

    // 2. 启动 gRPC-Gateway
    startGateway()
}

func startGRPCServer() {
    lis, err := net.Listen("tcp", ":50051")
    if err != nil {
        log.Fatalf("failed to listen: %v", err)
    }

    s := grpc.NewServer()
    pb.RegisterOrderServiceServer(s, &orderService{})
    log.Println("gRPC server listening on :50051")
    s.Serve(lis)
}

func startGateway() {
    ctx := context.Background()
    mux := runtime.NewServeMux(
        // 自定义错误处理
        runtime.WithErrorHandler(customErrorHandler),
        // 自定义 marshaler
        runtime.WithMarshalerOption(runtime.MIMEWildcard,
            &runtime.JSONPb{
                MarshalOptions: protojson.MarshalOptions{
                    EmitUnpopulated: true,  // 输出零值字段
                    UseProtoNames:   true,  // 使用 proto 字段名 (snake_case)
                },
                UnmarshalOptions: protojson.UnmarshalOptions{
                    DiscardUnknown: true,
                },
            },
        ),
    )

    opts := []grpc.DialOption{
        grpc.WithTransportCredentials(insecure.NewCredentials()),
    }

    // 注册 Gateway
    err := pb.RegisterOrderServiceHandlerFromEndpoint(
        ctx, mux, "localhost:50051", opts,
    )
    if err != nil {
        log.Fatalf("failed to register gateway: %v", err)
    }

    log.Println("gRPC-Gateway listening on :8080")
    if err := http.ListenAndServe(":8080", mux); err != nil {
        log.Fatalf("failed to serve gateway: %v", err)
    }
}

// 自定义错误处理
func customErrorHandler(
    ctx context.Context,
    mux *runtime.ServeMux,
    marshaler runtime.Marshaler,
    w http.ResponseWriter,
    r *http.Request,
    err error) {

    // 将 gRPC 错误转换为 HTTP 错误
    st, ok := status.FromError(err)
    if !ok {
        st = status.Convert(err)
    }

    httpStatus := runtime.HTTPStatusFromCode(st.Code())

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(httpStatus)

    // 返回标准错误格式
    errResp := map[string]interface{}{
        "error": map[string]interface{}{
            "code":    httpStatus,
            "status":  st.Code().String(),
            "message": st.Message(),
            "details": st.Details(),
        },
    }

    json.NewEncoder(w).Encode(errResp)
}
```

### 2.6 同进程 vs 独立进程

```
方式1: 同进程 (推荐)
┌──────────────────────────────┐
│          Go 进程              │
│  ┌────────┐  ┌────────────┐ │
│  │gRPC    │  │Gateway     │ │
│  │Server  │◀─│(HTTP Proxy)│ │
│  │:50051  │  │:8080       │ │
│  └────────┘  └────────────┘ │
└──────────────────────────────┘

方式2: 独立进程
┌─────────────┐     ┌─────────────┐
│ Gateway 进程 │────▶│ gRPC 进程    │
│ :8080       │     │ :50051      │
└─────────────┘     └─────────────┘

方式3: 反向代理
┌─────────┐     ┌─────────┐     ┌─────────┐
│ Nginx   │────▶│Gateway  │────▶│gRPC     │
│ :443    │     │ :8080   │     │:50051   │
└─────────┘     └─────────┘     └─────────┘
```

### 2.7 测试 REST API

```bash
# 创建订单
curl -X POST http://localhost:8080/v1/orders \
    -H "Content-Type: application/json" \
    -d '{
        "user_id": "user-001",
        "items": [
            {
                "product_id": "prod-001",
                "product_name": "Go Programming",
                "quantity": 2,
                "unit_price": 49.99
            }
        ],
        "shipping_address": "123 Main St"
    }'

# 获取订单
curl http://localhost:8080/v1/orders/ORD-abc123

# 列出订单
curl "http://localhost:8080/v1/orders?user_id=user-001&page_size=10"

# 更新订单
curl -X PUT http://localhost:8080/v1/orders/ORD-abc123 \
    -H "Content-Type: application/json" \
    -d '{"user_id": "user-001", "items": [...]}'

# 删除订单
curl -X DELETE http://localhost:8080/v1/orders/ORD-abc123
```

---

## 3. OpenAPI/Swagger 文档生成

### 3.1 生成 OpenAPI 文档

```bash
# 安装插件
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest

# 生成
protoc \
    -I proto \
    -I third_party \
    --openapiv2_out=docs \
    --openapiv2_opt=logtostderr=true \
    proto/order/order.proto
```

### 3.2 Proto 中添加 OpenAPI 信息

```protobuf
syntax = "proto3";

package order;

import "google/api/annotations.proto";
import "protoc-gen-openapiv2/options/annotations.proto";

// API 级别信息
option (grpc.gateway.protoc_gen_openapiv2.options.openapiv2_swagger) = {
    info: {
        title: "Order Service API";
        version: "1.0";
        description: "订单管理服务 REST API";
        contact: {
            name: "API Support";
            email: "support@example.com";
        };
    };
    host: "api.example.com";
    base_path: "/v1";
    schemes: HTTPS;
    consumes: "application/json";
    produces: "application/json";
    security_definitions: {
        security: {
            key: "BearerAuth";
            value: {
                type: TYPE_API_KEY;
                in: IN_HEADER;
                name: "Authorization";
            }
        }
    };
    security: {
        security_requirement: {
            key: "BearerAuth";
            value: {};
        }
    };
};

service OrderService {
    rpc CreateOrder(CreateOrderRequest) returns (Order) {
        option (google.api.http) = {
            post: "/v1/orders"
            body: "*"
        };
        option (grpc.gateway.protoc_gen_openapiv2.options.openapiv2_operation) = {
            summary: "创建订单";
            description: "创建一个新的订单";
            tags: "Orders";
            security: {
                security_requirement: {
                    key: "BearerAuth";
                    value: {};
                }
            };
        };
    }
}
```

---

## 4. gRPC-Web

### 4.1 概述

gRPC-Web 允许浏览器直接调用 gRPC 服务：

```
┌──────────┐                ┌──────────┐    ┌──────────┐
│ 浏览器    │───gRPC-Web────▶│ Envoy/   │───▶│ gRPC     │
│ (JS)     │   (HTTP/1.1)   │ Gateway  │    │ Server   │
└──────────┘                └──────────┘    └──────────┘
```

### 4.2 安装 Go gRPC-Web 插件

```bash
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
```

### 4.3 使用 Envoy 作为 gRPC-Web 代理

```yaml
# envoy.yaml
static_resources:
  listeners:
    - name: listener
      address:
        socket_address:
          address: 0.0.0.0
          port_value: 8080
      filter_chains:
        - filters:
            - name: envoy.filters.network.http_connection_manager
              typed_config:
                "@type": type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager
                codec_type: AUTO
                stat_prefix: grpc_web
                route_config:
                  name: local_route
                  virtual_hosts:
                    - name: local_service
                      domains: ["*"]
                      routes:
                        - match: { prefix: "/" }
                          route:
                            cluster: grpc_backend
                            max_stream_duration:
                              grpc_timeout_header_max: 0s
                http_filters:
                  - name: envoy.filters.http.grpc_web
                  - name: envoy.filters.http.router

  clusters:
    - name: grpc_backend
      connect_timeout: 5s
      type: LOGICAL_DNS
      http2_protocol_options: {}
      lb_policy: ROUND_ROBIN
      load_assignment:
        cluster_name: grpc_backend
        endpoints:
          - lb_endpoints:
              - endpoint:
                  address:
                    socket_address:
                      address: localhost
                      port_value: 50051
```

### 4.4 前端 TypeScript 调用

```typescript
// 安装依赖
// npm install grpc-web google-protobuf

import { OrderServiceClient } from './generated/order_grpc_web_pb';
import { CreateOrderRequest, OrderItem } from './generated/order_pb';

const client = new OrderServiceClient('http://localhost:8080');

// 创建订单
const request = new CreateOrderRequest();
request.setUserId('user-001');

const item = new OrderItem();
item.setProductId('prod-001');
item.setProductName('Go Book');
item.setQuantity(2);
item.setUnitPrice(49.99);
request.addItems(item);
request.setShippingAddress('123 Main St');

client.createOrder(request, {}, (err, response) => {
    if (err) {
        console.error('Error:', err.message);
        return;
    }
    console.log('Order created:', response.getOrderId());
    console.log('Total amount:', response.getTotalAmount());
});

// 服务端流
const listRequest = new ListOrdersRequest();
listRequest.setUserId('user-001');

const stream = client.listOrders(listRequest);
stream.on('data', (response) => {
    console.log('Order:', response.getOrderId());
});
stream.on('end', () => {
    console.log('Stream ended');
});
stream.on('error', (err) => {
    console.error('Stream error:', err);
});
```

---

## 5. 手写 REST 代理 (不使用 Gateway)

### 5.1 Go HTTP 代理

```go
package main

import (
    "encoding/json"
    "io"
    "log"
    "net/http"

    pb "github.com/example/grpc-demo/gen/order"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
)

type RESTProxy struct {
    client pb.OrderServiceClient
}

func NewRESTProxy(addr string) *RESTProxy {
    conn, err := grpc.Dial(addr,
        grpc.WithTransportCredentials(insecure.NewCredentials()),
    )
    if err != nil {
        log.Fatal(err)
    }

    return &RESTProxy{
        client: pb.NewOrderServiceClient(conn),
    }
}

func (p *RESTProxy) CreateOrder(w http.ResponseWriter, r *http.Request) {
    var req struct {
        UserID          string  `json:"user_id"`
        Items           []Item  `json:"items"`
        ShippingAddress string  `json:"shipping_address"`
    }

    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // 转换为 gRPC 请求
    grpcReq := &pb.CreateOrderRequest{
        UserId:          req.UserID,
        ShippingAddress: req.ShippingAddress,
    }

    for _, item := range req.Items {
        grpcReq.Items = append(grpcReq.Items, &pb.OrderItem{
            ProductId:   item.ProductID,
            ProductName: item.ProductName,
            Quantity:    item.Quantity,
            UnitPrice:   item.UnitPrice,
        })
    }

    // 调用 gRPC
    resp, err := p.client.CreateOrder(r.Context(), grpcReq)
    if err != nil {
        handleGRPCErr(w, err)
        return
    }

    writeJSON(w, http.StatusCreated, resp)
}

func (p *RESTProxy) GetOrder(w http.ResponseWriter, r *http.Request) {
    orderID := chi.URLParam(r, "orderID")

    resp, err := p.client.GetOrder(r.Context(),
        &pb.GetOrderRequest{OrderId: orderID})
    if err != nil {
        handleGRPCErr(w, err)
        return
    }

    writeJSON(w, http.StatusOK, resp)
}

func handleGRPCErr(w http.ResponseWriter, err error) {
    st, _ := status.FromError(err)
    httpStatus := runtime.HTTPStatusFromCode(st.Code())
    writeJSON(w, httpStatus, map[string]string{
        "error":  st.Message(),
        "status": st.Code().String(),
    })
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(code)
    json.NewEncoder(w).Encode(v)
}
```

---

## 6. gRPC-Gateway vs gRPC-Web vs 手写代理

| 方案 | 协议 | 浏览器支持 | 流式支持 | 复杂度 | 适用场景 |
|------|------|-----------|----------|--------|----------|
| gRPC-Gateway | REST/JSON | 完全 | 仅一元和服务端流 | 低 | 外部 API |
| gRPC-Web | gRPC-Web | 需要代理 | 一元和服务端流 | 中 | 前端应用 |
| 手写代理 | REST/JSON | 完全 | 自定义 | 高 | 高度定制 |
| Envoy | 原生gRPC | 需要代理 | 全部 | 中 | Service Mesh |

---

## 7. 总结

1. **gRPC-Gateway**: 通过 proto 注解自动生成 REST 代理，最常用
2. **OpenAPI**: 自动生成 Swagger 文档
3. **gRPC-Web**: 让浏览器直接调用 gRPC（需代理）
4. **手写代理**: 适合高度定制场景
5. **Envoy**: 统一代理，同时支持 gRPC-Web 和负载均衡

下一步: 学习 [gRPC 生产实践与最佳实践](12-GRPC生产实践与最佳实践.md)
