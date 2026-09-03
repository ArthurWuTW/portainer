# SERVICE_INSTANCE_DESIGN — Service Instance MVP 設計

## 1. Domain Model

新增於 `api/portainer.go`（沿用既有 `type (...)` 區塊與 JSON tag 風格）：

```go
// ServiceInstanceID
ServiceInstanceID int

// ServiceInstanceTargetType: 1 = GROUP, 2 = ENVIRONMENTS
ServiceInstanceTargetType int
const (
    ServiceInstanceTargetGroup ServiceInstanceTargetType = iota + 1
    ServiceInstanceTargetEnvironments
)

// ServiceInstanceStatus: 0 unknown, 1 deploying, 2 running, 3 stopped, 4 partial, 5 failed
ServiceInstanceStatus int
const (
    ServiceInstanceStatusUnknown ServiceInstanceStatus = iota
    ServiceInstanceStatusDeploying
    ServiceInstanceStatusRunning
    ServiceInstanceStatusStopped
    ServiceInstanceStatusPartial
    ServiceInstanceStatusFailed
)

type ServiceInstance struct {
    ID           ServiceInstanceID       `json:"Id"`
    Name         string                  `json:"Name"`
    Description  string                  `json:"Description"`
    TargetType   ServiceInstanceTargetType `json:"TargetType"`
    GroupID      EndpointGroupID         `json:"GroupId,omitempty"`
    EnvironmentIDs []EndpointID          `json:"EnvironmentIds,omitempty"`
    StackName    string                  `json:"StackName"`   // deterministic, e.g. si-102-production-web
    ComposeFile  string                  `json:"ComposeFile"` // desired compose (web editor source)
    Env          []Pair                  `json:"Env"`
    Status       ServiceInstanceStatus   `json:"Status"`
    CreatedBy    string                  `json:"CreatedBy"`
    CreatedAt    int64                   `json:"CreatedAt"`
    UpdatedAt    int64                   `json:"UpdatedAt"`
}

// ServiceInstanceOperationType: 1 deploy, 2 start, 3 stop, 4 redeploy, 5 refresh
ServiceInstanceOperationType int
const (
    ServiceInstanceOperationDeploy ServiceInstanceOperationType = iota + 1
    ServiceInstanceOperationStart
    ServiceInstanceOperationStop
    ServiceInstanceOperationRedeploy
    ServiceInstanceOperationRefresh
)

// ServiceInstanceOperationStatus: 1 pending, 2 running, 3 success, 4 partial_success, 5 failed, 6 cancelled
ServiceInstanceOperationStatus int
const (
    ServiceInstanceOperationStatusPending ServiceInstanceOperationStatus = iota + 1
    ServiceInstanceOperationStatusRunning
    ServiceInstanceOperationStatusSuccess
    ServiceInstanceOperationStatusPartialSuccess
    ServiceInstanceOperationStatusFailed
    ServiceInstanceOperationStatusCancelled
)

// ServiceInstanceTargetStatus: 1 pending, 2 running, 3 success, 4 failed, 5 skipped
ServiceInstanceTargetStatus int
const (
    ServiceInstanceTargetStatusPending ServiceInstanceTargetStatus = iota + 1
    ServiceInstanceTargetStatusRunning
    ServiceInstanceTargetStatusSuccess
    ServiceInstanceTargetStatusFailed
    ServiceInstanceTargetStatusSkipped
)

type ServiceInstanceTargetResult struct {
    EnvironmentID EndpointID                `json:"EnvironmentId"`
    Status        ServiceInstanceTargetStatus `json:"Status"`
    Error         string                    `json:"Error,omitempty"`
}

type ServiceInstanceOperation struct {
    ID                ServiceInstanceOperationID `json:"Id"`
    ServiceInstanceID ServiceInstanceID          `json:"ServiceInstanceId"`
    Type              ServiceInstanceOperationType `json:"Type"`
    Status            ServiceInstanceOperationStatus `json:"Status"`
    UserID            UserID                     `json:"UserId"`
    StartedAt         int64                      `json:"StartedAt"`
    FinishedAt        *int64                     `json:"FinishedAt,omitempty"`
    Results           []ServiceInstanceTargetResult `json:"Results"`
}
```

