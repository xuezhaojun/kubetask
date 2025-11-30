# Authentication Decision Guide

## 🎯 你说得对！

将**个人 ADC credentials** 复制到 Kubernetes 确实不是最佳实践。

## ⚠️ ADC 方法的问题

当你运行 `./scripts/create-vertex-secret.sh` 时，它会：
1. 读取你的个人 ADC credentials (`~/.config/gcloud/application_default_credentials.json`)
2. 创建 Kubernetes Secret 包含这些凭据
3. Pod 使用**你的个人身份**访问 Vertex AI

### 为什么这不好？

| 问题 | 影响 |
|------|------|
| **身份混用** | Pod 使用你的个人身份，而不是服务账户 |
| **权限过大** | 你的个人账户可能有很多不必要的权限 |
| **审计困难** | 无法区分是你本人操作还是 Pod 操作 |
| **凭据过期** | ADC token 会过期，需要定期更新 Secret |
| **安全风险** | 如果 Pod 被攻击，你的个人凭据可能泄露 |
| **不符合最佳实践** | Google 推荐使用 Workload Identity |

## ✅ 推荐方案：Workload Identity

### 你提到有创建 Service Account 的权限

太好了！这意味着你可以使用 **Workload Identity**，这是最佳方案：

```bash
./scripts/setup-workload-identity.sh
```

### Workload Identity 的优势

| 优势 | 说明 |
|------|------|
| **专用身份** | 每个 workload 有自己的 Service Account |
| **最小权限** | 只授予 Vertex AI 需要的权限 |
| **清晰审计** | 审计日志清楚显示是哪个服务账户 |
| **永不过期** | 不需要定期更新凭据 |
| **自动轮换** | GKE 自动管理 token 轮换 |
| **无凭据文件** | 不需要管理或挂载凭据文件 |
| **符合最佳实践** | Google Cloud 官方推荐 |

## 🤔 决策树

```
开始
  │
  ├─ 你有创建 Service Account 的权限？
  │   │
  │   ├─ YES ──→ ✅ 使用 Workload Identity (最佳)
  │   │          ./scripts/setup-workload-identity.sh
  │   │
  │   └─ NO ───→ 联系管理员
  │               │
  │               ├─ 选项 1: 请求创建 SA 的权限
  │               │
  │               ├─ 选项 2: 请管理员创建 SA 给你
  │               │
  │               └─ 选项 3: 临时使用 ADC (不推荐)
  │                          ./scripts/create-vertex-secret.sh
  │
  └─ 完成
```

## 📋 方法对比

### Method 1: Personal ADC (不推荐)

**何时使用**: 仅用于本地开发测试，或没有其他选择时

**优点**:
- ✅ 设置简单
- ✅ 不需要额外权限

**缺点**:
- ❌ 使用个人身份
- ❌ 可能有过多权限
- ❌ 难以审计
- ❌ 凭据会过期
- ❌ 安全风险

**步骤**:
```bash
# 1. 设置 ADC
gcloud auth application-default login
gcloud auth application-default set-quota-project cloudability-it-gemini

# 2. 创建 Secret
./scripts/create-vertex-secret.sh

# 3. 使用 ADC Job manifest
kubectl apply -f k8s/hello-world-experiment/manifests/03-hello-world-job-adc.yaml
```

**Job 配置**:
```yaml
envFrom:
- secretRef:
    name: claude-vertex-env
env:
- name: GOOGLE_APPLICATION_CREDENTIALS
  value: /var/secrets/google/application_default_credentials.json
volumeMounts:
- name: google-cloud-credentials
  mountPath: /var/secrets/google
volumes:
- name: google-cloud-credentials
  secret:
    secretName: claude-vertex-credentials
```

---

### Method 2: Workload Identity (✅ 推荐)

**何时使用**: 所有生产环境，以及你有创建 SA 权限的情况

**优点**:
- ✅ 专用服务账户
- ✅ 最小权限原则
- ✅ 清晰的审计日志
- ✅ 凭据永不过期
- ✅ 自动凭据轮换
- ✅ 无需管理凭据文件
- ✅ Google 官方推荐

**缺点**:
- ⚠️ 需要创建 SA 的权限
- ⚠️ 需要 GKE 集群启用 Workload Identity

**步骤**:
```bash
# 1. 运行 Workload Identity 设置
./scripts/setup-workload-identity.sh

# 脚本会：
# - 创建 GCP Service Account
# - 授予 Vertex AI 权限
# - 创建 K8s ServiceAccount
# - 绑定两者
# - 创建环境变量 Secret

# 2. 使用 Workload Identity Job manifest
kubectl apply -f k8s/hello-world-experiment/manifests/03-hello-world-job-wi.yaml
```

**Job 配置** (更简单！):
```yaml
serviceAccountName: codesweep-agent  # 只需要这一行！
envFrom:
- secretRef:
    name: claude-vertex-env
# 不需要挂载 credentials！
# 不需要 GOOGLE_APPLICATION_CREDENTIALS！
```

