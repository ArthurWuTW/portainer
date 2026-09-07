# Portainer AI Chatbot — Metadata / State Context Integration

你現在正在修改一份 **Portainer codebase**。

目標是新增一個內嵌於 Portainer Web UI 的 **AI Chatbot**，讓使用者可以透過自然語言詢問目前 Portainer Server 所管理的 environment、endpoint、stack、container，以及 Portainer 自己的 metadata / state。

AI 模型是地端 LLM，例如：

```text
Qwen3 / Qwen3.x
```

模型透過 OpenAI-compatible API 提供服務，因此 **不要假設模型本身是 OpenAI 官方 API**。

---

# 1. 核心目標

我要建立：

```text
Portainer Web UI
        │
        ├── existing Portainer UI
        │
        └── AI Chatbot
                │
                ▼
        AI Context / Tool API
                │
        ├── Portainer Metadata / State
        │
        └── Endpoint Runtime State
                │
                ├── Docker
                ├── Portainer Agent
                └── Kubernetes
```

AI Chatbot 的主要用途是：

> 讓地端 LLM 可以理解目前 Portainer Server 的狀態，並根據使用者 prompt 分析系統、找問題，以及未來進一步協助修改或操作系統。

---

# 2. 非常重要：先研究，不要直接改 code

在修改任何程式之前，先完整分析目前 Portainer architecture。

你必須先回答：

1. Portainer Server 的 metadata / state 存在哪裡？
2. Portainer 使用什麼 database？
3. `/data` 中有哪些重要資料？
4. 哪些資料是 persistent state？
5. 哪些資料是 runtime state？
6. Endpoint information 如何儲存？
7. Stack information 如何儲存？
8. User / Team / Access Control 如何儲存？
9. Server 如何與 Portainer Agent 通訊？
10. Container / image / network / volume / stats / logs 等資料如何取得？
11. 哪些資料來自 Portainer DB？
12. 哪些資料來自 Docker Engine？
13. 哪些資料來自 Agent？
14. 哪些資料來自 Kubernetes API？
15. 現有 Portainer API 是否已經提供上述資訊？

**不要因為需求而重新建立一套 duplicate database。**

優先重用 Portainer 現有的 service / API / repository / store / endpoint abstraction。

---

# 3. 先建立 Data Flow Map

在開始 implementation 前，畫出目前 Portainer 的資料流。

至少包含：

```text
Browser
   │
   ▼
Portainer Web UI
   │
   ▼
Portainer HTTP API
   │
   ▼
Portainer Server
   │
   ├── Portainer database
   │
   └── Endpoint abstraction
           │
           ├── Docker
           │      │
           │      └── Portainer Agent
           │
           └── Kubernetes
```

明確指出：

```text
Data Source
    ↓
Repository / Store / Service
    ↓
API Handler
    ↓
Frontend
```

以及：

```text
Data Source
    ↓
AI Context API
    ↓
LLM
```

---

# 4. AI 不應該直接讀 portainer.db

禁止採用：

```text
AI
 ↓
portainer.db
```

也不要讓 frontend 直接讀 database。

正確方向應該是：

```text
AI Chat
   │
   ▼
Portainer AI Context Layer
   │
   ├── existing Portainer services
   ├── existing stores
   ├── existing endpoint APIs
   └── existing Agent communication
```

也就是：

> AI 應該使用 Portainer 現有的 abstraction，而不是繞過 architecture 直接讀 database。

---

# 5. 建立 AI Context Layer

請設計一個新的 AI Context / Tool abstraction。

概念上：

```text
AI
 │
 ▼
AI Context Service
 │
 ├── Portainer metadata
 │
 ├── endpoint information
 │
 ├── stack information
 │
 ├── container information
 │
 ├── Docker runtime information
 │
 ├── Kubernetes information
 │
 └── system/runtime information
```

不要把所有資料一次 dump 給 LLM。

---

# 6. 建議的 AI Tools

請評估並實作合理的 read-only tools。

例如：

```text
get_portainer_state()

get_endpoints()

get_endpoint(endpoint_id)

get_stacks(endpoint_id)

get_stack(stack_id)

get_containers(endpoint_id)

get_container(endpoint_id, container_id)

get_container_logs(endpoint_id, container_id)

get_container_stats(endpoint_id, container_id)

get_images(endpoint_id)

get_networks(endpoint_id)

get_volumes(endpoint_id)

get_system_info(endpoint_id)
```

這些只是候選名稱。

**請先研究 Portainer 現有 architecture，再決定實際名稱與 interface。**

不要為了符合這份 prompt 而硬建立重複 API。

---

# 7. Metadata 與 Runtime State 必須分離

請明確區分：

## Portainer Metadata / State

例如：

```text
Endpoint
Endpoint settings
Endpoint name
Endpoint type
Endpoint status
Tags
Stacks
Registries
Users
Teams
Access control
Portainer settings
```