`Stack` 新增欄位（ownership metadata，backward compatible，`omitempty`）：

```go
// ServiceInstanceID is set when the stack is deployed by a Service Instance.
ServiceInstanceID ServiceInstanceID `json:"ServiceInstanceId,omitempty"`
```

Stack name 規則：`si-<instanceID>-<sanitized-name>`（deterministic；ID 為 internal identity，name 只作 display component）。

## 2. Persistence

- `api/dataservices/serviceinstance/serviceinstance.go` + `tx.go`
  - `const BucketName = "service_instances"`
  - `Service` 嵌入 `BaseDataService[portainer.ServiceInstance, portainer.ServiceInstanceID]`
  - 額外方法：`GetNextIdentifier()`、`ReadAllByStatus`（不需要，ReadAll predicate 即可）
- `api/dataservices/serviceinstanceoperation/serviceinstanceoperation.go` + `tx.go`
  - `const BucketName = "service_instance_operations"`
  - 額外方法：`GetNextIdentifier()`、`ReadAllByServiceInstanceID(id)`（predicate）
- 註冊：
  - `api/datastore/services.go`：Store 欄位、`initServices()`、`ServiceInstance()` / `ServiceInstanceOperation()` accessors、`storeExport` 欄位 + Export/Import 迴圈
  - `api/datastore/services_tx.go`：`StoreTx.ServiceInstance()` / `ServiceInstanceOperation()`
  - `api/dataservices/interface.go`：`DataStoreTx` 加兩個 accessor；新增 `ServiceInstanceService`、`ServiceInstanceOperationService` interface
- Migration：bucket 由 `SetServiceName` lazy 建立 → **不需 migration**（與所有既有 entity 相同）。

## 3. Service Layer

`api/serviceinstance/service.go`（新 package，business logic 集中處）：

```go
type Service struct {
    dataStore   dataservices.DataStore
    stackDeployer deployments.StackDeployer
    fileService portainer.FileService
    mu sync.Mutex // per-service guard map
    running map[portainer.ServiceInstanceID]bool
}
```

方法：
- `Create(ctx, instance) error` — 驗證 name 唯一、target 非空、compose 非空；`StackName = "si-" + id + "-" + sanitize(name)`；status = Unknown。
- `Update(ctx, id, instance) error` — 更新 metadata/compose/targets；compose 改變後 status 保持（UI 顯示 out-of-sync 由 targets 狀態推斷）。
- `Delete(ctx, id) error` — 刪除 instance + 其 operations（stacks 保留，標記 orphan 由既有 stack 機制處理；MVP 不自動 undeploy，避免危險）。
- `Read` / `ReadAll`。
- `ResolveTargets(instance) ([]portainer.Endpoint, error)`：
  - GROUP：`Endpoint().ReadAll(func(e) bool { return e.GroupID == instance.GroupID })`
  - ENVIRONMENTS：逐一 `Endpoint().Read`，missing → 回傳 `TargetMissing` 標記（不 crash）。
- `AggregateStatus(targetStatuses []ServiceInstanceTargetStatus) ServiceInstanceStatus`：
  - 全 success/running → Running；全 stopped → Stopped；混合 → Partial；任一 failed 且無 running → Failed；deploying 中 → Deploying；無 target → Unknown。
- `HasRunningOperation(id) bool` — per-instance guard（mutex + map）。
- `ExecuteOperation(ctx, instanceID, opType, userID) (*portainer.ServiceInstanceOperation, error)`：
  1. 檢查 guard → 若 RUNNING 回 `ErrOperationInProgress`（handler 轉 409）
  2. resolve targets snapshot（fail-fast：empty → 400 "No deployment targets"）
  3. 建立 operation record（PENDING→RUNNING），persist
  4. `go s.runOperation(...)`：sequential、fail-fast；每 target：
     - 找/建 stack record（`StackByName` + endpoint 過濾；ownership 檢查 `ServiceInstanceID`）
     - DEPLOY/REDEPLOY：寫 compose 到 project path（`fileService`，同 stack create 的 `ProjectPath` 模式）→ `StackDeployer.DeployComposeStack`
     - START：`DeployComposeStack`（pullImage=false）
     - STOP：`UndeployComposeStack`
     - REFRESH：只讀 stack status 更新 result（不部署）
     - 每 target 完成即更新 operation record（partial persistence，crash 可恢復）
  5. 完成：更新 operation status（SUCCESS / PARTIAL_SUCCESS / FAILED）+ instance.Status（aggregate）
