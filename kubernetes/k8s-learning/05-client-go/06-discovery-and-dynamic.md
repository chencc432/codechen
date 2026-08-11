# Discovery 与 Dynamic Client

写通用工具、平台组件、或操作 CRD 时，你往往在编译期不知道具体类型。这时靠两样东西：

1. **Discovery**：问集群「有哪些 API？」
2. **Dynamic Client + Unstructured**：用 GVR 做 CRUD，对象当 JSON 树操作

再配合 **RESTMapper**，就能在 GVK（YAML 里的 `apiVersion/kind`）和 REST 路径（URL 里的 resource）之间互转。

## 为什么需要 Discovery

Clientset 是生成代码写死的：`CoreV1().Pods()`。集群里还有：

- 不同版本的内置 API（`apps/v1`、`networking.k8s.io/v1`）
- 大量 CRD（`mysql.example.com/v1`）
- 聚合 API（metrics、自定义 apiserver）

Discovery 回答：

- 有哪些 Group / Version
- 每个 Version 下有哪些 Resource
- 资源是 namespaced 还是 cluster-scoped
- 支持哪些动词（get/list/watch/create/...）
- Preferred Version 是什么

```
kubectl api-resources
kubectl api-versions
```

这两条命令背后就是 Discovery API。

## Discovery Client 基本用法

```go
package main

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
)

func listAPIResources(clientset kubernetes.Interface) error {
	discoveryClient := clientset.Discovery()

	// 所有 GroupVersion
	groups, err := discoveryClient.ServerGroups()
	if err != nil {
		return err
	}
	for _, g := range groups.Groups {
		fmt.Printf("group=%s preferred=%s\n", g.Name, g.PreferredVersion.Version)
	}

	// 展开到具体资源
	_, resources, err := discoveryClient.ServerGroupsAndResources()
	if err != nil {
		// partial discovery error 在有不可达聚合 API 时很常见，需要按场景决定是否容忍
		fmt.Printf("warning: %v\n", err)
	}
	for _, list := range resources {
		gv, _ := schema.ParseGroupVersion(list.GroupVersion)
		for _, r := range list.APIResources {
			if r.Name == "pods" || r.Kind == "Pod" {
				fmt.Printf("%s %s namespaced=%v verbs=%v\n",
					gv, r.Name, r.Namespaced, r.Verbs)
			}
		}
	}
	return nil
}
```

### CachedDiscovery

频繁打 Discovery 会给 API Server 压力。生产里用带缓存的客户端：

```go
import (
	"path/filepath"
	"time"

	"k8s.io/client-go/discovery/cached/disk"
)

func newCachedDiscovery(host, cacheDir string) (discovery.CachedDiscoveryInterface, error) {
	return disk.NewCachedDiscoveryClientForConfig(
		&rest.Config{Host: host}, // 实际应传入完整 config
		filepath.Join(cacheDir, "discovery"),
		filepath.Join(cacheDir, "http"),
		6*time.Hour,
	)
}
```

Controller Runtime / kubectl 都大量依赖 cached discovery。CRD 刚注册时缓存可能过期，需要 `Invalidate()` 后再查。

## RESTMapper：GVK ↔ GVR

YAML 写的是：

```yaml
apiVersion: apps/v1
kind: Deployment
```

REST 路径却是 `/apis/apps/v1/namespaces/default/deployments`。中间差一个 **Kind → Resource** 映射（Deployment → deployments），以及 scope。

```go
import (
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/restmapper"
)

func mapperFor(disco discovery.DiscoveryInterface) meta.RESTMapper {
	return restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(disco))
}

func gvrOf(mapper meta.RESTMapper, apiVersion, kind string) (schema.GroupVersionResource, error) {
	gv, err := schema.ParseGroupVersion(apiVersion)
	if err != nil {
		return schema.GroupVersionResource{}, err
	}
	gk := schema.GroupKind{Group: gv.Group, Kind: kind}
	mapping, err := mapper.RESTMapping(gk, gv.Version)
	if err != nil {
		return schema.GroupVersionResource{}, err
	}
	return mapping.Resource, nil
}
```

