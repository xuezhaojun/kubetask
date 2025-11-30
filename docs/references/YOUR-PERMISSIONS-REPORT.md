# Your GCP Permissions Report

Generated: $(date)
Account: zxue@redhat.com
Project: itpc-gcp-hcm-pe-eng-claude

---

## ✅ Test Results

| Permission | Status | Details |
|-----------|--------|---------|
| **List Service Accounts** | ✅ YES | Can list service accounts in the project |
| **Create Service Accounts** | ✅ YES | Successfully created and deleted test SA |
| **View IAM Policy** | ❌ NO | Cannot view project IAM policy |
| **Grant IAM Roles** | ❓ MAYBE | Need to test (likely no based on IAM policy restriction) |

---

## 🎯 Conclusion

### What you CAN do:
✅ Create Service Accounts
✅ Delete Service Accounts (your own)
✅ List Service Accounts

### What you CANNOT do:
❌ View project-level IAM policies
❌ Grant project-level IAM roles (likely)

---

## 🚧 Implications

This means you have **partial permissions** for Workload Identity:

### ✅ Good News:
You CAN create the Service Account needed for Workload Identity

### ⚠️ Challenge:
You likely CANNOT grant the Service Account the `roles/aiplatform.user` role

---

## 🎯 Recommended Approach

### Option 1: Try Workload Identity (may need admin help)

**Step 1**: Try running the setup script
```bash
./scripts/setup-workload-identity.sh
```

**What will happen**:
- ✅ Will succeed: Creating Service Account
- ❌ Will likely fail: Granting `roles/aiplatform.user` role

**Step 2**: If it fails at granting roles
Contact your GCP admin and provide this information:

```
Hi,

I need help setting up Workload Identity for the CodeSweep project.

I've created a Service Account:
  Name: codesweep-agent
  Email: codesweep-agent@itpc-gcp-hcm-pe-eng-claude.iam.gserviceaccount.com

Could you please grant this Service Account the following role:
  Role: roles/aiplatform.user
  Project: itpc-gcp-hcm-pe-eng-claude

This is needed to access Vertex AI for Claude Code.

Thank you!
```

---

### Option 2: Use ADC Method (temporary workaround)

If you need to get started quickly while waiting for admin help:

```bash
# 1. Create secrets using your personal ADC
./scripts/create-vertex-secret.sh

# 2. Deploy using ADC job manifest
kubectl apply -f k8s/hello-world-experiment/manifests/03-hello-world-job-adc.yaml
```

**⚠️ Important**:
- This uses YOUR personal credentials
- Not recommended for production
- Should switch to Workload Identity once admin grants the role

---

### Option 3: Request Full Permissions (recommended for long-term)

Ask your GCP admin for these roles:

```
Request for GCP Permissions:

Project: itpc-gcp-hcm-pe-eng-claude
Account: zxue@redhat.com

Roles needed:
1. roles/iam.serviceAccountAdmin
   (Currently have partial - can create, but not grant roles)

2. roles/resourcemanager.projectIamAdmin (or similar)
   (To grant roles to service accounts I create)

Purpose:
- Set up Workload Identity for CodeSweep automation
- Follow GCP security best practices
- Avoid using personal credentials in Kubernetes

This is for automated code sweeping tasks using Claude Code with Vertex AI.
```

---

## 📝 Next Steps

### Immediate (Try this now):

```bash
# Try the Workload Identity setup
./scripts/setup-workload-identity.sh

# If it fails at granting roles, note the error message
# and contact your admin with the information above
```

### Short-term (If WI setup fails):

```bash
# Use ADC method temporarily
./scripts/create-vertex-secret.sh

# Deploy with ADC
kubectl apply -f k8s/hello-world-experiment/manifests/03-hello-world-job-adc.yaml
```

### Long-term (Recommended):

1. Get admin to grant role to your Service Account, OR
2. Request full IAM admin permissions for yourself
3. Switch to Workload Identity for production

---

## 🔍 Understanding Your Permissions

You have what's called "**Service Account User**" permissions, which allows you to:
- Create and manage Service Accounts
- Use Service Accounts in applications

But you don't have "**IAM Admin**" permissions, which would allow you to:
- Grant roles to Service Accounts
- Modify project-level IAM policies

This is a common security pattern - separation of duties.

---

## 💡 Why Workload Identity Still Worth Pursuing

Even though you need admin help to complete the setup, Workload Identity is still the best approach because:

1. **One-time setup**: Admin only needs to grant the role once
2. **No credential management**: No need to update secrets when credentials expire
3. **Better security**: Uses dedicated service account, not your personal identity
4. **Audit trail**: Clear logs of what the service account does
5. **Production ready**: Follows Google Cloud best practices

The ADC method requires YOU to update the secret every time your personal credentials expire, which is ongoing maintenance.

---

## 📞 Template Email to Admin

Subject: Request: Grant IAM Role for CodeSweep Service Account

```
Hi [Admin Name],

I'm setting up Workload Identity for our CodeSweep automation project.

I've created a Service Account:
  - Name: codesweep-agent
  - Email: codesweep-agent@itpc-gcp-hcm-pe-eng-claude.iam.gserviceaccount.com
  - Project: itpc-gcp-hcm-pe-eng-claude

Could you please grant this Service Account the following permission:
  - Role: roles/aiplatform.user

This role is needed to access Vertex AI API for Claude Code automation.

Alternatively, if you prefer, you could grant me the permission to manage
IAM roles (roles/resourcemanager.projectIamAdmin) so I can do this myself
for future service accounts.

Background:
- Using Workload Identity instead of personal credentials (security best practice)
- Following Google Cloud's recommended approach
- Project: Automated code maintenance using Claude Code

Let me know if you need any additional information!

Thanks,
Zxue
```

---

## 🔗 Related Documentation

- [AUTHENTICATION-DECISION-GUIDE.md](AUTHENTICATION-DECISION-GUIDE.md) - Full comparison of auth methods
- [scripts/README-VERTEX-SECRET.md](scripts/README-VERTEX-SECRET.md) - Detailed setup guides
- [scripts/setup-workload-identity.sh](scripts/setup-workload-identity.sh) - The setup script