- `RefreshStatus(ctx, instanceID)` — 同步重算各 target stack 狀態並更新 instance status。

## 4. API

`api/http/handler/serviceinstances/`：

| Method | Path | Handler |
|---|---|---|
| GET | `/service-instances` | `serviceInstanceList` |
| POST | `/service-instances` | `serviceInstanceCreate` |
| GET | `/service-instances/{id}` | `serviceInstanceInspect` |
| PUT | `/service-instances/{id}` | `serviceInstanceUpdate` |
| DELETE | `/service-instances/{id}` | `serviceInstanceDelete` |
| POST | `/service-instances/{id}/deploy` | `serviceInstanceDeploy` |
| POST | `/service-instances/{id}/start` | `serviceInstanceStart` |
| POST | `/service-instances/{id}/stop` | `serviceInstanceStop` |
| POST | `/service-instances/{id}/redeploy` | `serviceInstanceRedeploy` |
| POST | `/service-instances/{id}/refresh` | `serviceInstanceRefresh` |
| GET | `/service-instances/{id}/targets` | `serviceInstanceTargets` |
| GET | `/service-instances/{id}/operations` | `serviceInstanceOperations` |
| GET | `/service-instance-operations/{id}` | `serviceInstanceOperationInspect` |

- 全部 `bouncer.AuthenticatedAccess(httperror.LoggerHandler(...))`。
- Lifecycle endpoints 回傳 `202 + operation`（async）；refresh 同步回傳 instance。
- 註冊：`handler/handler.go`（struct 欄位 + switch case `/api/service-instances`、`/api/service-instance-operations`）；`http/server.go` 建立 handler。

## 5. Authorization

- Create/Update：resolve targets → 對每個 endpoint 呼叫 `requestBouncer.AuthorizedEndpointOperation(r, endpoint)`；任一失敗 → 403（conservative，無 partial authorization）。
- Lifecycle（deploy/start/stop/redeploy/refresh）：同樣重新檢查所有 targets（Day-11 場景）。
- Delete：檢查所有 targets 權限（避免刪掉有權限的 instance 留下 orphan stacks 給無權限者）。
- List/Inspect：admin 看全部；非 admin 只看「所有 targets 都有權限」的 instances（filter，不 403）。

## 6. Operation Model / Deployment Flow

```
POST /service-instances/{id}/deploy
  → guard check (409 if running)
  → resolve targets snapshot
  → create operation (RUNNING)
  → 202 {operation}
  → goroutine:
      for each target (sequential, fail-fast):
          result = execute on endpoint (via StackDeployer)
          persist operation result
      finalize operation status + instance status
```

- Concurrency：per-instance `sync.Mutex` + running map；同 instance 第二個 operation → 409 Conflict。
- Fail-fast：B 失敗 → C 標記 SKIPPED，operation = PARTIAL_SUCCESS（有 success 有 fail）或 FAILED（全 fail）。
- 無自動 rollback（§26）。
- Crash recovery：server 啟動時把 RUNNING operations 標記 FAILED（同 `recoverStaleDeployingStacks` 模式，加在 main.go 的 recover 區塊）。

## 7. Error Handling

- 400：invalid payload / empty targets / invalid compose
- 403：任一 target 無權限
- 404：instance/operation 不存在
- 409：operation in progress / stack ownership conflict / stack name collision
- 500：DB / deploy 錯誤
- 每 target 錯誤記錄在 `Results[].Error`，不中斷 HTTP 回應（operation 已建立）。

## 8. Frontend Architecture

