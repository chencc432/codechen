# 💾 Volume 与持久化存储

## 一、先理解问题：容器里的数据为什么会丢？

在学习 Volume 之前，我们需要先理解一个根本问题：**容器的文件系统是临时的**。

### 什么叫"临时的"？

你可以把容器想象成一个**一次性的沙盒**：

```
普通电脑（虚拟机）：
┌──────────────────────────────┐
│  硬盘（永久保存数据）          │  ← 关机重启后，文件还在
│  ├── /home/user/文档/         │
│  ├── /var/log/日志/           │
│  └── /data/数据库文件/         │
└──────────────────────────────┘

容器：
┌──────────────────────────────┐
│  容器文件层（临时的！）         │  ← 容器重启/删除后，文件全部消失
│  ├── /app/上传的文件/          │  ❌ 没了
│  ├── /var/log/运行日志/        │  ❌ 没了
│  └── /data/数据库文件/         │  ❌ 没了
└──────────────────────────────┘
```

**为什么会这样？** 因为容器的设计哲学是"用完即扔"——每次启动都是一个全新的环境。这对于"无状态"的 Web 应用来说没问题（比如 Nginx 只负责转发请求），但对于需要保存数据的应用来说就是灾难（比如 MySQL 数据库）。

### 实际场景举例

| 场景 | 不用 Volume 会怎样？ |
|------|---------------------|
| MySQL 数据库 | Pod 重启后，所有表和数据全部丢失 |
| 用户上传的图片 | Pod 重建后，用户上传过的文件全没了 |
| 应用运行日志 | 容器崩溃后，无法查看崩溃前的日志 |
| 多个容器共享文件 | 同一个 Pod 里的两个容器无法互相读取文件 |

**Volume（卷）就是用来解决这些问题的。**

---

## 二、Volume 是什么？通俗理解

### 类比：U盘

可以把 Volume 想象成一个 **U盘**：

```
没有 Volume 的容器：
┌──────────────┐
│   容器        │  数据写在容器"内部"
│  ┌─────────┐ │  容器删了 → 数据也没了
│  │ 数据文件 │ │
│  └─────────┘ │
└──────────────┘

有 Volume 的容器：
┌──────────────┐      ┌────────────────┐
│   容器        │      │   Volume       │
│  ┌─────────┐ │      │  (像一个 U盘)   │
│  │ /data ──┼─┼──────┤  真正存数据     │
│  └─────────┘ │      │  容器删了也不丢  │
└──────────────┘      └────────────────┘
```

- 容器里的 `/data` 目录，实际指向了外部的 Volume
- 容器被删除、重建，只要 Volume 还在，数据就还在
- 就像你的电脑坏了换一台，只要 U盘没丢，数据就能恢复

### Volume 的本质

Volume 本质上就是一个**目录**（或文件），它被"挂载"（mount）到容器内部的某个路径上。容器对这个路径的读写操作，实际上都是在操作这个外部目录。

