# ARCHITECTURE_ANALYSIS — Service Instance 整合分析

> 本文件為 code-change-request.md STEP 1/2 的產出：repository inspection 結果。
> 目標：在 Portainer CE 上加入「Service Instance」orchestration layer，重用既有
> Environment(Endpoint) / EndpointGroup / Stack / StackDeployer / 權限 / 持久化機制。

## 1. Backend entry point

- `api/cmd/portainer/main.go` — `main()` → `buildServer()` (line 376) 組裝所有 service，回傳 `portainer.Server`（即 `api/http/server.go` 的 `http.Server`）。
- `api/http/server.go` — `Server.Start(ctx)` 建立 middleware chain（offline gate → admin monitor → panic logger → CSRF），啟動 HTTP/HTTPS server。
- Handler 在 `Server.Start` 內以 `xxx.NewHandler(requestBouncer, ...)` 建立（line 134–303），最後填入 `handler.Handler{...}`（line 331 附近）。

## 2. API router 註冊

- 兩層路由：
  1. `api/http/handler/handler.go` — `Handler.ServeHTTP` 以 `strings.HasPrefix(r.URL.Path, "/api/xxx")` switch 分發到各 sub-handler，並 `http.StripPrefix("/api", ...)`。
  2. 各 sub-handler 用 gorilla/mux：`h.Handle("/stacks/{id}/start", bouncer.AuthenticatedAccess(httperror.LoggerHandler(h.stackStart))).Methods(http.MethodPost)`（見 `api/http/handler/stacks/handler.go:57-96`）。
- 新增 Service Instance 需：
  - 在 `api/http/handler/handler.go` 的 `Handler` struct 加 `ServiceInstanceHandler *serviceinstances.Handler`，並在 `ServeHTTP` switch 加 `/api/service-instances` 與 `/api/service-instance-operations` 前綴。
  - 在 `api/http/server.go` 建立 handler 並填入 struct。

## 3. Stack HTTP handlers

- 目錄：`api/http/handler/stacks/`
  - `handler.go` — `Handler` struct（含 `DataStore`、`requestBouncer`、`StackDeployer`、`ComposeStackManager`、`SwarmStackManager`、`teardownService` 等）+ 路由註冊 + 權限 helper（`userCanAccessStack`、`userCanManageStacks`、`checkUniqueStackNameInDocker`）。
  - `stack_create.go` — `stackCreate` → `createComposeStack` / `createSwarmStack` / `createKubernetesStack`。
  - `stack_start.go` — `stackStart`：讀 stack → 狀態檢查 → endpoint 權限 → resource control 權限 → `startStack()`（compose: `StackDeployer.DeployComposeStack`；swarm: `DeploySwarmStack`）→ 更新 `stack.Status`。
  - `stack_stop.go` — `stackStop`：compose → `StackDeployer.UndeployComposeStack`；swarm → `SwarmStackManager.Remove`。
  - `stack_update.go` — `stackUpdate` + 非同步 `stackDeploy`（goroutine）。
  - `stack_delete.go` — `stackDelete`（使用 `teardownService`）。
  - `stack_list.go` / `stack_inspect.go`。
- 權限檢查模式（每個 handler 重複）：
  ```go
  securityContext, _ := security.RetrieveRestrictedRequestContext(r)
  err = handler.requestBouncer.AuthorizedEndpointOperation(r, endpoint) // 403 if denied
  resourceControl, _ := handler.DataStore.ResourceControl().ResourceControlByResourceIDAndType(stackutils.ResourceControlID(endpointID, name), portainer.StackResourceControl)
  access, _ := handler.userCanAccessStack(securityContext, resourceControl)
  ```

## 4. Stack service layer

- **部署**：`api/stacks/deployments/deployer.go`
  ```go
  type BaseStackDeployer interface {
      DeploySwarmStack(ctx, stack, endpoint, registries, prune, pullImage) error
      DeployComposeStack(ctx, stack, endpoint, registries, prune, forcePullImage, forceRecreate) error
      UndeployComposeStack(ctx, stack, endpoint) error
      DeployKubernetesStack(ctx, stack, endpoint, user) error
      GetDockerClientFactory() *dockerclient.ClientFactory
  }
  type StackDeployer interface { BaseStackDeployer; RemoteStackDeployer }
  ```
  實作 `stackDeployer`（`NewStackDeployer(...)`），內部有 `sync.Mutex`。
