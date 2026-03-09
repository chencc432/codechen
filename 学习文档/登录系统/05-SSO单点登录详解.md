# 🔗 05 - SSO 单点登录详解

> 在企业中，员工往往要使用很多内部系统（OA、邮箱、CRM、项目管理...）。如果每个系统都要单独登录一次，那会疯掉的。SSO（单点登录）解决的就是"登录一次，访问所有"的问题。

---

## 一、什么是 SSO？

### 1.1 生活类比

想象你去一个大型游乐园：

- **没有 SSO**：每个游乐项目都要单独买票、排队验票 → 每个系统都要登录
- **有 SSO**：在门口买一张通票（手环），进任何项目只需要扫一下手环 → 登录一次，通行所有系统

### 1.2 技术定义

**SSO（Single Sign-On，单点登录）** 是指用户只需要在一个地方（认证中心）登录一次，就可以访问所有相互信任的系统，无需重复登录。

```mermaid
graph TD
    subgraph 没有SSO
        U1[用户] -->|登录| A1[OA系统]
        U1 -->|再次登录| B1[邮箱系统]
        U1 -->|又登录| C1[CRM系统]
        U1 -->|还要登录| D1[项目管理]
    end

    subgraph 有SSO ✅
        U2[用户] -->|只登录一次| SSO[🔐 SSO认证中心]
        SSO -->|自动通行| A2[OA系统]
        SSO -->|自动通行| B2[邮箱系统]
        SSO -->|自动通行| C2[CRM系统]
        SSO -->|自动通行| D2[项目管理]
    end

    style SSO fill:#c8e6c9
```

---

## 二、SSO 的核心概念

| 概念 | 说明 |
|------|------|
| **认证中心（IdP）** | Identity Provider，统一的登录服务，负责验证用户身份 |
| **服务提供者（SP）** | Service Provider，各个业务系统，依赖认证中心判断用户是否登录 |
| **全局会话** | 用户在认证中心的登录状态 |
| **局部会话** | 用户在各个业务系统的登录状态 |
| **令牌（Ticket/Token）** | 认证中心签发的凭证，业务系统用它来验证用户身份 |

---

## 三、SSO 登录流程

### 3.1 首次登录（系统A）

```mermaid
sequenceDiagram
    participant 用户 as 👤 用户浏览器
    participant 系统A as 📱 系统A（OA）
    participant SSO as 🔐 SSO认证中心

    用户->>系统A: ① 访问 OA 系统
    系统A->>系统A: ② 检查：没有局部会话
    系统A-->>用户: ③ 302 重定向到 SSO 登录页<br/>Location: sso.com/login?redirect=oa.com
    
    用户->>SSO: ④ 浏览器跳转到 SSO 登录页
    SSO->>SSO: ⑤ 检查：没有全局会话
    SSO-->>用户: ⑥ 显示登录表单
    
    用户->>SSO: ⑦ 提交用户名 + 密码
    SSO->>SSO: ⑧ 验证凭证
    SSO->>SSO: ⑨ 创建全局会话（设置 SSO Cookie）
    SSO->>SSO: ⑩ 生成 Ticket/Token
    SSO-->>用户: ⑪ 302 重定向回 OA<br/>Location: oa.com?ticket=ST-xxx
    
    用户->>系统A: ⑫ 浏览器跳转回 OA（携带 ticket）
    系统A->>SSO: ⑬ 后端验证 ticket 有效性
    SSO-->>系统A: ⑭ ticket 有效，返回用户信息
    系统A->>系统A: ⑮ 创建局部会话
    系统A-->>用户: ⑯ 登录成功！显示 OA 页面
```

### 3.2 再访问另一个系统（系统B）—— 无需再次登录！

```mermaid
sequenceDiagram
    participant 用户 as 👤 用户浏览器
    participant 系统B as 📱 系统B（邮箱）
    participant SSO as 🔐 SSO认证中心

    用户->>系统B: ① 访问邮箱系统
    系统B->>系统B: ② 检查：没有局部会话
    系统B-->>用户: ③ 302 重定向到 SSO
    
    用户->>SSO: ④ 浏览器跳转到 SSO
    SSO->>SSO: ⑤ 检查：已有全局会话 ✅（之前登录OA时创建的）
    Note over SSO: 不需要再次输入密码！
    SSO->>SSO: ⑥ 直接生成新的 Ticket
    SSO-->>用户: ⑦ 302 重定向回邮箱系统<br/>Location: mail.com?ticket=ST-yyy
    
    用户->>系统B: ⑧ 浏览器跳转回邮箱（携带 ticket）
    系统B->>SSO: ⑨ 后端验证 ticket
    SSO-->>系统B: ⑩ 有效，返回用户信息
    系统B->>系统B: ⑪ 创建局部会话
    系统B-->>用户: ⑫ 自动登录成功！显示邮箱页面 🎉
```

