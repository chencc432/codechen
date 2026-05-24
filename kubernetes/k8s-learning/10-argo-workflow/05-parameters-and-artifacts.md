# 📦 参数、制品与数据传递

> 这一篇讲清楚两件事：**参数（parameters）和制品（artifacts）到底有什么区别、怎么写、怎么传、怎么调试**。这是 Argo 写复杂 Workflow 时翻车最多的地方。

## 1. 一句话区分

| 概念 | 用途 | 存在哪 | 大小限制 |
|------|------|--------|----------|
| **Parameter** | 传字符串（版本号、URL、JSON 串） | 存在 etcd 里（Workflow CR 的 status） | 单个字段建议 < 256KB |
| **Artifact** | 传文件 / 目录 | 存在对象存储（S3 / OSS / MinIO） | 取决于对象存储 |

**经验法则**：能用一行字符串说清楚的，用 parameter；要传文件、模型、二进制的，用 artifact。

## 2. Parameters：四种用法

### 2.1 全局参数（workflow 入参）

```yaml
spec:
  arguments:
    parameters:
      - name: version
        value: v1.2.0
      - name: env
        value: staging
```

提交时可以覆盖：

```bash
argo submit wf.yaml -p version=v2.0.0 -p env=prod
```

容器里引用：`{{workflow.parameters.version}}`

### 2.2 模板入参

```yaml
- name: deploy
  inputs:
    parameters:
      - name: image
      - name: replicas
        default: "1"            # 注意：默认值要用字符串
  container:
    image: bitnami/kubectl
    command: [sh, -c]
    args: ["kubectl set image deploy/app app={{inputs.parameters.image}}"]
```

调用时传值：

```yaml
- name: deploy
  template: deploy
  arguments:
    parameters:
      - name: image
        value: "registry.local/app:{{workflow.uid}}"
      - name: replicas
        value: "3"
```

### 2.3 模板输出参数（让下游能用）

三种获取来源：

```yaml
outputs:
  parameters:
    # (1) 从容器内文件读
    - name: line-count
      valueFrom:
        path: /tmp/lines.txt

    # (2) 直接给固定值
    - name: image
      value: "registry.local/app:{{workflow.uid}}"

    # (3) 从 JSON 提取
    - name: status
      valueFrom:
        jsonPath: '{.metadata.status}'      # 多用于 resource 类型 template
```

### 2.4 引用上一步的输出

DAG 里：

```yaml
- name: deploy
  depends: build
  arguments:
    parameters:
      - name: image
        value: "{{tasks.build.outputs.parameters.image}}"
```

Steps 里：

```yaml
- - name: deploy
    template: deploy
    arguments:
      parameters:
        - name: image
          value: "{{steps.build.outputs.parameters.image}}"
```

## 3. script 模板的 result 输出

`script` 类型有个特别的隐式输出 `result`：脚本的标准输出（限制 256KB）会自动作为输出参数。

```yaml
- name: gen
  script:
    image: python:3.11
    command: [python]
    source: |
      import json
      print(json.dumps(["a","b","c"]), end='')

- name: process
  depends: gen
  template: do
  withParam: "{{tasks.gen.outputs.result}}"
```

这是 **fan-out 最常见的写法**。

## 4. Artifacts：文件传递

### 4.1 基本概念

artifact 实际上是一段路径在两端的"自动上传 / 下载"：

```text
        ┌── 上一步 Pod ──┐                  ┌── 下一步 Pod ──┐
        │ /workspace/.. │                   │ /input/..     │
        └──────┬────────┘                   └──────▲────────┘
               │ outputs.artifacts            inputs│artifacts
               ▼                                    │
            ┌─────────────────────────────────────────────┐
            │     S3 / MinIO / OSS / GCS（对象存储）       │
            └─────────────────────────────────────────────┘
```

输出端的容器把文件写到 `outputs.artifacts.x.path`，Argo 的 wait 容器在 Pod 结束时自动打包上传；输入端启动前自动下载并解压到 `inputs.artifacts.y.path`。

### 4.2 输出 artifact

```yaml
- name: build
  container:
    image: golang:1.22
    command: [sh, -c]
    args: ["go build -o /out/app ./cmd/app"]
  outputs:
    artifacts:
      - name: binary
        path: /out                 # 整个目录会被打包
        archive:                   # 可选：归档方式
          tar:
            compressionLevel: 6    # tar.gz；0 表示不压缩
```

### 4.3 输入 artifact