**"挂载"是什么意思？**
> 挂载 = 把一个存储设备"接入"到文件系统的某个目录上。就像你插入 U盘后，电脑上出现一个 `E:\` 盘符一样。在 Linux 中，这个动作叫 `mount`。容器中的挂载也是同样的道理：把外部存储"接入"到容器内的某个目录。

---

## 三、Kubernetes 存储体系全景

Kubernetes 提供了多种 Volume 类型，按用途可以分为以下几类：

```
┌────────────────────────────────────────────────────────────────────┐
│                     Kubernetes 存储体系                             │
├────────────────────────────────────────────────────────────────────┤
│                                                                    │
│  ① 临时存储（Pod 删除后数据丢失）                                    │
│  ┌──────────────────────────────────────────────────┐              │
│  │  emptyDir  →  同一个 Pod 内多个容器共享数据        │              │
│  └──────────────────────────────────────────────────┘              │
│                                                                    │
│  ② 节点存储（数据存在 Node 的硬盘上）                                │
│  ┌──────────────────────────────────────────────────┐              │
│  │  hostPath  →  直接使用 Node 节点上的目录/文件      │              │
│  └──────────────────────────────────────────────────┘              │
│                                                                    │
│  ③ 配置存储（把配置/密钥注入容器）                                    │
│  ┌──────────────────────────────────────────────────┐              │
│  │  ConfigMap / Secret  →  挂载为文件给容器使用       │              │
│  └──────────────────────────────────────────────────┘              │
│                                                                    │
│  ④ 持久化存储（真正的"数据库级别"持久化，重点！）                      │
│  ┌──────────────────────────────────────────────────┐              │
│  │  PV + PVC + StorageClass                          │              │
│  │  → 数据独立于 Pod 生命周期，安全可靠               │              │
│  └──────────────────────────────────────────────────┘              │
│                                                                    │
└────────────────────────────────────────────────────────────────────┘
```

**选择建议**：
- 临时缓存、容器间传数据 → `emptyDir`
- 读取节点日志、特殊调试 → `hostPath`（生产环境少用）
- 注入配置文件 → `ConfigMap` / `Secret`（前一章已介绍）
- **数据库、文件存储等需要持久化的场景 → PV + PVC**（本章重点）

---

## 四、Volume 类型详解

### 4.1 emptyDir —— 临时共享存储

#### 是什么？

`emptyDir` 是最简单的 Volume 类型。当 Pod 被创建时，Kubernetes 会在 Node 节点上创建一个**空目录**，Pod 内的所有容器都可以读写这个目录。

#### 关键特点

| 特点 | 说明 |
|------|------|
| 生命周期 | 和 Pod 一样长——Pod 删除，数据就没了 |
| 共享范围 | 同一个 Pod 内的多个容器之间共享 |
| 存储位置 | 默认存在 Node 的硬盘上，也可以用内存（更快但容量有限） |
| 初始状态 | 空的（empty），所以叫 emptyDir |

#### 典型场景

```
┌─────────────── 一个 Pod ──────────────────┐
│                                            │
│  容器A（写日志）         容器B（读日志）      │
│  ┌──────────┐          ┌──────────┐       │
│  │ 写 →     │          │     → 读 │       │
│  │ /shared/ ├──────────┤ /shared/ │       │
│  └──────────┘          └──────────┘       │
│         │                    │             │
│         └────────┬───────────┘             │
│                  ▼                         │
│         ┌──────────────┐                  │
│         │  emptyDir    │                  │
│         │  (共享目录)   │                  │
│         └──────────────┘                  │
└────────────────────────────────────────────┘
```

#### 完整示例

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: emptydir-demo
spec:
  containers:
  # 第一个容器：负责往共享目录写数据
  - name: writer
    image: busybox
    command: ["sh", "-c", "echo 'Hello from writer' > /data/hello.txt && sleep 3600"]
    volumeMounts:                     # 声明要挂载哪个 Volume
    - name: shared-data               # Volume 的名字（和下面 volumes 中的 name 对应）
      mountPath: /data                 # 挂载到容器内的哪个目录

  # 第二个容器：从共享目录读数据
  - name: reader
    image: busybox
    command: ["sh", "-c", "cat /data/hello.txt && sleep 3600"]
    volumeMounts:
    - name: shared-data               # 同一个 Volume 名字 → 两个容器共享同一个目录
      mountPath: /data

  # 定义 Volume（在 Pod 级别声明，所有容器都可以引用）
  volumes:
  - name: shared-data                  # Volume 名字
    emptyDir: {}                       # 类型是 emptyDir，{} 表示使用默认配置
```

> **YAML 关键配置说明**：
> - `volumes`：在 Pod 级别定义"有哪些 Volume 可用"
> - `volumeMounts`：在容器级别定义"要使用哪个 Volume，挂载到容器的哪个路径"
> - 两者通过 `name` 字段关联

#### 使用内存作为存储介质

```yaml
volumes:
- name: cache
  emptyDir:
    medium: Memory          # 用内存代替硬盘，速度极快
    sizeLimit: 100Mi        # 限制最大使用 100MB 内存
```

> 使用内存的好处是读写速度很快，适合做缓存。但要注意内存是有限的，不要设得太大。

---

### 4.2 hostPath —— 使用节点本地路径

#### 是什么？

`hostPath` 把 Node 节点（宿主机）上的一个目录或文件，直接挂载到 Pod 容器中。

#### 通俗理解

