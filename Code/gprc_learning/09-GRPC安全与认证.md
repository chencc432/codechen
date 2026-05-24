# gRPC 安全与认证

## 1. 安全概述

gRPC 安全主要包括三个方面：

```
gRPC 安全
├── 传输安全 (Transport Security)
│   └── TLS/SSL 加密通道
├── 认证 (Authentication)
│   ├── 证书认证 (mTLS)
│   ├── Token 认证 (JWT/OAuth2)
│   └── 基础认证 (Basic Auth)
└── 授权 (Authorization)
    └── 基于角色的访问控制 (RBAC)
```

---

## 2. TLS/SSL 传输加密

### 2.1 证书体系

```
┌─────────────────────────────────────────┐
│              PKI 证书体系                │
│                                         │
│  Root CA (自签名)                       │
│    ├── Intermediate CA                  │
│    │     ├── Server Certificate         │
│    │     └── Client Certificate         │
│    └── Intermediate CA 2                │
│          └── ...                        │
│                                         │
│  单向 TLS: 客户端验证服务端证书          │
│  双向 TLS: 双方互相验证证书 (mTLS)      │
└─────────────────────────────────────────┘
```

### 2.2 生成测试证书

```bash
# 1. 生成 CA 密钥和证书
openssl genrsa -out ca.key 4096
openssl req -new -x509 -days 365 -key ca.key \
    -subj "/C=CN/ST=Beijing/L=Beijing/O=MyOrg/CN=MyCA" \
    -out ca.crt

# 2. 生成服务端密钥和 CSR
openssl genrsa -out server.key 4096
openssl req -new -key server.key \
    -subj "/C=CN/ST=Beijing/L=Beijing/O=MyOrg/CN=localhost" \
    -out server.csr

# 3. 用 CA 签发服务端证书
openssl x509 -req -days 365 -in server.csr \
    -CA ca.crt -CAkey ca.key -CAcreateserial \
    -extfile <(echo "subjectAltName=DNS:localhost,IP:127.0.0.1") \
    -out server.crt

# 4. 生成客户端密钥和 CSR (mTLS)
openssl genrsa -out client.key 4096
openssl req -new -key client.key \
    -subj "/C=CN/ST=Beijing/L=Beijing/O=MyOrg/CN=client1" \
    -out client.csr

# 5. 用 CA 签发客户端证书
openssl x509 -req -days 365 -in client.csr \
    -CA ca.crt -CAkey ca.key -CAcreateserial \
    -out client.crt
```

### 2.3 Go TLS 配置

**单向 TLS (服务端证书)**:

```go
// 服务端
func main() {
    creds, err := credentials.NewServerTLSFromFile(
        "server.crt", "server.key",
    )
    if err != nil {
        log.Fatalf("failed to create credentials: %v", err)
    }

    s := grpc.NewServer(grpc.Creds(creds))
    // ...注册服务

    lis, _ := net.Listen("tcp", ":50051")
    s.Serve(lis)
}

// 客户端
func main() {
    creds, err := credentials.NewClientTLSFromFile(
        "ca.crt", "localhost",
    )
    if err != nil {
        log.Fatalf("failed to create credentials: %v", err)
    }

    conn, err := grpc.Dial("localhost:50051",
        grpc.WithTransportCredentials(creds),
    )
    // ...
}
```

**双向 TLS (mTLS)**:

```go
// 服务端
func main() {
    // 加载服务端证书
    serverCert, err := tls.LoadX509KeyPair("server.crt", "server.key")
    if err != nil {
        log.Fatal(err)
    }

    // 加载 CA 证书
    caCert, err := os.ReadFile("ca.crt")
    if err != nil {
        log.Fatal(err)
    }
    caCertPool := x509.NewCertPool()
    caCertPool.AppendCertsFromPEM(caCert)

    // TLS 配置
    tlsConfig := &tls.Config{
        Certificates: []tls.Certificate{serverCert},
        ClientAuth:   tls.RequireAndVerifyClientCert, // 要求客户端证书
        ClientCAs:    caCertPool,
        MinVersion:   tls.VersionTLS12,
    }

    creds := credentials.NewTLS(tlsConfig)

    s := grpc.NewServer(grpc.Creds(creds))
    // ...
}

// 客户端
func main() {
    // 加载客户端证书
    clientCert, err := tls.LoadX509KeyPair("client.crt", "client.key")
    if err != nil {
        log.Fatal(err)
    }

    // 加载 CA 证书
    caCert, err := os.ReadFile("ca.crt")
    if err != nil {
        log.Fatal(err)
    }
    caCertPool := x509.NewCertPool()
    caCertPool.AppendCertsFromPEM(caCert)

    // TLS 配置
    tlsConfig := &tls.Config{
        Certificates: []tls.Certificate{clientCert},
        RootCAs:      caCertPool,
        ServerName:   "localhost",
        MinVersion:   tls.VersionTLS12,
    }

    creds := credentials.NewTLS(tlsConfig)

    conn, err := grpc.Dial("localhost:50051",
        grpc.WithTransportCredentials(creds),
    )
    // ...
}
```

