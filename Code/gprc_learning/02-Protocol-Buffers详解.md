# Protocol Buffers 详解

## 1. Protocol Buffers 简介

Protocol Buffers (简称 Protobuf) 是 Google 开发的一种语言无关、平台无关的可扩展序列化结构数据格式。它是 gRPC 的默认 IDL (接口定义语言) 和序列化格式。

### 1.1 核心特性

- **高效**: 二进制编码，比 JSON 小 3-10 倍，比 XML 小 20-100 倍
- **快速**: 序列化/反序列化速度比 JSON 快 20-100 倍
- **强类型**: 编译时类型检查
- **向后兼容**: 新增字段不影响旧代码
- **多语言**: 支持 11+ 编程语言

### 1.2 版本对比

| 特性 | proto2 | proto3 |
|------|--------|--------|
| 语法 | `required`, `optional` | 默认全部可选 |
| 默认值 | 可区分未设置和默认值 | 无法区分 |
| 枚举首值 | 任意 | 必须为 0 |
| JSON 映射 | 不支持 | 原生支持 |
| Map | 不支持 | 支持 |
| Any | 不支持 | 支持 |
| 推荐度 | 遗留项目 | **推荐使用** |

---

## 2. proto3 语法详解

### 2.1 文件结构

```protobuf
// 1. 语法声明 (必须放在第一行)
syntax = "proto3";

// 2. 包声明
package mypackage;

// 3. 导入其他 proto 文件
import "google/protobuf/timestamp.proto";
import "google/protobuf/any.proto";

// 4. 选项
option go_package = "github.com/example/mypackage";
option java_package = "com.example.mypackage";
option csharp_namespace = "MyPackage";

// 5. 消息定义
message Person {
    // ...
}

// 6. 服务定义
service PersonService {
    // ...
}
```

### 2.2 标量类型 (Scalar Types)

| Protobuf 类型 | Go 类型 | Java 类型 | Python 类型 | C++ 类型 | 说明 |
|---------------|---------|-----------|-------------|----------|------|
| `double` | float64 | double | float | double | 双精度浮点 |
| `float` | float32 | float | float | float | 单精度浮点 |
| `int32` | int32 | int | int | int32 | 变长编码，负数效率低 |
| `int64` | int64 | long | int | int64 | 变长编码，负数效率低 |
| `sint32` | int32 | int | int | int32 | ZigZag 编码，负数效率高 |
| `sint64` | int64 | long | int | int64 | ZigZag 编码，负数效率高 |
| `uint32` | uint32 | int | int | uint32 | 无符号变长编码 |
| `uint64` | uint64 | long | int | uint64 | 无符号变长编码 |
| `fixed32` | uint32 | int | int | uint32 | 固定4字节，值>2^28时比uint32高效 |
| `fixed64` | uint64 | long | int | uint64 | 固定8字节，值>2^56时比uint64高效 |
| `sfixed32` | int32 | int | int | int32 | 固定4字节 |
| `sfixed64` | int64 | long | int | int64 | 固定8字节 |
| `bool` | bool | boolean | bool | bool | 布尔值 |
| `string` | string | String | str | string | UTF-8 字符串 |
| `bytes` | []byte | ByteString | bytes | string | 字节数组 |

### 2.3 字段编号与规则

```protobuf
message Example {
    // 字段编号: 1 ~ 536,870,911 (2^29 - 1)
    // 19000 ~ 19999 保留给 Protobuf 内部使用

    string name = 1;          // 编号 1
    int32 age = 2;            // 编号 2
    bool active = 3;          // 编号 3

    // 编号选择建议:
    // - 1~15: 最常用字段 (1字节编码)
    // - 16~2047: 次常用字段 (2字节编码)
    // - 尽量不要跳号，保持连续
}
```

**编号编码开销**:
```
字段编号 + 类型 → Varint 编码
  1~15    → 1 字节  (0xxxxxxx)
  16~2047 → 2 字节  (1xxxxxxx 0xxxxxxx)
  2048+   → 3+ 字节

所以: 频繁使用的字段用 1~15
```

### 2.4 默认值规则

proto3 中，所有字段都是可选的。未设置的字段使用类型的默认值：

