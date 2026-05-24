# gRPC Python 快速入门

## 1. 环境准备

### 1.1 安装 Python

```bash
# 需要 Python 3.8+
python --version
# Python 3.11.x

# 创建虚拟环境
python -m venv venv
source venv/bin/activate  # Linux/macOS
venv\Scripts\activate     # Windows
```

### 1.2 安装 gRPC 依赖

```bash
pip install grpcio grpcio-tools
pip install protobuf

# 验证
python -c "import grpc; print(grpc.__version__)"
```

---

## 2. 项目结构

```
grpc-demo/
├── proto/
│   └── order.proto
├── generated/
│   ├── __init__.py
│   ├── order_pb2.py          # 消息类型
│   └── order_pb2_grpc.py     # gRPC 服务
├── server/
│   ├── __init__.py
│   └── order_server.py
├── client/
│   ├── __init__.py
│   └── order_client.py
├── tests/
│   └── test_order_service.py
├── requirements.txt
└── Makefile
```

### 2.1 requirements.txt

```
grpcio>=1.60.0
grpcio-tools>=1.60.0
protobuf>=4.25.0
```

---

## 3. 定义 Protobuf 并生成代码

### 3.1 Proto 定义

```protobuf
// proto/order.proto
syntax = "proto3";

package order;

service OrderService {
    rpc CreateOrder(CreateOrderRequest) returns (Order) {}
    rpc GetOrder(GetOrderRequest) returns (Order) {}
    rpc ListOrders(ListOrdersRequest) returns (stream Order) {}
    rpc BatchCreateOrders(stream CreateOrderRequest) returns (BatchCreateResponse) {}
    rpc OrderChat(stream ChatMessage) returns (stream ChatMessage) {}
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

message ChatMessage {
    string user = 1;
    string text = 2;
    int64 timestamp = 3;
}
```

### 3.2 生成代码

```bash
python -m grpc_tools.protoc \
    -I proto \
    --python_out=generated \
    --grpc_python_out=generated \
    proto/order.proto
```

**Makefile**:
```makefile
.PHONY: proto server client test

proto:
	python -m grpc_tools.protoc \
		-I proto \
		--python_out=generated \
		--grpc_python_out=generated \
		proto/order.proto

server:
	python server/order_server.py

client:
	python client/order_client.py

test:
	pytest tests/ -v
```

---

## 4. 服务端实现

### 4.1 完整服务端