### 2.4 Java TLS 配置

```java
// 单向 TLS
// 服务端
InputStream certChain = new FileInputStream("server.crt");
InputStream privateKey = new FileInputStream("server.key");

Server server = ServerBuilder.forPort(8443)
    .useTransportSecurity(certChain, privateKey)
    .addService(new OrderServiceImpl())
    .build();

// 客户端
InputStream caCert = new FileInputStream("ca.crt");
ManagedChannel channel = ManagedChannelBuilder
    .forTarget("localhost:8443")
    .useTransportSecurity()
    .build();
```

```java
// 双向 TLS (mTLS)
// 服务端
SslContextBuilder sslBuilder = SslContextBuilder.forServer(
    new File("server.crt"), new File("server.key")
);
sslBuilder.trustManager(new File("ca.crt"));
sslBuilder.clientAuth(ClientAuth.REQUIRE);  // 要求客户端证书

Server server = NettyServerBuilder.forPort(8443)
    .sslContext(sslBuilder.build())
    .addService(new OrderServiceImpl())
    .build();

// 客户端
SslContextBuilder sslBuilder = GrpcSslContexts.forClient();
sslBuilder.trustManager(new File("ca.crt"));
sslBuilder.keyManager(new File("client.crt"), new File("client.key"));

ManagedChannel channel = NettyChannelBuilder
    .forTarget("localhost:8443")
    .sslContext(sslBuilder.build())
    .build();
```

---

## 3. Token 认证

### 3.1 JWT 认证

```go
// JWT 认证拦截器
func JWTAuthInterceptor() grpc.UnaryServerInterceptor {
    return func(
        ctx context.Context,
        req interface{},
        info *grpc.UnaryServerInfo,
        handler grpc.UnaryHandler,
    ) (interface{}, error) {

        // 跳过不需要认证的方法
        if isPublicMethod(info.FullMethod) {
            return handler(ctx, req)
        }

        // 从 metadata 获取 token
        md, ok := metadata.FromIncomingContext(ctx)
        if !ok {
            return nil, status.Error(codes.Unauthenticated, "missing metadata")
        }

        authHeader := md.Get("authorization")
        if len(authHeader) == 0 {
            return nil, status.Error(codes.Unauthenticated, "missing authorization")
        }

        token := strings.TrimPrefix(authHeader[0], "Bearer ")
        if token == authHeader[0] {
            return nil, status.Error(codes.Unauthenticated, "invalid token format")
        }

        // 验证 JWT
        claims, err := validateJWT(token)
        if err != nil {
            return nil, status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
        }

        // 将用户信息存入 context
        ctx = context.WithValue(ctx, "userID", claims.UserID)
        ctx = context.WithValue(ctx, "roles", claims.Roles)

        return handler(ctx, req)
    }
}

// JWT 验证
type JWTClaims struct {
    UserID string   `json:"user_id"`
    Roles  []string `json:"roles"`
    jwt.RegisteredClaims
}

func validateJWT(tokenString string) (*JWTClaims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{},
        func(token *jwt.Token) (interface{}, error) {
            // 验证签名算法
            if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
                return nil, fmt.Errorf("unexpected signing method: %v",
                    token.Header["alg"])
            }
            return []byte("your-secret-key"), nil
        },
    )

    if err != nil {
        return nil, err
    }

    claims, ok := token.Claims.(*JWTClaims)
    if !ok || !token.Valid {
        return nil, fmt.Errorf("invalid token")
    }

    return claims, nil
}

// 生成 JWT
func generateJWT(userID string, roles []string) (string, error) {
    claims := JWTClaims{
        UserID: userID,
        Roles:  roles,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
            Issuer:    "grpc-demo",
        },
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte("your-secret-key"))
}
```

### 3.2 OAuth2 认证

```go
// 客户端 OAuth2 Token 注入
func OAuth2TokenInterceptor(tokenSource oauth2.TokenSource) grpc.UnaryClientInterceptor {
    return func(
        ctx context.Context,
        method string,
        req, reply interface{},
        cc *grpc.ClientConn,
        invoker grpc.UnaryInvoker,
        opts ...grpc.CallOption,
    ) error {
        token, err := tokenSource.Token()
        if err != nil {
            return status.Errorf(codes.Unauthenticated,
                "failed to get token: %v", err)
        }

        ctx = metadata.AppendToOutgoingContext(ctx,
            "authorization", "Bearer "+token.AccessToken,
        )

        return invoker(ctx, method, req, reply, cc, opts...)
    }
}

// 使用
config := &oauth2.Config{
    ClientID:     "client-id",
    ClientSecret: "client-secret",
    TokenURL:     "https://oauth.example.com/token",
}

tokenSource := config.TokenSource(context.Background(),
    &oauth2.Token{RefreshToken: "refresh-token"})

conn, err := grpc.Dial("localhost:50051",
    grpc.WithTransportCredentials(insecure.NewCredentials()),
    grpc.WithUnaryInterceptor(OAuth2TokenInterceptor(tokenSource)),
)
```