```protobuf
message Defaults {
    int32 count = 1;        // 默认: 0
    bool active = 2;        // 默认: false
    string name = 3;        // 默认: "" (空字符串)
    bytes data = 4;         // 默认: [] (空字节)
    Status status = 5;      // 默认: 枚举的第一个值 (0)
    Person person = 6;      // 默认: nil / null
    repeated int32 tags = 7; // 默认: [] (空列表)
}

enum Status {
    UNKNOWN = 0;  // 必须以 0 开头
    ACTIVE = 1;
    INACTIVE = 2;
}
```

> **重要**: proto3 无法区分"字段未设置"和"字段设置为默认值"。如果需要区分，请使用 `optional` 关键字或 `wrapper` 类型。

### 2.5 optional 关键字 (proto3.15+)

```protobuf
message WithOptional {
    // 普通: 无法区分未设置和默认值 0
    int32 count = 1;

    // optional: 可以区分
    optional int32 optional_count = 2;

    // 生成的 Go 代码:
    // Count        int32
    // OptionalCount *int32   // 指针类型，nil 表示未设置
}
```

### 2.6 repeated 字段 (列表/数组)

```protobuf
message RepeatedExample {
    // 重复字段 → 有序列表
    repeated string tags = 1;         // Go: []string
    repeated int32 scores = 2;        // Go: []int32
    repeated Person people = 3;       // Go: []*Person

    // packed 编码 (标量类型默认 packed)
    // 连续的数值字段编码为一个块，更高效
    repeated int32 packed_nums = 4 [packed = true];  // 默认就是 packed
}
```

### 2.7 map 字段 (字典/映射)

```protobuf
message MapExample {
    // map<key_type, value_type> name = number;
    map<string, string> headers = 1;       // Go: map[string]string
    map<int64, Person> users = 2;          // Go: map[int64]*Person
    map<string, google.protobuf.Any> data = 3;

    // 注意:
    // - key 不能是 float/double/bytes/enum/message
    // - map 不能是 repeated
    // - map 不保证顺序
    // - map 内部等价于: message MapEntry { key_type key = 1; value_type value = 2; }
    //   repeated MapEntry name = number;
}
```

### 2.8 嵌套消息

```protobuf
message Outer {
    string name = 1;

    // 嵌套消息定义
    message Inner {
        string value = 1;
        int32 count = 2;
    }

    // 使用嵌套消息
    Inner detail = 2;
    repeated Inner details = 3;
}

// 外部引用嵌套消息
message Other {
    Outer.Inner inner = 1;
}
```

### 2.9 枚举类型

```protobuf
message EnumExample {
    enum Status {
        // 必须以 0 开头 (作为默认值)
        UNKNOWN = 0;
        ACTIVE = 1;
        INACTIVE = 2;
        PENDING = 3;
    }

    Status status = 1;

    // 枚举别名 (允许相同值的不同名称)
    enum Type {
        option allow_alias = true;
        TYPE_UNSPECIFIED = 0;
        TYPE_A = 1;
        TYPE_ALPHA = 1;   // 别名，与 TYPE_A 值相同
    }

    Type type = 2;
}

// 嵌套枚举在外部引用
message OtherMessage {
    EnumExample.Status status = 1;
}
```

### 2.10 Oneof (多选一)

```protobuf
message OneofExample {
    string name = 1;

    // oneof: 同一时刻只有一个字段有值
    oneof contact {
        string email = 2;
        string phone = 3;
        Address address = 4;
    }

    // 设置一个会自动清除其他
    // Go 代码: contact 是 interface{} 类型
    // 可以用类型断言获取具体值
}

// Go 使用示例:
// switch c := msg.GetContact().(type) {
// case *OneofExample_Email:    fmt.Println(c.Email)
// case *OneofExample_Phone:    fmt.Println(c.Phone)
// case *OneofExample_Address:  fmt.Println(c.Address)
// }
```

### 2.11 Well-Known Types (WKT)

Protobuf 提供了一组预定义的常用类型：