```python
# server/order_server.py
import grpc
from concurrent import futures
import time
import uuid
import logging
from generated import order_pb2, order_pb2_grpc

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


class OrderServiceServicer(order_pb2_grpc.OrderServiceServicer):
    """订单服务实现"""

    def __init__(self):
        self.orders = {}  # order_id -> Order

    # ========== 一元 RPC ==========

    def CreateOrder(self, request, context):
        """创建订单"""
        # 验证
        if not request.user_id:
            context.set_code(grpc.StatusCode.INVALID_ARGUMENT)
            context.set_details("user_id is required")
            return order_pb2.Order()

        if not request.items:
            context.set_code(grpc.StatusCode.INVALID_ARGUMENT)
            context.set_details("at least one item is required")
            return order_pb2.Order()

        # 计算总金额
        total_amount = sum(
            item.quantity * item.unit_price for item in request.items
        )

        # 创建订单
        order_id = f"ORD-{uuid.uuid4().hex[:8]}"
        order = order_pb2.Order(
            order_id=order_id,
            user_id=request.user_id,
            items=request.items,
            total_amount=total_amount,
            status=order_pb2.ORDER_STATUS_PENDING,
            created_at=int(time.time()),
        )

        self.orders[order_id] = order
        logger.info(f"Created order: {order_id}")
        return order

    def GetOrder(self, request, context):
        """获取订单"""
        order = self.orders.get(request.order_id)
        if not order:
            context.set_code(grpc.StatusCode.NOT_FOUND)
            context.set_details(f"Order {request.order_id} not found")
            return order_pb2.Order()
        return order

    # ========== 服务端流 ==========

    def ListOrders(self, request, context):
        """列出订单 (服务端流)"""
        count = 0
        for order in self.orders.values():
            # 按用户过滤
            if request.user_id and order.user_id != request.user_id:
                continue

            # 检查客户端是否还在线
            if not context.is_active():
                logger.info("Client disconnected")
                return

            yield order
            count += 1

            if request.page_size > 0 and count >= request.page_size:
                break

    # ========== 客户端流 ==========

    def BatchCreateOrders(self, request_iterator, context):
        """批量创建订单 (客户端流)"""
        created_orders = []
        success_count = 0
        failure_count = 0

        for request in request_iterator:
            try:
                order = self.CreateOrder(request, context)
                if order.order_id:  # 创建成功
                    created_orders.append(order)
                    success_count += 1
                else:
                    failure_count += 1
            except Exception as e:
                logger.error(f"Batch create error: {e}")
                failure_count += 1

        return order_pb2.BatchCreateResponse(
            orders=created_orders,
            success_count=success_count,
            failure_count=failure_count,
        )

    # ========== 双向流 ==========

    def OrderChat(self, request_iterator, context):
        """订单聊天 (双向流)"""
        for message in request_iterator:
            if not context.is_active():
                return

            # 处理消息并返回响应
            yield order_pb2.ChatMessage(
                user="Server",
                text=f"Received from {message.user}: {message.text}",
                timestamp=int(time.time()),
            )


def serve():
    """启动 gRPC 服务器"""
    # 创建服务器
    server = grpc.server(
        futures.ThreadPoolExecutor(max_workers=10),
        options=[
            ("grpc.max_receive_message_length", 4 * 1024 * 1024),  # 4MB
            ("grpc.max_send_message_length", 4 * 1024 * 1024),     # 4MB
        ],
    )

    # 注册服务
    order_pb2_grpc.add_OrderServiceServicer_to_server(
        OrderServiceServicer(), server
    )

    # 监听端口
    server.add_insecure_port("[::]:50051")

    # 启动
    server.start()
    logger.info("Server started on port 50051")

    # 等待终止
    try:
        server.wait_for_termination()
    except KeyboardInterrupt:
        logger.info("Shutting down...")
        server.stop(grace=5)


if __name__ == "__main__":
    serve()
```

### 4.2 异步服务端 (asyncio)

```python
# server/order_server_async.py
import grpc
from grpc import aio
import time
import uuid
import logging
import asyncio
from generated import order_pb2, order_pb2_grpc

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


class OrderServiceAsyncServicer(order_pb2_grpc.OrderServiceServicer):
    """异步订单服务实现"""

    def __init__(self):
        self.orders = {}

    async def CreateOrder(self, request, context):
        """异步创建订单"""
        if not request.user_id:
            await context.abort(
                grpc.StatusCode.INVALID_ARGUMENT,
                "user_id is required",
            )

        total_amount = sum(
            item.quantity * item.unit_price for item in request.items
        )

        order_id = f"ORD-{uuid.uuid4().hex[:8]}"
        order = order_pb2.Order(
            order_id=order_id,
            user_id=request.user_id,
            items=request.items,
            total_amount=total_amount,
            status=order_pb2.ORDER_STATUS_PENDING,
            created_at=int(time.time()),
        )

        self.orders[order_id] = order
        return order

    async def GetOrder(self, request, context):
        """异步获取订单"""
        order = self.orders.get(request.order_id)
        if not order:
            await context.abort(
                grpc.StatusCode.NOT_FOUND,
                f"Order {request.order_id} not found",
            )
        return order

    async def ListOrders(self, request, context):
        """异步列出订单"""
        count = 0
        for order in self.orders.values():
            if request.user_id and order.user_id != request.user_id:
                continue
            yield order
            count += 1
            if request.page_size > 0 and count >= request.page_size:
                break


async def serve():
    """启动异步 gRPC 服务器"""
    server = aio.server(
        futures.ThreadPoolExecutor(max_workers=10),
        options=[
            ("grpc.max_receive_message_length", 4 * 1024 * 1024),
        ],
    )

    order_pb2_grpc.add_OrderServiceServicer_to_server(
        OrderServiceAsyncServicer(), server
    )

    server.add_insecure_port("[::]:50051")
    await server.start()
    logger.info("Async server started on port 50051")

    try:
        await server.wait_for_termination()
    except KeyboardInterrupt:
        await server.stop(grace=5)


if __name__ == "__main__":
    asyncio.run(serve())
```

