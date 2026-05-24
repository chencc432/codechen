# gRPC 负载均衡与服务发现

## 1. 概述

gRPC 基于 HTTP/2 的长连接特性，使得负载均衡与 HTTP/1.1 有显著不同。

### 1.1 为什么 gRPC 负载均衡特殊

```
HTTP/1.1 (短连接):
┌────────┐    ┌───┐    ┌──────────┐
│ Client │───▶│ LB│───▶│ Server 1 │
│        │───▶│   │───▶│ Server 2 │
│        │───▶│   │───▶│ Server 3 │
└────────┘    └───┘    └──────────┘
每次请求可以分配到不同服务器 ✅

HTTP/2 (长连接 + 多路复用):
┌────────┐    ┌───┐    ┌──────────┐
│ Client │═══▶│ LB│═══▶│ Server 1 │ ← 所有请求都走同一连接！
│        │    │   │    │ Server 2 │ ← 永远收不到请求
│        │    │   │    │ Server 3 │ ← 永远收不到请求
└────────┘    └───┘    └──────────┘
L4 负载均衡器无法感知多路复用 ❌
```

### 1.2 gRPC 负载均衡方案

```
gRPC 负载均衡方案
├── 客户端负载均衡 (Client-side LB)
│   ├── 内置 Name Resolver + LB Policy
│   └── 自定义 Name Resolver
│
├── 代理端负载均衡 (Proxy-side LB)
│   ├── L7 代理 (Envoy, Nginx)
│   └── Service Mesh (Istio, Linkerd)
│
└── 外部负载均衡器 (External LB)
    └── xDS 协议
```

---

## 2. 客户端负载均衡

### 2.1 内置负载均衡策略

```
gRPC Go 内置策略:

1. pick_first (默认)
   - 选择第一个可用地址
   - 所有请求发往同一后端
   - 适用于不需要负载均衡的场景

2. round_robin
   - 轮询所有可用地址
   - 请求均匀分配到所有后端
   - 适用于后端性能相近的场景
```

```
pick_first:
  Client ──── Server 1 (所有请求)
             Server 2 (空闲)
             Server 3 (空闲)

round_robin:
  Client ──── Server 1 (请求1, 4, 7...)
  Client ──── Server 2 (请求2, 5, 8...)
  Client ──── Server 3 (请求3, 6, 9...)
```

### 2.2 Go 客户端负载均衡

```go
// 方式1: 使用 DNS 服务发现 + round_robin
conn, err := grpc.Dial(
    "dns:///my-service.default.svc.cluster.local:50051",
    grpc.WithDefaultServiceConfig(`{
        "loadBalancingConfig": [{"round_robin": {}}]
    }`),
    grpc.WithTransportCredentials(insecure.NewCredentials()),
)
```

```go
// 方式2: 静态地址列表 + round_robin
func main() {
    // 创建 resolver
    resolver.Register(&exampleResolverBuilder{})

    conn, err := grpc.Dial(
        "example:///my-service",
        grpc.WithDefaultServiceConfig(`{
            "loadBalancingConfig": [{"round_robin": {}}]
        }`),
        grpc.WithTransportCredentials(insecure.NewCredentials()),
    )
}

// 自定义 Resolver
type exampleResolverBuilder struct{}

func (b *exampleResolverBuilder) Build(
    target resolver.Target,
    cc resolver.ClientConn,
    opts resolver.BuildOptions) (resolver.Resolver, error) {

    r := &exampleResolver{cc: cc}
    r.start()
    return r, nil
}

func (b *exampleResolverBuilder) Scheme() string { return "example" }

type exampleResolver struct {
    cc resolver.ClientConn
}

func (r *exampleResolver) start() {
    // 解析服务地址
    addresses := []resolver.Address{
        {Addr: "localhost:50051"},
        {Addr: "localhost:50052"},
        {Addr: "localhost:50053"},
    }

    r.cc.UpdateState(resolver.State{
        Addresses: addresses,
    })
}

func (r *exampleResolver) ResolveNow(opts resolver.ResolveNowOptions) {}

func (r *exampleResolver) Close() {}
```

### 2.3 健康检查感知的负载均衡

