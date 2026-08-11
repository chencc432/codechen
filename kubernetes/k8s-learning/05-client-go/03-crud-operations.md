# 资源的 CRUD 与写路径机制

CRUD 是 client-go 的基本功，但生产代码里真正决定稳定性的往往是：**怎么改（Update / Patch / Apply）**、**冲突怎么重试**、**List 怎么分页**。本章先覆盖常见资源操作，再把这些机制讲透。

读路径若要长期跟着变化走，不要停在本章末尾的裸 Watch；下一章 Informer 才是标准做法。

## Pod 操作

### 列出 Pod

```go
package main

import (
    "context"
    "fmt"
    
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/client-go/kubernetes"
)

func ListPods(clientset *kubernetes.Clientset) error {
    // 列出所有命名空间的 Pod
    pods, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
    if err != nil {
        return err
    }
    
    for _, pod := range pods.Items {
        fmt.Printf("命名空间: %s, 名称: %s, 状态: %s\n", 
            pod.Namespace, pod.Name, pod.Status.Phase)
    }
    return nil
}

// 带标签筛选
func ListPodsWithLabel(clientset *kubernetes.Clientset, namespace, labelSelector string) error {
    pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
        LabelSelector: labelSelector,  // 例如: "app=nginx"
    })
    if err != nil {
        return err
    }
    
    for _, pod := range pods.Items {
        fmt.Printf("Pod: %s\n", pod.Name)
    }
    return nil
}

// 带字段筛选
func ListRunningPods(clientset *kubernetes.Clientset, namespace string) error {
    pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
        FieldSelector: "status.phase=Running",
    })
    if err != nil {
        return err
    }
    
    for _, pod := range pods.Items {
        fmt.Printf("运行中的 Pod: %s\n", pod.Name)
    }
    return nil
}
```

### 获取单个 Pod

```go
func GetPod(clientset *kubernetes.Clientset, namespace, name string) error {
    pod, err := clientset.CoreV1().Pods(namespace).Get(context.TODO(), name, metav1.GetOptions{})
    if err != nil {
        return err
    }
    
    fmt.Printf("Pod 名称: %s\n", pod.Name)
    fmt.Printf("Pod IP: %s\n", pod.Status.PodIP)
    fmt.Printf("节点: %s\n", pod.Spec.NodeName)
    fmt.Printf("状态: %s\n", pod.Status.Phase)
    
    for _, container := range pod.Spec.Containers {
        fmt.Printf("容器: %s, 镜像: %s\n", container.Name, container.Image)
    }
    return nil
}
```

### 创建 Pod

```go
import (
    corev1 "k8s.io/api/core/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreatePod(clientset *kubernetes.Clientset, namespace string) error {
    pod := &corev1.Pod{
        ObjectMeta: metav1.ObjectMeta{
            Name:      "nginx-pod",
            Namespace: namespace,
            Labels: map[string]string{
                "app": "nginx",
            },
        },
        Spec: corev1.PodSpec{
            Containers: []corev1.Container{
                {
                    Name:  "nginx",
                    Image: "nginx:1.21",
                    Ports: []corev1.ContainerPort{
                        {
                            ContainerPort: 80,
                        },
                    },
                },
            },
        },
    }
    
    createdPod, err := clientset.CoreV1().Pods(namespace).Create(
        context.TODO(), 
        pod, 
        metav1.CreateOptions{},
    )
    if err != nil {
        return err
    }
    
    fmt.Printf("Pod %s 创建成功\n", createdPod.Name)
    return nil
}
```

### 更新 Pod

```go
func UpdatePodLabels(clientset *kubernetes.Clientset, namespace, name string) error {
    // 获取现有 Pod
    pod, err := clientset.CoreV1().Pods(namespace).Get(context.TODO(), name, metav1.GetOptions{})
    if err != nil {
        return err
    }
    
    // 修改标签
    if pod.Labels == nil {
        pod.Labels = make(map[string]string)
    }
    pod.Labels["version"] = "v2"
    
    // 更新
    updatedPod, err := clientset.CoreV1().Pods(namespace).Update(
        context.TODO(), 
        pod, 
        metav1.UpdateOptions{},
    )
    if err != nil {
        return err
    }
    
    fmt.Printf("Pod %s 更新成功\n", updatedPod.Name)
    return nil
}
```

### 删除 Pod