---

## 5. 客户端实现

### 5.1 完整客户端

```python
# client/order_client.py
import grpc
import time
import logging
from generated import order_pb2, order_pb2_grpc

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


class OrderClient:
    """gRPC 订单客户端"""

    def __init__(self, host="localhost", port=50051):
        self.channel = grpc.insecure_channel(
            f"{host}:{port}",
            options=[
                ("grpc.max_receive_message_length", 4 * 1024 * 1024),
            ],
        )
        self.stub = order_pb2_grpc.OrderServiceStub(self.channel)

    def close(self):
        self.channel.close()

    # ========== 一元 RPC ==========

    def create_order(self, user_id, items, shipping_address=""):
        """创建订单"""
        request = order_pb2.CreateOrderRequest(
            user_id=user_id,
            items=items,
            shipping_address=shipping_address,
        )

        try:
            order = self.stub.CreateOrder(request)
            logger.info(f"Created order: {order.order_id}, total: ${order.total_amount:.2f}")
            return order
        except grpc.RpcError as e:
            logger.error(f"CreateOrder failed: {e.code()} - {e.details()}")
            return None

    def get_order(self, order_id):
        """获取订单"""
        request = order_pb2.GetOrderRequest(order_id=order_id)

        try:
            order = self.stub.GetOrder(request, timeout=5)
            logger.info(f"Got order: {order.order_id}, status: {order.status}")
            return order
        except grpc.RpcError as e:
            if e.code() == grpc.StatusCode.NOT_FOUND:
                logger.warning(f"Order {order_id} not found")
            else:
                logger.error(f"GetOrder failed: {e.code()} - {e.details()}")
            return None

    # ========== 服务端流 ==========

    def list_orders(self, user_id="", page_size=0):
        """列出订单 (服务端流)"""
        request = order_pb2.ListOrdersRequest(
            user_id=user_id,
            page_size=page_size,
        )

        try:
            for order in self.stub.ListOrders(request):
                logger.info(f"Order: {order.order_id}, "
                          f"status={order.status}, "
                          f"amount=${order.total_amount:.2f}")
        except grpc.RpcError as e:
            logger.error(f"ListOrders failed: {e.code()} - {e.details()}")

    # ========== 客户端流 ==========

    def batch_create_orders(self, requests):
        """批量创建订单 (客户端流)"""
        def request_generator():
            for req in requests:
                yield req

        try:
            response = self.stub.BatchCreateOrders(request_generator())
            logger.info(f"Batch result: {response.success_count} success, "
                       f"{response.failure_count} failure")
            return response
        except grpc.RpcError as e:
            logger.error(f"BatchCreateOrders failed: {e.code()} - {e.details()}")
            return None

    # ========== 双向流 ==========

    def order_chat(self, user_name, messages):
        """订单聊天 (双向流)"""
        def request_generator():
            for text in messages:
                yield order_pb2.ChatMessage(
                    user=user_name,
                    text=text,
                    timestamp=int(time.time()),
                )

        try:
            for response in self.stub.OrderChat(request_generator()):
                logger.info(f"[{response.user}]: {response.text}")
        except grpc.RpcError as e:
            logger.error(f"OrderChat failed: {e.code()} - {e.details()}")

    # ========== 带元数据的调用 ==========

    def get_order_with_metadata(self, order_id):
        """带元数据的调用"""
        metadata = (
            ("authorization", "Bearer token-123"),
            ("x-trace-id", "trace-abc-123"),
        )

        request = order_pb2.GetOrderRequest(order_id=order_id)

        try:
            order = self.stub.GetOrder(
                request,
                metadata=metadata,
                timeout=5,
            )
            return order
        except grpc.RpcError as e:
            logger.error(f"GetOrder failed: {e.code()} - {e.details()}")
            return None


def main():
    client = OrderClient()

    try:
        # 1. 创建订单
        items = [
            order_pb2.OrderItem(
                product_id="prod-001",
                product_name="Python Programming",
                quantity=2,
                unit_price=49.99,
            ),
            order_pb2.OrderItem(
                product_id="prod-002",
                product_name="gRPC Guide",
                quantity=1,
                unit_price=29.99,
            ),
        ]

        order = client.create_order(
            user_id="user-001",
            items=items,
            shipping_address="123 Main St",
        )

        if order:
            # 2. 获取订单
            client.get_order(order.order_id)

            # 3. 列出订单
            client.list_orders(user_id="user-001")

        # 4. 批量创建
        batch_requests = [
            order_pb2.CreateOrderRequest(
                user_id=f"user-batch-{i}",
                items=[order_pb2.OrderItem(
                    product_id=f"prod-batch-{i}",
                    product_name=f"Product {i}",
                    quantity=1,
                    unit_price=9.99,
                )],
                shipping_address="Batch Address",
            )
            for i in range(5)
        ]
        client.batch_create_orders(batch_requests)

        # 5. 聊天
        client.order_chat("Alice", ["Hello", "How are you?", "Goodbye"])

    finally:
        client.close()


if __name__ == "__main__":
    main()
```

