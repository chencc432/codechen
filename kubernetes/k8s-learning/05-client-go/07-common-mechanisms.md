# 常用机制与工具包

Clientset / Informer / WorkQueue 是主干。这一章补齐写生产组件时几乎总会碰到的周边机制：Scheme、Event、Leader Election、retry/wait、Fake client、以及若干 `tools` / `util` 包。

## Scheme：类型注册与编解码

Kubernetes 对象在网上走 JSON/YAML/Protobuf，在进程里是 Go struct。`runtime.Scheme` 负责：

- 注册 GVK ↔ Go 类型
- 提供默认值填充、转换、编解码器工厂

```go
import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
	// CRD：_ = myapi.AddToScheme(scheme)
	return scheme
}

func codecsFor(scheme *runtime.Scheme) serializer.CodecFactory {
	return serializer.NewCodecFactory(scheme)
}
```

Clientset 内部已带好内置类型 Scheme。你写 CRD typed client、Admission Webhook、conversion 时，需要自己 `AddToScheme`。Dynamic/Unstructured 路径对 Scheme 依赖更少，但做 typed 转换时仍需要。

## EventRecorder：让 `kubectl describe` 可观测

控制器除了改对象，还应留下人类可读事件：

```bash
kubectl describe pod nginx
# Events:
#   Type    Reason     Age   Message
#   Normal  Annotated  5s    added annotation by pod-annotator
```

```go
import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
)

func newRecorder(clientset kubernetes.Interface) record.EventRecorder {
	broadcaster := record.NewBroadcaster()
	broadcaster.StartStructuredLogging(0)
	broadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{
		Interface: clientset.CoreV1().Events(""),
	})
	return broadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "pod-annotator"})
}

// 使用
// recorder.Event(pod, corev1.EventTypeNormal, "Annotated", "added annotation")
// recorder.Eventf(pod, corev1.EventTypeWarning, "FailedUpdate", "update failed: %v", err)
```

建议：

- 关键状态转换打 Normal；失败打 Warning
- Reason 用稳定短词，Message 可含细节
- 不要在热循环里刷屏，事件有限流与聚合

## Leader Election：多副本只跑一个活跃循环

Deployment `replicas: 3` 时，若三个副本都 Reconcile，会打架。标准做法：用 Lease 做选主，只有 Leader 跑控制循环。

```go
import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
)

func runWithLeader(ctx context.Context, client clientset.Interface, ns, name, id string, run func(context.Context)) {
	lock := &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		Client:    client.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity: id, // 通常是 pod name
		},
	}

	leaderelection.RunOrDie(ctx, leaderelection.LeaderElectionConfig{
		Lock:            lock,
		ReleaseOnCancel: true,
		LeaseDuration:   15 * time.Second,
		RenewDeadline:   10 * time.Second,
		RetryPeriod:     2 * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: run,
			OnStoppedLeading: func() {
				// 失去领导权，应停止写路径
			},
		},
	})
}
```

注意：

- 需要 `coordination.k8s.io` Lease 的 create/get/update 权限
- 选主只保证「同一时刻一个 Leader」，不保证「切换瞬间零重叠」——Reconcile 仍必须幂等
- 深度时序与坑见 [Leader Election 与 Finalizer](../11-controller-deep-dive/05-leader-election-and-finalizers.md)

## retry：冲突与瞬时错误

```go
import (
	"k8s.io/client-go/util/retry"
)

// Update 冲突
err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
	// get → mutate → update
	return nil
})

// 任意可重试错误
err = retry.OnError(retry.DefaultBackoff, isRetriable, func() error {
	return callAPI()
})
```

与 WorkQueue 的关系：

- **单次 Reconcile 内部**的 409 → `RetryOnConflict`
- **整个 Reconcile 失败** → 返回 error，让 WorkQueue `AddRateLimited`

两者常一起用，不是二选一。

## wait：等到条件成立

