# gRPC Java 快速入门

## 1. 环境准备

### 1.1 安装 JDK

```bash
# 需要 JDK 11+
java -version
# openjdk version "11.0.x"

# macOS
brew install openjdk@11

# Ubuntu
sudo apt install openjdk-11-jdk
```

### 1.2 安装 Maven / Gradle

```bash
# Maven
mvn --version

# 或 Gradle
gradle --version
```

---

## 2. 项目搭建

### 2.1 Maven 项目

**pom.xml**:
```xml
<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0"
         xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
         xsi:schemaLocation="http://maven.apache.org/POM/4.0.0
         http://maven.apache.org/xsd/maven-4.0.0.xsd">
    <modelVersion>4.0.0</modelVersion>

    <groupId>com.example</groupId>
    <artifactId>grpc-demo</artifactId>
    <version>1.0-SNAPSHOT</version>
    <packaging>jar</packaging>

    <properties>
        <java.version>11</java.version>
        <grpc.version>1.60.0</grpc.version>
        <protobuf.version>3.25.1</protobuf.version>
        <project.build.sourceEncoding>UTF-8</project.build.sourceEncoding>
    </properties>

    <dependencyManagement>
        <dependencies>
            <dependency>
                <groupId>io.grpc</groupId>
                <artifactId>grpc-bom</artifactId>
                <version>${grpc.version}</version>
                <type>pom</type>
                <scope>import</scope>
            </dependency>
        </dependencies>
    </dependencyManagement>

    <dependencies>
        <!-- gRPC 核心依赖 -->
        <dependency>
            <groupId>io.grpc</groupId>
            <artifactId>grpc-netty-shaded</artifactId>
            <scope>runtime</scope>
        </dependency>
        <dependency>
            <groupId>io.grpc</groupId>
            <artifactId>grpc-protobuf</artifactId>
        </dependency>
        <dependency>
            <groupId>io.grpc</groupId>
            <artifactId>grpc-stub</artifactId>
        </dependency>

        <!-- Protobuf -->
        <dependency>
            <groupId>com.google.protobuf</groupId>
            <artifactId>protobuf-java</artifactId>
            <version>${protobuf.version}</version>
        </dependency>

        <!-- javax.annotation (Java 9+) -->
        <dependency>
            <groupId>org.apache.tomcat</groupId>
            <artifactId>annotations-api</artifactId>
            <version>6.0.53</version>
            <scope>provided</scope>
        </dependency>

        <!-- JSON 支持 -->
        <dependency>
            <groupId>com.google.protobuf</groupId>
            <artifactId>protobuf-java-util</artifactId>
            <version>${protobuf.version}</version>
        </dependency>

        <!-- 测试 -->
        <dependency>
            <groupId>org.junit.jupiter</groupId>
            <artifactId>junit-jupiter</artifactId>
            <version>5.10.1</version>
            <scope>test</scope>
        </dependency>
    </dependencies>

    <build>
        <extensions>
            <extension>
                <groupId>kr.motd.maven</groupId>
                <artifactId>os-maven-plugin</artifactId>
                <version>1.7.1</version>
            </extension>
        </extensions>

        <plugins>
            <!-- Protobuf 编译插件 -->
            <plugin>
                <groupId>org.xolstice.maven.plugins</groupId>
                <artifactId>protobuf-maven-plugin</artifactId>
                <version>0.6.1</version>
                <configuration>
                    <protocArtifact>
                        com.google.protobuf:protoc:${protobuf.version}:exe:${os.detected.classifier}
                    </protocArtifact>
                    <pluginId>grpc-java</pluginId>
                    <pluginArtifact>
                        io.grpc:protoc-gen-grpc-java:${grpc.version}:exe:${os.detected.classifier}
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

            <!-- Java 编译插件 -->
            <plugin>
                <groupId>org.apache.maven.plugins</groupId>
                <artifactId>maven-compiler-plugin</artifactId>
                <version>3.11.0</version>
                <configuration>
                    <source>${java.version}</source>
                    <target>${java.version}</target>
                </configuration>
            </plugin>

            <!-- Exec 插件 (运行 main 方法) -->
            <plugin>
                <groupId>org.codehaus.mojo</groupId>
                <artifactId>exec-maven-plugin</artifactId>
                <version>3.1.0</version>
            </plugin>
        </plugins>
    </build>
</project>
```

### 2.2 Gradle 项目