```protobuf
import "google/protobuf/timestamp.proto";
import "google/protobuf/duration.proto";
import "google/protobuf/any.proto";
import "google/protobuf/empty.proto";
import "google/protobuf/wrappers.proto";
import "google/protobuf/struct.proto";
import "google/protobuf/field_mask.proto";

message WKTExample {
    // 时间戳
    google.protobuf.Timestamp created_at = 1;
    google.protobuf.Timestamp updated_at = 2;

    // 时间间隔
    google.protobuf.Duration timeout = 3;

    // 任意类型 (动态类型)
    google.protobuf.Any detail = 4;

    // 空消息 (用于无参数/无返回值的 RPC)
    // rpc Ping(google.protobuf.Empty) returns (google.protobuf.Empty);

    // 包装类型 (区分未设置和默认值)
    google.protobuf.Int32Value count = 5;    // 可以为 nil
    google.protobuf.StringValue name = 6;    // 可以为 nil
    google.protobuf.BoolValue active = 7;    // 可以为 nil
    google.protobuf.DoubleValue score = 8;   // 可以为 nil

    // 动态结构 (类似 JSON 对象)
    google.protobuf.Struct metadata = 9;

    // 字段掩码 (部分更新)
    google.protobuf.FieldMask update_mask = 10;
}
```

#### WKT 各类型详解

**Timestamp**:
```protobuf
message Timestamp {
    int64 seconds = 1;  // Unix 纪元以来的秒数
    int32 nanos = 2;    // 纳秒偏移量
}

// Go 使用:
// t := timestamppb.New(time.Now())
// goTime := t.AsTime()
```

**Duration**:
```protobuf
message Duration {
    int64 seconds = 1;  // 秒数部分
    int32 nanos = 2;    // 纳秒部分 (±999,999,999)
}

// Go 使用:
// d := durationpb.New(5 * time.Second)
// goDuration := d.AsDuration()
```

**Any**:
```protobuf
message Any {
    string type_url = 1;  // 类型 URL: "type.googleapis.com/packagename.MessageType"
    bytes value = 2;      // 序列化的消息
}

// Go 使用:
// anyMsg, _ := anypb.New(&person)
// var p pb.Person
// anyMsg.UnmarshalTo(&p)
```

**Wrappers**:
```protobuf
// 每种标量类型对应一个包装类型
message Int32Value  { int32  value = 1; }
message Int64Value  { int64  value = 1; }
message UInt32Value { uint32 value = 1; }
message UInt64Value { uint64 value = 1; }
message FloatValue  { float  value = 1; }
message DoubleValue { double value = 1; }
message BoolValue   { bool   value = 1; }
message StringValue { string value = 1; }
message BytesValue  { bytes  value = 1; }
```

**Struct**:
```protobuf
message Struct {
    map<string, Value> fields = 1;
}

message Value {
    oneof kind {
        NullValue   null_value  = 1;
        double      number_value = 2;
        string      string_value = 3;
        bool        bool_value   = 4;
        Struct      struct_value = 5;
        ListValue   list_value   = 6;
    }
}

message ListValue {
    repeated Value values = 1;
}
```

### 2.12 reserved 关键字

```protobuf
message ReservedExample {
    // 保留字段编号 (防止被复用)
    reserved 2, 15, 9 to 11;

    // 保留字段名 (防止被复用)
    reserved "foo", "bar";

    // 为什么需要 reserved?
    // 如果删除了一个字段，其编号不应被新字段复用，
    // 否则旧数据反序列化时会出错
}
```

---

## 3. Protobuf 编码原理

### 3.1 Varint 编码

Varint 是一种可变长度的整数编码方式，每个字节的最高位 (MSB) 表示是否还有后续字节：

```
数值 1 的 Varint 编码:
  0000 0001          → 1 字节

数值 300 的 Varint 编码:
  二进制: 100101100
  分组:   0000010  0101100
  编码:   0000010  10101100   (加上 MSB)
  十六进制: 0x02    0xAC       → 2 字节

解码过程:
  1. 读取字节: 0xAC → MSB=1, 还有后续
     有效位: 0101100 (44)
  2. 读取字节: 0x02 → MSB=0, 结束
     有效位: 0000010 (2)
  3. 组合: 0000010 || 0101100 = 256 + 44 = 300
```

### 3.2 Wire Type (线型)

每个字段在编码时包含两部分: 字段编号 + 线型

