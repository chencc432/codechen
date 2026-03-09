# 🔓 04 - OAuth 2.0 协议详解

> 当你在某个网站看到"使用微信登录"、"使用 GitHub 登录"时，背后使用的就是 OAuth 2.0 协议。本章将完整介绍这个互联网最重要的授权协议。

---

## 一、为什么需要 OAuth？

### 1.1 一个场景引出问题

假设你在使用一个"在线简历网站"，它想读取你 GitHub 上的项目信息来自动填充简历。

**没有 OAuth 的做法（危险！❌）：**
- 简历网站问你要 GitHub 的用户名和密码
- 简历网站用你的密码去 GitHub 拿数据
- **问题**：你把密码给了第三方，它可以做任何事情（删除仓库、修改代码...）

**有了 OAuth 的做法（安全！✅）：**
- 简历网站把你引导到 GitHub 的授权页面
- 你在 GitHub 上确认"只允许读取项目列表"
- GitHub 给简历网站一个"限时、限权"的访问令牌
- 简历网站用这个令牌获取数据，**永远拿不到你的密码**

```mermaid
graph LR
    subgraph 危险做法 ❌
        A[用户] -->|给出GitHub密码| B[简历网站]
        B -->|用密码登录| C[GitHub]
        B -->|可以删库!| C
    end
    
    subgraph OAuth做法 ✅
        D[用户] -->|在GitHub授权| E[GitHub]
        E -->|发放限权Token| F[简历网站]
        F -->|只能读取项目| E
    end

    style B fill:#ffcdd2
    style F fill:#c8e6c9
```

### 1.2 OAuth 2.0 的定义

**OAuth 2.0** 是一个**授权框架**（Authorization Framework），它允许第三方应用在用户授权的情况下，**安全地、有限地**访问用户在另一个服务上的资源，而不需要获取用户的密码。

> 🔑 **关键词**：授权（Authorization），不是认证（Authentication）  
> OAuth 解决的是"允许别人访问我的资源"，而不是"证明我是谁"

---

## 二、OAuth 2.0 的四个角色

```mermaid
graph TD
    A["👤 Resource Owner<br/>资源拥有者（用户）<br/>拥有GitHub账号的你"] 
    B["📱 Client<br/>客户端（第三方应用）<br/>在线简历网站"]
    C["🔐 Authorization Server<br/>授权服务器<br/>GitHub的授权服务"]
    D["📦 Resource Server<br/>资源服务器<br/>GitHub的API服务器"]
    
    A -->|授权| C
    C -->|发放Token| B
    B -->|携带Token请求| D
    D -->|返回资源| B

    style A fill:#e3f2fd
    style B fill:#fff3e0
    style C fill:#f3e5f5
    style D fill:#e8f5e9
```

| 角色 | 英文 | 说明 | 举例 |
|------|------|------|------|
| **资源拥有者** | Resource Owner | 拥有数据的用户 | 你（GitHub 用户） |
| **客户端** | Client | 想要访问用户数据的第三方应用 | 在线简历网站 |
| **授权服务器** | Authorization Server | 负责认证用户身份、颁发令牌 | GitHub OAuth 服务 |
| **资源服务器** | Resource Server | 存储用户数据的服务 | GitHub API |

> 💡 在实际中，授权服务器和资源服务器往往是同一家公司（如 GitHub），但逻辑上是两个不同的角色。

---

## 三、OAuth 2.0 的四种授权模式

OAuth 2.0 定义了四种获取令牌的方式（Grant Types），适用于不同的应用场景：

```mermaid
graph TD
    A[OAuth 2.0 授权模式] --> B["🔑 授权码模式<br/>Authorization Code<br/>⭐ 最常用、最安全"]
    A --> C["🔑 隐式模式<br/>Implicit<br/>⚠️ 已不推荐"]
    A --> D["🔑 密码模式<br/>Resource Owner Password<br/>⚠️ 仅限高度信任场景"]
    A --> E["🔑 客户端凭证模式<br/>Client Credentials<br/>适用于机器对机器"]

    style B fill:#c8e6c9
    style C fill:#ffcdd2
    style D fill:#fff3e0
    style E fill:#e3f2fd
```

---

## 四、授权码模式（Authorization Code）—— 最重要！

这是**最安全、最常用**的模式，适用于有后端服务器的 Web 应用。微信登录、GitHub 登录、Google 登录都用的是这种模式。

### 4.1 完整流程

