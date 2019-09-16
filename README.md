# lxcfs-sidecar-injector

lxcfs-sidecar-injector 利用 Admission Controller 对请求的资源对象修改实现 lxcfs 托管文件挂载。

## 如何开启 Admission Controller

在 Kubernetes API server 的启动参数中带上：

```
--enable-admission-plugins=ValidatingAdmissionWebhook,MutatingAdmissionWebhook
```

> `–-admission-control` 在 1.10 版本中就被废除，取而代之的是 `–-enable-admission-plugins`。

## 构建 & 部署

```bash
git clone https://github.com/crazytaxii/lxcfs-sidecar-injector.git
cd lxcfs-sidecar-injector
export GO111MODULE=on
make build
make image
```

```bash
./kubernetes/webhook-create-signed-cert.sh
cat ./kubernetes/mutatingwebhook.yaml |\
    ./kubernetes/webhook-patch-ca-bundle.sh >\
    ./kubernetes/mutatingwebhook-ca-bundle.yaml
kubectl apply -f ./kubernetes/deployment.yaml
kubectl apply -f ./kubernetes/service.yaml
kubectl apply -f ./kubernetes/mutatingwebhook-ca-bundle.yaml
```

使用时需要给 pod 带上 annotation `sidecar-injector.lxcfs/inject: "true"`：

```yaml
spec:
  template:
    metadata:
      annotations:
        sidecar-injector.lxcfs/inject: "true"
```

```bash
kubectl apply -f ./kubernetes/test-deloyment.yaml
```
