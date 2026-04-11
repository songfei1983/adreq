# CI/CD（Kubernetes）提案 Spec

## Why
目前仓库只有本地运行/测试与性能压测工具，缺少可重复、可审计的发布流程。引入基于 Kubernetes 的 CI/CD，可以把构建、测试、制品产出、部署与回滚标准化，便于后续接入真实 HTTP 服务与线上环境。

## What Changes
- 增加容器化交付：构建 adreq 的容器镜像并发布到镜像仓库（建议 GHCR）。
- 增加 Kubernetes 部署清单：Deployment/Service（以及可选的 HPA、PDB、Config/Secrets）。
- 增加 CI：PR 上执行单测、lint（可选）、镜像构建验证（不推送或推送到临时 tag）。
- 增加 CD：合并到主分支后发布镜像并部署到集群（先 staging，prod 采用手动审批）。
- 增加可观测性最小集：readiness/liveness、资源 requests/limits、日志输出约定。
- **BREAKING（可选）**：如需通过 HTTP 对外服务，可能需要新增一个真正的 HTTP server 入口（例如 `cmd/api`），否则 Kubernetes 部署只能跑 demo/worker 进程。

## Impact
- Affected specs:
  - Build/Release：容器镜像制品
  - Deployment：Kubernetes manifests/Helm/Kustomize
  - CI：PR 验证
  - CD：自动部署/回滚
- Affected code:
  - 新增：Dockerfile（或 build 目录）、k8s 清单目录（如 `deploy/`）
  - 新增：GitHub Actions workflows（`.github/workflows/*.yml`）
  - 可能新增：`cmd/api`（如果要在集群内提供 HTTP 服务）

## ADDED Requirements

### Requirement: Container Image
系统 SHALL 能够在 CI 中构建可运行的容器镜像，且镜像包含可执行文件与最小运行时依赖。

#### Scenario: Image Build
- **WHEN** 在 PR 或主分支触发 CI
- **THEN** 镜像可以成功构建
- **AND** 镜像标签遵循约定（例如：`sha-<GITHUB_SHA>`，主分支额外打 `latest` 或 `main`）

### Requirement: Kubernetes Deployment (Staging)
系统 SHALL 提供 Kubernetes 部署清单，使服务能在 staging 命名空间部署与滚动更新。

#### Scenario: Deploy Success
- **WHEN** 合并到主分支触发 CD
- **THEN** staging 的 Deployment 版本更新到新镜像
- **AND** readiness 通过后才算部署完成

### Requirement: Admission/Backpressure Configurability
系统 SHALL 能在 Kubernetes 环境下通过配置控制请求准入/背压关键参数（例如 max-requests、admission、worker/queue）。

#### Scenario: Configure via Env/Args
- **WHEN** 部署清单设置参数（env 或 args）
- **THEN** 容器启动时按配置生效

## MODIFIED Requirements

### Requirement: Build and Test
现有 `go test ./...` SHALL 在 CI 中作为必选检查项执行，并阻止失败的 PR 合并。

## REMOVED Requirements
无。

## Design Notes（提案选型）
- GitHub Actions 作为 CI/CD orchestrator（仓库已在 GitHub）。
- Kubernetes 清单建议从 Kustomize 开始（轻量、贴近原生），后续如需多环境与参数化再引入 Helm。
- 部署安全：
  - 使用 GitHub OIDC + 云厂商（如 GKE/EKS/AKS）或自建集群的短期凭证优先；否则使用加密的 kubeconfig secret。
  - 镜像仓库使用最小权限 token。
- 环境建议：
  - staging：自动部署
  - production：手动审批/受保护环境（GitHub Environments）
