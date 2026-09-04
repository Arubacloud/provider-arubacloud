# Authentication

## Credentials format

The provider authenticates against ArubaCloud using OAuth2 client credentials. Credentials are stored in a Kubernetes Secret as a JSON object.

**Required fields:**

```json
{
  "client_id": "your-client-id",
  "client_secret": "your-client-secret"
}
```

**Optional fields:**

```json
{
  "client_id": "...",
  "client_secret": "...",
  "base_url": "",
  "token_issuer_url": "",
  "resource_timeout": "30m"
}
```

| Field | Description | Default |
|---|---|---|
| `client_id` | OAuth2 client identifier | required |
| `client_secret` | OAuth2 client secret | required |
| `base_url` | Override the ArubaCloud API base URL | ArubaCloud default |
| `token_issuer_url` | Override the OAuth2 token endpoint | ArubaCloud default |
| `resource_timeout` | Default timeout for async operations | `30m` |

For KaaS clusters (which take 10–30 minutes to create) set `resource_timeout` to at least `60m`.

## Creating the credentials Secret

```bash
kubectl create secret generic arubacloud-credentials \
  --namespace crossplane-system \
  --from-literal=credentials='{
    "client_id": "YOUR_CLIENT_ID",
    "client_secret": "YOUR_CLIENT_SECRET",
    "resource_timeout": "30m"
  }'
```

## Creating the ProviderConfig

```yaml
apiVersion: arubacloud.crossplane.io/v1beta1
kind: ProviderConfig
metadata:
  name: default
spec:
  credentials:
    source: Secret
    secretRef:
      name: arubacloud-credentials
      namespace: crossplane-system
      key: credentials
```

## Multiple projects

ArubaCloud organizes resources by project. Every managed resource has a `project_id` field (or a `projectIdRef`). The ProviderConfig does **not** set a global project — each resource must specify its own.

To manage resources in multiple projects with different credentials:

```yaml
# Project A credentials
apiVersion: arubacloud.crossplane.io/v1beta1
kind: ProviderConfig
metadata:
  name: project-a
spec:
  credentials:
    source: Secret
    secretRef:
      name: project-a-credentials
      namespace: crossplane-system
      key: credentials
---
# Resource in project A
apiVersion: arubacloud.crossplane.io/v1alpha1
kind: VPC
metadata:
  name: vpc-project-a
spec:
  forProvider:
    projectIdRef:
      name: my-project-a
  providerConfigRef:
    name: project-a
```

## Environment variables (local development)

For local development with `make run`, set:

```bash
export ARUBACLOUD_CLIENT_ID=your-client-id
export ARUBACLOUD_CLIENT_SECRET=your-client-secret
```