```go
func DeletePod(clientset *kubernetes.Clientset, namespace, name string) error {
    err := clientset.CoreV1().Pods(namespace).Delete(
        context.TODO(), 
        name, 
        metav1.DeleteOptions{},
    )
    if err != nil {
        return err
    }
    
    fmt.Printf("Pod %s 删除成功\n", name)
    return nil
}

// 强制删除
func ForceDeletePod(clientset *kubernetes.Clientset, namespace, name string) error {
    gracePeriod := int64(0)
    err := clientset.CoreV1().Pods(namespace).Delete(
        context.TODO(), 
        name, 
        metav1.DeleteOptions{
            GracePeriodSeconds: &gracePeriod,
        },
    )
    return err
}
```

## Deployment 操作

### 创建 Deployment

```go
import (
    appsv1 "k8s.io/api/apps/v1"
    corev1 "k8s.io/api/core/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreateDeployment(clientset *kubernetes.Clientset, namespace string) error {
    replicas := int32(3)
    
    deployment := &appsv1.Deployment{
        ObjectMeta: metav1.ObjectMeta{
            Name:      "nginx-deployment",
            Namespace: namespace,
        },
        Spec: appsv1.DeploymentSpec{
            Replicas: &replicas,
            Selector: &metav1.LabelSelector{
                MatchLabels: map[string]string{
                    "app": "nginx",
                },
            },
            Template: corev1.PodTemplateSpec{
                ObjectMeta: metav1.ObjectMeta{
                    Labels: map[string]string{
                        "app": "nginx",
                    },
                },
                Spec: corev1.PodSpec{
                    Containers: []corev1.Container{
                        {
                            Name:  "nginx",
                            Image: "nginx:1.21",
                            Ports: []corev1.ContainerPort{
                                {
                                    ContainerPort: 80,
                                },
                            },
                            Resources: corev1.ResourceRequirements{
                                Requests: corev1.ResourceList{
                                    corev1.ResourceCPU:    resource.MustParse("100m"),
                                    corev1.ResourceMemory: resource.MustParse("128Mi"),
                                },
                                Limits: corev1.ResourceList{
                                    corev1.ResourceCPU:    resource.MustParse("200m"),
                                    corev1.ResourceMemory: resource.MustParse("256Mi"),
                                },
                            },
                        },
                    },
                },
            },
        },
    }
    
    result, err := clientset.AppsV1().Deployments(namespace).Create(
        context.TODO(), 
        deployment, 
        metav1.CreateOptions{},
    )
    if err != nil {
        return err
    }
    
    fmt.Printf("Deployment %s 创建成功\n", result.Name)
    return nil
}
```

### 扩缩容 Deployment

```go
func ScaleDeployment(clientset *kubernetes.Clientset, namespace, name string, replicas int32) error {
    // 获取当前 Deployment
    deployment, err := clientset.AppsV1().Deployments(namespace).Get(
        context.TODO(), 
        name, 
        metav1.GetOptions{},
    )
    if err != nil {
        return err
    }
    
    // 修改副本数
    deployment.Spec.Replicas = &replicas
    
    // 更新
    _, err = clientset.AppsV1().Deployments(namespace).Update(
        context.TODO(), 
        deployment, 
        metav1.UpdateOptions{},
    )
    if err != nil {
        return err
    }
    
    fmt.Printf("Deployment %s 扩缩容到 %d 副本\n", name, replicas)
    return nil
}

// 使用 Scale 子资源
func ScaleDeploymentSubresource(clientset *kubernetes.Clientset, namespace, name string, replicas int32) error {
    scale, err := clientset.AppsV1().Deployments(namespace).GetScale(
        context.TODO(), 
        name, 
        metav1.GetOptions{},
    )
    if err != nil {
        return err
    }
    
    scale.Spec.Replicas = replicas
    
    _, err = clientset.AppsV1().Deployments(namespace).UpdateScale(
        context.TODO(), 
        name, 
        scale, 
        metav1.UpdateOptions{},
    )
    return err
}
```

### 更新镜像

```go
func UpdateDeploymentImage(clientset *kubernetes.Clientset, namespace, name, containerName, newImage string) error {
    deployment, err := clientset.AppsV1().Deployments(namespace).Get(
        context.TODO(), 
        name, 
        metav1.GetOptions{},
    )
    if err != nil {
        return err
    }
    
    // 更新容器镜像
    for i := range deployment.Spec.Template.Spec.Containers {
        if deployment.Spec.Template.Spec.Containers[i].Name == containerName {
            deployment.Spec.Template.Spec.Containers[i].Image = newImage
            break
        }
    }
    
    _, err = clientset.AppsV1().Deployments(namespace).Update(
        context.TODO(), 
        deployment, 
        metav1.UpdateOptions{},
    )
    return err
}
```