- **拆除**：`api/stacks/teardown/teardown.go` — `Service` interface：`RemoveResources`、`DeleteRecords(tx, stack)`、`RemoveFiles(stack)`。
- **持久化**：`api/dataservices/stack/stack.go`（見 §7）。
- **工具**：`api/stacks/stackutils/` — `UserIsAdminOrEndpointAdmin`、`ResourceControlID(endpointID, name)`、`UpdateStackStatusFromUndeploymentResult`、`IsRelativePathStack`（CE 恆 false）。

## 5. Environment model

- 「Environment」在 codebase 中名為 **Endpoint**：`api/portainer.go:452` `Endpoint struct`。
  - 關鍵欄位：`ID EndpointID`、`Name`、`Type EndpointType`、`GroupID EndpointGroupID`（line 464，FK 在 endpoint 上）、`SecuritySettings`、`Status`。
- ID 型別：`EndpointID int`（line 707）、`EndpointType int`（line 741）。
- Data service：`api/dataservices/endpoint/endpoint.go`，bucket `"endpoints"`；`ReadAll(predicates ...func(portainer.Endpoint) bool)` 可用 predicate 過濾 `GroupID`。

## 6. Group model

- 「Group」名為 **EndpointGroup**：`api/portainer.go:549` `EndpointGroup struct`（`ID EndpointGroupID`、`Name`、`Description`、`UserAccessPolicies`、`TeamAccessPolicies`、`TagIDs`）。
- 關係：`Endpoint.GroupID`（一個 endpoint 只屬一個 group）。
- Data service：`api/dataservices/endpointgroup/endpointgroup.go`，bucket `"endpoint_groups"`。
- 預設 "Unassigned" group 在 `api/datastore/init.go` 建立。

## 7. Stack persistence

- `api/dataservices/stack/stack.go`：
  - `const BucketName = "stacks"`
  - `Service` 嵌入 `dataservices.BaseDataService[portainer.Stack, portainer.StackID]`
  - `NewService(connection)` → `connection.SetServiceName(BucketName)`（lazy 建 bucket）
  - `Create` → `Connection.CreateObjectWithId(BucketName, int(stack.ID), stack)`
  - key = 8-byte big-endian uint64（`api/database/boltdb/db.go:269 ConvertToKey`）
  - 值 = JSON（`api/database/boltdb/json.go` `MarshalObject`，可選 AES-GCM 加密）
- `BaseCRUD[T, I]` / `BaseDataService` / `BaseDataServiceTx`：`api/dataservices/base.go`、`base_tx.go`。
- 註冊：
  - `api/datastore/services.go` — `Store` struct 欄位、`initServices()`、accessor `Store.Stack()`、`storeExport`（backup/restore）。
  - `api/datastore/services_tx.go` — `StoreTx.Stack()`。
  - `api/dataservices/interface.go` — `DataStoreTx` / `DataStore` interface + `StackService` interface。

## 8. Datastore / migration

- `api/datastore/datastore.go` — `NewStore`、`Open()`、`UpdateTx`/`ViewTx`。
- Migrations：`api/datastore/migrator/migrator.go` — `addMigrations(version, funcs...)`；`Migrate()`（`migrate_ce.go`）比對 DB schema version 與 `portainer.APIVersion`。
- **新 bucket 不需 migration**：bucket 由 `SetServiceName` → `CreateBucketIfNotExists` lazy 建立（`api/database/boltdb/tx.go:19`）。Service Instance 只需在 `initServices()` 建立 service 即可。

## 9. 權限 / Authorization

- `api/http/security/bouncer.go` — `BouncerService`：
  - middleware：`AuthenticatedAccess`、`AdminAccess`、`RestrictedAccess`...
  - `AuthorizedEndpointOperation(r, endpoint) error`（line 156）：admin 直接過；否則查 team memberships + endpoint group 的 access policies → `AuthorizedEndpointAccess`。
- `api/http/security/authorization.go` — `AuthorizedEndpointAccess(endpoint, group, userID, memberships)`、`AuthorizedAccess(...)`。
- `api/http/security/context.go` — `RetrieveRestrictedRequestContext(r)` → `RestrictedRequestContext{IsAdmin, UserID, User, UserMemberships}`。
- Resource control：`api/internal/authorization/access_control.go` + `pkg/authorization/access_control.go` — `UserCanAccessResource(userID, teamIDs, resourceControl)`。
- 對 Service Instance 的 conservative policy：對每個 target endpoint 呼叫 `AuthorizedEndpointOperation`（或 `AuthorizedEndpointAccess`），任一失敗 → 整個 operation reject（403）。

## 10. Background job / task