這些主要來自：

```text
Portainer Server
```

---

## Runtime State

例如：

```text
Container status
Container health
CPU usage
Memory usage
Network usage
Processes
Container logs
Docker events
Images
Networks
Volumes
Kubernetes workloads
```

這些應該透過 Portainer 現有 Endpoint abstraction / Agent / Docker / Kubernetes integration 取得。

不要假設 runtime state 永久存在 Portainer DB。

---

# 8. AI Context 不要一次塞全部資料

避免：

```text
get_everything()
```

然後把幾 MB / 幾十 MB JSON 丟給 LLM。

應該採用：

```text
User prompt
      │
      ▼
LLM
      │
      ▼
Tool call
      │
      ▼
取得必要資料
      │
      ▼
LLM
```

例如使用者問：

```text
為什麼 production 的 nginx container 一直重啟？
```

AI 應該可以：

```text
get_endpoints()
        ↓
找到 production
        ↓
get_containers(production)
        ↓
找到 nginx
        ↓
get_container(...)
        ↓
get_container_logs(...)
        ↓
get_container_stats(...)
        ↓
分析
```

而不是：

```text
把整個 Portainer state 全部送進 Qwen
```

---

# 9. AI Tool API 必須考慮 Token Cost

所有 AI Context response 都應該：

1. 最小化 JSON
2. 移除不必要欄位
3. 避免重複資料
4. 支援 pagination
5. 支援 filtering
6. 支援指定 endpoint
7. 支援指定 container / stack
8. 避免把 secrets 傳給 LLM

例如不要直接：

```json
{
  "container": {
    "Env": [
      "OPENAI_API_KEY=xxxxx",
      "DATABASE_PASSWORD=xxxxx"
    ]
  }
}
```

應該：

```json
{
  "container": {
    "name": "nginx",
    "status": "running",
    "image": "nginx:latest",
    "env": "[REDACTED]"
  }
}
```

---

# 10. Security

這一點非常重要。

AI Context API 不可以繞過 Portainer 原有 permission model。

例如：

```text
User
 ↓
Portainer permission
 ↓
AI Context
 ↓
Endpoint
```

不能：

```text
User
 ↓
AI
 ↓
直接讀所有 endpoint
```

AI 只能看到：

> 目前登入使用者本來就有權限看到的資料。

尤其注意：

```text
password
token
API key
registry credential
JWT
SSH key
environment secrets
database password
```

不要送進 LLM。

---

# 11. Frontend AI Chatbot

請在 Portainer Web UI 增加：

```text
┌──────────────────────────────┐
│ Portainer                    │
│                              │
│                         ┌────┤
│                         │ AI │
│                         │    │
│                         │ Hi │
│                         │    │
│                         │ ...│
│                         │    │
│                         └────┤
└──────────────────────────────┘
```

需求：

* Chat panel 可以展開 / 收起
* 可以調整寬度
* 不應破壞原本 Portainer layout
* 支援 streaming response
* 顯示 AI 正在執行哪個 tool
* 顯示 tool execution 狀態
* 顯示錯誤
* 支援清除 conversation
* 支援設定 LLM endpoint

---

# 12. LLM Endpoint 設定

AI Chatbot 必須支援 OpenAI-compatible endpoint。

例如：

```text
Endpoint:
http://localhost:8000/v1

Model:
Qwen3-27B
```

或：

```text
http://192.168.1.100:8000/v1
```

不要 hard-code OpenAI。

設定至少包含：

```text
Base URL
API Key (optional)
Model
Temperature
Max Tokens
```

API Key 必須安全處理，不可以出現在 log。

---

# 13. Tool Calling Architecture

優先採用：

```text
User
 │
 ▼
LLM
 │
 ├── tool_call: get_endpoints()
 │
 ▼
Portainer AI Tool Layer
 │
 ▼
Portainer existing service
 │
 ▼
Result
 │
 ▼
LLM
 │
 ▼
Final answer
```

不要使用：

```text
LLM
 ↓
任意 HTTP request
```

Tool 必須是明確定義的。

---

# 14. 第一階段只做 Read-only

第一階段：

**只允許 AI 讀資料。**

例如：

```text
READ

get_endpoints
get_stacks
get_containers
get_container
get_logs
get_stats
get_images
get_networks
get_volumes
get_system_info
```

暫時禁止：

```text
restart
stop
start
remove
deploy
delete
update
```

不要在第一版實作 destructive operation。

---

# 15. 未來可以擴充 Write Tools

架構必須讓未來可以增加：

```text
restart_container()
stop_container()
start_container()
deploy_stack()
remove_container()
update_stack()
```

但目前不要實作。

未來 write operation 必須支援：

```text
AI proposes action
       ↓
User confirmation
       ↓
Permission check
       ↓
Execute
       ↓
Return result
```