## Service 操作

### 创建 Service

```go
func CreateService(clientset *kubernetes.Clientset, namespace string) error {
    service := &corev1.Service{
        ObjectMeta: metav1.ObjectMeta{
            Name:      "nginx-service",
            Namespace: namespace,
        },
        Spec: corev1.ServiceSpec{
            Type: corev1.ServiceTypeClusterIP,
            Selector: map[string]string{
                "app": "nginx",
            },
            Ports: []corev1.ServicePort{
                {
                    Name:     "http",
                    Port:     80,
                    TargetPort: intstr.FromInt(80),
                    Protocol: corev1.ProtocolTCP,
                },
            },
        },
    }
    
    result, err := clientset.CoreV1().Services(namespace).Create(
        context.TODO(), 
        service, 
        metav1.CreateOptions{},
    )
    if err != nil {
        return err
    }
    
    fmt.Printf("Service %s 创建成功, ClusterIP: %s\n", result.Name, result.Spec.ClusterIP)
    return nil
}
```

## ConfigMap 和 Secret 操作

### 创建 ConfigMap

```go
func CreateConfigMap(clientset *kubernetes.Clientset, namespace string) error {
    configMap := &corev1.ConfigMap{
        ObjectMeta: metav1.ObjectMeta{
            Name:      "app-config",
            Namespace: namespace,
        },
        Data: map[string]string{
            "DATABASE_HOST": "mysql.example.com",
            "DATABASE_PORT": "3306",
            "app.properties": `
                server.port=8080
                log.level=INFO
            `,
        },
    }
    
    result, err := clientset.CoreV1().ConfigMaps(namespace).Create(
        context.TODO(), 
        configMap, 
        metav1.CreateOptions{},
    )
    if err != nil {
        return err
    }
    
    fmt.Printf("ConfigMap %s 创建成功\n", result.Name)
    return nil
}
```

### 创建 Secret

```go
func CreateSecret(clientset *kubernetes.Clientset, namespace string) error {
    secret := &corev1.Secret{
        ObjectMeta: metav1.ObjectMeta{
            Name:      "app-secret",
            Namespace: namespace,
        },
        Type: corev1.SecretTypeOpaque,
        StringData: map[string]string{  // 自动 base64 编码
            "username": "admin",
            "password": "secretpassword",
        },
    }
    
    result, err := clientset.CoreV1().Secrets(namespace).Create(
        context.TODO(), 
        secret, 
        metav1.CreateOptions{},
    )
    if err != nil {
        return err
    }
    
    fmt.Printf("Secret %s 创建成功\n", result.Name)
    return nil
}
```

## 写路径机制：Update / Patch / Apply

### 三种改法对比

| 方式 | 提交内容 | 并发友好度 | 典型用途 |
|------|----------|------------|----------|
| **Update** | 整对象 | 低（易 409） | 逻辑简单、对象小 |
| **Patch** | 变更片段 | 较高 | 改注解、改个别字段 |
| **Apply（SSA）** | 声明式字段 + fieldManager | 高（按字段归属合并） | 多控制器共建同一对象 |

Update 依赖对象里的 `resourceVersion` 做乐观锁：你读到的版本若已被别人改过，API Server 返回 `409 Conflict`。

### Strategic Merge Patch / Merge Patch / JSON Patch

```go
import (
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// Strategic Merge Patch：对内置资源最常用，懂 list 合并策略（如 containers 按 name 合并）
func PatchPodAnnotation(clientset kubernetes.Interface, ns, name, key, value string) error {
	patch := map[string]interface{}{
		"metadata": map[string]interface{}{
			"annotations": map[string]string{
				key: value,
			},
		},
	}
	data, err := json.Marshal(patch)
	if err != nil {
		return err
	}
	_, err = clientset.CoreV1().Pods(ns).Patch(
		context.TODO(),
		name,
		types.StrategicMergePatchType,
		data,
		metav1.PatchOptions{},
	)
	return err
}

// JSON Patch：精确操作路径，适合删字段、改数组下标
func JSONPatchReplicas(clientset kubernetes.Interface, ns, name string, replicas int32) error {
	patch := []map[string]interface{}{
		{"op": "replace", "path": "/spec/replicas", "value": replicas},
	}
	data, err := json.Marshal(patch)
	if err != nil {
		return err
	}
	_, err = clientset.AppsV1().Deployments(ns).Patch(
		context.TODO(),
		name,
		types.JSONPatchType,
		data,
		metav1.PatchOptions{},
	)
	return err
}
```

