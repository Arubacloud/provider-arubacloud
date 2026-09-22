# Management Policies

## Overview

Crossplane management policies control which lifecycle operations the provider may perform on an external resource. This is configured via `spec.managementPolicies` on each managed resource.

## Available policies

| Policy | Description |
|---|---|
| `Observe` | Read and report the resource state from ArubaCloud. No create, update, or delete. |
| `Create` | Allow Crossplane to create the resource in ArubaCloud. |
| `Update` | Allow Crossplane to update the resource in ArubaCloud when the spec changes. |
| `Delete` | Allow Crossplane to delete the resource from ArubaCloud when the object is deleted. |
| `LateInitialize` | Allow Crossplane to copy Optional+Computed fields from the ArubaCloud API response into `spec.forProvider`. |

`FullControl` (the default) is shorthand for `[Observe, Create, Update, Delete, LateInitialize]`.

## Default: FullControl

If you do not set `managementPolicies`, the resource uses FullControl. Crossplane will create, update, and delete the resource in ArubaCloud.

```yaml
spec:
  forProvider:
    name: my-vpc
    location: ITBG-Bergamo
    projectIdRef:
      name: my-project
  providerConfigRef:
    name: default
# managementPolicies is omitted → defaults to FullControl
```

## Observe-only

Use `Observe` to read the current state of an existing ArubaCloud resource without managing its lifecycle. Useful for monitoring resources created outside Crossplane.

```yaml
spec:
  managementPolicies:
    - Observe
  forProvider:
    name: existing-vpc
    location: ITBG-Bergamo
    projectIdRef:
      name: my-project
```

## Importing existing resources

To bring an existing ArubaCloud resource under Crossplane management:

1. Set `crossplane.io/external-name` to the ArubaCloud resource ID.
2. Set `managementPolicies: [Observe]` initially.
3. Apply the resource and verify `status.atProvider` is populated.
4. Switch to `FullControl` once verified.

See `docs/import.md` for the full workflow.

## Recommendations by resource type

| Resource | Recommended policy | Reason |
|---|---|---|
| All standard resources | `FullControl` (default) | Safe for continuous reconciliation |
| `restore` | `[Create, Observe]` | One-shot operation — must not be re-triggered |
| `schedulejob` (OneShot type) | `[Create, Observe]` | After job fires, must not be recreated |
| `schedulejob` (Recurring type) | `FullControl` | Declarative recurring schedule |
| Externally managed resources | `[Observe]` | Adoption without lifecycle management |

## Policy transition: FullControl → Observe

To hand off a resource to external management (stop Crossplane from updating it):

```yaml
spec:
  managementPolicies:
    - Observe
    # Removed: Create, Update, Delete
```

The resource remains in ArubaCloud. Crossplane will continue to observe it but will not modify or delete it.

## Policy transition: Observe → FullControl

To resume full management after importing or monitoring:

```yaml
spec:
  managementPolicies:
    - Observe
    - Create
    - Update
    - Delete
    - LateInitialize
```

Or simply omit `managementPolicies` entirely to get the default FullControl.

## Restore and OneShot ScheduleJob guidance

Both `restore` and OneShot `schedulejob` are **imperative one-shot operations**. Crossplane's reconciler is designed for declarative resources and will attempt to recreate a resource that disappears from ArubaCloud state.

**For Restore**: After the restore completes, the operation is done. Set `managementPolicies: [Observe]` immediately after applying to prevent Crossplane from re-triggering the restore.

**For OneShot ScheduleJob**: After the job fires, set `managementPolicies: [Create, Observe]`. This prevents re-creation while still allowing Crossplane to track the job's final state.

See `docs/limitations.md` for full details.
