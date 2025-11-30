# KubeTask Architecture & API Design

## Table of Contents

1. [System Overview](#system-overview)
2. [API Design](#api-design)
3. [System Architecture](#system-architecture)
4. [Custom Resource Definitions](#custom-resource-definitions)
5. [Agent Template Discovery](#agent-template-discovery)
6. [Complete Examples](#complete-examples)
7. [kubectl Usage](#kubectl-usage)
8. [Migration Guide](#migration-guide)

---

## System Overview

KubeTask is a Kubernetes-native system that executes AI-powered tasks across multiple repositories using Custom Resources (CRs) and the Operator pattern.

### Core Goals

- Use Kubernetes CRDs to define Batch and Task resources
- Use Controller pattern to manage resource lifecycle
- Execute tasks as Kubernetes Jobs
- No external databases or message queues required
- Seamless integration with Kubernetes ecosystem

### Key Advantages

✅ **Simplified Architecture**: No PostgreSQL, Redis - reduced component dependencies
✅ **Native Integration**: Works seamlessly with Tekton, Argo, Flux and other K8s tools
✅ **Declarative Management**: Use K8s resource definitions, supports GitOps
✅ **Infrastructure Reuse**: Logs, monitoring, auth/authz all leverage K8s capabilities
✅ **Simplified Operations**: Manage with standard K8s tools (kubectl, dashboard)

---

## API Design

### Complete Resource Overview

| Resource | Purpose | Stability |
|----------|---------|-----------|
| **Batch** | Task batch template (WHAT + WHERE) | Stable - semantic name |
| **BatchRun** | Batch execution instance | Stable - follows Batch |
| **Task** | Single task execution (simplified API) | Stable - semantic name |
| **WorkspaceConfig** | Workspace environment config (HOW) | **Stable - independent of project name** |

### Key Design Decisions

#### 1. Batch (not Bundle)

**Rationale**: "Batch" better expresses batch processing concept and aligns with K8s `batch/v1`.

```yaml
apiVersion: kubetask.io/v1alpha1
kind: Batch
```

#### 2. WorkspaceConfig (not KubeTaskConfig)

**Rationale**:
- ✅ **Stable**: Independent of project name - won't change even if project renames
- ✅ **Semantic**: Reflects architecture philosophy: "Workspace = AI + permissions + tools"
- ✅ **Clear**: Configures the workspace environment for task execution

```yaml
apiVersion: kubetask.io/v1alpha1
kind: WorkspaceConfig
```

#### 3. AgentTemplateRef (not JobTemplateRef)

**Rationale**: More semantic - we're configuring AI agents, not generic jobs.

```yaml
spec:
  agentTemplateRef:
    name: claude-agent
    key: agent-template.yaml
```

#### 4. variableContexts (not targets/variants/contextSets)

**Rationale**: Highlights constant/variable dichotomy perfectly.

```yaml
spec:
  commonContext: [...]      # Constant
  variableContexts: [...]   # Variable
```

### Resource Hierarchy

```
Batch (template)
├── BatchSpec
│   ├── commonContext: []Context
│   └── variableContexts: [][]Context
│
BatchRun (execution)
├── BatchRunSpec
│   ├── batchRef: string
│   ├── batchSpec: *BatchSpec
│   └── agentTemplateRef: *AgentTemplateReference
└── BatchRunStatus
    ├── phase: BatchRunPhase
    ├── progress: ProgressStatus
    └── tasks: []TaskStatus

Task (single task)
├── TaskSpec
│   └── contexts: []Context
└── TaskExecutionStatus
    ├── phase: TaskPhase
    ├── jobName: string
    ├── startTime: Time
    └── completionTime: Time

WorkspaceConfig (environment)
└── WorkspaceConfigSpec
    └── agentTemplateRef: *AgentTemplateReference
```

### Complete Type Definitions

```go
// Core resources
type Batch struct {
    Spec BatchSpec
}

type BatchSpec struct {
    CommonContext    []Context
    VariableContexts [][]Context
}

type BatchRun struct {
    Spec   BatchRunSpec
    Status BatchRunStatus
}

type BatchRunSpec struct {
    BatchRef         string
    BatchSpec        *BatchSpec
    AgentTemplateRef *AgentTemplateReference
}

type Task struct {
    Spec   TaskSpec
    Status TaskExecutionStatus
}

type TaskSpec struct {
    Contexts []Context
}

type TaskExecutionStatus struct {
    Phase          TaskPhase
    JobName        string
    StartTime      *metav1.Time
    CompletionTime *metav1.Time
}

type WorkspaceConfig struct {
    Spec WorkspaceConfigSpec
}

type WorkspaceConfigSpec struct {
    AgentTemplateRef *AgentTemplateReference
}

// Context system
type Context struct {
    Type       ContextType
    File       *FileContext
    Repository *RepositoryContext
}

type ContextType string
const (
    ContextTypeFile       ContextType = "File"
    ContextTypeRepository ContextType = "Repository"
)
```

---

## System Architecture

### Component Layers

```
┌─────────────────────────────────────────────────────────────┐
│                   Kubernetes API Server                     │
│  - Custom Resource Definitions (CRDs)                       │
│  - RBAC & Authentication                                    │
│  - Event System                                             │
└─────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│              KubeTask Controller (Operator)                 │
│  - Watch Batch & BatchRun CRs                              │
│  - Reconcile loop                                           │
│  - Create Kubernetes Jobs for tasks                         │
│  - Update CR status fields                                  │
│  - Handle retries and failures                              │
└─────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│                   Kubernetes Jobs/Pods                      │
│  - Each task runs as a separate Job/Pod                    │
│  - Execute task using agent scripts                         │
│  - Git worktree management                                  │
│  - AI agent invocation                                      │
│  - Update parent CR status                                  │
└─────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│                      Storage Layer                          │
│  ┌──────────────────────────────────────────────────────┐   │
│  │ etcd (Kubernetes Backend)                            │   │
│  │  - Batch CRs                                         │   │
│  │  - BatchRun CRs                                      │   │
│  │  - CR status (execution state, results)              │   │
│  └──────────────────────────────────────────────────────┘   │
│  ┌──────────────────────────────────────────────────────┐   │
│  │ PersistentVolume (PVC)                               │   │
│  │  - /workspace (bare repos)                           │   │
│  │  - Pod logs (managed by Kubernetes)                  │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

---

## Custom Resource Definitions

### Design Principles

KubeTask follows **separation of concerns**:

- **Batch**: Defines task content (WHAT) and repository scope (WHERE)
- **WorkspaceConfig**: Defines execution config (HOW) - Agent template reference
- **BatchRun**: Execution instance, can override WorkspaceConfig settings

**Benefits**:
1. **Simple Batch**: Only focuses on task definition, easy to create and maintain
2. **Centralized Config**: All execution config managed in WorkspaceConfig
3. **Flexible Override**: BatchRun can override default config as needed
4. **Stability**: WorkspaceConfig name independent of project name, won't change if project renames

### Batch (Task Definition)

Batch defines the content and targets of batch processing tasks.

```yaml
apiVersion: kubetask.io/v1alpha1
kind: Batch
metadata:
  name: update-dependencies
  namespace: kubetask-system
spec:
  # Common context - shared by all tasks
  commonContext:
    - type: File
      file:
        name: task.md
        source:
          inline: |
            Update dependencies to latest versions.
            Run tests and create PR.

    - type: File
      file:
        name: guide.md
        source:
          configMapKeyRef:
            name: workflow-guides
            key: standard-pr-workflow.md

  # Variable context - different per task
  variableContexts:
    # Task 1: service-a + its config
    - - type: Repository
        repository:
          org: myorg
          repo: service-a
          branch: main

      - type: File
        file:
          name: config.json
          source:
            configMapKeyRef:
              name: dep-configs
              key: service-a.json

    # Task 2: service-b + its config + credentials
    - - type: Repository
        repository:
          org: myorg
          repo: service-b
          branch: main

      - type: File
        file:
          name: config.json
          source:
            configMapKeyRef:
              name: dep-configs
              key: service-b.json

      - type: File
        file:
          name: credentials.json
          source:
            secretKeyRef:
              name: private-credentials
              key: service-b-creds.json

    # Task 3: service-c on develop branch
    - - type: Repository
        repository:
          org: myorg
          repo: service-c
          branch: develop

      - type: File
        file:
          name: config.json
          source:
            configMapKeyRef:
              name: dep-configs
              key: service-c.json
```

**Field Description:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `spec.commonContext` | []Context | Yes | Common context, shared by all tasks |
| `spec.variableContexts` | [][]Context | Yes | Variable context, each element generates one task |

**Task Generation Formula:**

```
Task[i] = commonContext + variableContexts[i]
Total Tasks = len(variableContexts)
```

**Example:**
```
commonContext = [task.md, guide.md]
variableContexts = [
  [service-a:main, service-a.json],
  [service-b:main, service-b.json, creds.json],
  [service-c:develop, service-c.json]
]

Generated Tasks:
Task 1 = [task.md, guide.md, service-a:main, service-a.json]
Task 2 = [task.md, guide.md, service-b:main, service-b.json, creds.json]
Task 3 = [task.md, guide.md, service-c:develop, service-c.json]

Total: 3 tasks
```

**Context Types:**

1. **File Context**:
```yaml
type: File
file:
  name: task.md
  source:
    inline: "Task description"
    # OR
    configMapKeyRef:
      name: configs
      key: task.md
    # OR
    secretKeyRef:
      name: secrets
      key: credentials.json
```

2. **Repository Context**:
```yaml
type: Repository
repository:
  org: myorg
  repo: service-a
  branch: main
```

### BatchRun (Execution Instance)

BatchRun is a concrete execution instance of a Batch. Each execution creates a new Run.

```yaml
apiVersion: kubetask.io/v1alpha1
kind: BatchRun
metadata:
  name: update-dependencies-run-001
  namespace: kubetask-system
  labels:
    team: platform
    jira: PROJ-1234
spec:
  # Method 1: Reference existing Batch (recommended)
  batchRef: update-dependencies

  # Method 2: Inline Batch definition (optional, choose one)
  # batchSpec:
  #   commonContext: [...]
  #   variableContexts: [...]

  # (Optional) Agent template reference
  # If not specified, uses convention name "kubetask-agent"
  # If convention ConfigMap doesn't exist, uses built-in default template
  agentTemplateRef:
    name: claude-agent  # Optional: use specific agent
    key: agent-template.yaml  # Optional: default is "agent-template.yaml"

status:
  # Execution phase (BatchRunPhase enum)
  phase: Running  # Pending|Running|Succeeded|Failed

  # Start and end times
  startTime: "2025-01-18T10:00:00Z"
  completionTime: "2025-01-18T10:15:00Z"

  # Progress statistics
  progress:
    total: 2
    pending: 0
    running: 1
    completed: 1
    failed: 0

  # Task list
  tasks:
    - contexts:
        - type: Repository
          repository:
            org: myorg
            repo: service-a
            branch: main
      status: Succeeded  # TaskPhase enum: Pending|Running|Succeeded|Failed
      jobName: update-dependencies-run-001-task-0
      startTime: "2025-01-18T10:01:00Z"
      completionTime: "2025-01-18T10:05:00Z"

    - contexts:
        - type: Repository
          repository:
            org: myorg
            repo: service-b
            branch: main
      status: Running
      jobName: update-dependencies-run-001-task-1
      startTime: "2025-01-18T10:02:00Z"
```

**Status Field Description:**

| Field | Type | Description |
|-------|------|-------------|
| `status.phase` | BatchRunPhase | Execution phase: Pending\|Running\|Succeeded\|Failed |
| `status.startTime` | Timestamp | Start time |
| `status.completionTime` | Timestamp | End time |
| `status.progress` | Object | Progress statistics |
| `status.tasks` | []TaskStatus | Task details list |

### Task (Single Task Execution)

Task is a simplified API for users who want to execute a single task without creating a Batch. Unlike BatchRun which manages multiple tasks, Task is for simple one-off executions.

```yaml
apiVersion: kubetask.io/v1alpha1
kind: Task
metadata:
  name: update-service-a
  namespace: kubetask-system
  labels:
    team: platform
spec:
  # Contexts defines what this task operates on
  # This is a simple list of contexts (no common/variable separation)
  contexts:
    # File context - task description
    - type: File
      file:
        name: task.md
        source:
          inline: |
            Update dependencies to latest versions.
            Run tests and create PR.

    # File context - workflow guide
    - type: File
      file:
        name: guide.md
        source:
          configMapKeyRef:
            name: workflow-guides
            key: standard-pr-workflow.md

    # Repository context
    - type: Repository
      repository:
        org: myorg
        repo: service-a
        branch: main

    # Config file from ConfigMap
    - type: File
      file:
        name: config.json
        source:
          configMapKeyRef:
            name: dep-configs
            key: service-a.json

status:
  # Execution phase (TaskPhase enum)
  phase: Running  # Pending|Running|Succeeded|Failed

  # Kubernetes Job name
  jobName: update-service-a-xyz123

  # Start and end times
  startTime: "2025-01-18T10:00:00Z"
  completionTime: "2025-01-18T10:05:00Z"
```

**Field Description:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `spec.contexts` | []Context | Yes | List of contexts (files, repositories, etc.) |

**Status Field Description:**

| Field | Type | Description |
|-------|------|-------------|
| `status.phase` | TaskPhase | Execution phase: Pending\|Running\|Succeeded\|Failed |
| `status.jobName` | String | Kubernetes Job name |
| `status.startTime` | Timestamp | Start time |
| `status.completionTime` | Timestamp | End time |

**When to Use Task vs Batch:**

- **Use Task**: For simple one-off executions on a single repository
- **Use Batch + BatchRun**: For executing the same task across multiple repositories

**Example Comparison:**

```yaml
# Using Task (simple, single execution)
apiVersion: kubetask.io/v1alpha1
kind: Task
metadata:
  name: update-service-a
spec:
  contexts:
    - type: File
      file: {name: task.md, source: {inline: "Update deps"}}
    - type: Repository
      repository: {org: myorg, repo: service-a, branch: main}

# Using Batch (template for multiple executions)
apiVersion: kubetask.io/v1alpha1
kind: Batch
metadata:
  name: update-dependencies
spec:
  commonContext:
    - type: File
      file: {name: task.md, source: {inline: "Update deps"}}
  variableContexts:
    - [{type: Repository, repository: {org: myorg, repo: service-a, branch: main}}]
    - [{type: Repository, repository: {org: myorg, repo: service-b, branch: main}}]
    - [{type: Repository, repository: {org: myorg, repo: service-c, branch: main}}]
```

### WorkspaceConfig (Execution Configuration)

WorkspaceConfig defines Agent template reference. All execution details (how to execute tasks) are encapsulated in the Agent template.

```yaml
apiVersion: kubetask.io/v1alpha1
kind: WorkspaceConfig
metadata:
  name: default-workspace
  namespace: kubetask-system
spec:
  # Agent template reference
  # References Job template in ConfigMap
  # If not specified, controller uses convention name "kubetask-agent"
  # If convention ConfigMap doesn't exist, controller uses built-in default template
  agentTemplateRef:
    name: kubetask-agent  # ConfigMap name
    key: agent-template.yaml  # Key in ConfigMap, default is "agent-template.yaml"
```

**Field Description:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `spec.agentTemplateRef` | Object | No | Agent template reference |
| `spec.agentTemplateRef.name` | String | Yes | ConfigMap name |
| `spec.agentTemplateRef.key` | String | No | Key in ConfigMap (default: "agent-template.yaml") |

**Design Points:**
- WorkspaceConfig only cares about Agent template reference
- All execution details encapsulated in Agent template
- Users can freely define all aspects of Job (image, env vars, resource limits, etc.)
- Name independent of project name, won't change if project renames

---

## Agent Template Discovery

### Design Philosophy

**Core Idea**: KubeTask only needs to know "how to create Jobs", other workspace infrastructure (NetworkPolicy, Secrets, PVC, etc.) managed by users.

**User Philosophy**: Work is not produced by people (AI agents) alone, but by **workspace environments**. Workspace = AI intelligence + permissions + tools.

### Discovery Priority

Controller searches for Agent template in this priority order:

1. **BatchRun.spec.agentTemplateRef** (highest priority) - explicitly specified
2. **Convention ConfigMap** (default) - `kubetask-agent`
3. **Built-in default template** (fallback) - Controller built-in template

### Template Format

Agent template uses Go template syntax (same as Helm), supports these variables:

| Variable | Description | Example |
|----------|-------------|---------|
| `{{.TaskID}}` | Task unique ID | `update-deps-001-task-0` |
| `{{.BatchName}}` | Batch name | `update-dependencies` |
| `{{.BatchRunName}}` | BatchRun name | `update-dependencies-run-001` |
| `{{.Namespace}}` | Namespace | `kubetask-system` |
| `{{.Contexts}}` | Task context JSON | `[{"type":"Repository",...}]` |

### ConfigMap Template Examples

**Basic Template**:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: kubetask-agent
  namespace: kubetask-system
data:
  agent-template.yaml: |
    apiVersion: batch/v1
    kind: Job
    metadata:
      name: "{{.TaskID}}"
      namespace: "{{.Namespace}}"
      labels:
        kubetask.io/batch: "{{.BatchName}}"
        kubetask.io/batchrun: "{{.BatchRunName}}"
    spec:
      template:
        spec:
          serviceAccountName: kubetask-agent
          containers:
          - name: agent
            image: ghcr.io/myorg/claude-agent:v1.2.3
            env:
            - name: TASK_CONTEXTS
              value: '{{.Contexts}}'
            - name: ANTHROPIC_API_KEY
              valueFrom:
                secretKeyRef:
                  name: claude-credentials
                  key: api-key
            volumeMounts:
            - name: workspace
              mountPath: /workspace
          volumes:
          - name: workspace
            persistentVolumeClaim:
              claimName: kubetask-workspace
          restartPolicy: Never
```

**Advanced Template (Multi-AI Support)**:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: gemini-agent
  namespace: kubetask-system
data:
  agent-template.yaml: |
    apiVersion: batch/v1
    kind: Job
    metadata:
      name: "{{.TaskID}}"
      labels:
        kubetask.io/agent: "gemini"
    spec:
      template:
        spec:
          serviceAccountName: gemini-agent
          containers:
          - name: agent
            image: ghcr.io/myorg/gemini-agent:latest
            env:
            - name: TASK_CONTEXTS
              value: '{{.Contexts}}'
            - name: GOOGLE_API_KEY
              valueFrom:
                secretKeyRef:
                  name: gemini-credentials
                  key: api-key
            volumeMounts:
            - name: workspace
              mountPath: /workspace
          volumes:
          - name: workspace
            persistentVolumeClaim:
              claimName: kubetask-workspace
          restartPolicy: Never
```

### Usage Scenarios

#### Scenario 1: Standard Usage (90% of cases)

**User Operations**:
```bash
# 1. Create workspace resources (user's own way)
kubectl create namespace team-platform
kubectl apply -f workspace-setup.yaml  # PVC, Secrets, ServiceAccount, NetworkPolicy

# 2. Create standard Agent template
kubectl create configmap kubetask-agent \
  --from-file=agent-template.yaml \
  -n team-platform

# 3. Create Batch
kubectl apply -f batch-update-deps.yaml

# 4. Create BatchRun (automatically uses convention ConfigMap)
kubectl apply -f - <<EOF
apiVersion: kubetask.io/v1alpha1
kind: BatchRun
metadata:
  name: update-deps-001
  namespace: team-platform
spec:
  batchRef: update-dependencies
  # No need to specify agentTemplateRef, automatically uses "kubetask-agent"
EOF
```

#### Scenario 2: Testing New Template

**User Operations**:
```bash
# 1. Create new version template (doesn't affect existing)
kubectl create configmap kubetask-agent-v2 \
  --from-file=agent-template-v2.yaml \
  -n team-platform

# 2. Specific BatchRun uses new template
kubectl apply -f - <<EOF
apiVersion: kubetask.io/v1alpha1
kind: BatchRun
metadata:
  name: test-new-template
  namespace: team-platform
spec:
  batchRef: update-dependencies
  agentTemplateRef:
    name: kubetask-agent-v2  # Explicitly specify new template
EOF

# 3. Other BatchRun continues using old template
kubectl apply -f batchrun-production.yaml  # Uses default template
```

#### Scenario 3: Multi-AI Environment

**User Operations**:
```bash
# 1. Create multiple AI agent templates
kubectl create cm claude-agent --from-file=claude-template.yaml -n team-platform
kubectl create cm gemini-agent --from-file=gemini-template.yaml -n team-platform
kubectl create cm codex-agent --from-file=codex-template.yaml -n team-platform

# 2. Set default to Claude (create convention ConfigMap)
kubectl create cm kubetask-agent \
  --from-file=agent-template.yaml=claude-template.yaml \
  -n team-platform

# 3. Most tasks use default Claude
kubectl apply -f batchrun-normal.yaml

# 4. Specific task uses Gemini
kubectl apply -f - <<EOF
apiVersion: kubetask.io/v1alpha1
kind: BatchRun
metadata:
  name: try-gemini
  namespace: team-platform
spec:
  batchRef: update-dependencies
  agentTemplateRef:
    name: gemini-agent  # Use Gemini
EOF
```

### Advantages

✅ **Minimal**: 90% users only need to create `kubetask-agent` ConfigMap
✅ **Flexible**: Supports explicit override, can test new templates or use different AI
✅ **Good Performance**: Direct Get() call, no List() operation needed
✅ **Clear**: Error messages explicit: "ConfigMap 'kubetask-agent' not found"
✅ **Gradual Migration**: Can seamlessly switch template versions
✅ **User Control**: Users fully control workspace infrastructure, KubeTask only cares about Job creation
✅ **Stability**: WorkspaceConfig name independent of project name

---

## Complete Examples

### 1. Define Workspace (Environment)

```yaml
apiVersion: kubetask.io/v1alpha1
kind: WorkspaceConfig
metadata:
  name: default-workspace
  namespace: kubetask-system
spec:
  agentTemplateRef:
    name: kubetask-agent
    key: agent-template.yaml
```

### 2. Create Agent Template

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: kubetask-agent
  namespace: kubetask-system
data:
  agent-template.yaml: |
    apiVersion: batch/v1
    kind: Job
    metadata:
      name: "{{.TaskID}}"
      labels:
        kubetask.io/batch: "{{.BatchName}}"
        kubetask.io/batchrun: "{{.BatchRunName}}"
    spec:
      template:
        spec:
          serviceAccountName: kubetask-agent
          containers:
          - name: agent
            image: ghcr.io/myorg/claude-agent:latest
            env:
            - name: TASK_CONTEXTS
              value: '{{.Contexts}}'
            - name: ANTHROPIC_API_KEY
              valueFrom:
                secretKeyRef:
                  name: claude-credentials
                  key: api-key
            volumeMounts:
            - name: workspace
              mountPath: /workspace
          volumes:
          - name: workspace
            persistentVolumeClaim:
              claimName: kubetask-workspace
          restartPolicy: Never
```

### 3. Define Batch (What to do)

```yaml
apiVersion: kubetask.io/v1alpha1
kind: Batch
metadata:
  name: update-dependencies
  namespace: kubetask-system
spec:
  # Constant - shared by all tasks
  commonContext:
    - type: File
      file:
        name: task.md
        source:
          inline: |
            Update dependencies to latest versions.
            Run tests and create PR.

    - type: File
      file:
        name: guide.md
        source:
          configMapKeyRef:
            name: workflow-guides
            key: standard-pr-workflow.md

  # Variable - different per task
  variableContexts:
    # Task 1: service-a + config
    - - type: Repository
        repository:
          org: myorg
          repo: service-a
          branch: main

      - type: File
        file:
          name: config.json
          source:
            configMapKeyRef:
              name: dep-configs
              key: service-a.json

    # Task 2: service-b + config + credentials
    - - type: Repository
        repository:
          org: myorg
          repo: service-b
          branch: main

      - type: File
        file:
          name: config.json
          source:
            configMapKeyRef:
              name: dep-configs
              key: service-b.json

      - type: File
        file:
          name: credentials.json
          source:
            secretKeyRef:
              name: private-credentials
              key: service-b-creds.json
```

### 4. Execute Batch

```yaml
apiVersion: kubetask.io/v1alpha1
kind: BatchRun
metadata:
  name: update-dependencies-run-001
  namespace: kubetask-system
  labels:
    team: platform
    jira: PROJ-1234
spec:
  # Reference Batch
  batchRef: update-dependencies

  # Optional: override workspace agent
  # agentTemplateRef:
  #   name: gemini-agent
  #   key: agent-template.yaml
```

---

## kubectl Usage

### Single Task Operations

```bash
# Create and run a single task
kubectl apply -f - <<EOF
apiVersion: kubetask.io/v1alpha1
kind: Task
metadata:
  name: update-service-a
  namespace: kubetask-system
spec:
  contexts:
    - type: File
      file:
        name: task.md
        source:
          inline: "Update dependencies and create PR"
    - type: Repository
      repository:
        org: myorg
        repo: service-a
        branch: main
EOF

# List tasks
kubectl get tasks -n kubetask-system

# Watch task execution
kubectl get task update-service-a -n kubetask-system -w

# Check task status
kubectl get task update-service-a -o yaml

# View task logs
kubectl logs job/$(kubectl get task update-service-a -n kubetask-system -o jsonpath='{.status.jobName}') -n kubetask-system

# Delete completed task
kubectl delete task update-service-a -n kubetask-system
```

### Batch Operations

```bash
# List batches
kubectl get batches -n kubetask-system

# Create a batch
kubectl apply -f my-batch.yaml

# Run a batch
kubectl apply -f - <<EOF
apiVersion: kubetask.io/v1alpha1
kind: BatchRun
metadata:
  generateName: update-deps-
  namespace: kubetask-system
spec:
  batchRef: update-dependencies
EOF

# Watch batch execution
kubectl get batchrun -n kubetask-system -w

# Check batch run status
kubectl get batchrun update-deps-abc123 -o yaml

# View logs for a specific task in batch
kubectl logs job/update-deps-abc123-task-0 -n kubetask-system

# List all batch runs
kubectl get batchruns -n kubetask-system
```

### Workspace Configuration

```bash
# List workspace configs
kubectl get workspaceconfigs -n kubetask-system

# Create workspace config
kubectl apply -f workspace-config.yaml

# View workspace config details
kubectl get workspaceconfig default-workspace -o yaml
```

---

## Naming Stability Analysis

### Stable Names (Won't Change)

| Name | Reason |
|------|--------|
| **WorkspaceConfig** | Independent of project name; semantic concept |
| **AgentTemplateRef** | Semantic concept; describes what it is |
| **commonContext** | Programming concept; universal |
| **variableContexts** | Programming concept; universal |
| **Context** | Universal abstraction |

### Project-Dependent Names (May Change)

| Name | Current | If Project Renames |
|------|---------|-------------------|
| Namespace | `kubetask-system` | Would change |
| ConfigMap | `kubetask-agent` | Would change |
| Labels | `kubetask.io/*` | Would change |

**Mitigation**: Use convention-based discovery. WorkspaceConfig doesn't hardcode names.

---

## Migration Guide

### Migration from CodeSweep

#### Resource Mapping

| Old (CodeSweep) | New (KubeTask) |
|----------------|----------------|
| `Bundle` | `Batch` |
| `BundleRun` | `BatchRun` |
| `CodeSweepConfig` | `WorkspaceConfig` |
| `bundles.codesweep.io` | `batches.kubetask.io` |
| `bundleruns.codesweep.io` | `batchruns.kubetask.io` |
| `codesweepconfigs.codesweep.io` | `workspaceconfigs.kubetask.io` |

#### Field Mapping

| Old | New |
|-----|-----|
| `spec.repositories` | `spec.variableContexts` (with type: Repository) |
| `spec.context.files` | `spec.commonContext` (with type: File) |
| `spec.bundleRef` | `spec.batchRef` |
| `spec.jobTemplateRef` | `spec.agentTemplateRef` |

---

## Convention-Based Discovery

**Default agent template discovery order:**

1. **BatchRun.spec.agentTemplateRef** (explicit)
2. **WorkspaceConfig.spec.agentTemplateRef** (configured)
3. **Convention: `kubetask-agent`** (default ConfigMap name)
4. **Built-in template** (fallback)

This allows:
- ✅ Explicit override per BatchRun
- ✅ Default config per workspace
- ✅ Convention for simple cases
- ✅ Fallback for new users

---

## Benefits of Final Design

### 1. Semantic Clarity

- **Batch**: Clearly batch processing
- **WorkspaceConfig**: Clearly environment config
- **AgentTemplateRef**: Clearly AI agent template
- **commonContext/variableContexts**: Clearly constant/variable

### 2. Stability

- **WorkspaceConfig**: Won't change even if project renames
- **AgentTemplateRef**: Semantic, not project-specific
- **Core concepts**: Independent of project name

### 3. Flexibility

- Multiple context types (File, Repository, future: API, Database)
- Each task can have different contexts
- Easy to extend

### 4. K8s Alignment

- **Batch**: Aligns with `batch/v1`
- **WorkspaceConfig**: Follows K8s Config pattern
- Convention-based discovery: K8s standard practice

---

## Summary

**Final API**:
- ✅ **Batch** + **BatchRun** - semantic batch processing
- ✅ **WorkspaceConfig** - stable, project-independent
- ✅ **AgentTemplateRef** - semantic agent configuration
- ✅ **commonContext** + **variableContexts** - clear constant/variable model
- ✅ **Context** abstraction - flexible, extensible

**Philosophy**:
- **Stability**: Core concepts independent of project name
- **Semantics**: Names reflect what things are, not project branding
- **Flexibility**: Extensible context system
- **Alignment**: Follows K8s patterns and conventions

**Advantages**:
- ✅ **Simplified Architecture**: No external dependencies
- ✅ **Native Integration**: Works with K8s ecosystem
- ✅ **Declarative Management**: GitOps ready
- ✅ **Infrastructure Reuse**: Leverage K8s capabilities
- ✅ **Simplified Operations**: Standard K8s tools

---

## Related Documentation

- **[ADR 0003: Use Kubernetes-Native Architecture](adr/0003-kubernetes-native-architecture.md)** - Architecture decision record
- **[ADR 0005: Task Abstraction and Context System](adr/0005-task-abstraction-and-context-system.md)** - Task abstraction and context system
- **[FINAL-NAMING-DECISIONS.md](FINAL-NAMING-DECISIONS.md)** - Naming decision records

---

**Status**: ✅ FINAL
**Date**: 2025-12-01
**Version**: v2.0
**Approved**: @zxue
**Maintainer**: KubeTask Team
