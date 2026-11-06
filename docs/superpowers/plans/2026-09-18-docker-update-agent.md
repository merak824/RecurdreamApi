# Docker 在线更新代理 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make the admin “立即更新” button upgrade Docker deployments through a host-side blue/green update agent while preserving the existing binary update path.

**Architecture:** The application discovers the container runtime and talks to a host-side Go agent through a mounted Unix socket. The agent accepts only a semantic version, constructs the configured allowlisted image reference, and runs the existing `deploy/hot-deploy.sh` with `DRAIN_SECONDS=300`; the frontend polls a proxy status endpoint until the new slot is healthy. No Docker Socket is mounted into the application container.

**Tech Stack:** Go 1.27, net/http over Unix sockets, existing Gin handlers, Vue 3/TypeScript, Docker Compose, systemd.

**Spec:** `docs/superpowers/specs/2026-09-18-docker-update-agent-design.md`

## Global Constraints

- Docker update uses the fixed host deployment script and `DRAIN_SECONDS=300`.
- `KEEP_OLD=false` remains the default so the old slot is stopped after draining.
- The agent never accepts shell commands, deployment paths, or arbitrary image URLs from requests.
- The application container never receives Docker Socket access.
- Existing non-container binary update and rollback behavior remains available.
- Existing local modifications in `backend/internal/service/update_service.go` and its test file must be preserved.
- Do not commit, push, or deploy during implementation.

### Task 1: Build The Host Update Agent Core

**Files:**
- Create: `backend/internal/updateagent/agent.go`
- Create: `backend/internal/updateagent/agent_test.go`
- Create: `backend/cmd/update-agent/main.go`

**Interfaces:**
- `updateagent.Config` contains `SocketPath`, `DeployPath`, `HotDeployScript`, `ImageRepository`, `CurrentVersionFile`, `LogDir`, `DrainSeconds`, and `Timeout`.
- `updateagent.New(config, runner)` returns an agent with `POST /v1/update` and `GET /v1/update/{operation_id}` handlers.
- `updateagent.Runner` executes the fixed deployment command and receives `(context.Context, config, version, logWriter)`.
- `Operation` serializes `operation_id`, `version`, `status`, `message`, `started_at`, `finished_at`, and `log_path`.

- [ ] Write failing tests for exact semver validation, image argument construction, one active operation, status transitions, and failed deployment preserving the current-version file.
- [ ] Run `go test ./internal/updateagent -run Test -count=1` and confirm the new tests fail because the package does not exist.
- [ ] Implement the agent with a Unix-socket HTTP server, an in-memory operation registry, a single-operation mutex, background execution with a timeout, and JSON responses.
- [ ] Validate versions with `major.minor.patch`; reject path separators, whitespace, prerelease suffixes, and arbitrary image values.
- [ ] Compare the requested version with `CurrentVersionFile`; reject downgrades and write the new version only after the runner succeeds.
- [ ] Implement the default runner with `exec.CommandContext("/bin/bash", HotDeployScript, "--image", ImageRepository+":"+version)`, `cmd.Dir=DeployPath`, `DRAIN_SECONDS=300`, and output written to a per-operation log file.
- [ ] Implement `cmd/update-agent` configuration from environment variables, create the socket directory, remove a stale socket, listen only on the Unix socket, and set socket mode `0660`.
- [ ] Run the focused agent tests and confirm they pass.

### Task 2: Add The Application Unix-Socket Client And Update Model

**Files:**
- Create: `backend/internal/service/container_update_client.go`
- Create: `backend/internal/service/container_update_client_test.go`
- Modify: `backend/internal/service/update_service.go`
- Modify: `backend/internal/service/update_service_test.go`

**Interfaces:**
- `service.UpdateExecution` contains `OperationID`, `Version`, `Status`, `Message`, and `NeedRestart`.
- `service.ContainerUpdateClient` exposes `Start(ctx, version)` and `Status(ctx, operationID)`.
- `service.UpdateService.StartUpdate(ctx)` returns `*UpdateExecution`; `GetUpdateStatus(ctx, operationID)` returns `*UpdateExecution`.

- [ ] Write failing client tests using a temporary Unix socket HTTP server for request paths, JSON decoding, status errors, and socket-unavailable errors.
- [ ] Run the focused client tests and confirm they fail before implementation.
- [ ] Implement the Unix-socket HTTP transport with a bounded request timeout and no public TCP listener.
- [ ] Add `UPDATE_AGENT_SOCKET` discovery; in a container return a structured “agent unavailable” error when the socket is not configured or unreachable.
- [ ] Refactor update selection so `StartUpdate` checks the latest release once, sends only the validated release version to the agent in containers, and returns `NeedRestart=false` with the operation state.
- [ ] Keep the existing archive download/replacement path for non-container deployments and retain `PerformUpdate` as a compatibility wrapper.
- [ ] Make `GetUpdateStatus` proxy agent state in containers and return a clear unsupported error outside container mode.
- [ ] Run isolated service tests, including the existing container rejection test updated to cover agent-unavailable behavior.