```
┌─── Node 节点（宿主机）──────────────────────────┐
│                                                  │
│  宿主机目录: /var/log/app-logs/                   │
│        ▲                                         │
│        │  直接挂载（共用同一个目录）                 │
│        ▼                                         │
│  ┌─── Pod 容器 ────────────┐                     │
│  │  容器目录: /var/log/app  │                     │
│  │  （实际操作的是宿主机的   │                     │
│  │   /var/log/app-logs/）   │                     │
│  └──────────────────────────┘                    │
│                                                  │
└──────────────────────────────────────────────────┘
```

#### 完整示例

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: hostpath-demo
spec:
  containers:
  - name: app
    image: nginx
    volumeMounts:
    - name: host-logs
      mountPath: /var/log/app          # 容器内的路径
  volumes:
  - name: host-logs
    hostPath:
      path: /var/log/app-logs          # Node 节点上的真实路径
      type: DirectoryOrCreate          # 如果目录不存在，自动创建
```

#### hostPath type 选项

| type 值 | 含义 |
|---------|------|
| `""` | 不做任何检查（默认值） |
| `DirectoryOrCreate` | 目录不存在则自动创建 |
| `Directory` | 目录必须已经存在，否则 Pod 启动失败 |
| `FileOrCreate` | 文件不存在则自动创建 |
| `File` | 文件必须已经存在 |
| `Socket` | Unix Socket 必须已经存在 |

#### ⚠️ hostPath 的问题

| 问题 | 说明 |
|------|------|
| **安全风险** | 容器可以访问 Node 上的任意文件，可能被恶意利用 |
| **不可移植** | Pod 调度到不同 Node 时，数据不会跟着走 |
| **难以管理** | 需要手动在每个 Node 上创建目录 |

> **结论**：`hostPath` 主要用于开发测试、读取节点日志等场景。**生产环境中，请使用 PV/PVC 来做持久化存储。**

---

### 4.3 ConfigMap 和 Secret 作为 Volume

之前章节已经详细介绍过，这里简单回顾：

```yaml
volumes:
# 把 ConfigMap 挂载为文件
- name: config
  configMap:
    name: my-config            # ConfigMap 的名字

# 把 Secret 挂载为文件
- name: secret
  secret:
    secretName: my-secret      # Secret 的名字
```

> 详细用法参考 [ConfigMap 与 Secret](./04-configmap-secret.md)

---

## 五、PV 和 PVC —— 持久化存储（重点）

这是 Kubernetes 存储体系中最重要的部分。如果你只记一个知识点，就记住：**PV 是存储资源，PVC 是使用申请，Pod 通过 PVC 来使用 PV。**

### 5.1 为什么需要 PV/PVC？直接用 hostPath 不行吗？

假设你要给 MySQL 数据库做持久化存储：

```
方案1：直接在 Pod 中配置存储（不推荐）
────────────────────────────────────
Pod YAML 里直接写：用 NFS，地址是 192.168.1.100，路径是 /data/mysql

问题：
❌ 开发者需要知道存储的具体细节（NFS 地址、路径等）
❌ 存储细节硬编码在 Pod 配置里，换环境就要改
❌ 100 个 Pod 都要写一遍存储配置
❌ 没有统一管理，容易出错

方案2：使用 PV + PVC（推荐）
────────────────────────────────────
管理员创建 PV：这里有一块 10GB 的 NFS 存储可以用
开发者创建 PVC：我需要 5GB 的存储空间
Kubernetes 自动匹配：把 PV 分配给 PVC
Pod 引用 PVC：我要用这个 PVC

好处：
✅ 开发者不需要知道底层存储细节
✅ 管理员统一管理存储资源
✅ 解耦——存储的"提供"和"使用"分离
✅ 更安全、更规范
```

### 5.2 PV、PVC、StorageClass 三者的关系

#### 类比：租房子

这是理解 PV/PVC/StorageClass 最好的类比：

```
┌─────────────────────────────────────────────────────────────┐
│                                                             │
│  StorageClass（存储类）≈ 房屋中介/开发商                      │
│  定义"什么样的存储"，可以自动创建 PV                           │
│                                                             │
│        │ 自动创建（动态供应）                                 │
│        ▼                                                    │
│                                                             │
│  PV（PersistentVolume）≈ 一套具体的房子                      │
│  实际的存储资源，有大小、类型、位置等属性                       │
│  由管理员预先创建 或 StorageClass 自动创建                     │
│                                                             │
│        ▲                                                    │
│        │ 绑定（系统自动匹配）                                 │
│        ▼                                                    │
│                                                             │
│  PVC（PersistentVolumeClaim）≈ 租房需求单                    │
│  用户说"我需要多大的存储、什么访问模式"                         │
│  系统找到匹配的 PV 并绑定                                     │
│                                                             │
│        ▲                                                    │
│        │ 使用                                                │
│        │                                                    │
│                                                             │
│  Pod  ≈  住进房子的人                                        │
│  通过引用 PVC 来使用存储                                      │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

