# 06 — Docker 网络详解

> 容器之间怎么互相通信？容器怎么访问外网？外部怎么访问容器？看完这篇就全懂了。

---

## 一、Docker 网络模型概述

### 1.1 核心问题

每个容器都是一个隔离的"小房间"，它们之间默认是互相看不到的。Docker 网络要解决三个问题：

```
1. 容器 ↔ 容器：同一台机器上的容器怎么互相通信？
2. 容器 ↔ 宿主机：容器怎么访问宿主机的服务？
3. 容器 ↔ 外网：容器怎么上网？外部怎么访问容器？
```

### 1.2 Docker 的四种网络模式

| 模式 | 命令参数 | 特点 | 使用场景 |
|------|---------|------|---------|
| **bridge** | `--network bridge` | 默认模式，创建虚拟网桥 | 大多数场景 |
| **host** | `--network host` | 直接使用宿主机网络 | 对网络性能要求高 |
| **none** | `--network none` | 完全没有网络 | 安全隔离 |
| **container** | `--network container:xxx` | 和另一个容器共享网络 | Sidecar 模式 |

---

## 二、Bridge 网络（默认模式）

### 2.1 原理

Docker 安装后会创建一个名为 `docker0` 的虚拟网桥。每个容器都连到这个网桥上，就像插了一根虚拟网线。

```
┌──────────────────────────────────────────────┐
│                    宿主机                      │
│                                              │
│    ┌──────────┐    ┌──────────┐              │
│    │ 容器 A    │    │ 容器 B    │              │
│    │ 172.17.0.2│   │ 172.17.0.3│              │
│    └─────┬────┘    └─────┬────┘              │
│          │               │                   │
│    ┌─────┴───────────────┴────┐              │
│    │      docker0（网桥）       │              │
│    │       172.17.0.1         │              │
│    └───────────┬──────────────┘              │
│                │                             │
│    ┌───────────┴──────────────┐              │
│    │    eth0（物理网卡）        │              │
│    │    192.168.1.100         │              │
│    └──────────────────────────┘              │
└──────────────────────────────────────────────┘
```

### 2.2 验证默认网络

```bash
# 查看所有 Docker 网络
docker network ls

# 输出
NETWORK ID     NAME      DRIVER    SCOPE
a1b2c3d4e5f6   bridge    bridge    local      ← 默认网桥
b2c3d4e5f6g7   host      host      local      ← 宿主机模式
c3d4e5f6g7h8   none      null      local      ← 无网络模式

# 查看 bridge 网络的详细信息
docker network inspect bridge
```

### 2.3 容器间通信（默认 bridge 的限制）

```bash
# 启动两个容器
docker run -d --name web nginx
docker run -d --name app python:3.11-slim sleep infinity

# 查看 IP
docker inspect --format='{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' web
# 172.17.0.2

docker inspect --format='{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' app
# 172.17.0.3

# 从 app 容器 ping web 容器的 IP — 可以通
docker exec app ping -c 3 172.17.0.2

# 但用容器名 ping — 不通！
docker exec app ping -c 3 web
# ping: web: Name or service not known
```

**默认 bridge 网络的问题：不支持容器名 DNS 解析！**

### 2.4 自定义 Bridge 网络（推荐！）

```bash
# 创建自定义网络
docker network create my-net

# 在自定义网络中启动容器
docker run -d --name web --network my-net nginx
docker run -d --name app --network my-net python:3.11-slim sleep infinity

# 现在用容器名可以 ping 通了！
docker exec app ping -c 3 web
# PING web (172.18.0.2): 56 data bytes
# 64 bytes from 172.18.0.2: seq=0 ttl=64 time=0.150 ms
```

**自定义网络的优势**：

| 特性 | 默认 bridge | 自定义 bridge |
|------|------------|-------------|
| 容器名 DNS 解析 | ❌ 不支持 | ✅ 支持 |
| 网络隔离 | 所有容器同网段 | 不同网络互相隔离 |
| 动态连接/断开 | ❌ 需重建容器 | ✅ 运行时可操作 |

### 2.5 自定义网络的高级配置

```bash
# 创建自定义网段
docker network create \
    --driver bridge \
    --subnet 10.10.0.0/24 \
    --gateway 10.10.0.1 \
    my-custom-net

# 指定容器 IP
docker run -d --name web \
    --network my-custom-net \
    --ip 10.10.0.100 \
    nginx
```

---

## 三、Host 网络

### 3.1 原理

容器直接使用宿主机的网络栈，没有网络隔离。

```
┌──────────────────────────────┐
│            宿主机              │
│                              │
│    ┌──────────────────┐      │
│    │ 容器（host 模式）  │      │
│    │ 直接用宿主机 IP    │      │
│    │ 直接用宿主机端口   │      │
│    └──────────────────┘      │
│                              │
│    eth0: 192.168.1.100       │
└──────────────────────────────┘
```

### 3.2 使用方法

```bash
# 使用 host 网络（注意：不需要 -p 端口映射了！）
docker run -d --network host nginx

# 容器里的 Nginx 直接监听宿主机的 80 端口
# 直接访问 http://宿主机IP:80
```