### 5.2 异步客户端

```python
# client/order_client_async.py
import grpc
from grpc import aio
import time
import logging
import asyncio
from generated import order_pb2, order_pb2_grpc

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


class OrderAsyncClient:
    """异步 gRPC 订单客户端"""

    def __init__(self, host="localhost", port=50051):
        self.channel = aio.insecure_channel(f"{host}:{port}")
        self.stub = order_pb2_grpc.OrderServiceStub(self.channel)

    async def close(self):
        await self.channel.close()

    async def create_order(self, user_id, items):
        """异步创建订单"""
        request = order_pb2.CreateOrderRequest(
            user_id=user_id,
            items=items,
        )
        try:
            order = await self.stub.CreateOrder(request)
            logger.info(f"Created: {order.order_id}")
            return order
        except grpc.aio.AioRpcError as e:
            logger.error(f"CreateOrder failed: {e.code()}")
            return None

    async def list_orders(self, user_id=""):
        """异步列出订单"""
        request = order_pb2.ListOrdersRequest(user_id=user_id)
        try:
            async for order in self.stub.ListOrders(request):
                logger.info(f"Order: {order.order_id}")
        except grpc.aio.AioRpcError as e:
            logger.error(f"ListOrders failed: {e.code()}")

    async def concurrent_requests(self, num_requests=10):
        """并发请求"""
        tasks = []
        for i in range(num_requests):
            items = [order_pb2.OrderItem(
                product_id=f"prod-{i}",
                product_name=f"Product {i}",
                quantity=1,
                unit_price=9.99,
            )]
            tasks.append(self.create_order(f"user-{i}", items))

        results = await asyncio.gather(*tasks, return_exceptions=True)
        success = sum(1 for r in results if r is not None)
        logger.info(f"Concurrent: {success}/{num_requests} succeeded")


async def main():
    client = OrderAsyncClient()
    try:
        items = [order_pb2.OrderItem(
            product_id="prod-001",
            product_name="Test Product",
            quantity=1,
            unit_price=19.99,
        )]
        await client.create_order("user-001", items)
        await client.list_orders()
        await client.concurrent_requests(10)
    finally:
        await client.close()


if __name__ == "__main__":
    asyncio.run(main())
```

---

## 6. 拦截器

### 6.1 服务端拦截器