| 概念 | 角色 | 类比 | 谁来创建？ |
|------|------|------|-----------|
| **PV** | 实际存储资源 | 一套房子（100㎡，朝南，精装修） | 管理员 或 StorageClass 自动创建 |
| **PVC** | 存储使用申请 | 租房需求（要 80㎡以上，朝南） | 开发者 |
| **StorageClass** | 存储分类/自动供应 | 房屋中介（告诉你有哪些类型的房，按需自动建） | 管理员（一次配好即可） |

#### 绑定过程

```
1. 管理员创建 PV（或配置好 StorageClass 自动创建）
   PV: "我是一个 10GB 的 NFS 存储，访问模式 RWO"

2. 开发者创建 PVC
   PVC: "我需要一个 5GB 的存储，访问模式 RWO"

3. Kubernetes 自动匹配
   系统发现 PV（10GB, RWO）满足 PVC（5GB, RWO）的需求
   → 将 PV 和 PVC 绑定在一起

4. Pod 引用 PVC
   Pod: "我要使用 my-pvc 这个 PVC"
   → 容器就能读写 PV 上的数据了
```

### 5.3 访问模式（Access Modes）

访问模式决定了 Volume 可以被多少个节点/Pod 同时访问。

| 模式 | 缩写 | 含义 | 适用场景 |
|------|------|------|---------|
| ReadWriteOnce | **RWO** | 只能被**一个 Node** 上的 Pod 读写 | MySQL、PostgreSQL 等单实例数据库 |
| ReadOnlyMany | **ROX** | 可以被**多个 Node** 上的 Pod 只读挂载 | 共享配置文件、静态资源 |
| ReadWriteMany | **RWX** | 可以被**多个 Node** 上的 Pod 同时读写 | 共享文件存储（NFS）、多副本应用共享数据 |
| ReadWriteOncePod | **RWOP** | 只能被**一个 Pod** 读写（K8s 1.22+） | 需要严格独占的场景 |

> **注意**：不是所有存储都支持所有访问模式。比如 AWS EBS 只支持 RWO，NFS 支持 RWX。

### 5.4 回收策略（Reclaim Policy）

当 PVC 被删除后，PV 上的数据怎么处理？

| 策略 | 含义 | 适用场景 |
|------|------|---------|
| **Retain** | 保留数据，PV 变为 `Released` 状态，需要管理员手动处理 | 生产环境（数据安全第一） |
| **Delete** | 自动删除 PV 和底层存储资源 | 测试环境、临时数据 |
| ~~Recycle~~ | ~~清空数据后重新使用~~ | ~~已废弃，不要使用~~ |

### 5.5 PV 的完整示例

```yaml
apiVersion: v1
kind: PersistentVolume
metadata:
  name: my-pv                          # PV 的名字
spec:
  capacity:
    storage: 10Gi                       # 这个 PV 提供 10GB 存储空间
  accessModes:
  - ReadWriteOnce                       # 访问模式：单节点读写
  persistentVolumeReclaimPolicy: Retain # 回收策略：PVC 删除后保留数据
  storageClassName: manual              # 存储类名（和 PVC 匹配用）

  # --- 以下是底层存储的具体配置 ---

  # 示例1：使用 NFS 网络存储
  nfs:
    server: nfs-server.example.com      # NFS 服务器地址
    path: /exports/data                 # NFS 上的共享目录路径

  # 示例2：使用节点本地路径（仅测试用）
  # hostPath:
  #   path: /mnt/data
```

### 5.6 PVC 的完整示例

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: my-pvc                          # PVC 的名字（Pod 中要引用这个名字）
spec:
  accessModes:
  - ReadWriteOnce                       # 我需要的访问模式
  resources:
    requests:
      storage: 5Gi                      # 我需要 5GB 的存储空间
  storageClassName: manual              # 要匹配的存储类名（必须和 PV 一致）

  # 可选：直接指定要绑定哪个 PV（通常不需要，让系统自动匹配）
  # volumeName: my-pv