```mermaid
sequenceDiagram
    participant 用户 as 👤 用户
    participant 客户端 as 📱 第三方应用<br/>（如简历网站）
    participant 授权服务器 as 🔐 授权服务器<br/>（如GitHub OAuth）
    participant 资源服务器 as 📦 资源服务器<br/>（如GitHub API）

    rect rgb(232, 245, 233)
        Note over 用户,授权服务器: 第一步：获取授权码（Authorization Code）
        用户->>客户端: ① 点击"使用GitHub登录"
        客户端->>用户: ② 重定向到 GitHub 授权页面
        Note over 客户端,授权服务器: GET https://github.com/login/oauth/authorize<br/>?client_id=xxx<br/>&redirect_uri=https://resume.com/callback<br/>&scope=read:user,repo<br/>&state=random123
        用户->>授权服务器: ③ 用户在GitHub上登录并点击"授权"
        授权服务器->>用户: ④ 重定向回 redirect_uri，携带授权码
        Note over 授权服务器,用户: 302 Location: https://resume.com/callback<br/>?code=AUTH_CODE_HERE<br/>&state=random123
        用户->>客户端: ⑤ 浏览器跟随重定向，带着 code 到达简历网站
    end

    rect rgb(227, 242, 253)
        Note over 客户端,资源服务器: 第二步：用授权码换取 Token（后端进行，用户不可见）
        客户端->>授权服务器: ⑥ POST /oauth/access_token<br/>{code, client_id, client_secret, redirect_uri}
        授权服务器-->>客户端: ⑦ 返回 {access_token, refresh_token, expires_in}
    end

    rect rgb(243, 229, 245)
        Note over 客户端,资源服务器: 第三步：使用 Token 访问资源
        客户端->>资源服务器: ⑧ GET /api/user<br/>Authorization: Bearer access_token
        资源服务器-->>客户端: ⑨ 返回用户的GitHub信息
    end
```

### 4.2 关键参数解释

**第一步 - 请求授权码的参数：**

| 参数 | 说明 | 示例 |
|------|------|------|
| `client_id` | 第三方应用在 OAuth 平台注册时获得的ID | `abc123` |
| `redirect_uri` | 授权后的回调地址 | `https://resume.com/callback` |
| `scope` | 请求的权限范围 | `read:user repo` |
| `state` | 随机字符串，防 CSRF 攻击 | `xyzrandom` |
| `response_type` | 授权类型，固定为 `code` | `code` |

**第二步 - 用授权码换 Token 的参数：**

| 参数 | 说明 |
|------|------|
| `code` | 第一步获得的授权码（**一次性、有时效**） |
| `client_id` | 应用ID |
| `client_secret` | 应用密钥（**只在后端使用，绝不能暴露到前端！**） |
| `redirect_uri` | 必须与第一步一致 |
| `grant_type` | 固定为 `authorization_code` |

### 4.3 为什么要分两步？为什么不直接返回 Token？

```mermaid
graph TD
    A["为什么要先返回 Code，<br/>再用 Code 换 Token？"] --> B["安全原因！"]
    
    B --> C["第一步：Code 通过浏览器传递<br/>（前端可见，可能被截获）"]
    B --> D["第二步：Code换Token 在后端进行<br/>（需要 client_secret）<br/>（用户和浏览器看不到）"]
    
    C --> E["即使 Code 被截获：<br/>① Code 一次性使用，用过即废<br/>② 没有 client_secret 无法换Token<br/>③ Code 有效期极短（通常10分钟）"]
    
    D --> F["Token 只存在于后端<br/>永远不会暴露给浏览器 ✅"]

    style F fill:#c8e6c9
    style E fill:#fff3e0
```

---

## 五、其他授权模式简介

### 5.1 隐式模式（Implicit）—— ⚠️ 已不推荐

```mermaid
sequenceDiagram
    participant 用户 as 👤 用户
    participant SPA as 📱 纯前端应用
    participant 授权服务器 as 🔐 授权服务器

    用户->>SPA: 点击登录
    SPA->>授权服务器: 重定向到授权页面（response_type=token）
    用户->>授权服务器: 登录并授权
    授权服务器->>用户: 重定向回应用，Token 在 URL Fragment 中
    Note over 授权服务器,用户: redirect_uri#access_token=xxx&token_type=bearer
    用户->>SPA: 浏览器跟随重定向
    SPA->>SPA: 从 URL Fragment 中提取 Token
```

**特点**：
- 跳过了"授权码"步骤，直接返回 Token
- Token 暴露在 URL 中，安全性差
- **已被 OAuth 2.1 废弃**，建议使用授权码模式 + PKCE 替代

### 5.2 密码模式（Resource Owner Password）—— ⚠️ 仅限特殊场景

```mermaid
sequenceDiagram
    participant 用户 as 👤 用户
    participant 客户端 as 📱 自家应用
    participant 授权服务器 as 🔐 授权服务器

    用户->>客户端: 输入用户名和密码
    客户端->>授权服务器: POST /token<br/>{grant_type=password, username, password, client_id}
    授权服务器-->>客户端: {access_token, refresh_token}
```

**特点**：
- 用户直接把密码给客户端
- **只适用于用户高度信任的自家应用**（如公司内部系统）
- **不适用于第三方应用**