**build.gradle**:
```groovy
plugins {
    id 'java'
    id 'com.google.protobuf' version '0.9.4'
}

group = 'com.example'
version = '1.0-SNAPSHOT'
sourceCompatibility = '11'

def grpcVersion = '1.60.0'
def protobufVersion = '3.25.1'

repositories {
    mavenCentral()
}

dependencies {
    // gRPC
    implementation "io.grpc:grpc-netty-shaded:${grpcVersion}"
    implementation "io.grpc:grpc-protobuf:${grpcVersion}"
    implementation "io.grpc:grpc-stub:${grpcVersion}"

    // Protobuf
    implementation "com.google.protobuf:protobuf-java:${protobufVersion}"
    implementation "com.google.protobuf:protobuf-java-util:${protobufVersion}"

    // javax.annotation
    compileOnly 'org.apache.tomcat:annotations-api:6.0.53'

    // 测试
    testImplementation 'org.junit.jupiter:junit-jupiter:5.10.1'
}

protobuf {
    protoc {
        artifact = "com.google.protobuf:protoc:${protobufVersion}"
    }
    plugins {
        grpc {
            artifact = "io.grpc:protoc-gen-grpc-java:${grpcVersion}"
        }
    }
    generateProtoTasks {
        all()*.plugins {
            grpc {}
        }
    }
}
```

### 2.3 项目结构

```
grpc-demo/
├── src/
│   ├── main/
│   │   ├── java/com/example/grpc/
│   │   │   ├── server/
│   │   │   │   └── OrderServer.java
│   │   │   ├── client/
│   │   │   │   └── OrderClient.java
│   │   │   └── service/
│   │   │       └── OrderServiceImpl.java
│   │   └── proto/
│   │       └── order.proto
│   └── test/
│       └── java/com/example/grpc/
│           └── OrderServiceTest.java
├── pom.xml
└── build.gradle
```

---

## 3. 定义 Protobuf

```protobuf
// src/main/proto/order.proto
syntax = "proto3";

package order;

option java_multiple_files = true;
option java_package = "com.example.grpc.order";
option java_outer_classname = "OrderProto";

service OrderService {
    rpc CreateOrder(CreateOrderRequest) returns (Order) {}
    rpc GetOrder(GetOrderRequest) returns (Order) {}
    rpc ListOrders(ListOrdersRequest) returns (stream Order) {}
    rpc BatchCreateOrders(stream CreateOrderRequest) returns (BatchCreateResponse) {}
}

message Order {
    string order_id = 1;
    string user_id = 2;
    repeated OrderItem items = 3;
    double total_amount = 4;
    OrderStatus status = 5;
    int64 created_at = 6;
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

message CreateOrderRequest {
    string user_id = 1;
    repeated OrderItem items = 2;
    string shipping_address = 3;
}

message GetOrderRequest {
    string order_id = 1;
}

message ListOrdersRequest {
    string user_id = 1;
    int32 page_size = 2;
}

message BatchCreateResponse {
    repeated Order orders = 1;
    int32 success_count = 2;
    int32 failure_count = 3;
}
```

生成代码:
```bash
# Maven
mvn compile

# Gradle
gradle generateProto
```

生成的文件位于 `target/generated-sources/protobuf/`:
- `OrderProto.java` - 消息类型
- `OrderServiceGrpc.java` - gRPC 桩代码

---

## 4. 服务端实现

### 4.1 服务实现类