---

## 4. 授权

### 4.1 基于角色的授权

```go
// 授权拦截器
func RBACAuthInterceptor(roles map[string][]string) grpc.UnaryServerInterceptor {
    return func(
        ctx context.Context,
        req interface{},
        info *grpc.UnaryServerInfo,
        handler grpc.UnaryHandler,
    ) (interface{}, error) {

        // 获取用户角色
        userRoles, _ := ctx.Value("roles").([]string)
        if len(userRoles) == 0 {
            return nil, status.Error(codes.PermissionDenied, "no roles assigned")
        }

        // 获取方法所需角色
        requiredRoles, exists := roles[info.FullMethod]
        if !exists {
            // 未配置则允许
            return handler(ctx, req)
        }

        // 检查权限
        if !hasRole(userRoles, requiredRoles) {
            return nil, status.Errorf(codes.PermissionDenied,
                "required roles: %v, got: %v", requiredRoles, userRoles)
        }

        return handler(ctx, req)
    }
}

func hasRole(userRoles, requiredRoles []string) bool {
    roleSet := make(map[string]bool)
    for _, r := range userRoles {
        roleSet[r] = true
    }
    for _, r := range requiredRoles {
        if roleSet[r] {
            return true
        }
    }
    return false
}

// 配置方法-角色映射
methodRoles := map[string][]string{
    "/order.OrderService/CreateOrder": {"admin", "user"},
    "/order.OrderService/DeleteOrder": {"admin"},
    "/order.OrderService/GetOrder":    {"admin", "user", "viewer"},
}

s := grpc.NewServer(
    grpc.ChainUnaryInterceptor(
        JWTAuthInterceptor(),
        RBACAuthInterceptor(methodRoles),
    ),
)
```

### 4.2 方法级权限注解

```protobuf
// 使用 proto 自定义选项定义权限
syntax = "proto3";

import "google/protobuf/descriptor.proto";

extend google.protobuf.MethodOptions {
    repeated string required_roles = 50000;
}

service OrderService {
    rpc CreateOrder(CreateOrderRequest) returns (Order) {
        option (required_roles) = "admin";
        option (required_roles) = "user";
    }

    rpc DeleteOrder(DeleteOrderRequest) returns (google.protobuf.Empty) {
        option (required_roles) = "admin";
    }
}
```

---

## 5. gRPC 安全最佳实践

### 5.1 传输层安全

```
✅ 必做:
- 生产环境必须使用 TLS
- 使用 TLS 1.2 或更高版本
- 使用强加密套件
- 定期更新证书

❌ 避免:
- 使用 insecure 连接 (开发环境除外)
- 使用过时的 TLS 版本
- 禁用证书验证
```

### 5.2 认证安全

```
✅ 必做:
- 使用 JWT 或 mTLS 进行认证
- Token 设置合理的过期时间
- 使用 HTTPS 获取 Token
- 实现 Token 刷新机制
- 验证 Token 的签发者 (issuer)

❌ 避免:
- 在 URL 中传递 Token
- 使用不安全的签名算法 (如 none)
- 硬编码密钥
- 永不过期的 Token
```

### 5.3 常见安全配置

```go
// 服务端安全配置
s := grpc.NewServer(
    grpc.Creds(credentials),            // TLS 凭证
    grpc.MaxRecvMsgSize(4*1024*1024),    // 限制消息大小
    grpc.MaxConcurrentStreams(1000),      // 限制并发流
    grpc.ChainUnaryInterceptor(
        RecoveryInterceptor(),           // panic 恢复
        RateLimitInterceptor(limiter),   // 限流
        AuthInterceptor(),              // 认证
        RBACInterceptor(roles),         // 授权
        ValidationInterceptor(),        // 输入验证
    ),
)

// 客户端安全配置
conn, err := grpc.Dial(target,
    grpc.WithTransportCredentials(creds),  // TLS
    grpc.WithDefaultCallOptions(
        grpc.MaxCallRecvMsgSize(4*1024*1024),
    ),
    grpc.WithUnaryInterceptor(
        MetadataInterceptor(),             // 注入认证信息
    ),
    grpc.WithDefaultServiceConfig(retryConfig),  // 重试配置
)
```

---

## 6. 总结

1. **TLS**: 生产环境必须启用，推荐 mTLS
2. **JWT**: 常用的 Token 认证方式，配合拦截器使用
3. **mTLS**: 服务间零信任认证，最高安全级别
4. **RBAC**: 基于角色的细粒度授权
5. **纵深防御**: TLS + 认证 + 授权 + 限流 + 验证

下一步: 学习 [gRPC 负载均衡与服务发现](10-GRPC负载均衡与服务发现.md)