```
Tag = (field_number << 3) | wire_type

线型:
┌──────────┬─────┬─────────────────────────┐
│ 类型      │ 值   │ 含义                     │
├──────────┼─────┼─────────────────────────┤
│ Varint   │ 0   │ int32, int64, uint32,    │
│          │     │ uint64, sint32, sint64,  │
│          │     │ bool, enum               │
│ 64-bit   │ 1   │ fixed64, sfixed64,       │
│          │     │ double                   │
│ Length-  │ 2   │ string, bytes, embedded  │
│ delimited│     │ messages, packed         │
│          │     │ repeated fields          │
│ Start    │ 3   │ 已废弃 (group)            │
│ group    │     │                          │
│ End      │ 4   │ 已废弃 (group)            │
│ group    │     │                          │
│ 32-bit   │ 5   │ fixed32, sfixed32,       │
│          │     │ float                    │
└──────────┴─────┴─────────────────────────┘
```

### 3.3 消息编码示例

```protobuf
message Person {
    string name = 1;
    int32 age = 2;
}
```

编码 `Person{name: "Alice", age: 30}`:

```
字段1 (name="Alice"):
  Tag: (1 << 3) | 2 = 0x0A    → 字段编号1, 线型2 (length-delimited)
  Length: 5                    → "Alice" 的字节长度
  Value: 41 6C 69 63 65       → "Alice" 的 UTF-8 编码

字段2 (age=30):
  Tag: (2 << 3) | 0 = 0x10    → 字段编号2, 线型0 (varint)
  Value: 1E                   → 30 的 varint 编码

最终字节流: 0A 05 41 6C 69 63 65 10 1E
           ↑  ↑  └──── "Alice" ────┘ ↑  ↑
           │  长度                    │  30
           Tag(name)                  Tag(age)
```

### 3.4 ZigZag 编码

`sint32` 和 `sint64` 使用 ZigZag 编码将负数映射为正数，提高负数的编码效率：

```
原始值          ZigZag 编码值
 0        →        0
-1        →        1
 1        →        2
-2        →        3
 2        →        4
-3        →        5
 3        →        6

编码公式: (n << 1) ^ (n >> 31)    // sint32
         (n << 1) ^ (n >> 63)    // sint64

对比:
int32(-1)  → Varint: 10 字节 (因为补码是全1)
sint32(-1) → Varint: 1 字节 (ZigZag 后变为 1)
```

---

## 4. 服务定义

### 4.1 基本 service 定义

```protobuf
service UserService {
    // 一元 RPC
    rpc GetUser(GetUserRequest) returns (User) {}
    rpc CreateUser(CreateUserRequest) returns (User) {}
    rpc DeleteUser(DeleteUserRequest) returns (google.protobuf.Empty) {}

    // 服务端流式
    rpc ListUsers(ListUsersRequest) returns (stream User) {}

    // 客户端流式
    rpc UploadAvatars(stream AvatarUploadRequest) returns (AvatarUploadResponse) {}

    // 双向流式
    rpc Chat(stream ChatMessage) returns (stream ChatMessage) {}
}
```

### 4.2 请求和响应设计模式

```protobuf
// 标准请求模式: 包含资源名 + 可选字段
message GetUserRequest {
    string user_id = 1;
}

// 标准响应模式: 包含资源 + 元数据
message GetUserResponse {
    User user = 1;
}

// 列表请求模式: 分页
message ListUsersRequest {
    int32 page_size = 1;      // 每页大小
    string page_token = 2;    // 分页令牌
    string filter = 3;        // 过滤条件
}

// 列表响应模式: 结果 + 分页信息
message ListUsersResponse {
    repeated User users = 1;
    string next_page_token = 2;  // 下一页令牌
    int32 total_size = 3;        // 总数
}

// 批量操作请求
message BatchGetUsersRequest {
    repeated string user_ids = 1;
}

// 部分更新: 使用 FieldMask
message UpdateUserRequest {
    User user = 1;
    google.protobuf.FieldMask update_mask = 2;
}
```

---

## 5. 包管理与导入

### 5.1 包组织最佳实践

```
proto/
├── common/
│   ├── types.proto         # 公共类型
│   └── errors.proto        # 错误定义
├── user/
│   ├── user.proto          # 用户消息定义
│   └── user_service.proto  # 用户服务定义
├── order/
│   ├── order.proto         # 订单消息定义
│   └── order_service.proto # 订单服务定义
└── api/
    └── gateway.proto       # API 网关定义
```

