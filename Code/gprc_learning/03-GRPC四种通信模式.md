# gRPC 四种通信模式

## 1. 概述

gRPC 支持四种通信模式，基于 HTTP/2 的流式能力实现：

| 模式 | 客户端 | 服务端 | 典型场景 |
|------|--------|--------|----------|
| 一元 RPC | 单个请求 | 单个响应 | CRUD 操作 |
| 服务端流 | 单个请求 | 流式响应 | 查询列表、实时推送 |
| 客户端流 | 流式请求 | 单个响应 | 文件上传、批量导入 |
| 双向流 | 流式请求 | 流式响应 | 聊天、实时协作 |

---

## 2. 一元 RPC (Unary RPC)

### 2.1 定义

```protobuf
service OrderService {
    rpc GetOrder(GetOrderRequest) returns (Order) {}
    rpc CreateOrder(CreateOrderRequest) returns (Order) {}
}

message GetOrderRequest {
    string order_id = 1;
}

message CreateOrderRequest {
    string product_id = 1;
    int32 quantity = 2;
}

message Order {
    string order_id = 1;
    string product_id = 2;
    int32 quantity = 3;
    string status = 4;
}
```

### 2.2 Go 实现

**服务端**:
```go
package main

import (
    "context"
    "log"
    "net"

    pb "example.com/orderpb"
    "google.golang.org/grpc"
)

type orderServer struct {
    pb.UnimplementedOrderServiceServer
    orders map[string]*pb.Order
}

func (s *orderServer) GetOrder(ctx context.Context, req *pb.GetOrderRequest) (*pb.Order, error) {
    order, exists := s.orders[req.OrderId]
    if !exists {
        return nil, status.Errorf(codes.NotFound, "order %s not found", req.OrderId)
    }
    return order, nil
}

func (s *orderServer) CreateOrder(ctx context.Context, req *pb.CreateOrderRequest) (*pb.Order, error) {
    orderID := uuid.New().String()
    order := &pb.Order{
        OrderId:   orderID,
        ProductId: req.ProductId,
        Quantity:  req.Quantity,
        Status:    "CREATED",
    }
    s.orders[orderID] = order
    return order, nil
}

func main() {
    lis, err := net.Listen("tcp", ":50051")
    if err != nil {
        log.Fatalf("failed to listen: %v", err)
    }

    s := grpc.NewServer()
    pb.RegisterOrderServiceServer(s, &orderServer{
        orders: make(map[string]*pb.Order),
    })

    log.Println("Server listening on :50051")
    s.Serve(lis)
}
```

**客户端**:
```go
package main

import (
    "context"
    "log"

    pb "example.com/orderpb"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
)

func main() {
    conn, err := grpc.Dial("localhost:50051",
        grpc.WithTransportCredentials(insecure.NewCredentials()),
    )
    if err != nil {
        log.Fatalf("did not connect: %v", err)
    }
    defer conn.Close()

    client := pb.NewOrderServiceClient(conn)

    // 创建订单
    createResp, err := client.CreateOrder(context.Background(), &pb.CreateOrderRequest{
        ProductId: "prod-001",
        Quantity:  3,
    })
    if err != nil {
        log.Fatalf("CreateOrder failed: %v", err)
    }
    log.Printf("Created order: %s", createResp.OrderId)

    // 查询订单
    getResp, err := client.GetOrder(context.Background(), &pb.GetOrderRequest{
        OrderId: createResp.OrderId,
    })
    if err != nil {
        log.Fatalf("GetOrder failed: %v", err)
    }
    log.Printf("Got order: %+v", getResp)
}
```

### 2.3 Java 实现

**服务端**:
```java
public class OrderServiceImpl extends OrderServiceGrpc.OrderServiceImplBase {
    private final Map<String, Order> orders = new ConcurrentHashMap<>();

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
    public void createOrder(CreateOrderRequest request,
                            StreamObserver<Order> responseObserver) {
        String orderId = UUID.randomUUID().toString();
        Order order = Order.newBuilder()
            .setOrderId(orderId)
            .setProductId(request.getProductId())
            .setQuantity(request.getQuantity())
            .setStatus("CREATED")
            .build();
        orders.put(orderId, order);
        responseObserver.onNext(order);
        responseObserver.onCompleted();
    }
}
```