通用 apply/get 工具几乎都是：`解析 YAML → RESTMapper → Dynamic Client`。

## Dynamic Client

```go
import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

func createCR(dyn dynamic.Interface) error {
	gvr := schema.GroupVersionResource{
		Group:    "mysql.example.com",
		Version:  "v1",
		Resource: "mysqlclusters",
	}

	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "mysql.example.com/v1",
			"kind":       "MySQLCluster",
			"metadata": map[string]interface{}{
				"name":      "demo",
				"namespace": "default",
			},
			"spec": map[string]interface{}{
				"replicas": int64(3),
				"version":  "8.0",
			},
		},
	}

	_, err := dyn.Resource(gvr).Namespace("default").Create(
		context.TODO(), obj, metav1.CreateOptions{},
	)
	return err
}
```

### 读写 Unstructured 字段

```go
replicas, found, err := unstructured.NestedInt64(obj.Object, "spec", "replicas")
_ = unstructured.SetNestedField(obj.Object, int64(5), "spec", "replicas")
phase, _, _ := unstructured.NestedString(obj.Object, "status", "phase")
```

尽量用 `unstructured.Nested*` 辅助函数，少手写层层 `map` 断言。

### Dynamic Informer

```go
import "k8s.io/client-go/dynamic/dynamicinformer"

factory := dynamicinformer.NewDynamicSharedInformerFactory(dyn, 0)
informer := factory.ForResource(gvr).Informer()
lister := factory.ForResource(gvr).Lister()
```

事件里拿到的也是 `*unstructured.Unstructured`。

## Clientset vs Dynamic 怎么选

| 场景 | 选择 |
|------|------|
| 固定操作内置资源 | Clientset |
| 操作多个/未知 CRD | Dynamic + Discovery |
| 性能敏感且类型固定 | 优先生成 typed client（client-gen） |
| 写 kubectl 插件 / 通用同步器 | Dynamic + RESTMapper |
| 只要 metadata | Metadata client |

也可以混用：主资源用 typed，附属未知 CR 用 dynamic。

## 常见坑

1. **Resource 名用错**：Kind 是 `MySQLCluster`，Resource 通常是 `mysqlclusters`（复数、小写）。以 Discovery/`kubectl api-resources` 为准。
2. **CRD 刚创建就访问**：Discovery 缓存未刷新 → `no matches for kind`。Invalidate 或短暂重试。
3. **Namespaced 调错接口**：cluster-scoped 资源不要走 `.Namespace(ns)`。
4. **Partial discovery errors**：某聚合 API 挂了会导致 ServerGroupsAndResources 报错但仍有部分结果，要会判断。
5. **类型断言失败**：Dynamic Informer 回调里别假设是 typed Pod。

## 最小完整示例：按 GVK 获取对象

```go
func getByGVK(
	dyn dynamic.Interface,
	mapper meta.RESTMapper,
	apiVersion, kind, ns, name string,
) (*unstructured.Unstructured, error) {
	gvr, err := gvrOf(mapper, apiVersion, kind)
	if err != nil {
		return nil, err
	}
	mapping, err := mapper.RESTMapping(
		schema.FromAPIVersionAndKind(apiVersion, kind).GroupKind(),
		schema.FromAPIVersionAndKind(apiVersion, kind).Version,
	)
	if err != nil {
		return nil, err
	}
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		return dyn.Resource(gvr).Namespace(ns).Get(context.TODO(), name, metav1.GetOptions{})
	}
	return dyn.Resource(gvr).Get(context.TODO(), name, metav1.GetOptions{})
}
```

## 下一步

- [常用机制与工具包](./07-common-mechanisms.md)
- [排障与生产清单](./08-debugging-and-pitfalls.md)
- [自定义资源专题](../07-custom-resources/README.md)
