# Tasks

- [x] Task 1: 明确交付形态（Job vs HTTP 服务）
  - [x] 确认是否需要新增 `cmd/api`（提供 HTTP 接口）以便 Kubernetes Service/LB 暴露
  - [x] 若保持 demo 模式，定义在 k8s 中的运行方式（CronJob/Job/Deployment）与退出语义

- [x] Task 2: 容器化构建
  - [x] 增加 Dockerfile（多阶段构建，生成最小 runtime 镜像）
  - [x] 确定镜像入口命令与参数（包含 max-requests/admission/pool 配置等）

- [x] Task 3: Kubernetes 部署清单（staging）
  - [x] 增加 namespace/Deployment/Service（如适用）
  - [x] 增加资源 requests/limits、readiness/liveness
  - [x] 增加配置注入（env/args/configmap/secret，按需求取舍）
  - [x] 可选：HPA/PDB（若目标是高 QPS 稳态）

- [x] Task 4: CI（PR 验证）
  - [x] GitHub Actions：`go test ./...`
  - [x] 可选：`go test -bench` 仅手动触发或 nightly（避免 PR 里波动）
  - [x] 镜像 build 校验（不推送或推送到临时 tag）

- [x] Task 5: CD（主分支部署到 staging）
  - [x] 构建并 push 镜像到 GHCR（基于 commit SHA）
  - [x] 应用 k8s 清单并等待 rollout 完成
  - [x] 失败时输出诊断信息（kubectl describe/logs）

- [x] Task 6: 可选生产发布（手动审批）
  - [x] GitHub Environment：prod 需要审批
  - [x] 单独的 prod overlays（kustomize）或 values（helm）

# Task Dependencies
- Task 2 depends on Task 1
- Task 3 depends on Task 1 and Task 2
- Task 4 can be done in parallel with Task 2
- Task 5 depends on Task 2, Task 3, Task 4
- Task 6 depends on Task 5