**客户端**:
```java
ManagedChannel channel = ManagedChannelBuilder
    .forAddress("localhost", 50051)
    .usePlaintext()
    .build();

OrderServiceGrpc.OrderServiceBlockingStub client =
    OrderServiceGrpc.newBlockingStub(channel);

// 创建订单
Order created = client.createOrder(CreateOrderRequest.newBuilder()
    .setProductId("prod-001")
    .setQuantity(3)
    .build());

// 查询订单
Order order = client.getOrder(GetOrderRequest.newBuilder()
    .setOrderId(created.getOrderId())
    .build());
```

### 2.4 Python 实现

**服务端**:
```python
import grpc
from concurrent import futures
import order_pb2
import order_pb2_grpc
import uuid

class OrderServiceServicer(order_pb2_grpc.OrderServiceServicer):
    def __init__(self):
        self.orders = {}

    def GetOrder(self, request, context):
        order = self.orders.get(request.order_id)
        if not order:
            context.set_code(grpc.StatusCode.NOT_FOUND)
            context.set_details(f"Order {request.order_id} not found")
            return order_pb2.Order()
        return order

    def CreateOrder(self, request, context):
        order_id = str(uuid.uuid4())
        order = order_pb2.Order(
            order_id=order_id,
            product_id=request.product_id,
            quantity=request.quantity,
            status="CREATED",
        )
        self.orders[order_id] = order
        return order

def serve():
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    order_pb2_grpc.add_OrderServiceServicer_to_server(
        OrderServiceServicer(), server
    )
    server.add_insecure_port("[::]:50051")
    server.start()
    server.wait_for_termination()

if __name__ == "__main__":
    serve()
```

**客户端**:
```python
import grpc
import order_pb2
import order_pb2_grpc

def run():
    channel = grpc.insecure_channel("localhost:50051")
    stub = order_pb2_grpc.OrderServiceStub(channel)

    # 创建订单
    created = stub.CreateOrder(order_pb2.CreateOrderRequest(
        product_id="prod-001",
        quantity=3,
    ))
    print(f"Created order: {created.order_id}")

    # 查询订单
    order = stub.GetOrder(order_pb2.GetOrderRequest(
        order_id=created.order_id,
    ))
    print(f"Got order: {order}")
```

---

## 3. 服务端流式 RPC (Server Streaming RPC)

### 3.1 定义

```protobuf
service StockService {
    // 订阅股票价格变动
    rpc SubscribePrice(PriceRequest) returns (stream PriceResponse) {}

    // 查询订单历史
    rpc GetOrderHistory(OrderHistoryRequest) returns (stream Order) {}
}

message PriceRequest {
    string symbol = 1;
}

message PriceResponse {
    string symbol = 1;
    double price = 2;
    int64 timestamp = 3;
}
```

### 3.2 Go 实现

**服务端**:
```go
func (s *stockServer) SubscribePrice(req *pb.PriceRequest,
    stream pb.StockService_SubscribePriceServer) error {

    ticker := time.NewTicker(2 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-stream.Context().Done():
            // 客户端取消或断开连接
            log.Printf("Client disconnected for %s", req.Symbol)
            return stream.Context().Err()

        case t := <-ticker.C:
            price := generatePrice(req.Symbol)
            resp := &pb.PriceResponse{
                Symbol:    req.Symbol,
                Price:     price,
                Timestamp: t.Unix(),
            }
            if err := stream.Send(resp); err != nil {
                return err
            }
        }
    }
}
```

**客户端**:
```go
func subscribePrice(client pb.StockServiceClient, symbol string) {
    req := &pb.PriceRequest{Symbol: symbol}
    stream, err := client.SubscribePrice(context.Background(), req)
    if err != nil {
        log.Fatalf("SubscribePrice failed: %v", err)
    }

    for {
        resp, err := stream.Recv()
        if err == io.EOF {
            log.Println("Stream ended")
            break
        }
        if err != nil {
            log.Fatalf("Recv error: %v", err)
        }
        log.Printf("%s: $%.2f (ts: %d)", resp.Symbol, resp.Price, resp.Timestamp)
    }
}
```

### 3.3 Java 实现

