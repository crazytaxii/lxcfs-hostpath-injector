# lxcfs-sidecar-injector

lxcfs-sidecar-injector 利用 Admission Controller 对请求的资源对象修改实现 lxcfs 托管文件挂载。

## 如何开启 Admission Controller

在 Kubernetes API server 的启动参数中带上 `--enable-admission-plugins=ValidatingAdmissionWebhook,MutatingAdmissionWebhook`

> `–-admission-control` 在 1.10 版本中就被废除，取而代之的是 `–-enable-admission-plugins`。

## 构建 lxcfs-sidecar-injector

**需要提前配置好 Golang 环境！**

```bash
$ git clone https://github.com/crazytaxii/lxcfs-sidecar-injector.git
$ cd lxcfs-sidecar-injector
$ export GO111MODULE=on
$ make build
$ make image
```

## 部署 lxcfs-sidecar-injector

**需要提前部署好 lxcfs 守护进程！**

```bash
$ ./kubernetes/webhook-create-signed-cert.sh
$ cat ./kubernetes/mutatingwebhook.yaml |\
    ./kubernetes/webhook-patch-ca-bundle.sh >\
    ./kubernetes/mutatingwebhookconfigurations.yaml
$ kubectl apply -f ./kubernetes/deployment.yaml
$ kubectl apply -f ./kubernetes/service.yaml
$ kubectl apply -f ./kubernetes/mutatingwebhookconfigurations.yaml
```

给 pod 带上 annotation `sidecar-injector.lxcfs/inject: "true"` 即可完成 hostPath 注入：

```yaml
spec:
  template:
    metadata:
      annotations:
        sidecar-injector.lxcfs/inject: "true"
```

## 测试 lxcfs hostPath 注入

```bash
$ kubectl apply -f ./kubernetes/test-deloyment.yaml
```

```bash
$ kubectl exec -it $(kubectl get pod -l app=web -o jsonpath="{.items[0].metadata.name}") -- free -h
             total       used       free     shared    buffers     cached
Mem:          256M       2.9M       253M         0B         0B       364K
-/+ buffers/cache:       2.5M       253M
Swap:           0B         0B         0B
```
