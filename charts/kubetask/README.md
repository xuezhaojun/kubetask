# KubeTask Helm Chart

This Helm chart deploys KubeTask, a Kubernetes-native system for executing AI agent tasks across multiple repositories.

## Prerequisites

- Kubernetes 1.25+
- Helm 3.8+
- PersistentVolume provisioner support in the underlying infrastructure (for workspace storage)
- GitHub Personal Access Token with repo permissions
- Anthropic Claude API key or Vertex AI credentials

## Installing the Chart

### Quick Start

```bash
# Create namespace
kubectl create namespace kubetask-system

# Install with minimal configuration
helm install kubetask ./charts/kubetask \
  --namespace kubetask-system \
  --set github.token=<YOUR_GITHUB_TOKEN> \
  --set claude.apiKey=<YOUR_CLAUDE_API_KEY>
```

### Production Installation

```bash
# Create a values file with your configuration
cat > my-values.yaml <<EOF
controller:
  image:
    repository: ghcr.io/stolostron/kubetask-controller
    tag: v0.1.0

  resources:
    limits:
      cpu: 1000m
      memory: 1Gi
    requests:
      cpu: 200m
      memory: 256Mi

workspace:
  enabled: true
  storageClass: nfs-client  # Use your storage class
  size: 200Gi

github:
  token: <YOUR_GITHUB_TOKEN>

claude:
  apiKey: <YOUR_CLAUDE_API_KEY>

# Or use Vertex AI
# claude:
#   vertexAI:
#     enabled: true
#     projectID: your-gcp-project
#     location: us-central1
#     serviceAccountKey: <BASE64_ENCODED_KEY>

cleanup:
  enabled: true
  schedule: "0 2 * * *"  # Daily at 2 AM
  ttlDays: 7
EOF

# Install the chart
helm install kubetask ./charts/kubetask \
  --namespace kubetask-system \
  --values my-values.yaml
```

## Configuration

The following table lists the configurable parameters of the KubeTask chart and their default values.

### Controller Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `controller.image.repository` | Controller image repository | `ghcr.io/stolostron/kubetask-controller` |
| `controller.image.tag` | Controller image tag | `""` (uses chart appVersion) |
| `controller.image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `controller.replicas` | Number of controller replicas | `1` |
| `controller.resources.limits.cpu` | CPU limit | `500m` |
| `controller.resources.limits.memory` | Memory limit | `512Mi` |
| `controller.resources.requests.cpu` | CPU request | `100m` |
| `controller.resources.requests.memory` | Memory request | `128Mi` |

### Agent Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `agent.image.repository` | Agent image repository | `ghcr.io/stolostron/kubetask-agent` |
| `agent.image.tag` | Agent image tag | `latest` |

### Workspace Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `workspace.enabled` | Enable workspace PVC creation | `true` |
| `workspace.storageClass` | Storage class for PVC | `""` (default storage class) |
| `workspace.size` | Size of workspace volume | `100Gi` |
| `workspace.accessMode` | Access mode (must be ReadWriteMany) | `ReadWriteMany` |

### Authentication Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `github.token` | GitHub personal access token | `""` |
| `github.existingSecret` | Existing secret name for GitHub token | `""` |
| `claude.apiKey` | Claude API key | `""` |
| `claude.existingSecret` | Existing secret name for Claude API key | `""` |
| `claude.vertexAI.enabled` | Enable Vertex AI integration | `false` |
| `claude.vertexAI.projectID` | GCP project ID | `""` |
| `claude.vertexAI.location` | GCP location | `""` |

### Cleanup Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `cleanup.enabled` | Enable automatic cleanup CronJob | `true` |
| `cleanup.schedule` | Cron schedule for cleanup | `"0 2 * * *"` |
| `cleanup.ttlDays` | TTL for completed BatchRuns (days) | `3` |
| `cleanup.failedTTLDays` | TTL for failed BatchRuns (days) | `7` |

## Usage Examples

### Creating a Batch

```yaml
apiVersion: kubetask.io/v1alpha1
kind: Batch
metadata:
  name: upgrade-deps
  namespace: kubetask-system
spec:
  commonContext:
    - type: File
      file:
        name: task.md
        source:
          inline: |
            Update go.mod to Go 1.21 and run go mod tidy.
            Ensure all tests pass after the upgrade.
  variableContexts:
    - - type: Repository
        repository:
          org: myorg
          repo: repo1
          branch: main
    - - type: Repository
        repository:
          org: myorg
          repo: repo2
          branch: main
```

### Creating a BatchRun

```yaml
apiVersion: kubetask.io/v1alpha1
kind: BatchRun
metadata:
  name: upgrade-deps-001
  namespace: kubetask-system
spec:
  batchRef: upgrade-deps
```

### Monitoring Progress

```bash
# Watch BatchRun status
kubectl get batchrun -n kubetask-system -w

# View detailed status
kubectl describe batchrun upgrade-deps-001 -n kubetask-system

# Check Jobs
kubectl get jobs -n kubetask-system -l kubetask.io/batchrun=upgrade-deps-001

# View task logs
kubectl logs job/upgrade-deps-001-task-0 -n kubetask-system
```

## Uninstalling the Chart

```bash
helm uninstall kubetask --namespace kubetask-system
```

To also delete the namespace:

```bash
kubectl delete namespace kubetask-system
```

## Storage Requirements

KubeTask requires a PersistentVolume with `ReadWriteMany` access mode for the workspace. This allows multiple agent Jobs to run concurrently.

Recommended storage solutions:
- NFS
- CephFS
- Azure Files
- Google Cloud Filestore
- AWS EFS

## Security Considerations

1. **Secrets Management**: Never commit secrets to Git. Use:
   - Kubernetes Secrets
   - External Secrets Operator
   - Sealed Secrets
   - HashiCorp Vault

2. **RBAC**: The chart creates minimal RBAC permissions:
   - Controller: Manages CRs and Jobs
   - Agent: Updates BatchRun status only

3. **Network Policies**: Consider adding NetworkPolicies to restrict traffic

4. **Pod Security**: Runs with non-root user and dropped capabilities

## Troubleshooting

### Controller not starting

```bash
# Check controller logs
kubectl logs -n kubetask-system deployment/kubetask-controller

# Check RBAC permissions
kubectl auth can-i create batchruns --as=system:serviceaccount:kubetask-system:kubetask-controller -n kubetask-system
```

### Jobs failing

```bash
# List failed Jobs
kubectl get jobs -n kubetask-system --field-selector status.successful=0

# Check Job logs
kubectl logs job/<job-name> -n kubetask-system

# Check workspace PVC
kubectl get pvc -n kubetask-system
```

### Storage issues

```bash
# Verify PVC is bound
kubectl get pvc -n kubetask-system

# Check storage class
kubectl get storageclass

# Ensure ReadWriteMany support
kubectl describe pvc kubetask-workspace -n kubetask-system
```

## Contributing

See the main project [README](../../../README.md) for contribution guidelines.

## License

Copyright Contributors to the KubeTask project. Licensed under the Apache License 2.0.
