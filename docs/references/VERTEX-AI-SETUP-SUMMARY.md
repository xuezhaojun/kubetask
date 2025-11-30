# Vertex AI Setup Summary for CodeSweep

Complete setup guide for running CodeSweep with Google Vertex AI authentication in Kubernetes.

## 📂 What's Been Created

### 1. Secret Management Scripts (`scripts/`)

#### [scripts/create-vertex-secret.sh](scripts/create-vertex-secret.sh)
Creates Kubernetes Secrets for ADC-based authentication:
- Reads your local `~/.config/gcloud/application_default_credentials.json`
- Creates two secrets:
  - `claude-vertex-env`: Environment variables (CLAUDE_CODE_USE_VERTEX, etc.)
  - `claude-vertex-credentials`: ADC credentials file

**Usage**:
```bash
./scripts/create-vertex-secret.sh
NAMESPACE=codesweep-system ./scripts/create-vertex-secret.sh
```

#### [scripts/setup-workload-identity.sh](scripts/setup-workload-identity.sh)
Sets up Workload Identity for production (recommended):
- Creates GCP Service Account
- Grants Vertex AI permissions
- Binds GCP SA to Kubernetes ServiceAccount
- Creates environment variables secret

**Usage**:
```bash
./scripts/setup-workload-identity.sh
K8S_NAMESPACE=codesweep-system ./scripts/setup-workload-identity.sh
```

#### [scripts/README-VERTEX-SECRET.md](scripts/README-VERTEX-SECRET.md)
Complete documentation covering:
- ADC vs Workload Identity comparison
- Detailed setup instructions
- Verification steps
- Troubleshooting guide
- Security best practices

### 2. Updated Docker Image (`k8s/hello-world-experiment/`)

#### [k8s/hello-world-experiment/Dockerfile](k8s/hello-world-experiment/Dockerfile)
**Key improvements**:
- ✅ Node.js 20 LTS (from NodeSource, not old apt version)
- ✅ Google Cloud SDK (for debugging authentication)
- ✅ Proper directory structure (`/home/claude/.config/gcloud`)
- ✅ Enhanced health checks

#### [k8s/hello-world-experiment/app/entrypoint.sh](k8s/hello-world-experiment/app/entrypoint.sh)
**Enhanced features**:
- ✅ Validates all required environment variables
- ✅ Checks ADC credentials file (if using ADC method)
- ✅ Verifies Workload Identity metadata server (if using WI method)
- ✅ Tests gcloud authentication
- ✅ Detailed diagnostic output on failure

#### Documentation Files
- [DOCKERFILE-CHANGES.md](k8s/hello-world-experiment/DOCKERFILE-CHANGES.md): Detailed explanation of all changes
- [QUICK-REFERENCE.md](k8s/hello-world-experiment/QUICK-REFERENCE.md): Quick reference card

## 🚀 Quick Start Guide

### Step 1: Choose Authentication Method

| Method | Use Case | Complexity | Security |
|--------|----------|------------|----------|
| **ADC** | Dev/Testing | Simple | Good |
| **Workload Identity** | Production | Moderate | Excellent |

### Step 2: Set Up Secrets

**For ADC**:
```bash
# Ensure you've run gcloud ADC setup (per company docs)
gcloud auth application-default login
gcloud auth application-default set-quota-project cloudability-it-gemini

# Create secrets
./scripts/create-vertex-secret.sh
```

**For Workload Identity**:
```bash
# Run interactive setup
./scripts/setup-workload-identity.sh
```

### Step 3: Build and Deploy

```bash
# Build new Docker image
cd k8s/hello-world-experiment
docker build -t codesweep-agent:latest .

# Test locally (optional)
docker run --rm codesweep-agent:latest bash -c 'node --version && gcloud --version'

# Push to registry
docker tag codesweep-agent:latest your-registry/codesweep-agent:latest
docker push your-registry/codesweep-agent:latest
```

### Step 4: Deploy to Kubernetes

**ADC Method**:
```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: codesweep-task
spec:
  template:
    spec:
      containers:
      - name: agent
        image: your-registry/codesweep-agent:latest
        envFrom:
        - secretRef:
            name: claude-vertex-env
        env:
        - name: GOOGLE_APPLICATION_CREDENTIALS
          value: /var/secrets/google/application_default_credentials.json
        volumeMounts:
        - name: google-cloud-credentials
          mountPath: /var/secrets/google
          readOnly: true
      volumes:
      - name: google-cloud-credentials
        secret:
          secretName: claude-vertex-credentials
      restartPolicy: Never
```

**Workload Identity Method** (simpler!):
```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: codesweep-task
spec:
  template:
    spec:
      serviceAccountName: codesweep-agent  # That's it!
      containers:
      - name: agent
        image: your-registry/codesweep-agent:latest
        envFrom:
        - secretRef:
            name: claude-vertex-env
      restartPolicy: Never
```

## 📊 Architecture Overview

### ADC Authentication Flow
```
Developer Machine                    Kubernetes Cluster
─────────────────                   ──────────────────
~/.config/gcloud/
application_default_credentials.json
         │
         │ Read by script
         ▼
./scripts/create-vertex-secret.sh
         │
         │ Creates K8s Secret
         ▼
kubernetes.io/secret
  - claude-vertex-env
  - claude-vertex-credentials
         │
         │ Mounted to Pod
         ▼
Container Environment
  GOOGLE_APPLICATION_CREDENTIALS=/var/secrets/google/...
         │
         │ Used by Claude Code
         ▼
Google Vertex AI API
```