### Task 3: Wire The Admin API Without Breaking Existing Handler Stubs

**Files:**
- Modify: `backend/internal/handler/admin/system_handler.go`
- Modify: `backend/internal/server/routes/admin.go`
- Modify: `backend/internal/handler/admin/system_handler_test.go`

**Interfaces:**
- `POST /api/v1/admin/system/update` returns `operation_id`, `status`, `version`, `message`, and `need_restart` for agent-backed updates.
- `GET /api/v1/admin/system/update/:operation_id` returns the current `UpdateExecution` state.

- [ ] Add handler tests for queued update responses, status proxying, agent-unavailable errors, and legacy stub fallback.
- [ ] Run the focused handler tests and confirm the new assertions fail.
- [ ] Add an optional async-update interface assertion so existing test doubles implementing only `PerformUpdate` continue to compile and use the old behavior.
- [ ] Call `StartUpdate` under the existing admin system operation lock; release the HTTP lock after the agent has accepted the task because the agent owns deployment serialization.
- [ ] Add the status endpoint and preserve admin authentication and idempotency behavior.
- [ ] Run the focused handler tests and confirm they pass.

### Task 4: Make The Version Badge Poll Docker Deployment State

**Files:**
- Modify: `frontend/src/api/admin/system.ts`
- Modify: `frontend/src/components/common/VersionBadge.vue`
- Create or modify: the existing frontend system API test file discovered under `frontend/src`.

**Interfaces:**
- `UpdateResult` accepts `operation_id`, `status`, `version`, `message`, and `need_restart`.
- `getUpdateStatus(operationId)` calls `/admin/system/update/{operationId}`.

- [ ] Add API/component tests for a queued operation, transient polling failure, terminal success, and terminal failure.
- [ ] Run the focused frontend tests and confirm they fail before implementation.
- [ ] Add `getUpdateStatus` and a bounded polling loop with a 20-minute deadline, 3-second interval, and limited transient network retries.
- [ ] Keep the existing binary update behavior: successful binary updates still show the restart action; agent-backed Docker updates wait for success and refresh the version cache.
- [ ] Display the agent failure message and keep the retry action available.
- [ ] Run the focused frontend tests and production frontend build.

### Task 5: Add Docker/Systemd Installation And Documentation

**Files:**
- Create: `deploy/update-agent.service`
- Create: `deploy/install-update-agent.sh`
- Modify: `deploy/docker-compose.hot.yml`
- Modify: `deploy/.env.example`
- Modify: `deploy/HOT_DEPLOY_CN.md`
- Modify: `deploy/README.md`
- Modify: `.github/workflows/deploy.yml`
- Create or modify: `deploy/tests/update-agent-install-test.sh`

**Interfaces:**
- Host configuration uses `UPDATE_AGENT_SOCKET`, `UPDATE_AGENT_IMAGE`, `UPDATE_AGENT_DEPLOY_PATH`, `UPDATE_AGENT_CURRENT_VERSION_FILE`, and `UPDATE_AGENT_DRAIN_SECONDS`.
- Compose mounts only the configured Unix socket into the app containers and passes the container-side socket path as `UPDATE_AGENT_SOCKET`.

- [ ] Add a systemd unit running the fixed update-agent binary as the configured deployment user, with restart-on-failure and a private environment file.
- [ ] Add an installation script that validates the deployment directory, image repository, version file, and socket path before enabling the service.
- [ ] Mount the agent socket in both blue and green app slots without mounting `/var/run/docker.sock`.
- [ ] Set the default drain value to `300` and document that `KEEP_OLD` remains opt-in only.
- [ ] Add shell tests that reject Docker Socket mounts, require the fixed 300-second drain default, and check the systemd unit’s fixed command.
- [ ] Update the deployment workflow to package/build the agent and install or refresh it on the server before hot deployment, without exposing secrets to the application container.
- [ ] Run shell syntax and deployment tests.

### Task 6: Full Verification And Handoff

**Files:**
- Verify all modified files above.

- [ ] Run `gofmt` on changed Go files and `git diff --check`.
- [ ] Run isolated agent, client, and handler tests with `GOMAXPROCS=1` and `-p 1` because the Windows environment has a small paging file.
- [ ] Run the frontend production build.
- [ ] Run deployment shell tests.
- [ ] Run the available full backend test command and record unrelated pre-existing failures without attributing them to this feature.
- [ ] Verify `git status` contains only intended local changes and no generated binaries or archives.
- [ ] Report that server-side agent installation and production deployment remain pending unless separately authorized.