```go
import (
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
)

// 轮询直到返回 true 或超时
err := wait.PollUntilContextTimeout(ctx, 2*time.Second, 5*time.Minute, true, func(ctx context.Context) (bool, error) {
	deploy, err := client.AppsV1().Deployments(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return false, err
	}
	return deploy.Status.ReadyReplicas == *deploy.Spec.Replicas, nil
})

// 控制器 worker 循环常用
wait.Until(workerFunc, time.Second, stopCh)
```

测试或安装脚本里常见；长连接场景优先 Informer 事件，少用盲轮询。

## Fake Client：单测不连集群

```go
import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"testing"
)

func TestAddAnnotation(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "p", Namespace: "default"},
	}
	client := fake.NewSimpleClientset(pod)

	got, err := client.CoreV1().Pods("default").Get(context.TODO(), "p", metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	got = got.DeepCopy()
	got.Annotations = map[string]string{"k": "v"}
	_, err = client.CoreV1().Pods("default").Update(context.TODO(), got, metav1.UpdateOptions{})
	if err != nil {
		t.Fatal(err)
	}
}
```

配套还有：

- `dynamicfake`：测 Dynamic Client
- `client-go/tools/cache` 的假 Indexer：测 Lister 逻辑
- Informer 单测可用 `framework` / 手动 fixture，或上 controller-runtime 的 envtest（真 API Server）

Fake **不是**完整 API Server：校验、默认值、admission、部分子资源行为可能与真实集群不同。关键路径建议加 integration test。

## 其它高频工具

| 包 | 用途 |
|----|------|
| `tools/clientcmd` | 加载 kubeconfig、切换 context |
| `tools/cache` | Informer/Indexer/Lister 核心 |
| `util/workqueue` | 去重限速队列 |
| `util/homedir` | 找 `~/.kube/config` |
| `rest.Config` 的 `WrapTransport` / `UserAgent` | 注入追踪头、标识客户端 |
| `kubernetes/scheme` | 内置类型 Scheme 单例 |

### UserAgent 与 WrapTransport（运维友好）

```go
config, _ := rest.InClusterConfig()
config.UserAgent = "pod-annotator/1.2.3"
config.WrapTransport = func(rt http.RoundTripper) http.RoundTripper {
	return roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		req.Header.Set("X-Request-Id", newID())
		return rt.RoundTrip(req)
	})
}
```

出问题时，API Server 审计日志能直接看出是哪个组件、哪次请求。

## 机制怎么拼在一起

一个生产级控制器常见组合：

```
rest.Config（QPS/UserAgent）
    ├── Clientset / Dynamic
    ├── Discovery + RESTMapper（若有 CR / 泛型）
    ├── SharedInformerFactory（或 dynamic/metadata informer）
    ├── WorkQueue + Workers
    ├── EventRecorder
    ├── Leader Election（多副本时）
    └── metrics / healthz（本课从略，深度专题与 Operator 课会涉及）
```

读路径走 Informer/Lister，写路径走 Clientset/Dynamic + RetryOnConflict，可观测性靠 Event/metrics，高可用靠 Leader Election。

## 与后续课程的分工

| 主题 | 本模块 | 更深内容 |
|------|--------|----------|
| Informer 数据结构 | [04](./04-informer.md) | [11-02](../11-controller-deep-dive/02-informer-internals.md) |
| WorkQueue 限速 | [04](./04-informer.md) | [11-03](../11-controller-deep-dive/03-workqueue.md) |
| Leader / Finalizer | 本章入门 | [11-05](../11-controller-deep-dive/05-leader-election-and-finalizers.md) |
| CRD / Operator | [06](./06-discovery-and-dynamic.md) + 实战章 | [07 自定义资源专题](../07-custom-resources/README.md) |
| controller-runtime | 概念对比 | [11-07](../11-controller-deep-dive/07-controller-runtime-vs-clientgo.md) |

## 下一步

- 回到 [模块总览](./README.md) 查缺补漏
- 做完 [自定义控制器实战](./05-controller-demo.md) 后，把 EventRecorder 与 Leader Election 加进去
- 用 [排障与生产清单](./08-debugging-and-pitfalls.md) 做上线前自检
- 进入 [控制器深度专题](../11-controller-deep-dive/README.md)