```java
// src/main/java/com/example/grpc/service/OrderServiceImpl.java
package com.example.grpc.service;

import com.example.grpc.order.*;
import io.grpc.Status;
import io.grpc.stub.StreamObserver;

import java.util.*;
import java.util.concurrent.ConcurrentHashMap;

public class OrderServiceImpl extends OrderServiceGrpc.OrderServiceImplBase {

    private final Map<String, Order> orders = new ConcurrentHashMap<>();

    @Override
    public void createOrder(CreateOrderRequest request,
                            StreamObserver<Order> responseObserver) {
        // 验证
        if (request.getUserId().isEmpty()) {
            responseObserver.onError(Status.INVALID_ARGUMENT
                .withDescription("user_id is required")
                .asRuntimeException());
            return;
        }

        if (request.getItemsCount() == 0) {
            responseObserver.onError(Status.INVALID_ARGUMENT
                .withDescription("at least one item is required")
                .asRuntimeException());
            return;
        }

        // 计算总金额
        double totalAmount = request.getItemsList().stream()
            .mapToDouble(item -> item.getQuantity() * item.getUnitPrice())
            .sum();

        // 创建订单
        String orderId = "ORD-" + UUID.randomUUID().toString().substring(0, 8);
        Order order = Order.newBuilder()
            .setOrderId(orderId)
            .setUserId(request.getUserId())
            .addAllItems(request.getItemsList())
            .setTotalAmount(totalAmount)
            .setStatus(OrderStatus.ORDER_STATUS_PENDING)
            .setCreatedAt(System.currentTimeMillis() / 1000)
            .build();

        orders.put(orderId, order);

        responseObserver.onNext(order);
        responseObserver.onCompleted();
    }

    @Override
    public void getOrder(GetOrderRequest request,
                         StreamObserver<Order> responseObserver) {
        Order order = orders.get(request.getOrderId());
        if (order == null) {
            responseObserver.onError(Status.NOT_FOUND
                .withDescription("Order not found: " + request.getOrderId())
                .asRuntimeException());
            return;
        }
        responseObserver.onNext(order);
        responseObserver.onCompleted();
    }

    @Override
    public void listOrders(ListOrdersRequest request,
                           StreamObserver<Order> responseObserver) {
        int count = 0;
        for (Order order : orders.values()) {
            // 按用户过滤
            if (!request.getUserId().isEmpty() &&
                !order.getUserId().equals(request.getUserId())) {
                continue;
            }

            responseObserver.onNext(order);
            count++;

            if (request.getPageSize() > 0 && count >= request.getPageSize()) {
                break;
            }
        }
        responseObserver.onCompleted();
    }

    @Override
    public StreamObserver<CreateOrderRequest> batchCreateOrders(
            StreamObserver<BatchCreateResponse> responseObserver) {

        return new StreamObserver<CreateOrderRequest>() {
            private final List<Order> createdOrders = new ArrayList<>();
            private int successCount = 0;
            private int failureCount = 0;

            @Override
            public void onNext(CreateOrderRequest request) {
                try {
                    String orderId = "ORD-" + UUID.randomUUID().toString().substring(0, 8);
                    double totalAmount = request.getItemsList().stream()
                        .mapToDouble(item -> item.getQuantity() * item.getUnitPrice())
                        .sum();

                    Order order = Order.newBuilder()
                        .setOrderId(orderId)
                        .setUserId(request.getUserId())
                        .addAllItems(request.getItemsList())
                        .setTotalAmount(totalAmount)
                        .setStatus(OrderStatus.ORDER_STATUS_PENDING)
                        .setCreatedAt(System.currentTimeMillis() / 1000)
                        .build();

                    orders.put(orderId, order);
                    createdOrders.add(order);
                    successCount++;
                } catch (Exception e) {
                    failureCount++;
                }
            }

            @Override
            public void onError(Throwable t) {
                System.err.println("BatchCreateOrders error: " + t.getMessage());
            }

            @Override
            public void onCompleted() {
                responseObserver.onNext(BatchCreateResponse.newBuilder()
                    .addAllOrders(createdOrders)
                    .setSuccessCount(successCount)
                    .setFailureCount(failureCount)
                    .build());
                responseObserver.onCompleted();
            }
        };
    }
}
```

### 4.2 服务端启动

```java
// src/main/java/com/example/grpc/server/OrderServer.java
package com.example.grpc.server;

import com.example.grpc.service.OrderServiceImpl;
import io.grpc.Server;
import io.grpc.ServerBuilder;
import io.grpc.ServerInterceptors;

public class OrderServer {

    private Server server;

    public void start() throws Exception {
        int port = 50051;

        server = ServerBuilder.forPort(port)
            .addService(new OrderServiceImpl())
            // 配置选项
            .maxInboundMessageSize(4 * 1024 * 1024)  // 4MB
            .maxInboundMetadataSize(8 * 1024)         // 8KB
            .build()
            .start();

        System.out.println("Server started, listening on " + port);

        // 注册 JVM 关闭钩子
        Runtime.getRuntime().addShutdownHook(new Thread(() -> {
            System.err.println("Shutting down gRPC server...");
            server.shutdown();
            System.err.println("Server shut down.");
        }));

        server.awaitTermination();
    }

    public static void main(String[] args) throws Exception {
        new OrderServer().start();
    }
}
```

---

## 5. 客户端实现

### 5.1 完整客户端