### 5.2 跨文件引用

```protobuf
// common/types.proto
syntax = "proto3";
package common;

message Pagination {
    int32 page_size = 1;
    string page_token = 2;
}

message PaginationResponse {
    string next_page_token = 1;
    int32 total_size = 2;
}
```

```protobuf
// user/user_service.proto
syntax = "proto3";
package user;

import "common/types.proto";  // 导入公共类型

service UserService {
    rpc ListUsers(ListUsersRequest) returns (ListUsersResponse) {}
}

message ListUsersRequest {
    common.Pagination pagination = 1;  // 使用导入的类型
}

message ListUsersResponse {
    repeated User users = 1;
    common.PaginationResponse pagination = 2;
}
```

### 5.3 import public

```protobuf
// a.proto
message A { string name = 1; }

// b.proto
import public "a.proto";   // public: 导入 a 的定义也对外可见
import "other.proto";      // 非 public: 仅 b 内部可见

// c.proto
import "b.proto";
// 可以使用 A，因为 b 通过 import public 暴露了 a 的定义
```

---

## 6. 代码生成

### 6.1 Go 代码生成

```bash
# 基本用法
protoc \
    --go_out=. \              # 生成消息类型代码
    --go_opt=paths=source_relative \  # 输出路径相对于源文件
    --go-grpc_out=. \         # 生成 gRPC 服务代码
    --go-grpc_opt=paths=source_relative \
    user_service.proto

# 多文件批量生成
protoc \
    --go_out=. \
    --go_opt=module=github.com/example/project \
    --go-grpc_out=. \
    --go-grpc_opt=module=github.com/example/project \
    proto/user/user.proto \
    proto/order/order.proto

# 生成的文件:
# user_service.pb.go         ← 消息类型的序列化/反序列化代码
# user_service_grpc.pb.go    ← gRPC 客户端桩和服务端接口
```

### 6.2 Java 代码生成

```bash
# 命令行
protoc \
    --java_out=src/main/java \
    --grpc-java_out=src/main/java \
    user_service.proto

# Maven 配置 (推荐)
# 见 01 章节 Maven 配置
```

### 6.3 Python 代码生成

```bash
python -m grpc_tools.protoc \
    -I proto \
    --python_out=generated \
    --grpc_python_out=generated \
    proto/user/user_service.proto

# 生成的文件:
# user_service_pb2.py        ← 消息类型
# user_service_pb2_grpc.py   ← gRPC 服务
```

---

## 7. 使用 buf 工具链 (现代化方案)

### 7.1 为什么用 buf

```bash
# 传统 protoc 的问题:
# 1. 插件管理复杂
# 2. 编译命令冗长
# 3. 无 lint 检查
# 4. 无 Breaking Change 检测
# 5. 依赖管理困难

# buf 的优势:
# 1. 简化的配置文件 (buf.yaml, buf.gen.yaml)
# 2. 内置 lint 规则
# 3. Breaking Change 检测
# 4. 自动依赖管理 (BSR)
# 5. 格式化
```

### 7.2 buf 配置

```yaml
# buf.yaml
version: v2
lint:
  use:
    - STANDARD
  enum_zero_value_suffix: _UNSPECIFIED
breaking:
  use:
    - FILE
```

```yaml
# buf.gen.yaml
version: v2
managed:
  enabled: true
  override:
    - file_option: go_package_prefix
      value: github.com/example/project/gen
plugins:
  - remote: buf.build/protocolbuffers/go
    out: gen
    opt: paths=source_relative
  - remote: buf.build/grpc/go
    out: gen
    opt: paths=source_relative
```

### 7.3 buf 命令

```bash
# 初始化
buf init

# Lint 检查
buf lint

# 格式化
buf format -w

# 生成代码
buf generate

# Breaking Change 检测
buf breaking --against '.git#branch=main'

# 构建
buf build
```

---

## 8. Protobuf 验证 (protoc-gen-validate)

### 8.1 安装

```bash
# Go
go install github.com/envoyproxy/protoc-gen-validate@latest
```

### 8.2 定义验证规则