`app/react/portainer/service-instances/`：
- `types.ts` — ServiceInstance、ServiceInstanceOperation、TargetResult、enums
- `queries/` — `useServiceInstances`、`useServiceInstance`、`useCreateServiceInstance`、`useUpdateServiceInstance`、`useDeleteServiceInstance`、`useServiceInstanceOperation`（polling `refetchInterval` while running）、`useServiceInstanceOperations`、`useServiceInstanceTargets`
- `ListView/ListView.tsx` — PageHeader + datatable（Name / Target / Status / Actions）
- `ItemView/ItemView.tsx` — header（status + Deploy/Start/Stop/Redeploy 按鈕）+ tabs：Overview / Targets / Compose / Operations
- `CreateView/CreateView.tsx` — name、description、target mode（group select / environment multi-select）、compose editor（web editor；MVP 不做 upload/git）
- `EditView/EditView.tsx` — 重用 CreateView form
- 註冊：`app/portainer/react/views/service-instances.ts`（r2a + withUIRouter + withCurrentUser + withReactQuery）→ `views/index.ts`
- 路由：`app/portainer/__module.js` — `portainer.service-instances`（list）、`portainer.service-instances.item`（detail）、`portainer.service-instances.new`（create）
- Sidebar：`app/react/sidebar/Sidebar.tsx` 頂層加 `SidebarItem to="portainer.service-instances"`（icon: Layers/Boxes）

## 9. Testing

- Backend unit：service layer（target resolution、status aggregation、stack name gen、operation lifecycle、partial failure、concurrency guard、ownership check）
- API tests：handler tests（create/list/inspect/update/delete/deploy/start/stop，用 `MustNewTestStore` + `NewTestRequestBouncer` + security context 注入）
- Frontend：list renders、detail renders、deploy action、operation polling、partial failure display、permission error

## 10. Migration / Backward Compatibility

- 新 bucket lazy 建立，舊 DB 不需 migration。
- `Stack.ServiceInstanceID` 為 `omitempty` 新欄位，舊 stack 不受影響。
- 舊 backup 無 `service_instances` 欄位 → Import 時為空 slice，無影響。
- 無 breaking API 變更（純新增 endpoints）。

## 11. Existing code reused

- `BaseDataService` / `BaseCRUD`（persistence 模式）
- `deployments.StackDeployer`（deploy/undeploy compose）
- `security.BouncerService.AuthorizedEndpointOperation`（endpoint 權限）
- `httperror` / `response` / `request` helpers
- `fileService`（compose file 寫入 project path）
- React Query + axios + r2a/ui-router hybrid 模式
- `PageHeader`、datatable、`confirmDelete`、`notifySuccess`

## 12. New code required

- `api/portainer.go`：ServiceInstance/Operation/TargetResult 型別 + Stack 欄位
- `api/dataservices/serviceinstance/`、`api/dataservices/serviceinstanceoperation/`
- `api/serviceinstance/`（business logic + operation executor）
- `api/http/handler/serviceinstances/`
- `app/react/portainer/service-instances/`
- 註冊點：`datastore/services.go`、`services_tx.go`、`dataservices/interface.go`、`handler/handler.go`、`http/server.go`、`main.go`（crash recovery）、`app/portainer/__module.js`、`app/portainer/react/views/`、`app/react/sidebar/Sidebar.tsx`

## 13. Potential breaking changes

- 無。純新增。唯一共享模型變更是 `Stack` 加 `omitempty` 欄位（JSON 序列化向後相容）。

## 14. Changed Files (implementation)

### Backend (new)
- `api/dataservices/serviceinstance/serviceinstance.go` + `tx.go` — `service_instances` bucket data service
- `api/dataservices/serviceinstanceoperation/serviceinstanceoperation.go` + `tx.go` — `service_instance_operations` bucket data service
- `api/serviceinstance/service.go` + `service_test.go` — business logic + operation executor
- `api/http/handler/serviceinstances/` — `handler.go` + 13 endpoint handlers + `serviceinstance_test.go`

### Backend (modified)
- `api/portainer.go` — `ServiceInstance`、`ServiceInstanceOperation`、`ServiceInstanceTargetResult` 型別 + 各 enum 常數；`Stack.ServiceInstanceID` 欄位
- `api/dataservices/interface.go` — `DataStoreTx` 加 `ServiceInstance()` / `ServiceInstanceOperation()`；新增兩個 service interface
- `api/datastore/services.go` — Store 欄位、`initServices()`、accessors、`storeExport` + Export/Import
- `api/datastore/services_tx.go` — `StoreTx` accessors
- `api/datastore/test_data/output_24_to_latest.json` — migration 測試 fixture 加兩個新 bucket（空）
- `api/http/handler/handler.go` — `ServiceInstanceHandler` 欄位 + 兩個 route prefix case
- `api/http/server.go` — 建立 handler + service 並注入
- `api/cmd/portainer/main.go` — `recoverStaleServiceInstanceOperations`（crash recovery）
- `api/internal/testhelpers/datastore.go` — `testDatastore` 實作新 interface 方法