### Workload Identity Flow
```
GCP IAM                              Kubernetes Cluster
───────                             ──────────────────
GCP Service Account
codesweep-agent@...
  - roles/aiplatform.user
         │
         │ Bound via IAM
         ▼
Kubernetes ServiceAccount
  annotations:
    iam.gke.io/gcp-service-account: ...
         │
         │ Referenced in Pod
         ▼
Container Environment
  (Automatic token injection by GKE)
         │
         │ Used by Claude Code
         ▼
Google Vertex AI API
```

## 🔧 Configuration Details

### Environment Variables (Required)

| Variable | Value | Source |
|----------|-------|--------|
| `CLAUDE_CODE_USE_VERTEX` | `1` | Your ~/.zshrc |
| `CLOUD_ML_REGION` | `us-east5` | Your ~/.zshrc |
| `ANTHROPIC_VERTEX_PROJECT_ID` | `itpc-gcp-hcm-pe-eng-claude` | Your ~/.zshrc |

### Secrets Created

**ADC Method**:
- `claude-vertex-env` (type: generic)
  - CLAUDE_CODE_USE_VERTEX
  - CLOUD_ML_REGION
  - ANTHROPIC_VERTEX_PROJECT_ID
- `claude-vertex-credentials` (type: generic)
  - application_default_credentials.json (file)

**Workload Identity Method**:
- `claude-vertex-env` (type: generic)
  - Same as above
- Kubernetes ServiceAccount with annotation

## 🧪 Verification Steps

### 1. Verify Secrets
```bash
# Check secrets exist
kubectl get secret claude-vertex-env
kubectl get secret claude-vertex-credentials  # ADC only

# Check ServiceAccount (WI only)
kubectl get sa codesweep-agent -o yaml
```

### 2. Test Docker Image
```bash
docker run --rm codesweep-agent:latest bash -c '
  echo "Node.js: $(node --version)"
  echo "gcloud: $(gcloud --version | head -n 1)"
  echo "Claude: $(claude --version 2>&1 | head -n 1)"
'
```

Expected output:
```
Node.js: v20.x.x
gcloud: Google Cloud SDK xxx.x.x
Claude: Claude Code ...
```

### 3. Test in Kubernetes
```bash
# Deploy test job
kubectl apply -f your-job.yaml

# Watch logs
kubectl logs -f job/codesweep-task
```

Look for:
```
✓ ANTHROPIC_VERTEX_PROJECT_ID: itpc-gcp-hcm-pe-eng-claude
✓ Credentials file found: /var/secrets/google/...  (ADC)
✓ Workload Identity metadata server accessible  (WI)
✓ Claude CLI version: ...
```

## 🔍 Troubleshooting

### Issue: Node.js version too old
**Symptom**: Claude Code fails to start or shows Node.js compatibility errors

**Solution**: Rebuild Docker image (Dockerfile is now updated to use Node.js 20)

### Issue: Authentication fails
**Symptom**: "Permission denied" or "credentials not found"

**For ADC**:
```bash
# Verify secret contains credentials
kubectl get secret claude-vertex-credentials -o jsonpath='{.data.application_default_credentials\.json}' | base64 -d | jq .

# Verify environment variable in pod
kubectl exec -it your-pod -- env | grep GOOGLE_APPLICATION_CREDENTIALS
```

**For Workload Identity**:
```bash
# Check SA annotation
kubectl get sa codesweep-agent -o jsonpath='{.metadata.annotations.iam\.gke\.io/gcp-service-account}'

# Check IAM binding
gcloud iam service-accounts get-iam-policy \
  codesweep-agent@${PROJECT_ID}.iam.gserviceaccount.com
```

### Issue: Secret not mounted
**Symptom**: Pod can't find credentials file

**Solution**: Check volume mount matches GOOGLE_APPLICATION_CREDENTIALS:
```yaml
env:
- name: GOOGLE_APPLICATION_CREDENTIALS
  value: /var/secrets/google/application_default_credentials.json  # Path must match
volumeMounts:
- name: google-cloud-credentials
  mountPath: /var/secrets/google  # Parent directory
```

## 📚 Documentation Index

| Document | Purpose |
|----------|---------|
| This file | Overall summary and quick start |
| [scripts/README-VERTEX-SECRET.md](scripts/README-VERTEX-SECRET.md) | Detailed authentication setup |
| [k8s/hello-world-experiment/QUICK-REFERENCE.md](k8s/hello-world-experiment/QUICK-REFERENCE.md) | Quick reference for Dockerfile changes |
| [k8s/hello-world-experiment/DOCKERFILE-CHANGES.md](k8s/hello-world-experiment/DOCKERFILE-CHANGES.md) | Detailed Dockerfile changes |

## 🎯 Next Steps

1. **Choose your authentication method** (ADC for dev, WI for prod)
2. **Run the appropriate setup script** (create-vertex-secret.sh or setup-workload-identity.sh)
3. **Build the Docker image** (Dockerfile is already updated)
4. **Deploy a test job** to verify everything works
5. **Integrate with CodeSweep** CRDs when ready

## 💡 Best Practices

1. **Use Workload Identity in production** - No credential files to manage
2. **Test locally first** - Build and test Docker image before deploying
3. **Keep secrets in sync** - Re-run scripts when credentials change
4. **Monitor logs** - The entrypoint provides detailed diagnostic output
5. **Follow principle of least privilege** - Grant only necessary IAM roles

## 🔗 Related Resources

- Company Installation Guide: Internal Google Doc
- [Google Cloud ADC Docs](https://cloud.google.com/docs/authentication/application-default-credentials)
- [Workload Identity Guide](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity)
- [Vertex AI Authentication](https://cloud.google.com/vertex-ai/docs/authentication)