注意：CRD 默认通常**没有** strategic merge patch 策略，对 CR 更常用 Merge Patch、JSON Patch 或 SSA。

### Server-Side Apply（SSA）

SSA 让 API Server 记录「哪个 fieldManager 拥有哪些字段」。两个控制器改同一对象的不同字段可以共存；改同一字段会按强制/冲突规则处理。

```go
import (
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func ApplyDeploymentLabels(clientset kubernetes.Interface, ns string, deploy *appsv1.Deployment) error {
	force := true
	_, err := clientset.AppsV1().Deployments(ns).Apply(
		context.TODO(),
		deploy, // 建议只填你拥有的字段；ApplyConfiguration 生成代码更稳妥
		metav1.ApplyOptions{
			FieldManager: "my-controller",
			Force:        force, // 生产中 Force 要谨慎，先理解冲突再决定
		},
	)
	return err
}
```

Kubernetes 1.21+ 起 SSA 已是推荐的声明式写入方式之一；Controller Runtime / Kubebuilder 默认也偏向 SSA。

### 冲突重试（RetryOnConflict）

控制器里 Update 撞车是常态，不要把 409 当致命错误。

```go
import (
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
)

func AddAnnotationWithRetry(clientset kubernetes.Interface, ns, name, key, value string) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		pod, err := clientset.CoreV1().Pods(ns).Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		pod = pod.DeepCopy() // 若对象来自 Informer 缓存，必须拷贝
		if pod.Annotations == nil {
			pod.Annotations = map[string]string{}
		}
		pod.Annotations[key] = value
		_, err = clientset.CoreV1().Pods(ns).Update(context.TODO(), pod, metav1.UpdateOptions{})
		return err
	})
}

func IsUselessToRetry(err error) bool {
	return errors.IsNotFound(err) || errors.IsInvalid(err) || errors.IsForbidden(err)
}
```

原则：

1. 每次重试都**重新 Get**（或重新从缓存读再 DeepCopy）
2. 不要修改 Lister 返回的原对象
3. NotFound / Forbidden / Invalid 通常不应无限重试

### List 分页（Continue + Limit）

大集群一次 List 全量会拖垮 API Server 和客户端内存。用分页：

```go
func ListAllPodsPaged(clientset kubernetes.Interface, ns string) ([]corev1.Pod, error) {
	var all []corev1.Pod
	var continueToken string
	for {
		list, err := clientset.CoreV1().Pods(ns).List(context.TODO(), metav1.ListOptions{
			Limit:    500,
			Continue: continueToken,
		})
		if err != nil {
			return nil, err
		}
		all = append(all, list.Items...)
		continueToken = list.Continue
		if continueToken == "" {
			break
		}
	}
	return all, nil
}
```

Informer 内部的 Reflector 也会处理分块 List；你手写 List 工具时记得同样做。

## Watch 只作理解，生产用 Informer

裸 Watch 能帮你理解事件流，但连接断开、RV 过期（`410 Gone`）、重 List、本地查询都要自己扛。

```go
func WatchPods(clientset kubernetes.Interface, namespace string) error {
	watcher, err := clientset.CoreV1().Pods(namespace).Watch(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	defer watcher.Stop()

	for event := range watcher.ResultChan() {
		pod, ok := event.Object.(*corev1.Pod)
		if !ok {
			continue
		}
		switch event.Type {
		case watch.Added:
			fmt.Printf("[新增] %s\n", pod.Name)
		case watch.Modified:
			fmt.Printf("[修改] %s (%s)\n", pod.Name, pod.Status.Phase)
		case watch.Deleted:
			fmt.Printf("[删除] %s\n", pod.Name)
		case watch.Bookmark:
			// 用于推进 resourceVersion，通常可忽略业务处理
		case watch.Error:
			return fmt.Errorf("watch error: %v", event.Object)
		}
	}
	return nil
}
```

长期运行请直接看 [Informer 机制详解](./04-informer.md)。