### Frontend (new)
- `app/react/portainer/service-instances/types.ts`
- `app/react/portainer/service-instances/service-instance.service.ts` — axios API client
- `app/react/portainer/service-instances/queries/` — query-keys + 8 個 hooks（list/item/targets/operations/operation + create/update/delete/lifecycle mutations；operations 在 RUNNING/PENDING 時 3s polling）
- `app/react/portainer/service-instances/ListView/` — `ListView.tsx` + `columns.tsx` + `ListView.test.tsx`
- `app/react/portainer/service-instances/ItemView/` — `ItemView.tsx`、`ServiceInstanceResourceHeader.tsx`（Deploy/Start/Stop/Redeploy/Refresh 按鈕）、`OverviewTab.tsx`、`TargetsTab.tsx`、`ComposeTab.tsx`、`OperationsTab.tsx`、`ItemView.test.tsx`
- `app/react/portainer/service-instances/CreateView/` — `CreateView.tsx` + `ServiceInstanceForm.tsx`（Formik + yup；group select / environment multi-select / compose editor）
- `app/react/portainer/service-instances/EditView/EditView.tsx` — 重用 form
- `app/react/portainer/service-instances/test-utils/mocks.ts`
- `app/portainer/react/views/service-instances.ts` — Angular module 註冊 4 個 React views

### Frontend (modified)
- `app/portainer/react/views/index.ts` — 註冊 `serviceInstancesModule`
- `app/portainer/__module.js` — 4 個 ui-router states：`portainer.service-instances`、`.item`、`.new`、`.item.edit`
- `app/react/sidebar/Sidebar.tsx` — 頂層 "Service Instances" 導覽項目

## 15. API Documentation

Base path: `/api`。全部 endpoints 需 authenticated（JWT 或 API key）。

| Method | Path | Description | Success | Errors |
|---|---|---|---|---|
| GET | `/service-instances` | List service instances（非 admin 只回傳所有 targets 都有權限的） | 200 `ServiceInstance[]` | 500 |
| POST | `/service-instances` | Create。Body: `{Name, Description, TargetType, GroupId?, EnvironmentIds?, ComposeFile, Env?}` | 200 `ServiceInstance` | 400 invalid payload / no targets, 403 no target permission, 500 |
| GET | `/service-instances/{id}` | Inspect | 200 `ServiceInstance` | 400, 404, 500 |
| PUT | `/service-instances/{id}` | Update | 200 `ServiceInstance` | 400, 403, 404, 500 |
| DELETE | `/service-instances/{id}` | Delete（stacks 保留） | 204 | 400, 403, 404, 500 |
| POST | `/service-instances/{id}/deploy` | Async deploy to all targets | 202 `ServiceInstanceOperation` | 400 no targets, 403, 404, 409 in progress, 500 |
| POST | `/service-instances/{id}/start` | Async start | 202 `ServiceInstanceOperation` | 同上 |
| POST | `/service-instances/{id}/stop` | Async stop | 202 `ServiceInstanceOperation` | 同上 |
| POST | `/service-instances/{id}/redeploy` | Async redeploy | 202 `ServiceInstanceOperation` | 同上 |
| POST | `/service-instances/{id}/refresh` | Sync 重算 aggregated status | 200 `ServiceInstance` | 400, 403, 404, 500 |
| GET | `/service-instances/{id}/targets` | Resolved targets + per-target stack status + missing 標記 | 200 `ServiceInstanceTarget[]` | 400, 404, 500 |
| GET | `/service-instances/{id}/operations` | Operation history（newest first） | 200 `ServiceInstanceOperation[]` | 400, 404, 500 |
| GET | `/service-instance-operations/{id}` | Inspect single operation（含 per-target results） | 200 `ServiceInstanceOperation` | 400, 404, 500 |