**服务端**:
```java
@Override
public void subscribePrice(PriceRequest request,
                           StreamObserver<PriceResponse> responseObserver) {
    ScheduledExecutorService scheduler = Executors.newSingleThreadScheduledExecutor();
    AtomicBoolean running = new AtomicBoolean(true);

    scheduler.scheduleAtFixedRate(() -> {
        if (!running.get()) return;
        PriceResponse response = PriceResponse.newBuilder()
            .setSymbol(request.getSymbol())
            .setPrice(generatePrice(request.getSymbol()))
            .setTimestamp(System.currentTimeMillis() / 1000)
            .build();
        responseObserver.onNext(response);
    }, 0, 2, TimeUnit.SECONDS);

    // 在 onCancel 时停止
    // 注意: 需要实现 ServerStreamObserver 的 onCancel 回调
}
```

**客户端 (异步)**:
```java
StreamObserver<PriceResponse> responseObserver = new StreamObserver<PriceResponse>() {
    @Override
    public void onNext(PriceResponse response) {
        System.out.printf("%s: $%.2f%n", response.getSymbol(), response.getPrice());
    }

    @Override
    public void onError(Throwable t) {
        System.err.println("Error: " + t.getMessage());
    }

    @Override
    public void onCompleted() {
        System.out.println("Stream completed");
    }
};

asyncStub.subscribePrice(PriceRequest.newBuilder()
    .setSymbol("AAPL")
    .build(), responseObserver);
```

### 3.4 Python 实现

**服务端**:
```python
import time
import random

class StockServiceServicer(stock_pb2_grpc.StockServiceServicer):
    def SubscribePrice(self, request, context):
        symbol = request.symbol
        while context.is_active():
            price = random.uniform(100, 200)
            yield stock_pb2.PriceResponse(
                symbol=symbol,
                price=price,
                timestamp=int(time.time()),
            )
            time.sleep(2)
```

**客户端**:
```python
def subscribe_price(stub, symbol):
    request = stock_pb2.PriceRequest(symbol=symbol)
    responses = stub.SubscribePrice(request)
    for response in responses:
        print(f"{response.symbol}: ${response.price:.2f}")
```

---

## 4. 客户端流式 RPC (Client Streaming RPC)

### 4.1 定义

```protobuf
service FileService {
    // 上传文件 (分块上传)
    rpc UploadFile(stream FileChunk) returns (UploadStatus) {}

    // 批量导入用户
    rpc ImportUsers(stream User) returns (ImportResult) {}
}

message FileChunk {
    string file_name = 1;
    int64 offset = 2;
    bytes data = 3;
}

message UploadStatus {
    bool success = 1;
    string message = 2;
    int64 total_bytes = 3;
}
```

### 4.2 Go 实现

**服务端**:
```go
func (s *fileServer) UploadFile(stream pb.FileService_UploadFileServer) error {
    var totalBytes int64
    var fileName string
    var file *os.File

    for {
        chunk, err := stream.Recv()
        if err == io.EOF {
            // 客户端发送完毕，返回结果
            if file != nil {
                file.Close()
            }
            return stream.SendAndClose(&pb.UploadStatus{
                Success:    true,
                Message:    "Upload complete",
                TotalBytes: totalBytes,
            })
        }
        if err != nil {
            return err
        }

        // 首个 chunk 获取文件名并创建文件
        if file == nil {
            fileName = chunk.GetFileName()
            file, err = os.Create("/tmp/" + fileName)
            if err != nil {
                return status.Errorf(codes.Internal, "create file failed: %v", err)
            }
            defer file.Close()
        }

        // 写入数据
        n, err := file.Write(chunk.GetData())
        if err != nil {
            return status.Errorf(codes.Internal, "write failed: %v", err)
        }
        totalBytes += int64(n)
    }
}
```

**客户端**:
```go
func uploadFile(client pb.FileServiceClient, filePath string) error {
    stream, err := client.UploadFile(context.Background())
    if err != nil {
        return err
    }

    file, err := os.Open(filePath)
    if err != nil {
        return err
    }
    defer file.Close()

    buf := make([]byte, 64*1024) // 64KB chunks
    fileName := filepath.Base(filePath)

    for {
        n, err := file.Read(buf)
        if err == io.EOF {
            break
        }
        if err != nil {
            return err
        }

        chunk := &pb.FileChunk{
            FileName: fileName,
            Data:     buf[:n],
        }
        if err := stream.Send(chunk); err != nil {
            return err
        }
    }

    status, err := stream.CloseAndRecv()
    if err != nil {
        return err
    }

    log.Printf("Upload: success=%v, bytes=%d, msg=%s",
        status.Success, status.TotalBytes, status.Message)
    return nil
}
```