```

### 5.7 Pod 使用 PVC

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: pvc-pod
spec:
  containers:
  - name: app
    image: nginx
    volumeMounts:
    - name: data                        # 引用下面 volumes 中定义的名字
      mountPath: /usr/share/nginx/html  # 挂载到容器的这个目录
  volumes:
  - name: data
    persistentVolumeClaim:
      claimName: my-pvc                 # 引用 PVC 的名字
```

### 5.8 PV 的生命周期

```
┌──────────────────────────────────────────────────────────────┐
│                       PV 生命周期                             │
│                                                              │
│  ┌───────────┐  PVC匹配  ┌───────────┐  PVC删除  ┌────────┐│
│  │ Available │ ────────> │  Bound    │ ───────> │Released││
│  │  (可用)    │           │ (已绑定)   │          │(已释放) ││
│  └───────────┘           └───────────┘          └───┬────┘│
│       ▲                                             │      │
│       │                                             ▼      │
│       │              ┌──────────────────────────────────┐  │
│       │              │ 根据回收策略（Reclaim Policy）：   │  │
│       │              │                                  │  │
│       │              │ Retain → 保持 Released 状态      │  │
│       │              │          管理员手动清理后可重新用  │  │
│       │              │                                  │  │
│       │              │ Delete → PV 被自动删除           │  │
│       │              └──────────────────────────────────┘  │
│       │                           │                        │
│       └───────────────────────────┘                        │
│                      (手动清理后重新使用)                     │
│                                                              │
│  Failed 状态: 当自动回收/清理过程中出错时进入此状态             │
└──────────────────────────────────────────────────────────────┘
```

---

## 六、StorageClass —— 自动创建 PV（动态供应）

### 6.1 为什么需要 StorageClass？

前面讲的 PV 都是管理员**手动创建**的（叫"静态供应"）。但如果有 100 个应用都要存储，管理员就要手动创建 100 个 PV，非常累。

**StorageClass 就是来解决这个问题的**——开发者只要创建 PVC，StorageClass 会**自动创建**匹配的 PV。

```
静态供应（没有 StorageClass）：
  管理员手动创建 PV → 开发者创建 PVC → 系统匹配绑定

动态供应（有 StorageClass）：
  开发者创建 PVC → StorageClass 自动创建 PV → 自动绑定
  （管理员只需要配置一次 StorageClass，之后全自动）
```

### 6.2 StorageClass 定义

```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: fast-storage                    # StorageClass 的名字
  annotations:
    # 设为默认 StorageClass（PVC 不指定 storageClassName 时自动使用）
    storageclass.kubernetes.io/is-default-class: "true"
provisioner: kubernetes.io/aws-ebs      # 存储供应商（谁来实际创建存储）
parameters:                             # 供应商特定的参数
  type: gp3                             # AWS EBS 卷类型
  fsType: ext4                          # 文件系统类型
reclaimPolicy: Delete                   # 默认回收策略
volumeBindingMode: WaitForFirstConsumer # 等到 Pod 调度时才创建 PV（推荐）
allowVolumeExpansion: true              # 允许之后扩容（增大存储空间）
```

> **`volumeBindingMode` 的两个选项**：
> - `Immediate`：PVC 创建后立即创建 PV（可能和 Pod 不在同一个可用区）
> - `WaitForFirstConsumer`：等 Pod 调度到某个 Node 后再创建 PV（推荐，避免可用区不匹配）

### 6.3 常见存储供应商（Provisioner）

| 存储类型 | Provisioner | 说明 |
|----------|-------------|------|
| AWS EBS | kubernetes.io/aws-ebs | AWS 弹性块存储，只支持 RWO |
| GCE PD | kubernetes.io/gce-pd | Google 云持久磁盘 |
| Azure Disk | kubernetes.io/azure-disk | Azure 磁盘存储 |
| 本地存储 | kubernetes.io/no-provisioner | 不自动创建，需手动管理 |
| NFS | 需要额外安装 provisioner | 网络文件系统，支持 RWX |

### 6.4 PVC 使用 StorageClass（动态供应）

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: dynamic-pvc
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 20Gi
  storageClassName: fast-storage        # 指定使用哪个 StorageClass
  # 提交后，fast-storage 会自动创建一个 20Gi 的 PV 并绑定