```yaml
- name: deploy
  inputs:
    artifacts:
      - name: binary
        path: /app                 # 自动下载并解压到这里
  container:
    image: alpine
    command: [/app/app, --start]
```

调用时把上一步的输出连过来：

```yaml
- name: deploy
  depends: build
  template: deploy
  arguments:
    artifacts:
      - name: binary
        from: "{{tasks.build.outputs.artifacts.binary}}"
```

### 4.4 直接从外部源拉

inputs.artifacts 也可以直接指定外部源：

```yaml
inputs:
  artifacts:
    - name: dataset
      path: /data
      s3:
        endpoint: minio.example.com:9000
        bucket: datasets
        key: imagenet.tar.gz
        accessKeySecret: {name: my-s3-cred, key: ak}
        secretKeySecret: {name: my-s3-cred, key: sk}
```

支持 `s3 / git / http / gcs / oss / artifactory / hdfs`。

## 5. 制品仓库（Artifact Repository）

### 5.1 三种配置层级

| 层级 | 谁定义 | 优先级 |
|------|--------|--------|
| 工作流内每个 artifact 自己写 | 写 yaml 的人 | 高 |
| Workflow 级 `artifactRepositoryRef` | 写 yaml 的人 | 中 |
| Namespace 级默认（ConfigMap `artifact-repositories`） | 平台管理员 | 低 |

推荐做法：**平台管理员配置 namespace 级默认**，业务方只写 path、不重复写 endpoint/credential。

### 5.2 默认仓库 ConfigMap 示例

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: artifact-repositories
  namespace: argo
  annotations:
    workflows.argoproj.io/default-artifact-repository: default-v1
data:
  default-v1: |
    s3:
      endpoint: minio.argo.svc:9000
      bucket: argo-artifacts
      insecure: true
      accessKeySecret: {name: minio-cred, key: accesskey}
      secretKeySecret: {name: minio-cred, key: secretkey}
      keyFormat: "{{workflow.namespace}}/{{workflow.name}}/{{pod.name}}"
```

`keyFormat` 控制存储 key 的命名规则，强烈建议加上 `{{workflow.name}}` 方便排查。

## 6. 参数 vs 制品的选择对照表

| 想传的东西 | 选 |
|-----------|-----|
| 镜像 tag、版本号、commit id | parameter |
| URL、配置 JSON 字符串 | parameter |
| 处理产出的小段统计信息 | parameter（valueFrom.path） |
| 训练好的模型文件 | artifact |
| 构建好的二进制 / 打包好的 zip | artifact |
| 一份要分发的代码目录 | artifact |
| 大段日志 / 大块 JSON | artifact（写到文件再上传） |

> 千万**不要把大段文本塞进 parameter**——会把 etcd 撑大、UI 卡死、控制器变慢。

## 7. 参数 / 制品传递常见坑

| 坑 | 现象 | 解决 |
|----|------|------|
| 默认值没加引号 | yaml 解析错（`default: 0` 被当成 int） | 始终用字符串：`default: "0"` |
| 跨 namespace artifact | 下载失败 | 制品仓库配置 + RBAC 必须能访问 |
| 上传 artifact 401/403 | wait 容器报错 | 检查 ServiceAccount 是否有读 secret 权限；secret 里 ak/sk 是否对 |
| Output parameter 没值 | 下游拿到空字符串 | 容器里 `print` 没加 `end=''` 末尾混了换行；或者文件没写 |
| Result 超过 256KB | 控制器报 result too large | 改用 artifact |
| Artifact path 写错 | 上传内容是空目录 | path 必须是容器内**绝对路径**，且确实生成了文件 |

## 8. 调试技巧

调参数 / 制品有问题时优先看：

```bash
# 1. wait 容器（Argo 注入的）的日志，里面有上传/下载详情
kubectl logs <pod> -c wait

# 2. 主容器的日志，看你的脚本输出对不对
kubectl logs <pod> -c main

# 3. 查 Workflow CR 里的 status，每个节点的 outputs 都在那里
kubectl get wf <name> -o yaml | less
```

UI 上点进每个节点的 ARTIFACTS / PARAMETERS 标签页，可以直接看输入输出。生产环境一定打开 archive，跑完才能在 UI 上回看历史。

## 下一步

数据传递学完后，下一步是把 Workflow 复用化、定时化：

- [WorkflowTemplate、ClusterWorkflowTemplate 与 CronWorkflow](./06-workflowtemplate-and-cron.md)
