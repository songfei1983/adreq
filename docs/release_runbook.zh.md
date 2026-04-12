# 发布手顺（Kubernetes）

状态：已验证（docker-desktop k8s；/healthz、/readyz、/bid 已验证）

本文描述基于当前仓库的 CI/CD 与 Kubernetes 清单的发布/回滚手顺，覆盖本地（docker-desktop/kind）与 GitHub Actions（staging/prod）。

## 交付形态

- 服务入口：`cmd/api`（HTTP）
- 端点：
  - `GET /healthz`
  - `GET /readyz`
  - `POST /bid`
- 部署清单：`deploy/kustomize/base` + `deploy/kustomize/overlays/{staging,prod}`
- 镜像：
  - 本地测试：`adreq:local`
  - 远端发布：`ghcr.io/<owner>/<repo>:sha-<sha>`（以及 `:latest`）

## 本地发布（docker-desktop Kubernetes）

### 前置

- `kubectl config current-context` 指向 `docker-desktop`
- 本机 Docker 可用（`docker build` 可成功）

### 步骤

1) 构建本地镜像：

```bash
docker build -t adreq:local .
```

2) 部署到 staging overlay：

```bash
kubectl apply -k deploy/kustomize/overlays/staging
```

3) 将 Deployment 镜像切到本地镜像：

```bash
kubectl -n adreq-staging set image deployment/adreq adreq=adreq:local
```

4) 等待滚动完成并查看状态：

```bash
kubectl -n adreq-staging rollout status deployment/adreq --timeout=180s
kubectl -n adreq-staging get deploy,svc,pods,pdb -o wide
```

5) 端口转发并验证：

```bash
kubectl -n adreq-staging port-forward svc/adreq 18080:80
```

另一个终端：

```bash
curl -i http://127.0.0.1:18080/healthz
curl -i http://127.0.0.1:18080/readyz
curl -i -X POST http://127.0.0.1:18080/bid -H 'content-type: application/json' \
  -d '{"id":"req1","imp":[{"id":"imp1","bidfloor":1.0}]}'
```

### 回滚

```bash
kubectl -n adreq-staging rollout undo deployment/adreq
```

## 本地发布（kind）

说明：kind 集群的 node 是容器，无法直接使用宿主机的本地镜像标签。需要将镜像 load 到 kind 集群。

### 前置

- `kind` CLI 可用
- `kubectl config current-context` 指向对应 kind context

### 步骤

1) 构建镜像：

```bash
docker build -t adreq:local .
```

2) 将镜像导入 kind：

```bash
kind load docker-image adreq:local --name <kind-cluster-name>
```

3) 应用清单并切换镜像：

```bash
kubectl apply -k deploy/kustomize/overlays/staging
kubectl -n adreq-staging set image deployment/adreq adreq=adreq:local
kubectl -n adreq-staging rollout status deployment/adreq --timeout=180s
```

## staging 手动发布（GitHub Actions）

### 前置（一次性）

1) GitHub Environments 创建 `staging`
2) 在 `staging` 环境增加 secret：`KUBE_CONFIG_DATA`
   - 值为 kubeconfig 的 base64 编码（建议最小权限 serviceaccount）

示例（mac/linux）：

```bash
base64 < ~/.kube/config | tr -d '\n'
```

### 流程

- 触发：手动触发 `cd-staging` 工作流（workflow_dispatch）
- 工作流：`.github/workflows/cd-staging.yml`
  - build & push：`ghcr.io/<owner>/<repo>:sha-<sha>` + `:latest`
  - `kubectl apply -k deploy/kustomize/overlays/staging`
  - `kubectl set image ...:sha-<sha>`
  - `kubectl rollout status`

说明：如果你希望 staging 自动发布，可以把 `cd-staging` 的触发器改回 `push`（但需要先准备好 `KUBE_CONFIG_DATA` 等部署环境）。

### 验证

```bash
kubectl -n adreq-staging get deploy adreq -o wide
kubectl -n adreq-staging describe deploy adreq | grep -i image
```

## prod 手动发布（GitHub Actions + 审批）

### 前置（一次性）

1) GitHub Environments 创建 `production` 并开启审批
2) 在 `production` 环境增加 secret：`KUBE_CONFIG_DATA`

### 发布步骤

在 GitHub Actions 手动触发工作流：`.github/workflows/cd-prod.yml`，输入参数：

- `image`: `ghcr.io/<owner>/<repo>:sha-<sha>`

### 回滚

- 方式 1：再次手动触发 `cd-prod`，把 `image` 指向上一个 `sha-...`
- 方式 2：集群侧回滚：

```bash
kubectl -n adreq-prod rollout undo deployment/adreq
```

## 常见问题排查

### Pod 起不来 / CrashLoopBackOff

```bash
kubectl -n adreq-staging get pods -o wide
kubectl -n adreq-staging describe pod <pod>
kubectl -n adreq-staging logs <pod>
```

### readiness 一直失败

- 先看 `/readyz` 是否 200
- 再看应用是否被错误参数启动（env/args）

### 镜像拉取失败

- 确认 image tag 是否存在
- kind 场景下确认是否已执行 `kind load docker-image`