```java
// src/main/java/com/example/grpc/client/OrderClient.java
package com.example.grpc.client;

import com.example.grpc.order.*;
import io.grpc.*;
import io.grpc.stub.StreamObserver;

import java.util.concurrent.CountDownLatch;
import java.util.concurrent.TimeUnit;

public class OrderClient {

    private final ManagedChannel channel;
    private final OrderServiceGrpc.OrderServiceBlockingStub blockingStub;
    private final OrderServiceGrpc.OrderServiceFutureStub futureStub;
    private final OrderServiceGrpc.OrderServiceStub asyncStub;

    public OrderClient(String host, int port) {
        this.channel = ManagedChannelBuilder
            .forAddress(host, port)
            .usePlaintext()
            .maxInboundMessageSize(4 * 1024 * 1024)
            .build();

        this.blockingStub = OrderServiceGrpc.newBlockingStub(channel);
        this.futureStub = OrderServiceGrpc.newFutureStub(channel);
        this.asyncStub = OrderServiceGrpc.newStub(channel);
    }

    public void shutdown() throws InterruptedException {
        channel.shutdown().awaitTermination(5, TimeUnit.SECONDS);
    }

    // ========== 一元 RPC ==========

    // 同步阻塞调用
    public void createOrder() {
        System.out.println("=== CreateOrder (Blocking) ===");

        CreateOrderRequest request = CreateOrderRequest.newBuilder()
            .setUserId("user-001")
            .addItems(OrderItem.newBuilder()
                .setProductId("prod-001")
                .setProductName("Java Programming Book")
                .setQuantity(2)
                .setUnitPrice(59.99)
                .build())
            .addItems(OrderItem.newBuilder()
                .setProductId("prod-002")
                .setProductName("gRPC T-Shirt")
                .setQuantity(1)
                .setUnitPrice(29.99)
                .build())
            .setShippingAddress("123 Main St, City")
            .build();

        try {
            Order order = blockingStub.createOrder(request);
            System.out.printf("Created: id=%s, total=$%.2f%n",
                order.getOrderId(), order.getTotalAmount());
        } catch (StatusRuntimeException e) {
            System.err.printf("RPC failed: %s%n", e.getStatus());
        }
    }

    // 异步 Future 调用
    public void createOrderFuture() {
        System.out.println("=== CreateOrder (Future) ===");

        CreateOrderRequest request = CreateOrderRequest.newBuilder()
            .setUserId("user-002")
            .addItems(OrderItem.newBuilder()
                .setProductId("prod-003")
                .setProductName("Design Patterns")
                .setQuantity(1)
                .setUnitPrice(49.99)
                .build())
            .setShippingAddress("456 Oak Ave")
            .build();

        try {
            Order order = futureStub.createOrder(request).get(5, TimeUnit.SECONDS);
            System.out.printf("Created: id=%s%n", order.getOrderId());
        } catch (Exception e) {
            System.err.printf("Future RPC failed: %s%n", e.getMessage());
        }
    }

    // 异步回调调用
    public void createOrderAsync() {
        System.out.println("=== CreateOrder (Async) ===");

        CreateOrderRequest request = CreateOrderRequest.newBuilder()
            .setUserId("user-003")
            .addItems(OrderItem.newBuilder()
                .setProductId("prod-004")
                .setProductName("Clean Code")
                .setQuantity(1)
                .setUnitPrice(39.99)
                .build())
            .setShippingAddress("789 Pine Rd")
            .build();

        asyncStub.createOrder(request, new StreamObserver<Order>() {
            @Override
            public void onNext(Order order) {
                System.out.printf("Created: id=%s%n", order.getOrderId());
            }

            @Override
            public void onError(Throwable t) {
                System.err.printf("Async RPC failed: %s%n", t.getMessage());
            }

            @Override
            public void onCompleted() {
                System.out.println("Async call completed");
            }
        });
    }

    // ========== 服务端流 ==========

    public void listOrders() {
        System.out.println("=== ListOrders (Server Streaming) ===");

        ListOrdersRequest request = ListOrdersRequest.newBuilder()
            .setUserId("user-001")
            .setPageSize(10)
            .build();

        try {
            blockingStub.listOrders(request).forEachRemaining(order -> {
                System.out.printf("Order: %s, status=%s, amount=$%.2f%n",
                    order.getOrderId(),
                    order.getStatus(),
                    order.getTotalAmount());
            });
        } catch (StatusRuntimeException e) {
            System.err.printf("ListOrders failed: %s%n", e.getStatus());
        }
    }

    // ========== 客户端流 ==========

    public void batchCreateOrders() throws InterruptedException {
        System.out.println("=== BatchCreateOrders (Client Streaming) ===");

        CountDownLatch latch = new CountDownLatch(1);

        StreamObserver<BatchCreateResponse> responseObserver =
            new StreamObserver<BatchCreateResponse>() {
                @Override
                public void onNext(BatchCreateResponse resp) {
                    System.out.printf("Batch result: %d success, %d failure%n",
                        resp.getSuccessCount(), resp.getFailureCount());
                }

                @Override
                public void onError(Throwable t) {
                    System.err.printf("BatchCreateOrders error: %s%n", t.getMessage());
                    latch.countDown();
                }

                @Override
                public void onCompleted() {
                    System.out.println("Batch create completed");
                    latch.countDown();
                }
            };

        StreamObserver<CreateOrderRequest> requestObserver =
            asyncStub.batchCreateOrders(responseObserver);

        // 发送多个创建请求
        for (int i = 0; i < 5; i++) {
            CreateOrderRequest request = CreateOrderRequest.newBuilder()
                .setUserId("user-batch-" + i)
                .addItems(OrderItem.newBuilder()
                    .setProductId("prod-batch-" + i)
                    .setProductName("Product " + i)
                    .setQuantity(1)
                    .setUnitPrice(9.99)
                    .build())
                .setShippingAddress("Batch Address " + i)
                .build();

            requestObserver.onNext(request);
        }

        requestObserver.onCompleted();
        latch.await(30, TimeUnit.SECONDS);
    }

    // ========== 带超时和元数据的调用 ==========

    public void getOrderWithMetadata(String orderId) {
        System.out.println("=== GetOrder with Metadata ===");

        // 添加元数据
        Metadata metadata = new Metadata();
        Metadata.Key<String> authKey =
            Metadata.Key.of("authorization", Metadata.ASCII_STRING_MARSHALLER);
        metadata.put(authKey, "Bearer token-123");

        Metadata.Key<String> traceKey =
            Metadata.Key.of("x-trace-id", Metadata.ASCII_STRING_MARSHALLER);
        metadata.put(traceKey, "trace-abc-123");

        // 带超时和元数据的调用
        GetOrderRequest request = GetOrderRequest.newBuilder()
            .setOrderId(orderId)
            .build();

        try {
            Order order = blockingStub
                .withInterceptors(MetadataUtils.newAttachHeadersInterceptor(metadata))
                .withDeadlineAfter(5, TimeUnit.SECONDS)
                .getOrder(request);

            System.out.printf("Got order: %s%n", order.getOrderId());
        } catch (StatusRuntimeException e) {
            if (e.getStatus().getCode() == Status.Code.DEADLINE_EXCEEDED) {
                System.err.println("Request timed out");
            } else if (e.getStatus().getCode() == Status.Code.NOT_FOUND) {
                System.err.println("Order not found");
            } else {
                System.err.printf("RPC failed: %s%n", e.getStatus());
            }
        }
    }

    public static void main(String[] args) throws Exception {
        OrderClient client = new OrderClient("localhost", 50051);

        try {
            client.createOrder();
            client.createOrderFuture();
            client.createOrderAsync();
            Thread.sleep(1000); // 等待异步调用完成
            client.listOrders();
            client.batchCreateOrders();
        } finally {
            client.shutdown();
        }
    }
}
```