### 4.3 Java 实现

**服务端**:
```java
@Override
public StreamObserver<FileChunk> uploadFile(
        StreamObserver<UploadStatus> responseObserver) {

    return new StreamObserver<FileChunk>() {
        private long totalBytes = 0;
        private String fileName;
        private FileOutputStream fos;

        @Override
        public void onNext(FileChunk chunk) {
            try {
                if (fos == null) {
                    fileName = chunk.getFileName();
                    fos = new FileOutputStream("/tmp/" + fileName);
                }
                fos.write(chunk.getData().toByteArray());
                totalBytes += chunk.getData().size();
            } catch (IOException e) {
                responseObserver.onError(Status.INTERNAL
                    .withDescription(e.getMessage())
                    .asRuntimeException());
            }
        }

        @Override
        public void onError(Throwable t) {
            if (fos != null) {
                try { fos.close(); } catch (IOException ignored) {}
            }
        }

        @Override
        public void onCompleted() {
            if (fos != null) {
                try { fos.close(); } catch (IOException ignored) {}
            }
            responseObserver.onNext(UploadStatus.newBuilder()
                .setSuccess(true)
                .setMessage("Upload complete")
                .setTotalBytes(totalBytes)
                .build());
            responseObserver.onCompleted();
        }
    };
}
```

**客户端**:
```java
StreamObserver<FileChunk> requestObserver = asyncStub.uploadFile(
    new StreamObserver<UploadStatus>() {
        @Override
        public void onNext(UploadStatus status) {
            System.out.printf("Upload: success=%b, bytes=%d%n",
                status.getSuccess(), status.getTotalBytes());
        }

        @Override
        public void onError(Throwable t) {
            System.err.println("Upload failed: " + t.getMessage());
        }

        @Override
        public void onCompleted() {
            System.out.println("Upload completed");
        }
    }
);

// 发送文件块
byte[] buffer = new byte[64 * 1024];
FileInputStream fis = new FileInputStream(filePath);
String fileName = new File(filePath).getName();
int bytesRead;
while ((bytesRead = fis.read(buffer)) != -1) {
    requestObserver.onNext(FileChunk.newBuilder()
        .setFileName(fileName)
        .setData(ByteString.copyFrom(buffer, 0, bytesRead))
        .build());
}
fis.close();
requestObserver.onCompleted();
```

### 4.4 Python 实现

**服务端**:
```python
class FileServiceServicer(file_pb2_grpc.FileServiceServicer):
    def UploadFile(self, request_iterator, context):
        total_bytes = 0
        file_handle = None
        file_name = None

        for chunk in request_iterator:
            if file_handle is None:
                file_name = chunk.file_name
                file_handle = open(f"/tmp/{file_name}", "wb")
            file_handle.write(chunk.data)
            total_bytes += len(chunk.data)

        if file_handle:
            file_handle.close()

        return file_pb2.UploadStatus(
            success=True,
            message="Upload complete",
            total_bytes=total_bytes,
        )
```

**客户端**:
```python
def upload_file(stub, file_path):
    def chunk_generator():
        file_name = os.path.basename(file_path)
        with open(file_path, "rb") as f:
            while True:
                chunk_data = f.read(64 * 1024)
                if not chunk_data:
                    break
                yield file_pb2.FileChunk(
                    file_name=file_name,
                    data=chunk_data,
                )

    status = stub.UploadFile(chunk_generator())
    print(f"Upload: success={status.success}, bytes={status.total_bytes}")
```

---

## 5. 双向流式 RPC (Bidirectional Streaming RPC)

### 5.1 定义

```protobuf
service ChatService {
    // 实时聊天
    rpc Chat(stream ChatMessage) returns (stream ChatMessage) {}
}

message ChatMessage {
    string user = 1;
    string text = 2;
    int64 timestamp = 3;
}
```

### 5.2 Go 实现