```python
class ServerLoggingInterceptor(grpc.ServerInterceptor):
    """服务端日志拦截器"""

    def intercept_service(self, continuation, handler_call_details):
        method = handler_call_details.method
        logger.info(f"[SERVER] {method} called")

        # 获取元数据
        metadata = dict(handler_call_details.invocation_metadata)
        trace_id = metadata.get("x-trace-id", "unknown")
        logger.info(f"[SERVER] trace_id={trace_id}")

        return continuation(handler_call_details)


class ServerAuthInterceptor(grpc.ServerInterceptor):
    """服务端认证拦截器"""

    def intercept_service(self, continuation, handler_call_details):
        metadata = dict(handler_call_details.invocation_metadata)
        token = metadata.get("authorization", "")

        if not token.startswith("Bearer "):
            return self._unauthenticated_handler()

        # 验证 token
        if not self._validate_token(token[7:]):
            return self._unauthenticated_handler()

        return continuation(handler_call_details)

    def _unauthenticated_handler(self):
        def unauthenticated(request, context):
            context.abort(grpc.StatusCode.UNAUTHENTICATED, "Invalid token")

        return grpc.unary_unary_rpc_method_handler(unauthenticated)

    def _validate_token(self, token):
        return token == "valid-token"  # 实际项目中应验证 JWT 等


# 注册拦截器
server = grpc.server(
    futures.ThreadPoolExecutor(max_workers=10),
    interceptors=[
        ServerLoggingInterceptor(),
        ServerAuthInterceptor(),
    ],
)
```

### 6.2 客户端拦截器

```python
class ClientLoggingInterceptor(grpc.UnaryUnaryClientInterceptor):
    """客户端日志拦截器"""

    def intercept_unary_unary(self, continuation, client_call_details, request):
        method = client_call_details.method
        logger.info(f"[CLIENT] Calling {method}")

        start = time.time()
        response = continuation(client_call_details, request)
        duration = time.time() - start

        logger.info(f"[CLIENT] {method} completed in {duration:.3f}s")
        return response


class ClientMetadataInterceptor(grpc.UnaryUnaryClientInterceptor):
    """客户端元数据拦截器"""

    def intercept_unary_unary(self, continuation, client_call_details, request):
        # 添加元数据
        metadata = list(client_call_details.metadata or [])
        metadata.append(("x-trace-id", str(uuid.uuid4())))
        metadata.append(("authorization", "Bearer valid-token"))

        new_details = _ClientCallDetails(
            client_call_details.method,
            client_call_details.timeout,
            metadata,
            client_call_details.credentials,
            client_call_details.wait_for_ready,
            client_call_details.compression,
        )

        return continuation(new_details, request)


class _ClientCallDetails(
    grpc.ClientCallDetails,
    namedtuple(
        "_ClientCallDetails",
        ["method", "timeout", "metadata", "credentials",
         "wait_for_ready", "compression"],
    ),
):
    pass


# 注册拦截器
channel = grpc.insecure_channel(
    "localhost:50051",
    options=[
        ("grpc.max_receive_message_length", 4 * 1024 * 1024),
    ],
)
intercepted_channel = grpc.intercept_channel(
    channel,
    ClientLoggingInterceptor(),
    ClientMetadataInterceptor(),
)
stub = order_pb2_grpc.OrderServiceStub(intercepted_channel)
```

---

## 7. 测试

### 7.1 单元测试

