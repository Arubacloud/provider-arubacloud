# Async Resources and Timeouts

## All resources are async

Every ArubaCloud resource is asynchronous. When you call `Create` or `Delete` on a resource, the ArubaCloud API starts the operation and returns immediately. The Terraform provider polls the API until the operation completes before returning to Crossplane.

From Crossplane's perspective the operations appear synchronous — Upjet calls into the Terraform provider binary, which blocks until ready. The polling is handled inside the Terraform provider binary via `WaitUntilReady()` and `WaitForResourceDeleted()`.

## Default timeout

The Terraform provider's `resource_timeout` field (default: `30m`) controls how long it waits for async operations. You set this in the credentials JSON in your ProviderConfig secret:

```json
{
  "client_id": "...",
  "client_secret": "...",
  "resource_timeout": "30m"
}
```

This applies to all resources managed by that ProviderConfig.

## KaaS timeout

KaaS (Kubernetes as a Service) cluster creation can take **10–30 minutes**. The default `resource_timeout` of 30 minutes may not be sufficient in all environments.

**Recommended**: Set `resource_timeout` to at least `60m` for ProviderConfigs that manage KaaS resources.

```bash
kubectl create secret generic arubacloud-credentials \
  --namespace crossplane-system \
  --from-literal=credentials='{
    "client_id": "YOUR_CLIENT_ID",
    "client_secret": "YOUR_CLIENT_SECRET",
    "resource_timeout": "60m"
  }'
```

Alternatively, the `timeout` field available in all resource schemas allows per-resource override:

```yaml
spec:
  forProvider:
    timeout: "90m"
    name: my-cluster
    ...
```

## Crossplane reconcile interval

Crossplane reconciles managed resources on a default poll interval of 10 minutes. For long-running operations (e.g. KaaS creation), the Terraform provider will block during the reconcile until the operation completes or times out. This means the Crossplane controller pod for that resource will be occupied for the duration.

To avoid this, consider setting a generous `resource_timeout` and allowing the reconcile to complete naturally rather than relying on retries.

## Deletion timeouts

Deletion is also async. The Terraform provider polls until the resource is confirmed deleted. Deletion is usually faster than creation but may still take several minutes for resources like KaaS clusters and DBaaS instances.

## Monitoring progress

While a resource is being created or deleted, `status.conditions` will show:

```yaml
conditions:
  - type: Synced
    status: "False"
    reason: ReconcileError
    message: "waiting for resource to be ready..."
  - type: Ready
    status: "False"
```

Once the operation completes:

```yaml
conditions:
  - type: Synced
    status: "True"
  - type: Ready
    status: "True"
```