```protobuf
syntax = "proto3";

package validate;

import "validate/validate.proto";

message Person {
    // 字符串验证
    string name = 1 [(validate.rules).string = {
        min_len: 1,
        max_len: 100,
        pattern: "^[a-zA-Z]+$"
    }];

    // 整数验证
    int32 age = 2 [(validate.rules).int32 = {
        gte: 0,
        lte: 150
    }];

    // 枚举验证
    enum Status {
        UNKNOWN = 0;
        ACTIVE = 1;
        INACTIVE = 2;
    }
    Status status = 3 [(validate.rules).enum = {
        defined_only: true,
        not_in: [0]  // 不允许 UNKNOWN
    }];

    // 邮箱验证
    string email = 4 [(validate.rules).string.email = true];

    // URL 验证
    string website = 5 [(validate.rules).string.uri = true];

    // UUID 验证
    string id = 6 [(validate.rules).string.uuid = true];

    // Map 验证
    map<string, string> labels = 7 [(validate.rules).map = {
        min_pairs: 1,
        max_pairs: 10
    }];

    // Repeated 验证
    repeated string tags = 8 [(validate.rules).repeated = {
        min_items: 1,
        max_items: 20,
        unique: true
    }];

    // 嵌套消息验证
    Address address = 9 [(validate.rules).message.required = true];
}

message Address {
    string street = 1 [(validate.rules).string.min_len = 1];
    string city = 2 [(validate.rules).string.min_len = 1];
    string zip = 3 [(validate.rules).string.pattern = "^[0-9]{6}$"];
}
```

---

## 9. 常见陷阱与最佳实践

### 9.1 字段编号永远不要复用

```protobuf
// ❌ 错误: 删除字段后复用编号
message Bad {
    string name = 1;     // 已删除
    string full_name = 1; // 复用了编号 1！旧数据会错乱
}

// ✅ 正确: 使用 reserved 保留编号
message Good {
    reserved 1;
    reserved "name";
    string full_name = 2;  // 使用新编号
}
```

### 9.2 注意默认值问题

```protobuf
message SearchRequest {
    int32 page_size = 1;  // 默认 0，但业务上可能期望 10
}

// 解决方案 1: 使用 optional
message SearchRequest {
    optional int32 page_size = 1;  // 可区分未设置
}

// 解决方案 2: 使用 wrapper
message SearchRequest {
    google.protobuf.Int32Value page_size = 1;
}

// 解决方案 3: 在业务代码中处理
// if req.PageSize == 0 { req.PageSize = 10 }
```

### 9.3 大消息处理

```protobuf
// ❌ 避免: 单个消息过大 (>1MB)
message Bad {
    bytes file_content = 1;  // 可能几百 MB
}

// ✅ 推荐: 使用流式传输
service FileService {
    rpc Upload(stream FileChunk) returns (UploadResponse) {}
    rpc Download(DownloadRequest) returns (stream FileChunk) {}
}

message FileChunk {
    string file_name = 1;
    int64 offset = 2;
    bytes chunk_data = 3;  // 每个 chunk 建议 64KB - 4MB
}
```

### 9.4 向后兼容变更指南

```
✅ 兼容的变更:
- 新增 optional/repeated 字段 (新编号)
- 新增枚举值
- 新增 oneof 字段
- 将 singular 改为 repeated (部分语言)
- 重命名字段 (线格式只看编号)

❌ 不兼容的变更:
- 删除字段 (应使用 reserved)
- 复用字段编号
- 修改字段类型 (不兼容的类型)
- 修改字段编号
- 将 required 改为 optional (proto2)
- 将 repeated 改为 singular
```

---

## 10. 总结

Protocol Buffers 是 gRPC 的基石，掌握其语法和编码原理对于高效使用 gRPC 至关重要：

1. **语法**: proto3 简化了 proto2，推荐使用
2. **编码**: Varint + ZigZag + Length-delimited，理解原理有助于优化
3. **类型系统**: 标量、枚举、嵌套、oneof、map、Any 等丰富类型
4. **WKT**: 善用 Timestamp、Duration、Empty、Wrappers 等内置类型
5. **验证**: protoc-gen-validate 提供声明式验证
6. **工具链**: buf 是现代化的 Protobuf 工具链

下一步: 学习 [gRPC 四种通信模式](03-GRPC四种通信模式.md)