> 🎉 用户的体验是：访问邮箱系统时，页面闪了一下（重定向），然后就直接进去了，完全不需要输入密码！

### 3.3 全局会话与局部会话的关系

```mermaid
graph TD
    subgraph SSO认证中心
        GS["🔐 全局会话<br/>Global Session<br/>SSO Cookie: sso_token=xxx"]
    end
    
    subgraph 各业务系统
        LS1["📱 系统A 局部会话<br/>Cookie: oa_session=aaa"]
        LS2["📱 系统B 局部会话<br/>Cookie: mail_session=bbb"]
        LS3["📱 系统C 局部会话<br/>Cookie: crm_session=ccc"]
    end
    
    GS -.->|信任关系| LS1
    GS -.->|信任关系| LS2
    GS -.->|信任关系| LS3

    style GS fill:#c8e6c9
```

**关键理解**：
- **全局会话**存在于 SSO 认证中心（`sso.company.com` 域的 Cookie）
- **局部会话**存在于各业务系统（各自域的 Cookie/Session）
- 全局会话是"母会话"，局部会话是"子会话"
- 全局会话存在 → 可以自动创建局部会话（无需输入密码）

---

## 四、SSO 登出流程

登出需要"单点登出"（Single Logout），即在一个系统登出后，所有系统都要登出。

```mermaid
sequenceDiagram
    participant 用户 as 👤 用户
    participant 系统A as 📱 系统A
    participant SSO as 🔐 SSO认证中心
    participant 系统B as 📱 系统B
    participant 系统C as 📱 系统C

    用户->>系统A: ① 点击"退出登录"
    系统A->>系统A: ② 销毁系统A的局部会话
    系统A->>SSO: ③ 通知 SSO 用户要登出
    
    SSO->>SSO: ④ 销毁全局会话
    SSO->>系统B: ⑤ 通知系统B销毁局部会话
    SSO->>系统C: ⑥ 通知系统C销毁局部会话
    系统B->>系统B: ⑦ 销毁局部会话
    系统C->>系统C: ⑧ 销毁局部会话
    
    SSO-->>用户: ⑨ 重定向到登录页
```

---

## 五、CAS 协议 —— 经典的 SSO 实现

**CAS（Central Authentication Service）** 是耶鲁大学开发的一套开源 SSO 协议，是最经典的 SSO 实现方案。

### 5.1 CAS 核心概念

| 术语 | 说明 |
|------|------|
| **TGT** (Ticket Granting Ticket) | 全局票据，存储在 SSO 服务端，代表全局会话 |
| **TGC** (Ticket Granting Cookie) | 全局 Cookie，存储在浏览器中，关联到 TGT |
| **ST** (Service Ticket) | 服务票据，一次性的，用于业务系统验证用户身份 |

### 5.2 CAS 票据流程

```mermaid
graph TD
    A["用户登录 SSO"] --> B["SSO 创建 TGT（全局票据）"]
    B --> C["SSO 设置 TGC Cookie 到浏览器"]
    C --> D["SSO 生成 ST（服务票据）"]
    D --> E["用户携带 ST 回到业务系统"]
    E --> F["业务系统拿 ST 到 SSO 验证"]
    F --> G["SSO 验证 ST 有效"]
    G --> H["业务系统创建局部会话"]
    
    I["用户访问另一个系统"] --> J["重定向到 SSO"]
    J --> K{"浏览器有 TGC Cookie?"}
    K -->|有| L["SSO 根据 TGC 找到 TGT"]
    L --> M["不需要再次登录，直接签发新的 ST"]
    K -->|无| N["显示登录页面"]

    style M fill:#c8e6c9
    style B fill:#e3f2fd
```

---

## 六、基于 OAuth 2.0 的 SSO

现代系统更倾向于使用 OAuth 2.0 / OIDC 来实现 SSO。

### 6.1 与 CAS 的对比

| 对比项 | CAS | OAuth 2.0 / OIDC |
|--------|-----|-------------------|
| 定位 | 专用于 SSO | 通用的授权/认证框架 |
| 协议复杂度 | 相对简单 | 更灵活但更复杂 |
| Token 格式 | Service Ticket（不透明字符串） | JWT（自包含信息） |
| 跨组织 | 主要用于组织内部 | 支持跨组织（如第三方登录） |
| 移动端支持 | 较弱 | 完善 |
| 生态 | 较老 | 丰富，各大厂都支持 |

