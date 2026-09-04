# Known Limitations

## 1. `restore` is a one-shot imperative operation

`arubacloud_restore` triggers a data restoration operation in ArubaCloud. This is **not** a declarative resource — applying it once performs the restore. Crossplane's reconciler continuously checks that the resource exists and will attempt to recreate it if it disappears.

**Consequence**: If you leave `managementPolicies` at FullControl, Crossplane may retrigger the restore operation on subsequent reconcile cycles.

**Required action**: Immediately after the restore resource becomes `Ready`, switch to Observe-only:

```yaml
spec:
  managementPolicies:
    - Observe
```

Alternatively, use `[Create, Observe]` from the start (the resource is created once and then only observed):

```yaml
spec:
  managementPolicies:
    - Create
    - Observe
  forProvider:
    name: boot-vol-restore
    projectIdRef:
      name: my-project
    backupIdRef:
      name: boot-vol-backup
    volumeIdRef:
      name: boot-vol
```

## 2. `schedulejob` with OneShot type must not be recreated

A OneShot `schedulejob` fires once at `scheduleAt` time and is done. If Crossplane detects the job has been removed from ArubaCloud state, it will recreate it — triggering the action again.

**Required action**: After a OneShot job fires, set `managementPolicies: [Create, Observe]` to prevent re-creation.

Recurring jobs (`scheduleJobType: Recurring`) are safe for FullControl and do not have this limitation.

## 3. `keypair.value` is write-only

The public key value is sent to ArubaCloud on creation but is never returned by the API. After creation (or import), the `value` field will be absent from `status.atProvider`.

**Consequence**: Crossplane cannot verify that the key stored in ArubaCloud matches what is in `spec.forProvider.value`. Preserve the original key value in your spec.

## 4. `dbaasuser.password` is write-only and immutable

The DBaaS user password is:
- Sent to ArubaCloud on creation (base64-encoded)
- Never returned by the API
- Treated as ForceNew — any change to `passwordSecretRef` replaces the DBaaS user

If you rotate the password by updating `passwordSecretRef`, Terraform will delete and recreate the `dbaasuser` resource. Plan accordingly and update any `databasegrant` resources referencing the old user.

## 5. `cloudserver` is nearly fully immutable

Almost all CloudServer fields are `ForceNew`. Changing any of the following replaces the server (delete + create):
- `location`, `zone`
- `settings.flavor_name`, `settings.key_pair_uri_ref`
- `network.*` (all network configuration)
- `storage.*` (all storage configuration)

Only `timeout` can be changed without replacing the server.

**Consequence**: Be sure your CloudServer spec is final before applying. Use `managementPolicies: [Create, Observe]` if you want to protect against accidental replacement.

## 6. CLI execution mode overhead

The provider runs Terraform as a subprocess (CLI fork mode) on every reconcile. Under high resource count or high reconcile frequency, this creates significant system load. Each reconcile forks a new Terraform process.

**Mitigation**: Increase the poll interval for non-critical resources. The default is 10 minutes; consider 20–30 minutes for stable resources. This is configurable in the Crossplane ProviderRevision.

## 7. No provider-level project default

`project_id` must be specified in every managed resource. There is no global default in ProviderConfig. Use `projectIdRef` to reference a `Project` resource by name, or `projectIdSelector` to select by label.

## 8. `database` name is immutable

Despite the Terraform provider's Update code applying name changes to databases, the schema marks `name` as ForceNew. The schema is authoritative. Changing `name` in an existing Database resource will delete and recreate the database.

## 9. `databasegrant` and `databasebackup` import format requires verification

The import ID format for `arubacloud_databasegrant` and `arubacloud_databasebackup` has not been verified against a live ArubaCloud environment. The current implementation uses a best-effort composite ID based on the resource schema. If import fails, check the Terraform provider source for the authoritative format.

## 10. `vpcpeeringroute` import format requires verification

The full import path for `arubacloud_vpcpeeringroute` was assumed to be `project_id/vpc_id/vpc_peering_id/route_id`. Verify this against a live environment before using the import workflow for this resource type.

## 11. KaaS `pod_cidr` is excluded from late initialization

The `pod_cidr` field on KaaS is user-controlled. The Terraform provider never overwrites it from the API response. Crossplane follows the same behavior — `pod_cidr` will not be late-initialized from `status.atProvider`.