```

---

## 七、完整实战示例

### 示例 1：StatefulSet + 持久化存储（数据库场景）

这是实际工作中最常见的存储场景——为数据库的每个副本创建独立的持久化存储。

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: mysql
spec:
  serviceName: mysql                    # 关联的 Headless Service
  replicas: 3                           # 3 个 MySQL 副本
  selector:
    matchLabels:
      app: mysql
  template:
    metadata:
      labels:
        app: mysql
    spec:
      containers:
      - name: mysql
        image: mysql:8.0
        ports:
        - containerPort: 3306
        env:
        - name: MYSQL_ROOT_PASSWORD
          valueFrom:
            secretKeyRef:
              name: mysql-secret
              key: password
        volumeMounts:
        - name: data
          mountPath: /var/lib/mysql      # MySQL 默认数据目录

  # volumeClaimTemplates 是 StatefulSet 特有的
  # 它会为每个 Pod 自动创建一个独立的 PVC
  # 3 个副本 → 3 个 PVC → 3 个 PV（各自独立的数据）
  volumeClaimTemplates:
  - metadata:
      name: data                        # 和 volumeMounts 中的 name 对应
    spec:
      accessModes: ["ReadWriteOnce"]    # 每个 PVC 独占一个 PV
      storageClassName: fast-storage
      resources:
        requests:
          storage: 20Gi
```

> **为什么用 StatefulSet 而不是 Deployment？**
> - Deployment 的 Pod 是"无差别"的，共享同一个 PVC
> - StatefulSet 的每个 Pod 有固定身份（mysql-0, mysql-1, mysql-2），各自有独立的 PVC
> - 数据库需要每个实例有独立的数据目录，所以必须用 StatefulSet

---

## 八、常用操作命令

```bash
# ============ PV 操作 ============
kubectl get pv                          # 查看所有 PV
kubectl get pv -o wide                  # 查看更多细节（存储类、回收策略等）
kubectl describe pv my-pv               # 查看某个 PV 的详细信息
kubectl apply -f pv.yaml                # 创建 PV
kubectl delete pv my-pv                 # 删除 PV

# ============ PVC 操作 ============
kubectl get pvc                         # 查看当前命名空间的所有 PVC
kubectl get pvc -n my-namespace         # 查看指定命名空间的 PVC
kubectl describe pvc my-pvc             # 查看 PVC 详情（绑定状态、使用的 PV 等）
kubectl apply -f pvc.yaml               # 创建 PVC
kubectl delete pvc my-pvc               # 删除 PVC

# ============ StorageClass 操作 ============
kubectl get storageclass                # 查看所有 StorageClass
kubectl get sc                          # 简写形式
kubectl describe sc fast-storage        # 查看某个 StorageClass 详情

# 设置默认 StorageClass
kubectl patch storageclass fast-storage \
  -p '{"metadata": {"annotations":{"storageclass.kubernetes.io/is-default-class":"true"}}}'

# ============ 排查问题 ============
kubectl get pv,pvc                      # 同时查看 PV 和 PVC 的状态
kubectl get events --sort-by='.lastTimestamp'  # 查看事件，排查绑定失败等问题
```

**常见状态说明**：

| PVC 状态 | 含义 |
|----------|------|
| `Pending` | 等待绑定（可能没有匹配的 PV 或 StorageClass 创建中） |
| `Bound` | 已绑定到某个 PV，可以正常使用 |
| `Lost` | 绑定的 PV 被删除了 |

---

## 九、实践练习

### 练习 1：emptyDir —— 容器间共享数据

```bash
# 创建一个 Pod，包含两个容器共享 emptyDir
cat << EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: emptydir-pod
spec:
  containers:
  - name: producer
    image: busybox
    command: ["sh", "-c", "while true; do date >> /shared/log.txt; sleep 5; done"]
    volumeMounts:
    - name: shared
      mountPath: /shared
  - name: consumer
    image: busybox
    command: ["sh", "-c", "tail -f /shared/log.txt"]
    volumeMounts:
    - name: shared
      mountPath: /shared
  volumes:
  - name: shared
    emptyDir: {}
EOF

# 查看 consumer 容器的输出（应该能看到 producer 写入的时间戳）
kubectl logs emptydir-pod -c consumer -f

# 清理
kubectl delete pod emptydir-pod
```