## 完整示例程序

```go
package main

import (
    "context"
    "flag"
    "fmt"
    "path/filepath"
    "time"
    
    appsv1 "k8s.io/api/apps/v1"
    corev1 "k8s.io/api/core/v1"
    "k8s.io/apimachinery/pkg/api/resource"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/util/intstr"
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/tools/clientcmd"
    "k8s.io/client-go/util/homedir"
)

func main() {
    // 配置
    var kubeconfig *string
    if home := homedir.HomeDir(); home != "" {
        kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "")
    }
    flag.Parse()
    
    config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
    if err != nil {
        panic(err)
    }
    
    clientset, err := kubernetes.NewForConfig(config)
    if err != nil {
        panic(err)
    }
    
    namespace := "default"
    
    // 1. 创建 Deployment
    fmt.Println("创建 Deployment...")
    if err := createDeployment(clientset, namespace); err != nil {
        fmt.Printf("创建 Deployment 失败: %v\n", err)
    }
    
    // 2. 等待 Pod 就绪
    time.Sleep(5 * time.Second)
    
    // 3. 列出 Pod
    fmt.Println("\n列出 Pod...")
    listPods(clientset, namespace)
    
    // 4. 创建 Service
    fmt.Println("\n创建 Service...")
    if err := createService(clientset, namespace); err != nil {
        fmt.Printf("创建 Service 失败: %v\n", err)
    }
    
    // 5. 扩容
    fmt.Println("\n扩容到 5 副本...")
    scaleDeployment(clientset, namespace, "demo-app", 5)
    
    // 6. 清理
    fmt.Println("\n清理资源...")
    cleanup(clientset, namespace)
}

func createDeployment(clientset *kubernetes.Clientset, namespace string) error {
    replicas := int32(3)
    deployment := &appsv1.Deployment{
        ObjectMeta: metav1.ObjectMeta{
            Name: "demo-app",
        },
        Spec: appsv1.DeploymentSpec{
            Replicas: &replicas,
            Selector: &metav1.LabelSelector{
                MatchLabels: map[string]string{"app": "demo"},
            },
            Template: corev1.PodTemplateSpec{
                ObjectMeta: metav1.ObjectMeta{
                    Labels: map[string]string{"app": "demo"},
                },
                Spec: corev1.PodSpec{
                    Containers: []corev1.Container{{
                        Name:  "nginx",
                        Image: "nginx:1.21",
                        Ports: []corev1.ContainerPort{{ContainerPort: 80}},
                    }},
                },
            },
        },
    }
    _, err := clientset.AppsV1().Deployments(namespace).Create(context.TODO(), deployment, metav1.CreateOptions{})
    return err
}

func listPods(clientset *kubernetes.Clientset, namespace string) {
    pods, _ := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
        LabelSelector: "app=demo",
    })
    for _, pod := range pods.Items {
        fmt.Printf("  - %s (%s)\n", pod.Name, pod.Status.Phase)
    }
}

func createService(clientset *kubernetes.Clientset, namespace string) error {
    service := &corev1.Service{
        ObjectMeta: metav1.ObjectMeta{Name: "demo-service"},
        Spec: corev1.ServiceSpec{
            Selector: map[string]string{"app": "demo"},
            Ports: []corev1.ServicePort{{
                Port:       80,
                TargetPort: intstr.FromInt(80),
            }},
        },
    }
    _, err := clientset.CoreV1().Services(namespace).Create(context.TODO(), service, metav1.CreateOptions{})
    return err
}

func scaleDeployment(clientset *kubernetes.Clientset, namespace, name string, replicas int32) {
    scale, _ := clientset.AppsV1().Deployments(namespace).GetScale(context.TODO(), name, metav1.GetOptions{})
    scale.Spec.Replicas = replicas
    clientset.AppsV1().Deployments(namespace).UpdateScale(context.TODO(), name, scale, metav1.UpdateOptions{})
}

func cleanup(clientset *kubernetes.Clientset, namespace string) {
    clientset.AppsV1().Deployments(namespace).Delete(context.TODO(), "demo-app", metav1.DeleteOptions{})
    clientset.CoreV1().Services(namespace).Delete(context.TODO(), "demo-service", metav1.DeleteOptions{})
}
```

## 下一步

- [Informer 机制详解](./04-informer.md) — List/Watch 地基、缓存、WorkQueue



