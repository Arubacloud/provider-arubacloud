# Troubleshooting

## Provider is not Healthy

```bash
kubectl get providers
# NAME                          INSTALLED  HEALTHY  PACKAGE  AGE
# provider-arubacloud  True       False    ...      5m
```

**Check the provider pod logs:**

```bash
kubectl -n crossplane-system logs -l pkg.crossplane.io/revision --tail=50
```

Common causes:
- The provider image cannot be pulled (check image name and pull secret)
- The provider CRDs are not installed (run `kubectl apply -f package/crds/`)
- A dependency is missing in `go.mod`

## ProviderConfig is not Ready

```bash
kubectl get providerconfigs
# NAME      READY  AGE
# default   False  1m
```

**Check the secret exists and has the correct key:**

```bash
kubectl -n crossplane-system get secret arubacloud-credentials -o jsonpath='{.data.credentials}' | base64 -d | jq .
```

The secret must contain a JSON object with at least `client_id` and `client_secret`. If the key name in the secret does not match the `key` field in the ProviderConfig `secretRef`, the credentials cannot be read.

## Resource stuck in "Synced: False" with ReconcileError

```bash
kubectl describe vpc main-vpc
# Events:
#   cannot get referenced ProviderConfig: ...
```

**Possible causes:**

1. **ProviderConfig not found**: Check `spec.providerConfigRef.name` matches an existing ProviderConfig.
2. **Reference not resolved**: A `*Ref` or `*Selector` field points to a resource that does not yet exist or is not yet Ready. Crossplane will retry automatically when the dependency becomes ready.
3. **Credentials error**: ArubaCloud API rejected the credentials. Verify `client_id` and `client_secret` are valid.

## Resource stuck creating

For resources like KaaS that take 10–30 minutes to create, the reconcile will block until the Terraform provider returns. This is expected behavior. Monitor progress via:

```bash
kubectl get kaas my-cluster -w
```

If the resource stays in a non-Ready state beyond the `resource_timeout` period (default 30 minutes), check:
1. The `resource_timeout` in the credentials secret — increase to `60m` for KaaS.
2. The ArubaCloud console to see if the operation is still running.

## External name annotation missing

If you created a resource and the external name annotation is empty:

```bash
kubectl get vpc main-vpc -o jsonpath='{.metadata.annotations.crossplane\.io/external-name}'
```

If empty, the resource may not have been created yet, or creation failed. Check `status.conditions` for error details.

## Import workflow fails

If applying a resource with `crossplane.io/external-name` does not result in a populated `status.atProvider`:

1. Verify the external name value is the correct leaf ID (not the full URI or composite import path).
2. Verify all required `*Ref` fields are set so that the composite import ID can be constructed. For example, `Subnet` requires both `projectIdRef` and `vpcIdRef`.
3. Check `kubectl describe <resource>` for a more detailed error from the Terraform provider.

## connection secret is empty

If `writeConnectionSecretToRef` is set but the secret has no data:

1. Verify the resource is Ready (`status.conditions[Ready].status == True`).
2. For KaaS, verify the cluster has a `kubeconfig` in `status.atProvider.kubeconfig`.
3. For ElasticIP, verify `status.atProvider.address` is populated.
4. For DBaaSUser, `password` is write-only — it is never returned by the API and will not appear in the connection secret.

## golangci-lint errors

To run the linter locally:

```bash
make lint
```

Or directly:

```bash
golangci-lint run ./...
```

Generated files (prefixed `zz_`) are excluded from linting via `.golangci.yml`.

## Stale generated files

If CI fails with `make generate && git diff --exit-code` detecting changes:

```bash
make generate
git diff --stat
```

Commit any changed generated files. This typically happens when `config/` changes without regenerating the CRDs and controller setup files.

## go build fails after adding a new resource

After adding a new resource configurator, run:

```bash
make generate
go build ./...
```

If you see import errors for generated packages, ensure `make generate` completed successfully.