```go
import "google.golang.org/grpc/health/grpc_health_v1"

// 服务端注册健康检查
healthServer := health.NewServer()
grpc_health_v1.RegisterHealthServer(s, healthServer)

// 设置服务健康状态
healthServer.SetServingStatus("order.OrderService",
    grpc_health_v1.HealthCheckResponse_SERVING)

// 当服务不健康时
healthServer.SetServingStatus("order.OrderService",
    grpc_health_v1.HealthCheckResponse_NOT_SERVING)

// 客户端配置健康检查
conn, err := grpc.Dial(
    "dns:///my-service:50051",
    grpc.WithDefaultServiceConfig(`{
        "loadBalancingConfig": [{"round_robin": {}}],
        "healthCheckConfig": {
            "serviceName": "order.OrderService"
        }
    }`),
    grpc.WithTransportCredentials(insecure.NewCredentials()),
)
// 客户端会自动检查后端健康状态，移除不健康的后端
```

### 2.4 自定义负载均衡策略

```go
// 实现自定义 LB 策略 (加权轮询)
type weightedRoundRobinBuilder struct{}

func (b *weightedRoundRobinBuilder) Build(cc balancer.ClientConn,
    opts balancer.BuildOptions) balancer.Balancer {

    return &weightedRoundRobin{
        cc:       cc,
        subConns: make(map[balancer.SubConn]int),
    }
}

func (b *weightedRoundRobinBuilder) Name() string {
    return "weighted_round_robin"
}

type weightedRoundRobin struct {
    cc       balancer.ClientConn
    subConns map[balancer.SubConn]int
    mu       sync.Mutex
    current  int
}

func (w *weightedRoundRobin) UpdateClientConnState(s balancer.ClientConnState) error {
    for _, addr := range s.ResolverState.Addresses {
        weight := 1 // 默认权重
        if w, ok := addr.Attributes.Value("weight").(int); ok {
            weight = w
        }

        subConn, err := w.cc.NewSubConn([]resolver.Address{addr},
            balancer.NewSubConnOptions{})
        if err != nil {
            return err
        }
        subConn.Connect()
        w.subConns[subConn] = weight
    }
    return nil
}

func (w *weightedRoundRobin) Pick(opts balancer.PickInfo) (
    balancer.PickResult, error) {

    w.mu.Lock()
    defer w.mu.Unlock()

    // 加权轮询逻辑
    var totalWeight int
    for _, weight := range w.subConns {
        totalWeight += weight
    }

    w.current = (w.current + 1) % totalWeight
    cumulative := 0
    for sc, weight := range w.subConns {
        cumulative += weight
        if w.current < cumulative {
            return balancer.PickResult{SubConn: sc}, nil
        }
    }

    return balancer.PickResult{}, balancer.ErrNoSubConnAvailable
}

// 注册自定义 LB
func init() {
    balancer.Register(&weightedRoundRobinBuilder{})
}
```

---

## 3. 服务发现

### 3.1 DNS 服务发现

```go
// Kubernetes DNS
// Service: my-service.default.svc.cluster.local
conn, err := grpc.Dial(
    "dns:///my-service.default.svc.cluster.local:50051",
    grpc.WithDefaultServiceConfig(`{
        "loadBalancingConfig": [{"round_robin": {}}]
    }`),
)

// 自定义 DNS 刷新间隔
conn, err := grpc.Dial(
    "dns:///my-service:50051",
    grpc.WithDefaultServiceConfig(`{
        "loadBalancingConfig": [{"round_robin": {}}]
    }`),
    // DNS 默认每 30 秒刷新一次
)
```

### 3.2 Consul 服务发现

```go
import "github.com/hashicorp/consul/api"

type consulResolverBuilder struct {
    client *api.Client
}

func (b *consulResolverBuilder) Build(
    target resolver.Target,
    cc resolver.ClientConn,
    opts resolver.BuildOptions) (resolver.Resolver, error) {

    r := &consulResolver{
        cc:     cc,
        client: b.client,
        target: target,
    }
    go r.watch()
    return r, nil
}

func (b *consulResolverBuilder) Scheme() string { return "consul" }

type consulResolver struct {
    cc     resolver.ClientConn
    client *api.Client
    target resolver.Target
}

func (r *consulResolver) watch() {
    serviceName := r.target.Endpoint()

    for {
        services, _, err := r.client.Health().Service(
            serviceName, "", true, nil,
        )
        if err != nil {
            r.cc.ReportError(err)
            time.Sleep(5 * time.Second)
            continue
        }

        var addresses []resolver.Address
        for _, svc := range services {
            addr := fmt.Sprintf("%s:%d",
                svc.Service.Address, svc.Service.Port)
            addresses = append(addresses, resolver.Address{Addr: addr})
        }

        r.cc.UpdateState(resolver.State{
            Addresses: addresses,
        })

        time.Sleep(10 * time.Second) // 定期刷新
    }
}

// 使用
func init() {
    consulClient, _ := api.NewClient(api.DefaultConfig())
    resolver.Register(&consulResolverBuilder{client: consulClient})
}

conn, err := grpc.Dial(
    "consul:///order-service",
    grpc.WithDefaultServiceConfig(`{
        "loadBalancingConfig": [{"round_robin": {}}]
    }`),
)
```