**服务端**:
```go
type chatServer struct {
    pb.UnimplementedChatServiceServer
    mu      sync.Mutex
    clients map[string]pb.ChatService_ChatServer
}

func (s *chatServer) Chat(stream pb.ChatService_ChatServer) error {
    // 注册客户端
    clientID := uuid.New().String()
    s.registerClient(clientID, stream)
    defer s.unregisterClient(clientID)

    // 接收客户端消息并广播
    for {
        msg, err := stream.Recv()
        if err == io.EOF {
            return nil
        }
        if err != nil {
            return err
        }

        // 广播给所有客户端
        s.broadcast(msg)
    }
}

func (s *chatServer) registerClient(id string, stream pb.ChatService_ChatServer) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.clients[id] = stream
}

func (s *chatServer) unregisterClient(id string) {
    s.mu.Lock()
    defer s.mu.Unlock()
    delete(s.clients, id)
}

func (s *chatServer) broadcast(msg *pb.ChatMessage) {
    s.mu.Lock()
    defer s.mu.Unlock()

    for id, stream := range s.clients {
        if err := stream.Send(msg); err != nil {
            log.Printf("Failed to send to %s: %v", id, err)
            delete(s.clients, id)
        }
    }
}
```

**客户端**:
```go
func chat(client pb.ChatServiceClient, userName string) {
    stream, err := client.Chat(context.Background())
    if err != nil {
        log.Fatalf("Chat failed: %v", err)
    }

    // 接收消息的 goroutine
    go func() {
        for {
            msg, err := stream.Recv()
            if err == io.EOF {
                return
            }
            if err != nil {
                log.Fatalf("Recv error: %v", err)
            }
            log.Printf("[%s]: %s", msg.User, msg.Text)
        }
    }()

    // 发送消息
    scanner := bufio.NewScanner(os.Stdin)
    for scanner.Scan() {
        text := scanner.Text()
        msg := &pb.ChatMessage{
            User:      userName,
            Text:      text,
            Timestamp: time.Now().Unix(),
        }
        if err := stream.Send(msg); err != nil {
            log.Fatalf("Send error: %v", err)
        }
    }

    stream.CloseSend()
}
```

### 5.3 Java 实现

**服务端**:
```java
public class ChatServiceImpl extends ChatServiceGrpc.ChatServiceImplBase {
    private final Set<StreamObserver<ChatMessage>> clients =
        ConcurrentHashMap.newKeySet();

    @Override
    public StreamObserver<ChatMessage> chat(
            StreamObserver<ChatMessage> responseObserver) {
        clients.add(responseObserver);

        return new StreamObserver<ChatMessage>() {
            @Override
            public void onNext(ChatMessage msg) {
                // 广播给所有客户端
                for (StreamObserver<ChatMessage> client : clients) {
                    client.onNext(msg);
                }
            }

            @Override
            public void onError(Throwable t) {
                clients.remove(responseObserver);
            }

            @Override
            public void onCompleted() {
                clients.remove(responseObserver);
                responseObserver.onCompleted();
            }
        };
    }
}
```

**客户端**:
```java
StreamObserver<ChatMessage> requestObserver = asyncStub.chat(
    new StreamObserver<ChatMessage>() {
        @Override
        public void onNext(ChatMessage msg) {
            System.out.printf("[%s]: %s%n", msg.getUser(), msg.getText());
        }

        @Override
        public void onError(Throwable t) {
            t.printStackTrace();
        }

        @Override
        public void onCompleted() {
            System.out.println("Chat ended");
        }
    }
);

// 从控制台读取输入并发送
Scanner scanner = new Scanner(System.in);
while (scanner.hasNextLine()) {
    String text = scanner.nextLine();
    requestObserver.onNext(ChatMessage.newBuilder()
        .setUser(userName)
        .setText(text)
        .setTimestamp(System.currentTimeMillis() / 1000)
        .build());
}
requestObserver.onCompleted();
```

### 5.4 Python 实现

**服务端**:
```python
import threading

class ChatServiceServicer(chat_pb2_grpc.ChatServiceServicer):
    def __init__(self):
        self.clients = []
        self.lock = threading.Lock()

    def Chat(self, request_iterator, context):
        # 为每个客户端创建一个队列
        queue = queue.Queue()
        with self.lock:
            self.clients.append(queue)

        def listen_requests():
            try:
                for msg in request_iterator:
                    # 广播给所有客户端
                    with self.lock:
                        for q in self.clients:
                            q.put(msg)
            finally:
                with self.lock:
                    self.clients.remove(queue)

        # 启动接收线程
        thread = threading.Thread(target=listen_requests, daemon=True)
        thread.start()

        # 从队列发送消息给当前客户端
        while context.is_active():
            try:
                msg = queue.get(timeout=1)
                yield msg
            except queue.Empty:
                continue
```