- **沒有 job queue framework**。CE 用：
  - 一般 goroutine：`api/stacks/stackbuilders/director.go` `BuildAndAsyncDeploy`（`go deploy(...)`，15 分鐘 timeout context，persist 最終狀態）；`api/stacks/deployments/deploy.go` webhook redeploy 用 `go func()` + `singleflight`。
  - cron：`api/scheduler/scheduler.go`（robfig/cron）；`pkg/schedule/ticker.go` `RunOnInterval`。
  - worker pool：`api/concurrent/concurrent.go` `Run(ctx, maxConcurrency, tasks...)`。
- Service Instance 的 async operation 用 **plain goroutine + per-instance mutex**（與 stack 現行做法一致），不需新 framework。

## 11. Audit / activity log

- **CE 沒有 activity log service**（EE/BE only；frontend `useActivityLogs` 在非 BE 回 mock）。
- 唯一 shape hint：`api/internal/testhelpers/user_activity_service.go` `LogUserActivity(username, context, action string, payload []byte) error`（no-op stub）。
- 決策：Service Instance 的 audit 資訊寫入 **operation record 本身**（`ServiceInstanceOperation` 含 `UserID`、`Type`、`Status`、`Results`、timestamps），並用 zerolog 記錄 lifecycle action。不另建獨立 audit system（符合 prompt §27「優先重用」精神；CE 無既有機制可重用）。

## 12. HTTP error / response / request helpers

- `pkg/libhttp/error` — `HandlerError`、`BadRequest/NotFound/Conflict/Forbidden/InternalServerError`、`LoggerHandler`。
- `pkg/libhttp/response` — `JSON`、`Empty`、`TxResponse`（`txresponse.go`）。
- `pkg/libhttp/request` — `RetrieveNumericRouteVariableValue`、`RetrieveNumericQueryParameter`、`RetrieveJSONQueryParameter`、`RetrieveMultiPartFormFile`。

## 13. 既有 backend tests

- Handler tests：`api/http/handler/stacks/*_test.go`
  - `datastore.MustNewTestStore(t, true, false)` 建真實 BoltDB store
  - `testhelpers.NewTestRequestBouncer()`（pass-through mock）
  - `security.StoreRestrictedRequestContext(req, &security.RestrictedRequestContext{IsAdmin: true, UserID: 1, User: ...})` 注入 context
  - `h.ServeHTTP(w, req)` 後斷言 `w.Code`
- Service tests：`api/internal/.../*_test.go`、`api/stacks/deployments/deploy_test.go` 等。
- Test store：`api/datastore/teststore.go`。

## 14. Frontend

### 14.1 架構：AngularJS + React hybrid
- `app/index.js` — Angular module `portainer`，含 `UI_ROUTER_REACT_HYBRID`（`@uirouter/react-hybrid`）。
- 路由：ui-router，註冊於 `app/portainer/__module.js`（`$stateRegistryProvider.register(...)`，line 483+）。
  - 例：`docker.stacks`（`app/docker/__module.js:405`）→ `templateUrl: '~@/portainer/views/stacks/stacks.html'`（Angular 頁）；`docker.stacks.stack` → `component: 'stackItemView'`（React 頁）。
  - 頂層 portainer 路由（`portainer.home`、`portainer.gitops.*`...）在 `app/portainer/__module.js`。
- React view 註冊為 Angular component：`app/portainer/react/views/*.ts`
  ```ts
  .component('workflowsView', r2a(withUIRouter(withCurrentUser(WorkflowsListView)), []))
  ```
  組合於 `app/portainer/react/views/index.ts`（`viewsModule`）。

### 14.2 Stack UI
- `app/react/docker/stacks/`
  - `ListView/StacksDatatable/`（datatable 模式：`StacksDatatable.tsx`、`columns/`、`store.ts`、`TableActions.tsx`）
  - `ItemView/ItemView.tsx`（tabs：StackInfoTab、StackEditorTab、containers/services datatables）
  - `CreateView/CreateView.tsx` + `CreateStackForm/`
- 共用 stack 邏輯：`app/react/common/stacks/`（`types.ts`、`queries/`：`useStacks`、`useCreateStack`、`useDeleteStackMutation`、`useStackFile`...）

### 14.3 Environment / Group UI
- `app/react/portainer/environments/`
  - `ListView/ListView.tsx`（`PageHeader` + `EnvironmentsDatatable`）
  - `ItemView/ItemView.tsx`
  - `environment-groups/`（`ListView`、`ItemView`、`CreateView`、`environment-groups.service.ts`、`queries/`）
  - `queries/`（`useEnvironmentList`、`useEnvironment`、`useCreateEnvironmentMutation`...）