### 3.3 Etcd 服务发现

```go
import "go.etcd.io/etcd/client/v3"

type etcdResolverBuilder struct {
    client *clientv3.Client
}

func (b *etcdResolverBuilder) Scheme() string { return "etcd" }

func (b *etcdResolverBuilder) Build(
    target resolver.Target,
    cc resolver.ClientConn,
    opts resolver.BuildOptions) (resolver.Resolver, error) {

    r := &etcdResolver{cc: cc, client: b.client, prefix: "/services/" + target.Endpoint()}
    go r.watch()
    return r, nil
}

type etcdResolver struct {
    cc     resolver.ClientConn
    client *clientv3.Client
    prefix string
}

func (r *etcdResolver) watch() {
    // 初始获取
    resp, err := r.client.Get(context.Background(), r.prefix,
        clientv3.WithPrefix())
    if err == nil {
        r.updateAddresses(resp.Kvs)
    }

    // Watch 变更
    watchCh := r.client.Watch(context.Background(), r.prefix,
        clientv3.WithPrefix())

    for wResp := range watchCh {
        for _, ev := range wResp.Events {
            switch ev.Type {
            case clientv3.EventTypePut:
                // 服务上线
                r.resolve()
            case clientv3.EventTypeDelete:
                // 服务下线
                r.resolve()
            }
        }
    }
}

func (r *etcdResolver) resolve() {
    resp, err := r.client.Get(context.Background(), r.prefix,
        clientv3.WithPrefix())
    if err != nil {
        return
    }
    r.updateAddresses(resp.Kvs)
}

func (r *etcdResolver) updateAddresses(kvs []*mvccpb.KeyValue) {
    var addresses []resolver.Address
    for _, kv := range kvs {
        addresses = append(addresses, resolver.Address{
            Addr: string(kv.Value),
        })
    }
    r.cc.UpdateState(resolver.State{Addresses: addresses})
}

// 服务注册
func registerService(client *clientv3.Client, serviceName, addr string) {
    key := fmt.Sprintf("/services/%s/%s", serviceName, addr)
    lease, _ := client.Grant(context.Background(), 10) // 10秒 TTL
    client.Put(context.Background(), key, addr,
        clientv3.WithLease(lease.ID))

    // 保持心跳
    ch, _ := client.KeepAlive(context.Background(), lease.ID)
    go func() {
        for range ch {
            // 保持租约
        }
    }()
}
```

---

## 4. 代理端负载均衡 (L7)

### 4.1 Envoy

```
┌────────┐    ┌─────────┐    ┌──────────┐
│ Client │───▶│  Envoy  │───▶│ Server 1 │
│        │    │  (L7)   │───▶│ Server 2 │
│        │    │         │───▶│ Server 3 │
└────────┘    └─────────┘    └──────────┘

Envoy 能理解 HTTP/2 帧:
- 为每个 Stream 选择不同的后端
- 支持加权路由
- 支持熔断、重试、超时
- 支持 xDS 动态配置
```

**Envoy 配置示例**:
```yaml
static_resources:
  listeners:
    - name: grpc_listener
      address:
        socket_address:
          address: 0.0.0.0
          port_value: 15001
      filter_chains:
        - filters:
            - name: envoy.filters.network.http_connection_manager
              typed_config:
                "@type": type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager
                codec_type: AUTO
                stat_prefix: grpc
                route_config:
                  name: local_route
                  virtual_hosts:
                    - name: backend
                      domains: ["*"]
                      routes:
                        - match: { prefix: "/" }
                          route:
                            cluster: grpc_cluster
                            timeout: 15s
                            retry_policy:
                              retry_on: 5xx
                              num_retries: 3
                http_filters:
                  - name: envoy.filters.http.router

  clusters:
    - name: grpc_cluster
      connect_timeout: 5s
      type: STRICT_DNS
      lb_policy: ROUND_ROBIN
      http2_protocol_options: {}
      health_checks:
        - timeout: 3s
          interval: 10s
          unhealthy_threshold: 3
          healthy_threshold: 2
          grpc_health_check: {}
      load_assignment:
        cluster_name: grpc_cluster
        endpoints:
          - lb_endpoints:
              - endpoint:
                  address:
                    socket_address:
                      address: server1
                      port_value: 50051
              - endpoint:
                  address:
                    socket_address:
                      address: server2
                      port_value: 50051
              - endpoint:
                  address:
                    socket_address:
                      address: server3
                      port_value: 50051
```