### 练习 2：PV + PVC 完整流程

```bash
# 第1步：创建 PV（模拟管理员操作）
cat << EOF | kubectl apply -f -
apiVersion: v1
kind: PersistentVolume
metadata:
  name: test-pv
spec:
  capacity:
    storage: 1Gi
  accessModes:
  - ReadWriteOnce
  persistentVolumeReclaimPolicy: Retain
  storageClassName: manual
  hostPath:
    path: /tmp/test-pv
EOF

# 第2步：创建 PVC（模拟开发者操作）
cat << EOF | kubectl apply -f -
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: test-pvc
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 500Mi
  storageClassName: manual
EOF

# 第3步：查看绑定状态（PVC 状态应该是 Bound）
kubectl get pv,pvc

# 第4步：创建使用 PVC 的 Pod
cat << EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: pvc-pod
spec:
  containers:
  - name: app
    image: nginx
    volumeMounts:
    - name: data
      mountPath: /usr/share/nginx/html
  volumes:
  - name: data
    persistentVolumeClaim:
      claimName: test-pvc
EOF

# 第5步：写入数据
kubectl exec pvc-pod -- sh -c 'echo "Hello PVC" > /usr/share/nginx/html/index.html'

# 第6步：验证数据
kubectl exec pvc-pod -- cat /usr/share/nginx/html/index.html

# 第7步：清理（按顺序：先删 Pod，再删 PVC，最后删 PV）
kubectl delete pod pvc-pod
kubectl delete pvc test-pvc
kubectl delete pv test-pv
```

---

## 十、核心概念速查表

| 概念 | 一句话说明 | 生命周期 |
|------|-----------|---------|
| **emptyDir** | Pod 内容器间共享的临时目录 | 随 Pod 消亡 |
| **hostPath** | 直接挂载 Node 节点的目录 | 独立于 Pod（但绑定了具体 Node） |
| **PV** | 集群级别的存储资源（一块"硬盘"） | 独立于 Pod，由管理员管理 |
| **PVC** | 用户对存储的"申请单" | 独立于 Pod，绑定到 PV |
| **StorageClass** | 存储分类 + 动态自动创建 PV | 一直存在，按需创建 PV |
| **volumeMounts** | 容器内的挂载配置 | 定义在 Pod spec 的容器中 |
| **volumes** | Pod 级别的 Volume 声明 | 定义在 Pod spec 中 |

---

## 十一、常见问题

**Q：PVC 一直是 Pending 状态怎么办？**
> 常见原因：① 没有匹配的 PV（容量不够或 storageClassName 不匹配）②StorageClass 的 provisioner 不可用。用 `kubectl describe pvc xxx` 查看 Events 获取详细原因。

**Q：PV 和 PVC 必须在同一个命名空间吗？**
> PV 是**集群级别**资源，不属于任何命名空间。PVC 是**命名空间级别**资源。任何命名空间的 PVC 都可以绑定集群中的 PV。

**Q：Pod 删除后，PVC 和 PV 会被删除吗？**
> Pod 删除不会影响 PVC 和 PV。PVC 需要手动删除（或由 StatefulSet 的删除策略控制）。PV 的去留取决于回收策略（Retain/Delete）。

**Q：可以多个 Pod 共用一个 PVC 吗？**
> 取决于 PV 的访问模式。如果是 RWX（ReadWriteMany），可以多个 Pod 共用。如果是 RWO（ReadWriteOnce），只能在同一个 Node 上的 Pod 使用。

**Q：emptyDir 和 PV 的区别是什么？**
> emptyDir 是临时的，Pod 删了数据就没了；PV 是持久的，Pod 删了数据还在。就像内存 vs 硬盘的区别。

---

## 最佳实践

1. **优先使用 StorageClass 动态供应** —— 避免手动管理大量 PV
2. **生产环境用 Retain 回收策略** —— 防止误删数据
3. **根据场景选择访问模式** —— 数据库用 RWO，共享文件用 RWX
4. **设置资源配额** —— 限制每个命名空间的存储使用量，防止个别应用占满所有存储
5. **监控存储使用率** —— 存储满了会导致 Pod 崩溃
6. **清理时注意顺序** —— 先删 Pod，再删 PVC，最后删 PV

---

## 下一步

- [Namespace - 资源隔离](./06-namespace.md)