---

### Method 3: Service Account Key (你没有权限)

**何时使用**: 当 Workload Identity 不可用时

**你的情况**: ❌ 没有创建 Key 的权限，所以不能用

**说明**:
- 需要在 GCP Console 创建 Service Account Key (JSON)
- 你提到没有 "add key" 和 "view key" 的权限
- 所以这个方法对你不适用

---

## 🚀 推荐的完整流程

既然你有创建 Service Account 的权限，请使用这个流程：

### Step 1: 验证权限
```bash
# 快速检查
./scripts/quick-permission-check.sh

# 或详细检查
./scripts/check-gcp-permissions.sh
```

### Step 2: 设置 Workload Identity
```bash
./scripts/setup-workload-identity.sh
```

### Step 3: 验证设置
```bash
# 检查 GCP Service Account
gcloud iam service-accounts list --project=$ANTHROPIC_VERTEX_PROJECT_ID | grep codesweep

# 检查 K8s ServiceAccount
kubectl get sa codesweep-agent -o yaml

# 应该看到 annotation:
#   iam.gke.io/gcp-service-account: codesweep-agent@....iam.gserviceaccount.com

# 检查环境变量 Secret
kubectl get secret claude-vertex-env -o yaml
```

### Step 4: 部署应用
```bash
# 创建其他资源
kubectl apply -f k8s/hello-world-experiment/manifests/00-namespace.yaml
kubectl apply -f k8s/hello-world-experiment/manifests/02-claude-settings-configmap.yaml

# Build 和 push image
cd k8s/hello-world-experiment
docker build -t YOUR_REGISTRY/codesweep/claude-hello-world:latest .
docker push YOUR_REGISTRY/codesweep/claude-hello-world:latest

# 部署 Job (使用 Workload Identity)
kubectl apply -f k8s/hello-world-experiment/manifests/03-hello-world-job-wi.yaml

# 查看日志
kubectl logs -f job/claude-hello-world-wi
```

### Step 5: 验证认证
日志应该显示：
```
==========================================
Authentication Verification
==========================================
Authentication method: Workload Identity
  Checking GCP metadata server...
✓ Workload Identity metadata server accessible
  Service Account: codesweep-agent@itpc-gcp-hcm-pe-eng-claude.iam.gserviceaccount.com
```

---

## 🔍 如果 Workload Identity 失败

### 常见问题

**1. GKE 集群未启用 Workload Identity**

检查:
```bash
gcloud container clusters describe YOUR_CLUSTER \
  --format="value(workloadIdentityConfig.workloadPool)"
```

如果返回空，说明集群未启用 Workload Identity。

解决: 联系集群管理员启用 Workload Identity

**2. 权限不足**

症状: `setup-workload-identity.sh` 脚本失败

解决:
- 联系管理员请求权限，或
- 请管理员帮你创建 Service Account

**3. IAM 绑定失败**

症状: Service Account 创建成功，但 IAM 绑定失败

解决: 你需要 `roles/resourcemanager.projectIamAdmin` 权限

---

## 📊 安全对比总结

```
安全性评分 (1-10，10 最高):

Workload Identity:        10/10 ⭐⭐⭐⭐⭐
  - 专用身份
  - 最小权限
  - 自动轮换
  - 无凭据文件

Service Account Key:       7/10 ⭐⭐⭐⭐
  - 专用身份
  - 最小权限
  - 需要管理密钥文件

Personal ADC:              3/10 ⭐
  - 个人身份
  - 可能有过多权限
  - 凭据会过期
  - 审计困难
```

---

## 💡 最终建议

### 如果你有创建 Service Account 的权限:

**✅ 使用 Workload Identity**
```bash
./scripts/setup-workload-identity.sh
```

### 如果你没有创建 Service Account 的权限:

**选项 1 (推荐)**: 请求权限
- 联系 GCP 管理员
- 请求 `roles/iam.serviceAccountAdmin` 或类似权限

**选项 2**: 请管理员帮忙
- 请管理员创建 Service Account: `codesweep-agent@$PROJECT_ID.iam.gserviceaccount.com`
- 授予 `roles/aiplatform.user` 权限
- 然后你可以配置 Workload Identity

**选项 3 (临时)**: 使用 ADC
```bash
./scripts/create-vertex-secret.sh
```
⚠️ 仅用于开发测试，不适合生产环境

---

## 📚 相关资源

- [Google Cloud Workload Identity Best Practices](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity)
- [IAM Service Accounts](https://cloud.google.com/iam/docs/service-accounts)
- [Vertex AI Authentication](https://cloud.google.com/vertex-ai/docs/authentication)
- [scripts/README-VERTEX-SECRET.md](scripts/README-VERTEX-SECRET.md) - 详细认证文档
