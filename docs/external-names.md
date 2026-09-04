# External Names

## Overview

Crossplane uses the `crossplane.io/external-name` annotation to track which cloud resource a managed object represents. For the ArubaCloud provider, the external name is always the **leaf resource ID** — the last segment of the resource URI.

ArubaCloud resource URIs follow the pattern:

```
/projects/{project_id}/network/vpcs/{vpc_id}
```

The external name for a VPC is `vpc_id`, not the full path.

## How external names map to Terraform import IDs

The Terraform ArubaCloud provider uses composite import IDs that include parent IDs. Crossplane reconstructs these at reconcile time from the resolved references in `spec.forProvider`. You never need to put the full composite ID in the annotation.

| Resource | External name | Full Terraform import ID |
|---|---|---|
| Project | `proj-abc` | `proj-abc` |
| VPC | `vpc-xyz` | `proj-abc/vpc-xyz` |
| Subnet | `sub-xyz` | `proj-abc/vpc-123/sub-xyz` |
| SecurityGroup | `sg-xyz` | `proj-abc/vpc-123/sg-xyz` |
| SecurityRule | `rule-xyz` | `proj-abc/vpc-123/sg-456/rule-xyz` |
| ElasticIP | `eip-xyz` | `proj-abc/eip-xyz` |
| KeyPair | `kp-xyz` | `proj-abc/kp-xyz` |
| BlockStorage | `vol-xyz` | `proj-abc/vol-xyz` |
| Snapshot | `snap-xyz` | `proj-abc/snap-xyz/Hour` |
| Backup | `bk-xyz` | `proj-abc/bk-xyz` |
| Restore | `rs-xyz` | `proj-abc/rs-xyz` |
| VPCPeering | `peer-xyz` | `proj-abc/vpc-123/peer-xyz` |
| VPCPeeringRoute | `rt-xyz` | `proj-abc/vpc-123/peer-456/rt-xyz` |
| VPNTunnel | `tun-xyz` | `proj-abc/tun-xyz` |
| VPNRoute | `vnrt-xyz` | `proj-abc/vnrt-xyz` |
| CloudServer | `srv-xyz` | `proj-abc/srv-xyz` |
| KaaS | `k8s-xyz` | `proj-abc/k8s-xyz` |
| ContainerRegistry | `cr-xyz` | `proj-abc/cr-xyz` |
| DBaaS | `dbaas-xyz` | `proj-abc/dbaas-xyz` |
| Database | `db-xyz` | `proj-abc/dbaas-123/db-xyz` |
| DBaaSUser | `user-xyz` | `proj-abc/dbaas-123/user-xyz` |
| DatabaseGrant | `grant-xyz` | `proj-abc/dbaas-123/db-456/grant-xyz` |
| DatabaseBackup | `dbbk-xyz` | `proj-abc/dbaas-123/dbbk-xyz` |
| KMS | `kms-xyz` | `proj-abc/kms-xyz` |
| ScheduleJob | `job-xyz` | `proj-abc/job-xyz` |

### Special case: Snapshot

Snapshot has a 3-part import ID that includes `billing_period`. The billing period is read from `spec.forProvider.billingPeriod` at reconcile time, not from the external name.

```yaml
metadata:
  annotations:
    crossplane.io/external-name: snap-xyz   # snapshot ID only
spec:
  forProvider:
    billingPeriod: Hour                     # used to reconstruct full import ID
    projectIdRef:
      name: my-project
    volumeUriRef:
      name: my-volume
```

## Setting an external name

External names are set via annotation:

```yaml
metadata:
  name: my-vpc           # Kubernetes object name
  annotations:
    crossplane.io/external-name: vpc-abc123  # ArubaCloud resource ID
```

When you create a new resource **without** this annotation, Crossplane creates the resource in ArubaCloud and populates the annotation automatically from the provider-generated ID.

When you set this annotation explicitly (for resource adoption / import), Crossplane will look up the existing resource in ArubaCloud rather than creating a new one.

## DisableNameInitializer

All ArubaCloud resources set `DisableNameInitializer: true`. This prevents Upjet from defaulting the external name to the Kubernetes object name, since ArubaCloud IDs are always provider-generated.