### 4.2 Nginx

```nginx
upstream grpc_backend {
    server server1:50051;
    server server2:50051;
    server server3:50051;
}

server {
    listen 15001 http2;

    location / {
        grpc_pass grpc://grpc_backend;
        grpc_connect_timeout 5s;
        grpc_read_timeout 15s;
        grpc_send_timeout 15s;
        grpc_set_header X-Request-Id $request_id;
    }
}
```

---

## 5. Service Mesh

### 5.1 Istio

```
┌──────────────────────────────────────────┐
│               Istio Mesh                 │
│                                          │
│  ┌─────────┐          ┌─────────┐       │
│  │  Pod A  │          │  Pod B  │       │
│  │┌───────┐│          │┌───────┐│       │
│  ││App    ││          ││App    ││       │
│  │└───┬───┘│          │└───▲───┘│       │
│  │┌───▼───┐│  gRPC    │┌───┴───┐│       │
│  ││Envoy  ││──────────▶│Envoy  ││       │
│  ││Sidecar││          ││Sidecar││       │
│  └───────┘│          └───────┘│       │
│  └─────────┘          └─────────┘       │
│                                          │
│  特性:                                   │
│  - 自动 mTLS                            │
│  - 流量管理 (路由/分流/镜像)              │
│  - 可观测性 (追踪/指标/日志)              │
│  - 熔断/重试/超时                        │
│  - 无需修改应用代码                      │
└──────────────────────────────────────────┘
```

**Istio VirtualService**:
```yaml
apiVersion: networking.istio.io/v1alpha3
kind: VirtualService
metadata:
  name: order-service
spec:
  hosts:
    - order-service
  http:
    - match:
        - headers:
            x-version:
              exact: v2
      route:
        - destination:
            host: order-service
            subset: v2
    - route:
        - destination:
            host: order-service
            subset: v1
          weight: 90
        - destination:
            host: order-service
            subset: v2
          weight: 10
```

**Istio DestinationRule**:
```yaml
apiVersion: networking.istio.io/v1alpha3
kind: DestinationRule
metadata:
  name: order-service
spec:
  host: order-service
  trafficPolicy:
    connectionPool:
      tcp:
        maxConnections: 100
      http:
        h2UpgradePolicy: DEFAULT
        http1MaxPendingRequests: 100
        http2MaxRequests: 100
    outlierDetection:
      consecutive5xxErrors: 5
      interval: 30s
      baseEjectionTime: 30s
      maxEjectionPercent: 50
  subsets:
    - name: v1
      labels:
        version: v1
    - name: v2
      labels:
        version: v2
```

---

## 6. 方案对比

| 方案 | 优点 | 缺点 | 适用场景 |
|------|------|------|----------|
| 客户端 LB (DNS) | 简单，无额外组件 | DNS 缓存延迟 | Kubernetes 内部 |
| 客户端 LB (Consul/Etcd) | 实时感知 | 需要服务发现组件 | 自建基础设施 |
| L7 代理 (Envoy) | 功能丰富，HTTP/2 感知 | 额外代理层 | 需要精细控制 |
| Service Mesh | 全自动，无侵入 | 复杂度高 | 大规模微服务 |

---

## 7. 总结

1. **客户端 LB**: round_robin + DNS，最简单的方案
2. **服务发现**: DNS / Consul / Etcd / Nacos
3. **L7 代理**: Envoy 是最佳选择，理解 HTTP/2 帧
4. **Service Mesh**: Istio 提供全自动的流量管理
5. **健康检查**: 必须配合负载均衡使用

下一步: 学习 [gRPC 网关与 REST 互操作](11-GRPC网关与REST互操作.md)