### 6.2 基于 OIDC 的 SSO 流程

```mermaid
sequenceDiagram
    participant 用户 as 👤 用户
    participant 应用A as 📱 应用A
    participant IdP as 🔐 OIDC Provider<br/>（认证中心）
    participant 应用B as 📱 应用B

    Note over 用户,应用B: 首次登录应用A
    用户->>应用A: 访问应用A
    应用A-->>用户: 重定向到 IdP（scope=openid profile）
    用户->>IdP: 登录（输入密码）
    IdP-->>用户: 重定向回应用A + 授权码
    应用A->>IdP: 用授权码换取 Token
    IdP-->>应用A: Access Token + ID Token（JWT）
    应用A-->>用户: 登录成功

    Note over 用户,应用B: 再访问应用B（无需再次登录）
    用户->>应用B: 访问应用B
    应用B-->>用户: 重定向到 IdP
    用户->>IdP: 已有 SSO Session ✅
    IdP-->>用户: 直接重定向回应用B + 授权码
    应用B->>IdP: 用授权码换取 Token
    IdP-->>应用B: Access Token + ID Token
    应用B-->>用户: 自动登录成功 🎉
```

---

## 七、同域 SSO vs 跨域 SSO

### 7.1 同域 SSO（简单场景）

当所有子系统在同一个主域下时（如 `*.company.com`），可以通过共享 Cookie 实现简单的 SSO。

```mermaid
graph TD
    subgraph "company.com 域"
        A["sso.company.com<br/>认证中心"] 
        B["oa.company.com<br/>OA系统"]
        C["mail.company.com<br/>邮箱系统"]
        D["crm.company.com<br/>CRM系统"]
    end
    
    E["Cookie: token=xxx<br/>Domain=.company.com"] --> A
    E --> B
    E --> C
    E --> D
    
    F["✅ 设置 Cookie 的 Domain 为 .company.com<br/>所有子域都可以读取这个 Cookie"]

    style E fill:#c8e6c9
    style F fill:#fff3e0
```

### 7.2 跨域 SSO（复杂场景）

当子系统在不同域名下时（如 `app1.com`、`app2.com`），Cookie 无法共享，必须通过重定向方式（CAS / OAuth）实现。

```mermaid
graph TD
    subgraph 不同域名
        A["app1.com"]
        B["app2.com"]
        C["app3.com"]
    end
    
    SSO["sso.company.com<br/>认证中心"]
    
    A -->|重定向| SSO
    B -->|重定向| SSO
    C -->|重定向| SSO
    SSO -->|携带Ticket重定向回| A
    SSO -->|携带Ticket重定向回| B
    SSO -->|携带Ticket重定向回| C

    D["❌ Cookie无法跨域共享<br/>必须通过重定向+Ticket方式"]
    
    style D fill:#ffcdd2
    style SSO fill:#c8e6c9
```

---

## 八、常见的 SSO 解决方案

| 方案 | 类型 | 特点 | 适用场景 |
|------|------|------|----------|
| **CAS** | 开源协议 | 经典、简单、成熟 | 高校、传统企业 |
| **Keycloak** | 开源平台 | 功能全面、支持 OIDC/SAML | 中大型企业 ✅ |
| **Auth0** | 商业服务 | 开箱即用、集成丰富 | 快速开发 |
| **Azure AD** | 微软云 | 与微软生态深度集成 | 微软技术栈 |
| **Okta** | 商业服务 | 企业级 IAM 平台 | 大型企业 |
| **自建** | 自研 | 完全可控 | 有技术能力的团队 |

---

## 九、本章小结

```mermaid
mindmap
  root((SSO 单点登录))
    核心思想
      登录一次,通行所有系统
      认证中心统一管理
    关键概念
      认证中心 IdP
      服务提供者 SP
      全局会话 vs 局部会话
    登录流程
      无全局会话 → 跳转登录
      有全局会话 → 自动签发Ticket
      业务系统验证Ticket → 创建局部会话
    登出流程
      通知所有系统销毁局部会话
      销毁全局会话
    实现协议
      CAS（经典）
      OAuth 2.0 / OIDC（现代）
    同域 vs 跨域
      同域: 共享Cookie
      跨域: 重定向+Ticket
```

---

> 📖 **上一篇**：[04-OAuth2.0协议详解](./04-OAuth2.0协议详解.md)  
> 📖 **下一篇**：[06-第三方登录实现](./06-第三方登录实现.md) —— 了解微信/GitHub 等第三方登录的实现细节