```python
# tests/test_order_service.py
import grpc
from concurrent import futures
import pytest
import time
from generated import order_pb2, order_pb2_grpc
from server.order_server import OrderServiceServicer


@pytest.fixture
def grpc_server():
    """测试用 gRPC 服务器"""
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=2))
    order_pb2_grpc.add_OrderServiceServicer_to_server(
        OrderServiceServicer(), server
    )
    port = server.add_insecure_port("[::]:0")
    server.start()
    yield f"localhost:{port}"
    server.stop(grace=0)


@pytest.fixture
def stub(grpc_server):
    """测试用客户端"""
    channel = grpc.insecure_channel(grpc_server)
    stub = order_pb2_grpc.OrderServiceStub(channel)
    yield stub
    channel.close()


class TestCreateOrder:
    def test_create_order_success(self, stub):
        request = order_pb2.CreateOrderRequest(
            user_id="user-001",
            items=[order_pb2.OrderItem(
                product_id="prod-001",
                product_name="Test Product",
                quantity=2,
                unit_price=19.99,
            )],
            shipping_address="123 Test St",
        )
        order = stub.CreateOrder(request)

        assert order.order_id != ""
        assert order.user_id == "user-001"
        assert order.total_amount == 39.98
        assert order.status == order_pb2.ORDER_STATUS_PENDING

    def test_create_order_empty_user_id(self, stub):
        request = order_pb2.CreateOrderRequest(
            user_id="",
            items=[order_pb2.OrderItem(
                product_id="prod-001",
                quantity=1,
                unit_price=10.0,
            )],
        )

        with pytest.raises(grpc.RpcError) as exc_info:
            stub.CreateOrder(request)

        assert exc_info.value.code() == grpc.StatusCode.INVALID_ARGUMENT


class TestGetOrder:
    def test_get_order_found(self, stub):
        # 先创建
        created = stub.CreateOrder(order_pb2.CreateOrderRequest(
            user_id="user-001",
            items=[order_pb2.OrderItem(
                product_id="p1", quantity=1, unit_price=10.0
            )],
        ))

        # 再获取
        order = stub.GetOrder(order_pb2.GetOrderRequest(
            order_id=created.order_id,
        ))
        assert order.order_id == created.order_id

    def test_get_order_not_found(self, stub):
        with pytest.raises(grpc.RpcError) as exc_info:
            stub.GetOrder(order_pb2.GetOrderRequest(
                order_id="nonexistent",
            ))

        assert exc_info.value.code() == grpc.StatusCode.NOT_FOUND
```

---

## 8. TLS 配置

### 8.1 生成证书

```bash
# 生成 CA 密钥和证书
openssl genrsa -out ca.key 4096
openssl req -new -x509 -key ca.key -sha256 -subj "/C=CN/ST=State/L=City/O=Org/CN=MyCA" -days 365 -out ca.crt

# 生成服务端密钥和证书
openssl genrsa -out server.key 4096
openssl req -new -key server.key -subj "/C=CN/ST=State/L=City/O=Org/CN=localhost" -out server.csr
openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt -days 365 -sha256
```

### 8.2 服务端 TLS

```python
def serve_tls():
    # 读取证书
    with open("server.crt", "rb") as f:
        server_cert = f.read()
    with open("server.key", "rb") as f:
        server_key = f.read()

    # 创建凭证
    server_credentials = grpc.ssl_server_credentials(
        [(server_key, server_cert)]
    )

    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    order_pb2_grpc.add_OrderServiceServicer_to_server(
        OrderServiceServicer(), server
    )
    server.add_secure_port("[::]:50051", server_credentials)
    server.start()
    server.wait_for_termination()
```

### 8.3 客户端 TLS

```python
def create_tls_client():
    with open("ca.crt", "rb") as f:
        ca_cert = f.read()

    credentials = grpc.ssl_channel_credentials(root_certificates=ca_cert)
    channel = grpc.secure_channel("localhost:50051", credentials)
    return order_pb2_grpc.OrderServiceStub(channel)
```

---

## 9. 总结

Python gRPC 开发核心要点:

1. **安装**: `grpcio` + `grpcio-tools`
2. **代码生成**: `grpc_tools.protoc`
3. **同步/异步**: 两种 API 风格 (同步 vs `grpc.aio`)
4. **拦截器**: ServerInterceptor / ClientInterceptor
5. **测试**: pytest + 临时服务器
6. **TLS**: `ssl_server_credentials` / `ssl_channel_credentials`

下一步: 学习 [gRPC 拦截器与中间件](07-GRPC拦截器与中间件.md)