- 型別：`app/react/portainer/environments/types.ts`（`Environment`、`EnvironmentId`、`EnvironmentGroupId`、`EnvironmentType`）。

### 14.4 API client
- `app/react/portainer/services/axios/axios.ts` — axios instance，`baseURL: 'api'`，interceptors（agent target、cache、401 logout）。
- 呼叫模式：`axios.get<Stack[]>(buildStackUrl())` + `parseAxiosError`。

### 14.5 State management
- **React Query**（`@tanstack/react-query`）：`useQuery`/`useMutation` + `withError`/`withInvalidate`（`app/react-tools/react-query.ts`）。
- 例：`app/react/common/stacks/queries/useStacks.ts`。
- Polling：React Query `refetchInterval`（gitops workflows 有使用）。

### 14.6 頁面結構模式
- List：`PageHeader`（`@@/PageHeader`）+ datatable（`@@/datatable`）或 `SortableList`（gitops 模式）。
- Item：`WorkflowResourceHeader` 風格 header + tabs（`app/react/portainer/gitops/workflows/ItemView/`：OverviewTab、TargetsSection、StacksSection...）。
- 刪除確認：`confirmDelete`（`@@/modals/confirm`）；通知：`notifySuccess`/`notifyError`（`@/portainer/services/notifications`）。

### 14.7 導航（sidebar）
- 頂層：`app/react/sidebar/Sidebar.tsx` — `Home`、`EnvironmentSidebar`、`AppDeliverySidebar`、`EdgeComputeSidebar`、`SettingsSidebar`。
- 新增頂層 section 模式：`AppDeliverySidebar.tsx`（`SidebarSection` + `SidebarItem`，`to="portainer.gitops.workflows"`）。
- Environment-scoped：`DockerSidebar.tsx`（`SidebarItem to="docker.stacks"`）。
- Service Instances 是跨 environment 的頂層概念 → 加在 `Sidebar.tsx` 頂層（新 `ServiceInstancesSidebar` 或直接 `SidebarItem`）。

### 14.8 Frontend tests
- Vitest + React Testing Library；`vitest.config.mts`；msw handlers（`app/setup-tests/setup-handlers/*.ts`）。
- 例：`app/react/portainer/gitops/workflows/ItemView/ItemView.test.tsx`、`app/react/common/stacks/EditGitSettings/EditGitSettingsModal.test.tsx`（`http.get('/api/...')` mock）。
- Test utils：`app/react-tools/test-mocks.ts`、`app/react/test-utils/`。

## 15. Service Instance 整合點總結

| 關注點 | 重用/新增 |
|---|---|
| Domain model | 新增 `ServiceInstance`、`ServiceInstanceOperation`、`ServiceInstanceTargetResult` 於 `api/portainer.go`；`Stack` 加 `ServiceInstanceID` 欄位（ownership metadata） |
| Persistence | 新增 `api/dataservices/serviceinstance/`、`api/dataservices/serviceinstanceoperation/`（BaseDataService 模式）；註冊於 `datastore/services.go`、`services_tx.go`、`dataservices/interface.go`；bucket lazy 建立，**不需 migration** |
| Target resolution | 重用 `Endpoint().ReadAll(pred)`（group → `e.GroupID == g`）與 `EndpointGroup().Read` |
| Deploy/Start/Stop | 重用 `deployments.StackDeployer`（`DeployComposeStack`/`UndeployComposeStack`）；每個 target 建立/更新一個 `portainer.Stack`（`Type=DockerComposeStack`、`ServiceInstanceID` 設為 instance ID） |
| Ownership | `Stack.ServiceInstanceID` 欄位；deploy 前檢查既有 stack 的 ownership，不匹配 → reject |
| Async operation | plain goroutine（同 `stackbuilders/director.go` 模式）+ per-instance `sync.Mutex` guard（RUNNING → 409） |
| Authorization | 重用 `bouncer.AuthorizedEndpointOperation` / `security.AuthorizedEndpointAccess`，對每個 target 檢查，任一失敗 → 403 |
| API | 新增 `api/http/handler/serviceinstances/`；註冊於 `handler/handler.go` + `http/server.go` |
| Audit | operation record + zerolog（CE 無 activity log service） |
| Frontend | 新增 `app/react/portainer/service-instances/`（types/queries/ListView/ItemView/CreateView/EditView）；view 註冊於 `app/portainer/react/views/service-instances.ts`；路由於 `app/portainer/__module.js`；sidebar 頂層項目 |