`ServiceInstanceOperation` 回應範例（partial failure）：

```json
{
  "Id": 123,
  "ServiceInstanceId": 10,
  "Type": 1,
  "Status": 4,
  "UserId": 1,
  "StartedAt": 1700000000,
  "FinishedAt": 1700000010,
  "Results": [
    { "EnvironmentId": 1, "Status": 3 },
    { "EnvironmentId": 2, "Status": 3 },
    { "EnvironmentId": 3, "Status": 4, "Error": "stack deployment failed" }
  ]
}
```

## 16. DB / Model Migration Notes

- 新 buckets `service_instances`、`service_instance_operations` 由 `SetServiceName` → `CreateBucketIfNotExists` lazy 建立，**不需 migration**（與所有既有 entity 相同模式）。
- `Stack` 新增 `ServiceInstanceId`（`omitempty`）：舊 stack 序列化/反序列化不受影響；舊 backup import 不受影響。
- Migration 測試 fixture `output_24_to_latest.json` 已加入兩個空 bucket 以符合新 export 格式。
- Crash recovery：server 啟動時 `recoverStaleServiceInstanceOperations` 把 RUNNING operations 標記 FAILED、PENDING/RUNNING targets 標記 SKIPPED（同 `recoverStaleDeployingStacks` 模式）。

## 17. Testing Results

- Backend: `go build ./...` 通過；`go test ./api/...` 全數通過（110 packages ok；唯一失敗為 pre-existing 的 `TestHandler_pingRegistry_DockerHubURL`，需真實網路連 registry-1.docker.io，與本變更無關，clean tree 上同樣 flaky）。
- 新增 backend tests：
  - `api/serviceinstance/service_test.go` — StackName、AggregateStatus、Create、ResolveTargets（group/environments/missing）、ExecuteOperation（no targets、concurrency guard、partial failure）
  - `api/http/handler/serviceinstances/serviceinstance_test.go` — 14 個 handler tests（create/invalid payload/list/inspect/not found/update/delete/deploy/no targets/start/stop/targets/operations/operation inspect）
- Frontend: `pnpm typecheck` 通過；`pnpm exec eslint`（新檔案）通過；`pnpm vitest run app/react/portainer/service-instances` 7/7 通過（list renders、empty state、detail renders、tabs、overview、partial failure display、missing targets）。
- Full frontend suite：312 files，唯一失敗為 pre-existing flaky `HelmApplicationView` test（clean tree full-suite 下同樣失敗，isolated 通過；timing-sensitive，與本變更無關）。

## 18. Known Limitations (MVP)

- Compose source 僅支援 web editor（MVP 不做 upload / git repository，符合 prompt §3 UX 但只實作 web editor）。
- 僅支援 Docker Compose stacks（swarm/k8s targets 未支援；target endpoint 為 swarm-only 時 deploy 會失敗並記錄在 result）。
- 無自動 rollback（§26）；partial failure 後需手動 redeploy。
- Concurrency guard 為 in-memory（per-process）；多 instance 部署 Portainer 時不跨 process 去重（BoltDB 單寫者模型下可接受）。
- 無 OUT_OF_SYNC status enum（§31 允許 MVP 省略）；desired vs actual 差異由 targets 狀態呈現。
- Delete 不 undeploy stacks（保留為 regular stacks，避免危險）。
- 無 server-side pagination/sorting（list 回傳全部；MVP 數量小）。
- Audit 僅為 operation record + zerolog（CE 無 activity log service 可重用）。

## 19. Future Roadmap

- Compose source：file upload、git repository（重用 `stackbuilders` git builder + `Source`/`Workflow` 機制）
- 平行部署（`parallelism = N`，重用 `api/concurrent`）
- Continue-on-failure 選項（取代 fail-fast）
- OUT_OF_SYNC / DRIFTED status（desired vs actual compose hash 比對）
- 自動 rollback / canary / blue-green（§42 明確排除於 MVP）
- 跨 instance dependency graph、health check framework
- 多 Portainer process 的 distributed lock（若未來支援多寫者）
- Kubernetes / Swarm targets 支援