### 3.3 适用场景

- 对网络性能有极高要求（省去 NAT 转换的开销）
- 需要监听大量端口
- 网络排查工具容器

**注意**：macOS 和 Windows 的 Docker Desktop 不完全支持 host 模式（因为 Docker 实际运行在虚拟机里）。

---

## 四、None 网络

```bash
# 完全隔离，没有任何网络
docker run -d --network none --name isolated alpine sleep infinity

# 验证：没有网卡
docker exec isolated ip addr
# 只有 lo（回环接口），没有 eth0
```

**适用场景**：运行纯计算任务（不需要网络），安全隔离环境。

---

## 五、Container 网络（共享网络）

```bash
# 容器 B 共享容器 A 的网络
docker run -d --name container-a nginx
docker run -d --network container:container-a --name container-b alpine sleep infinity

# 两个容器共享同一个网络命名空间
# 在 B 里 localhost 就能访问 A 的服务
docker exec container-b wget -qO- http://localhost:80
```

**适用场景**：Sidecar 模式（如日志收集器、代理）。在 Kubernetes 中，同一个 Pod 里的容器就是这种模式。

---

## 六、端口映射详解

### 6.1 基本语法

```bash
# 格式：-p 宿主机端口:容器端口
docker run -d -p 8080:80 nginx
# 访问宿主机的 8080 → 转发到容器的 80

# 映射到指定 IP
docker run -d -p 127.0.0.1:8080:80 nginx
# 只能通过 localhost:8080 访问

# 映射多个端口
docker run -d -p 80:80 -p 443:443 nginx

# 随机映射
docker run -d -P nginx
# Docker 自动分配宿主机端口

# 查看端口映射
docker port my-nginx
# 80/tcp -> 0.0.0.0:8080
```

### 6.2 端口映射原理

```
外部请求 → 宿主机 8080 → iptables NAT → 容器 172.17.0.2:80

实际上 Docker 在 iptables 中添加了 DNAT 规则：
-A DOCKER -p tcp --dport 8080 -j DNAT --to-destination 172.17.0.2:80
```

### 6.3 端口冲突处理

```bash
# 如果宿主机 80 端口已被占用
docker run -d -p 80:80 nginx
# Error: bind: address already in use

# 解决：换一个宿主机端口
docker run -d -p 8080:80 nginx

# 或者用 0 让 Docker 自动分配
docker run -d -p 0:80 nginx
# 查看分配了哪个端口
docker port <container_id>
```

---

## 七、容器 DNS 与服务发现

### 7.1 自定义网络中的 DNS

在自定义 bridge 网络中，Docker 内置了一个 DNS 服务器（127.0.0.11）：

```bash
docker network create app-net

docker run -d --name db --network app-net mysql:8.0 \
    -e MYSQL_ROOT_PASSWORD=secret

docker run -d --name app --network app-net my-app

# app 容器里可以直接用 "db" 作为主机名连接数据库
# 连接字符串：mysql://root:secret@db:3306/mydb
```

### 7.2 网络别名

```bash
# 给容器设置网络别名
docker run -d --name mysql-primary --network app-net \
    --network-alias db \
    mysql:8.0

docker run -d --name mysql-secondary --network app-net \
    --network-alias db \
    mysql:8.0

# "db" 会轮询解析到两个 MySQL 容器（简单的负载均衡）
```

---

## 八、网络管理命令

```bash
# 列出所有网络
docker network ls

# 创建网络
docker network create my-net
docker network create --driver bridge --subnet 10.10.0.0/24 my-net

# 查看网络详情
docker network inspect my-net

# 将容器连接到网络
docker network connect my-net my-container

# 将容器从网络断开
docker network disconnect my-net my-container

# 删除网络
docker network rm my-net

# 清理所有未使用的网络
docker network prune
```

---

## 九、常用网络排查技巧

```bash
# 1. 查看容器 IP
docker inspect --format='{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' my-container

# 2. 进入容器排查
docker exec -it my-container sh
# 然后用 ping、curl、wget 等工具

# 3. 使用网络排查工具容器
docker run -it --rm --network container:my-container nicolaka/netshoot
# netshoot 自带 curl, ping, dig, nslookup, tcpdump 等工具

# 4. 查看 iptables 规则
sudo iptables -t nat -L -n

# 5. 查看 docker0 网桥
ip addr show docker0
brctl show docker0
```

---

## 十、本章小结

```
✅ Docker 有四种网络模式：bridge（默认）、host、none、container
✅ 默认 bridge 不支持容器名 DNS，自定义 bridge 支持
✅ 生产环境推荐使用自定义 bridge 网络
✅ -p 8080:80 将宿主机 8080 映射到容器 80
✅ host 模式性能最好，但没有网络隔离
✅ 自定义网络自带 DNS 服务发现，容器名即主机名
✅ docker network connect/disconnect 可以动态管理
```

---

> 下一篇：[07-Docker数据管理与持久化](./07-Docker数据管理与持久化.md) — 容器的数据怎么不丢？