---

## 6. 高级特性

### 6.1 拦截器

```java
// 服务端拦截器
public class ServerLogInterceptor implements ServerInterceptor {

    @Override
    public <ReqT, RespT> ServerCall.Listener<ReqT> interceptCall(
            ServerCall<ReqT, RespT> call,
            Metadata headers,
            ServerCallHandler<ReqT, RespT> next) {

        String method = call.getMethodDescriptor().getFullMethodName();
        String traceId = headers.get(
            Metadata.Key.of("x-trace-id", Metadata.ASCII_STRING_MARSHALLER));

        System.out.printf("[SERVER] %s traceId=%s%n", method, traceId);

        return next.startCall(call, headers);
    }
}

// 注册拦截器
Server server = ServerBuilder.forPort(port)
    .addService(ServerInterceptors.intercept(
        new OrderServiceImpl(),
        new ServerLogInterceptor()
    ))
    .build();
```

```java
// 客户端拦截器
public class ClientLogInterceptor implements ClientInterceptor {

    @Override
    public <ReqT, RespT> ClientCall<ReqT, RespT> interceptCall(
            MethodDescriptor<ReqT, RespT> method,
            CallOptions callOptions,
            Channel next) {

        System.out.printf("[CLIENT] Calling %s%n", method.getFullMethodName());

        return next.newCall(method, callOptions);
    }
}

// 注册拦截器
ManagedChannel channel = ManagedChannelBuilder
    .forAddress("localhost", 50051)
    .intercept(new ClientLogInterceptor())
    .usePlaintext()
    .build();
```