### 5.3 客户端凭证模式（Client Credentials）—— 机器对机器

```mermaid
sequenceDiagram
    participant 服务A as 🖥️ 微服务A
    participant 授权服务器 as 🔐 授权服务器
    participant 服务B as 🖥️ 微服务B

    服务A->>授权服务器: POST /token<br/>{grant_type=client_credentials, client_id, client_secret}
    授权服务器-->>服务A: {access_token}
    服务A->>服务B: 携带 access_token 调用接口
    服务B-->>服务A: 返回数据
```

**特点**：
- 没有用户参与
- 适用于服务器之间的 API 调用
- 如：微服务 A 访问微服务 B 的数据

---

## 六、PKCE 扩展 —— 移动端/SPA 的安全增强

**PKCE**（Proof Key for Code Exchange，读作"pixie"）是授权码模式的安全增强，专为**无法安全存储 client_secret 的客户端**（如移动 App、SPA）设计。

### 6.1 PKCE 流程

```mermaid
sequenceDiagram
    participant App as 📱 移动App/SPA
    participant 授权服务器 as 🔐 授权服务器

    App->>App: ① 生成随机 code_verifier<br/>② 计算 code_challenge = SHA256(code_verifier)
    
    App->>授权服务器: ③ 请求授权码<br/>+ code_challenge + code_challenge_method=S256
    授权服务器-->>App: ④ 返回授权码 code
    
    App->>授权服务器: ⑤ 用 code + code_verifier 换Token
    授权服务器->>授权服务器: ⑥ 验证 SHA256(code_verifier) == 之前的 code_challenge
    授权服务器-->>App: ⑦ 验证通过，返回 Token
```

**核心思想**：即使授权码被拦截，攻击者没有 `code_verifier` 也无法换取 Token。

---

## 七、OAuth 2.0 中的 Scope（权限范围）

Scope 用来限制第三方应用能访问的资源范围：

```mermaid
graph TD
    A["OAuth Scope 权限控制"] --> B["GitHub 示例"]
    A --> C["微信示例"]
    A --> D["Google 示例"]
    
    B --> B1["read:user - 读取用户信息"]
    B --> B2["repo - 访问仓库"]
    B --> B3["delete_repo - 删除仓库"]
    B --> B4["gist - 管理Gist"]
    
    C --> C1["snsapi_base - 静默获取openid"]
    C --> C2["snsapi_userinfo - 获取用户信息"]
    
    D --> D1["email - 读取邮箱"]
    D --> D2["profile - 读取个人信息"]
    D --> D3["calendar - 读取日历"]
```

> 💡 **最小权限原则**：只申请应用真正需要的 Scope，不要过度申请权限。

---

## 八、OAuth 2.0 vs OpenID Connect

| 对比 | OAuth 2.0 | OpenID Connect (OIDC) |
|------|-----------|----------------------|
| **目的** | 授权（Authorization） | 认证 + 授权（Authentication + Authorization） |
| **回答** | "允许访问哪些资源" | "用户是谁" + "允许访问哪些资源" |
| **Token** | Access Token | Access Token + **ID Token** |
| **ID Token** | 无 | JWT 格式，包含用户身份信息 |
| **用户信息** | 需要额外调 API | ID Token 自带 |

```mermaid
graph TD
    A[OpenID Connect] --> B[OAuth 2.0 授权层]
    A --> C[身份认证层 新增]
    
    B --> B1["Access Token<br/>访问资源的凭证"]
    C --> C1["ID Token（JWT）<br/>包含用户身份信息<br/>sub: 用户唯一标识<br/>name: 用户名<br/>email: 邮箱"]
    
    B --> D["解决：第三方能访问什么"]
    C --> E["解决：用户是谁"]

    style C fill:#e8f5e9
    style C1 fill:#c8e6c9
```

> 💡 简单理解：**OIDC = OAuth 2.0 + 用户身份信息**

---

## 九、本章小结

```mermaid
mindmap
  root((OAuth 2.0))
    解决的问题
      安全地授权第三方访问资源
      用户不需要把密码给第三方
    四个角色
      资源拥有者（用户）
      客户端（第三方应用）
      授权服务器
      资源服务器
    四种授权模式
      授权码模式 ⭐最重要
      隐式模式（已废弃）
      密码模式（仅限自家应用）
      客户端凭证模式（机器对机器）
    授权码模式要点
      两步走：先获取Code，再换Token
      Code一次性使用
      client_secret只在后端
    PKCE扩展
      移动端/SPA安全增强
      code_verifier + code_challenge
    OIDC
      OAuth 2.0 + 身份认证
      新增 ID Token
```

---

> 📖 **上一篇**：[03-Token与JWT详解](./03-Token与JWT详解.md)  
> 📖 **下一篇**：[05-SSO单点登录详解](./05-SSO单点登录详解.md) —— 了解企业级的统一登录方案