**客户端**:
```python
def chat(stub, user_name):
    def request_generator():
        while True:
            text = input("> ")
            if text.lower() == "quit":
                break
            yield chat_pb2.ChatMessage(
                user=user_name,
                text=text,
                timestamp=int(time.time()),
            )

    responses = stub.Chat(request_generator())
    for msg in responses:
        print(f"[{msg.user}]: {msg.text}")
```

---

## 6. 流式 RPC 的流控 (Flow Control)

### 6.1 背压机制

HTTP/2 提供了内置的流控机制，防止发送方淹没接收方：

```
┌──────────┐                      ┌──────────┐
│  发送方   │                      │  接收方   │
│          │  DATA (1000 bytes)   │          │
│          │─────────────────────▶│          │
│          │                      │  剩余窗口: 50000
│          │                      │  - 1000
│          │                      │  = 49000
│          │                      │          │
│          │  WINDOW_UPDATE       │          │
│          │  (increment=1000)    │          │
│          │◀─────────────────────│          │
│          │                      │          │
│  窗口增大 │                      │          │
│  继续发送 │                      │          │
└──────────┘                      └──────────┘
```

### 6.2 Go 中的流控

```go
// 服务端: 控制发送速率
func (s *server) StreamData(req *pb.Request,
    stream pb.Service_StreamDataServer) error {

    for _, item := range largeDataset {
        // 发送前检查上下文是否已取消
        if stream.Context().Err() != nil {
            return stream.Context().Err()
        }

        if err := stream.Send(item); err != nil {
            return err  // 可能是背压导致的阻塞
        }

        // 主动控制发送速率
        time.Sleep(10 * time.Millisecond)
    }
    return nil
}
```

### 6.3 客户端取消流

```go
// 使用 WithCancel 取消流式 RPC
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

stream, err := client.SubscribePrice(ctx, &pb.PriceRequest{Symbol: "AAPL"})
if err != nil {
    log.Fatal(err)
}

// 10秒后取消
go func() {
    time.Sleep(10 * time.Second)
    cancel()
}()

for {
    resp, err := stream.Recv()
    if err == io.EOF || err == context.Canceled {
        log.Println("Stream cancelled")
        break
    }
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Price: %v", resp.Price)
}
```

---

## 7. 模式选择指南

### 7.1 决策树

```
需要流式通信吗？
├── 否 → 一元 RPC
│   └── 最简单、最常用
│
└── 是 → 哪一方需要流？
    ├── 只有服务端 → 服务端流
    │   ├── 实时数据推送
    │   ├── 大结果集分批返回
    │   └── 日志/事件流
    │
    ├── 只有客户端 → 客户端流
    │   ├── 文件上传
    │   ├── 批量数据导入
    │   └── 传感器数据采集
    │
    └── 双方都需要 → 双向流
        ├── 聊天应用
        ├── 实时协作
        └── 交互式查询
```

### 7.2 性能考虑

| 因素 | 一元 | 服务端流 | 客户端流 | 双向流 |
|------|------|----------|----------|--------|
| 连接开销 | 每次新建/复用 | 一次建立 | 一次建立 | 一次建立 |
| 内存占用 | 低 | 中 | 中 | 中 |
| 实现复杂度 | 低 | 中 | 中 | 高 |
| 调试难度 | 低 | 中 | 中 | 高 |
| 负载均衡 | 容易 | 长连接困难 | 长连接困难 | 长连接困难 |

---

## 8. 总结

1. **一元 RPC**: 最简单，适合请求-响应模式，如 CRUD 操作
2. **服务端流**: 适合推送场景，如实时数据、大结果集
3. **客户端流**: 适合上传场景，如文件传输、批量导入
4. **双向流**: 适合交互场景，如聊天、实时协作
5. **流控**: HTTP/2 提供背压机制，防止数据淹没
6. **取消**: 使用 Context 随时取消流式 RPC

下一步: 学习 [gRPC Go 快速入门](04-GRPC-Go快速入门.md)