### 6.2 TLS 配置

```java
// 服务端 TLS
InputStream certChain = new FileInputStream("server.crt");
InputStream privateKey = new FileInputStream("server.key");

Server server = ServerBuilder.forPort(8443)
    .useTransportSecurity(certChain, privateKey)
    .addService(new OrderServiceImpl())
    .build();
```

```java
// 客户端 TLS
InputStream caCert = new FileInputStream("ca.crt");

ManagedChannel channel = ManagedChannelBuilder
    .forAddress("localhost", 8443)
    .useTransportSecurity()
    .build();

// 自定义 CA
SslContext sslContext = GrpcSslContexts.forClient()
    .trustManager(new FileInputStream("ca.crt"))
    .build();

ManagedChannel channel = NettyChannelBuilder
    .forAddress("localhost", 8443)
    .sslContext(sslContext)
    .build();
```

### 6.3 重试策略

```java
// 通过 Service Config 配置重试
Map<String, Object> retryPolicy = new HashMap<>();
retryPolicy.put("maxAttempts", 3D);
retryPolicy.put("initialBackoff", "0.1s");
retryPolicy.put("maxBackoff", "1s");
retryPolicy.put("backoffMultiplier", 2D);
retryPolicy.put("retryableStatusCodes", Arrays.asList("UNAVAILABLE"));

Map<String, Object> methodConfig = new HashMap<>();
methodConfig.put("name", Arrays.asList(
    Collections.singletonMap("service", "order.OrderService")
));
methodConfig.put("retryPolicy", retryPolicy);

Map<String, Object> serviceConfig = new HashMap<>();
serviceConfig.put("methodConfig", Arrays.asList(methodConfig));

ManagedChannel channel = ManagedChannelBuilder
    .forAddress("localhost", 50051)
    .defaultServiceConfig(serviceConfig)
    .enableRetry()
    .usePlaintext()
    .build();
```

### 6.4 健康检查

```java
// 服务端
import io.grpc.protobuf.services.HealthStatusManager;

HealthStatusManager healthManager = new HealthStatusManager();

Server server = ServerBuilder.forPort(port)
    .addService(new OrderServiceImpl())
    .addService(healthManager.getHealthService())
    .build();

// 设置健康状态
healthManager.setStatus("order.OrderService",
    HealthCheckResponse.ServingStatus.SERVING);
```

---

## 7. Spring Boot 集成

### 7.1 依赖

```xml
<dependency>
    <groupId>net.devh</groupId>
    <artifactId>grpc-server-spring-boot-starter</artifactId>
    <version>2.15.0.RELEASE</version>
</dependency>
<dependency>
    <groupId>net.devh</groupId>
    <artifactId>grpc-client-spring-boot-starter</artifactId>
    <version>2.15.0.RELEASE</version>
</dependency>
```

### 7.2 服务端

```java
// application.yml
// grpc:
//   server:
//     port: 9090

@GrpcService
public class OrderGrpcService extends OrderServiceGrpc.OrderServiceImplBase {

    private final OrderService orderService; // 业务 Service

    public OrderGrpcService(OrderService orderService) {
        this.orderService = orderService;
    }

    @Override
    public void createOrder(CreateOrderRequest request,
                            StreamObserver<Order> responseObserver) {
        Order order = orderService.create(request);
        responseObserver.onNext(order);
        responseObserver.onCompleted();
    }
}
```

### 7.3 客户端

```java
@GrpcClient("order-service")
private OrderServiceGrpc.OrderServiceBlockingStub orderStub;

@GetMapping("/orders/{id}")
public Order getOrder(@PathVariable String id) {
    return orderStub.getOrder(
        GetOrderRequest.newBuilder().setOrderId(id).build()
    );
}
```

---

## 8. 总结

Java gRPC 开发核心要点:

1. **构建工具**: Maven 或 Gradle，使用 protobuf 插件自动生成代码
2. **三种 Stub**: Blocking (同步)、Future (异步+Future)、Async (异步+回调)
3. **StreamObserver**: 流式 RPC 的核心接口
4. **拦截器**: ClientInterceptor / ServerInterceptor
5. **Spring Boot**: grpc-spring-boot-starter 简化集成

下一步: 学习 [gRPC Python 快速入门](06-GRPC-Python快速入门.md)