例如：

```text
AI:
nginx container has been unhealthy.

I recommend restarting nginx.

[Confirm Restart] [Cancel]
```

---

# 16. Conversation Context

AI Chatbot 必須區分：

```text
Conversation state
```

與：

```text
Portainer state
```

例如：

```text
User:
production nginx 怎麼了？

AI:
我發現 nginx health check failed。

User:
那幫我看看 log。

AI:
[tool call get_container_logs(...)]
```

Conversation 不應該把所有 Portainer state 永久存入 database。

請設計合理的 session / conversation abstraction。

---

# 17. 不要過度修改 Portainer

這是一個既有大型 open-source project。

請遵守：

```text
Minimal invasive change
Reuse existing abstractions
Reuse existing APIs
Follow existing coding style
Follow existing architecture
Avoid duplicate implementation
Avoid unnecessary dependencies
```

不要因為 AI feature 而：

```text
rewrite existing endpoint architecture
rewrite database layer
rewrite Docker integration
rewrite Agent communication
```

---

# 18. Implementation 前先產生 Architecture Proposal

在真正修改 code 前，請先輸出：

## A. Existing Architecture

列出實際 source files：

```text
server/
    ...
api/
    ...
app/
    ...
agent/
    ...
```

以及：

```text
file
 ↓
function
 ↓
service
 ↓
data source
```

---

## B. Metadata Data Flow

說明：

```text
Portainer metadata
        ↓
actual source code
        ↓
database/store/service
```

---

## C. Runtime Data Flow

說明：

```text
Portainer Server
        ↓
Agent
        ↓
Docker Engine
```

或 Kubernetes 對應流程。

---

## D. Proposed AI Architecture

例如：

```text
Web UI
  ↓
AI Chat API
  ↓
AI Context / Tool Service
  ↓
Existing Portainer Services
  ↓
DB / Agent / Docker / Kubernetes
```

---

## E. Files To Modify

列出：

```text
file path
reason
expected change
```

不要先改一堆檔案再解釋。

---

# 19. Testing

至少增加：

### Backend tests

測試：

```text
AI Context API
permission checking
endpoint filtering
secret redaction
tool execution
error handling
```

### Frontend tests

測試：

```text
open chatbot
close chatbot
resize chatbot
send message
stream response
tool status
error handling
```

---

# 20. Definition of Done

完成後應該能做到：

```text
1. User opens Portainer

2. User opens AI Chatbot

3. User configures:

   Base URL:
   http://localhost:8000/v1

   Model:
   Qwen3

4. User asks:

   "現在有哪些 environment？"

5. Qwen calls:

   get_endpoints()

6. Portainer returns only authorized endpoints.

7. User asks:

   "production 有哪些 container？"

8. Qwen calls:

   get_containers(production)

9. User asks:

   "nginx 為什麼一直重啟？"

10. Qwen can call:

    get_container()
    get_container_logs()
    get_container_stats()

11. Qwen gives an explanation.
```

---

# 21. 最重要的開發原則

請把這個功能視為：

> **Portainer AI observability / operations layer**

而不是：

> 「在 Portainer 旁邊放一個 ChatGPT iframe」。

AI 必須真正理解：

```text
Portainer
    │
    ├── Metadata
    │
    ├── State
    │
    ├── Endpoints
    │
    ├── Stacks
    │
    └── Runtime
          │
          ├── Docker
          ├── Agent
          └── Kubernetes
```

並且使用 Portainer 本身已有的 architecture。

---

# 22. 工作順序

嚴格按照以下順序：

```text
Phase 1
↓
Reverse engineer existing Portainer architecture

Phase 2
↓
Identify metadata/state sources

Phase 3
↓
Identify existing APIs/services that expose the data

Phase 4
↓
Design AI Context / Tool abstraction

Phase 5
↓
Design security / permission / secret redaction

Phase 6
↓
Implement backend

Phase 7
↓
Implement frontend chatbot

Phase 8
↓
Implement OpenAI-compatible LLM configuration

Phase 9
↓
Implement tool calling

Phase 10
↓
Tests

Phase 11
↓
Run lint / type check / unit tests / build

Phase 12
↓
Provide final architecture summary
```

---

# 23. 最後的要求

**不要猜 Portainer source code。**

如果某個 function、service、store、database schema 或 API 不確定：

1. 先搜尋 codebase
2. 找到實際 implementation
3. 再進行設計

不要自己假設：

```text
Portainer probably does X
```

而應該確認：

```text
actual file
actual function
actual call chain
actual data structure
```

最終請提供：

```text
1. Existing architecture
2. Data flow
3. Proposed architecture
4. Files changed
5. Why each file changed
6. API / Tool schema
7. Security model
8. Implementation
9. Tests
10. Remaining limitations
```

**先分析，再提出 implementation plan；確認 architecture 合理後，再開始修改 code。**

