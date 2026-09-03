# Portainer CE — Service Instances

This fork of Portainer Community Edition adds **Service Instances**: a way to define a service once and roll it out to many environments in a single operation, with live monitoring and scheduled deployments.

A Service Instance is a logical orchestration object that groups a set of target environments (an endpoint group or a list of individual environments) and deploys a shared Compose definition to all of them.

## Features

### Service Instances

- **Define a service** — name, description, Compose file, and environment variables.
- **Multi-target deployment** — target an entire endpoint group or a hand-picked set of environments.
- **Full lifecycle control** — deploy, start, stop, redeploy, and refresh with a single action. Operations run asynchronously and are tracked end to end.
- **Per-target visibility** — every target reports its own status; the instance shows an aggregated status (running, partial, failed, etc.).
- **Operation history** — a full audit trail of every operation with per-target results and errors.

### Monitor

- A **Monitor tab** on each instance that shows live per-target status.
- Auto-refreshes target status every 3 seconds, with a toggle to disable auto-refresh.

### Deploy & Scheduled Builds

- A **Deploy tab** with a Compose editor to update the service definition and **Deploy now**.
- **Scheduled builds** — schedule a deploy for a future time. Images are pulled on all targets immediately so the deploy is fast and reliable when the time comes.
- **List and cancel** pending scheduled builds, with per-target build status (pending, pulling, image ready, deployed, failed, cancelled).

## How it works

1. Create a Service Instance with a Compose file and choose its targets.
2. Trigger a lifecycle operation (deploy, start, stop, redeploy, or refresh), or schedule a build for a future time.
3. Portainer resolves the target snapshot, runs the operation sequentially with fail-fast semantics, and persists per-target results as it goes.
4. Watch progress in the UI — the instance status, per-target results, and operation history update in real time.

Service Instances are available in the sidebar under **Service Instances** and are fully exposed through the REST API.

## REST API

All endpoints require authentication (JWT or API key).

| Method | Path | Description |
|---|---|---|
| GET | `/api/service-instances` | List service instances |
| POST | `/api/service-instances` | Create a service instance |
| GET | `/api/service-instances/{id}` | Inspect a service instance |
| PUT | `/api/service-instances/{id}` | Update a service instance |
| DELETE | `/api/service-instances/{id}` | Delete a service instance |
| POST | `/api/service-instances/{id}/deploy` | Deploy to all targets (async, 202 + operation) |
| POST | `/api/service-instances/{id}/start` | Start on all targets (async) |
| POST | `/api/service-instances/{id}/stop` | Stop on all targets (async) |
| POST | `/api/service-instances/{id}/redeploy` | Redeploy on all targets (async) |
| POST | `/api/service-instances/{id}/refresh` | Recompute aggregated status (sync) |
| POST | `/api/service-instances/{id}/schedule-build` | Schedule a build for a future time |
| GET | `/api/service-instances/{id}/scheduled-builds` | List scheduled builds |
| DELETE | `/api/service-instance-scheduled-builds/{id}` | Cancel a scheduled build |
| GET | `/api/service-instances/{id}/targets` | Resolved targets with per-target status |
| GET | `/api/service-instances/{id}/operations` | Operation history (newest first) |
| GET | `/api/service-instance-operations/{id}` | Inspect a single operation |

## Statuses

- **Instance status**: unknown, deploying, running, stopped, partial, failed.
- **Operation status**: pending, running, success, partial success, failed, cancelled.
- **Per-target status**: pending, running, success, failed, skipped.
- **Scheduled build status**: pending, pulling, image ready, deployed, failed, cancelled.

## Known limitations (MVP)

- Compose source is the web editor only (no file upload or git repository yet).
- Docker Compose stacks only (Swarm/Kubernetes targets are not supported).
- No automatic rollback; a partial failure requires a manual redeploy.
- Deleting an instance does not undeploy its stacks (they are kept as regular stacks).

## Design docs

- [SERVICE_INSTANCE_DESIGN.md](./SERVICE_INSTANCE_DESIGN.md) — domain model, persistence, service layer, API, and authorization design.
- [ARCHITECTURE_ANALYSIS.md](./ARCHITECTURE_ANALYSIS.md) — how Service Instances integrate with the existing Portainer codebase.
