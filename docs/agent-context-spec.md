# Agent Context Specification

This document defines how context items from Task are mounted and provided to AI agents in Kubernetes Pods.

## Overview

Context items provide information to AI agents during task execution. KubeTask supports two modes for delivering context to agents:

1. **Single File Mode**: Mount a single file at a specific path using `filePath`
2. **Directory Mode**: Mount a ConfigMap as a directory using `dirPath`

## Context Priority

Contexts are processed in the following priority order (lowest to highest):

1. `Agent.defaultContexts` - Base layer (organization-wide defaults)
2. `Task.contexts` - Task-specific contexts

Higher priority contexts take precedence. When multiple contexts target the same path, they are aggregated.

## File Context Types

### 1. Single File with Inline Content

```yaml
type: File
file:
  filePath: /workspace/task.md
  source:
    inline: |
      Update all dependencies to latest versions.
      Run tests and create a PR.
```

### 2. Single File from ConfigMap Key

```yaml
type: File
file:
  filePath: /workspace/guide.md
  source:
    configMapKeyRef:
      name: workflow-guides
      key: pr-workflow.md
```

### 3. Directory from ConfigMap

All keys in the ConfigMap become files in the specified directory:

```yaml
type: File
file:
  dirPath: /workspace/configs
  source:
    configMapRef:
      name: my-configs  # Each key becomes a file
```

## Content Aggregation

When multiple contexts target the same `filePath`, their contents are aggregated using XML tags:

```xml
<context index="0">
[Content from first context]
</context>

<context index="1">
[Content from second context]
</context>
```

## Workspace Structure Example

```
/
├── workspace/
│   ├── task.md              # filePath: /workspace/task.md
│   ├── guide.md             # filePath: /workspace/guide.md
│   └── configs/             # dirPath: /workspace/configs
│       ├── config.json      # From ConfigMap key "config.json"
│       └── settings.yaml    # From ConfigMap key "settings.yaml"
└── home/
    └── agent/
        └── .claude/
            └── CLAUDE.md    # filePath: /home/agent/.claude/CLAUDE.md
```

## Examples

### Example 1: Simple Task with Inline Content

```yaml
apiVersion: kubetask.io/v1alpha1
kind: Task
metadata:
  name: update-deps
spec:
  contexts:
    - type: File
      file:
        filePath: /workspace/task.md
        source:
          inline: |
            Update all dependencies to latest versions.
            Run tests and create a PR.
```

### Example 2: Task with ConfigMap Content

```yaml
apiVersion: kubetask.io/v1alpha1
kind: Task
metadata:
  name: deploy-service
spec:
  contexts:
    # Task description from inline
    - type: File
      file:
        filePath: /workspace/task.md
        source:
          inline: |
            Deploy the service to production.

    # Workflow guide from ConfigMap
    - type: File
      file:
        filePath: /workspace/guide.md
        source:
          configMapKeyRef:
            name: workflow-guides
            key: deploy-guide.md

    # Multiple config files as directory
    - type: File
      file:
        dirPath: /workspace/configs
        source:
          configMapRef:
            name: deploy-configs
```

### Example 3: Agent with Default Contexts

Agent provides organization-wide defaults that are merged with Task contexts:

```yaml
apiVersion: kubetask.io/v1alpha1
kind: Agent
metadata:
  name: default
spec:
  agentImage: quay.io/myorg/claude-agent:v1.0
  serviceAccountName: kubetask-agent
  defaultContexts:
    # Organization coding standards
    - type: File
      file:
        filePath: /workspace/org-standards.md
        source:
          configMapKeyRef:
            name: org-standards
            key: coding.md

    # Claude configuration at specific path
    - type: File
      file:
        filePath: /home/agent/.claude/CLAUDE.md
        source:
          configMapKeyRef:
            name: org-standards
            key: claude-config.md
---
apiVersion: kubetask.io/v1alpha1
kind: Task
metadata:
  name: update-service
spec:
  contexts:
    - type: File
      file:
        filePath: /workspace/task.md
        source:
          inline: |
            Update dependencies and create PR.
```

Result for the task:
- `/workspace/task.md`: Task description
- `/workspace/org-standards.md`: Organization standards (from Agent)
- `/home/agent/.claude/CLAUDE.md`: Claude configuration (from Agent)

## Credentials

Agent supports credentials for providing secrets to agents. Credentials can be exposed as:
- **Environment Variables**: For API tokens, passwords, etc.
- **File Mounts**: For SSH keys, service account files, etc.

### Credential Configuration

```yaml
apiVersion: kubetask.io/v1alpha1
kind: Agent
metadata:
  name: default
spec:
  agentImage: quay.io/myorg/claude-agent:v1.0
  serviceAccountName: kubetask-agent
  credentials:
    # GitHub token as environment variable
    - name: github-token
      secretRef:
        name: github-credentials
        key: token
      env: GITHUB_TOKEN

    # SSH key as file mount
    - name: ssh-key
      secretRef:
        name: ssh-credentials
        key: id_rsa
      mountPath: /home/agent/.ssh/id_rsa
      fileMode: 0400  # Read-only for SSH keys

    # Anthropic API key as environment variable
    - name: anthropic-api-key
      secretRef:
        name: ai-credentials
        key: anthropic-key
      env: ANTHROPIC_API_KEY

    # GCP service account as file mount
    - name: gcp-credentials
      secretRef:
        name: gcp-sa
        key: credentials.json
      mountPath: /home/agent/.config/gcloud/application_default_credentials.json
      fileMode: 0600
```

### Credential Fields

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Descriptive name for documentation |
| `secretRef.name` | Yes | Name of the Kubernetes Secret |
| `secretRef.key` | Yes | Key within the Secret to use |
| `env` | No | Environment variable name to expose the secret |
| `mountPath` | No | File path to mount the secret |
| `fileMode` | No | File permission mode (default: 0600) |

A credential can have both `env` and `mountPath` specified to expose the same secret value in both ways.

### Security Best Practices

1. **Use restrictive file modes**: Default is `0600` (read/write owner only). Use `0400` for read-only files like SSH keys.
2. **Avoid logging secrets**: Agents should never log secret values.
3. **Use short-lived tokens**: Prefer short-lived tokens over long-lived credentials.
4. **Principle of least privilege**: Only include credentials that are actually needed.

## Agent Implementation Guide

Agents should:

1. **Check for mounted context files** at the paths specified in the task
2. **Handle the aggregation format** (XML tags) when multiple contexts target the same path
3. **Use credentials securely**: Never log or expose credential values

### Environment Variables

The controller provides these environment variables to the agent:

| Variable | Description |
|----------|-------------|
| `TASK_NAME` | Name of the Task CR |
| `TASK_NAMESPACE` | Namespace of the Task CR |
| `GITHUB_TOKEN` | (if configured) GitHub API token |
| `ANTHROPIC_API_KEY` | (if configured) Anthropic API key |
| ... | Other credentials as configured in Agent |

### Recommended Agent Behavior

```
1. Read context files to understand the task
2. Check for any configuration files in specified directories
3. Execute the task as described
4. Report results via Task CR status updates
```

## Summary

| Context Type | Usage |
|--------------|-------|
| `filePath` + `inline` | Single file with inline content |
| `filePath` + `configMapKeyRef` | Single file from ConfigMap key |
| `dirPath` + `configMapRef` | Directory with all ConfigMap keys as files |

| Priority | Context Source | Description |
|----------|---------------|-------------|
| Lowest | `Agent.defaultContexts` | Organization defaults |
| Highest | `Task.contexts` | Task-specific context |
