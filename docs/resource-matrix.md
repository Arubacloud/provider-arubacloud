# ArubaCloud Crossplane Provider — Resource Matrix

> Status key: PLANNED · COMPLETE · COMPLETE WITH CUSTOM CONFIG · PARTIAL · BLOCKED · UNSUPPORTED

| Terraform Resource | Crossplane Kind | Priority | Auto Generated | Custom Config | References | Async | External Name | Risk | Planned Phase | Status |
|---|---|---:|---:|---:|---:|---:|---|---:|---:|---|
| `arubacloud_project` | `Project` | 1 | Yes | Yes | None | No | IdentifierFromProvider | Low | Phase 2 | PLANNED |
| `arubacloud_vpc` | `VPC` | 1 | Yes | Yes | ProjectRef | Yes | IdentifierFromProvider | Low | Phase 2 | PLANNED |
| `arubacloud_subnet` | `Subnet` | 1 | Yes | Yes | ProjectRef, VPCRef | Yes | IdentifierFromProvider + custom GetIDFn (3-part) | Medium | Phase 2 | PLANNED |
| `arubacloud_keypair` | `KeyPair` | 1 | Yes | Yes | ProjectRef | No | IdentifierFromProvider | Low | Phase 2 | PLANNED |
| `arubacloud_elasticip` | `ElasticIP` | 1 | Yes | Yes | ProjectRef | Yes | IdentifierFromProvider | Low | Phase 2 | PLANNED |
| `arubacloud_cloudserver` | `CloudServer` | 1 | Yes | Yes | ProjectRef, VPCRef, SubnetRef (list), SecurityGroupRef (list), ElasticIPRef, KeyPairRef, BlockStorageRef | Yes | IdentifierFromProvider + custom GetIDFn (2-part) | Medium | Phase 2 | PLANNED |
| `arubacloud_blockstorage` | `BlockStorage` | 2 | Yes | Yes | ProjectRef | Yes | IdentifierFromProvider | Low | Phase 3 | PLANNED |
| `arubacloud_snapshot` | `Snapshot` | 2 | Yes | Yes | ProjectRef, BlockStorageRef | Yes | IdentifierFromProvider + custom GetIDFn (3-part with billing_period) | Medium | Phase 3 | PLANNED |
| `arubacloud_backup` | `Backup` | 2 | Yes | Yes | ProjectRef, BlockStorageRef | Yes | IdentifierFromProvider | Low | Phase 3 | PLANNED |
| `arubacloud_restore` | `Restore` | 3 | Yes | Yes | ProjectRef, BackupRef, BlockStorageRef | Yes | IdentifierFromProvider | **High** | Phase 8 | PARTIAL |
| `arubacloud_securitygroup` | `SecurityGroup` | 2 | Yes | Yes | ProjectRef, VPCRef | Yes | IdentifierFromProvider + custom GetIDFn (3-part) | Medium | Phase 4 | PLANNED |
| `arubacloud_securityrule` | `SecurityRule` | 2 | Yes | Yes | ProjectRef, VPCRef, SecurityGroupRef | Yes | IdentifierFromProvider + custom GetIDFn (4-part) | Medium | Phase 4 | PLANNED |
| `arubacloud_vpcpeering` | `VPCPeering` | 2 | Yes | Yes | ProjectRef, VPCRef, peer VPCRef | Yes | IdentifierFromProvider + custom GetIDFn (3-part) | Medium | Phase 4 | PLANNED |
| `arubacloud_vpcpeeringroute` | `VPCPeeringRoute` | 2 | Yes | Yes | ProjectRef, VPCRef, VPCPeeringRef | Yes | IdentifierFromProvider + custom GetIDFn (REQUIRES VERIFICATION) | Medium | Phase 4 | PLANNED |
| `arubacloud_vpntunnel` | `VPNTunnel` | 2 | Yes | Yes | ProjectRef, VPCRef, SubnetRef, ElasticIPRef | Yes | IdentifierFromProvider | Medium | Phase 4 | PLANNED |
| `arubacloud_vpnroute` | `VPNRoute` | 2 | Yes | Yes | ProjectRef, VPNTunnelRef | Yes | IdentifierFromProvider (REQUIRES VERIFICATION) | Medium | Phase 4 | PLANNED |
| `arubacloud_kaas` | `KaaS` | 1 | Yes | Yes | ProjectRef, VPCRef, SubnetRef, SecurityGroupRef (by name — custom) | Yes | IdentifierFromProvider | **High** | Phase 5 | PLANNED |
| `arubacloud_containerregistry` | `ContainerRegistry` | 2 | Yes | Yes | ProjectRef, ElasticIPRef, VPCRef, SubnetRef, SecurityGroupRef, BlockStorageRef | Yes | IdentifierFromProvider | Medium | Phase 5 | PLANNED |
| `arubacloud_dbaas` | `DBaaS` | 1 | Yes | Yes | ProjectRef, VPCRef, SubnetRef, SecurityGroupRef, ElasticIPRef | Yes | IdentifierFromProvider | Medium | Phase 6 | PLANNED |
| `arubacloud_database` | `Database` | 1 | Yes | Yes | ProjectRef, DBaaSRef | Yes | IdentifierFromProvider + custom GetIDFn (3-part) | Medium | Phase 6 | PLANNED |
| `arubacloud_dbaasuser` | `DBaaSUser` | 1 | Yes | Yes | ProjectRef, DBaaSRef | Yes | IdentifierFromProvider + custom GetIDFn (3-part) | Medium | Phase 6 | PLANNED |
| `arubacloud_databasegrant` | `DatabaseGrant` | 2 | Yes | Yes | ProjectRef, DBaaSRef, DatabaseRef, DBaaSUserRef | Yes | IdentifierFromProvider (REQUIRES VERIFICATION) | Medium | Phase 6 | PLANNED |
| `arubacloud_databasebackup` | `DatabaseBackup` | 2 | Yes | Yes | ProjectRef, DBaaSRef, DatabaseRef | Yes | IdentifierFromProvider (REQUIRES VERIFICATION) | Medium | Phase 6 | PLANNED |
| `arubacloud_schedulejob` | `ScheduleJob` | 3 | Yes | Yes | ProjectRef | No | IdentifierFromProvider | **High** | Phase 7 | PARTIAL |
| `arubacloud_kms` | `KMS` | 2 | Yes | Yes | ProjectRef | Yes | IdentifierFromProvider | Low | Phase 7 | PLANNED |

---

## Data Source Inventory

| Terraform Data Source | Crossplane Recommendation | Reason |
|---|---|---|
| `data.arubacloud_project` | Reference / Selector | Discover existing projects by name; use with projectSelector |
| `data.arubacloud_vpc` | Reference / Selector | Discover existing VPCs; use with vpcSelector |
| `data.arubacloud_subnet` | Reference / Selector | Discover existing subnets |
| `data.arubacloud_securitygroup` | Reference / Selector | Discover existing SGs |
| `data.arubacloud_securityrule` | Reference only | Rules are tightly scoped to SGs; observation sufficient |
| `data.arubacloud_keypair` | Reference / Selector | Look up keypairs by name |
| `data.arubacloud_elasticip` | Reference / Selector | Discover pre-existing elastic IPs |
| `data.arubacloud_cloudserver` | Observation-only | Observe existing servers not managed by Crossplane |
| `data.arubacloud_blockstorage` | Reference / Selector | Look up volumes |
| `data.arubacloud_snapshot` | Reference / Selector | Look up snapshots |
| `data.arubacloud_vpcpeering` | Reference / Selector | Look up peerings |
| `data.arubacloud_vpcpeeringroute` | Observation-only | Granular route observation |
| `data.arubacloud_kaas` | Observation-only | Observe non-Crossplane clusters |
| `data.arubacloud_containerregistry` | Observation-only | Observe non-Crossplane registries |
| `data.arubacloud_backup` | Observation-only | Audit backup state |
| `data.arubacloud_restore` | Observation-only | Observe restore status |
| `data.arubacloud_dbaas` | Reference / Selector | Look up DB clusters |
| `data.arubacloud_database` | Reference / Selector | Look up databases |
| `data.arubacloud_dbaasuser` | Observation-only | Audit DB users |
| `data.arubacloud_databasegrant` | Observation-only | Audit grants |
| `data.arubacloud_databasebackup` | Observation-only | Audit DB backups |
| `data.arubacloud_kms` | Reference / Selector | Look up KMS instances |
| `data.arubacloud_schedulejob` | Observation-only | Observe job state |
| `data.arubacloud_vpntunnel` | Reference / Selector | Look up VPN tunnels |
| `data.arubacloud_vpnroute` | Observation-only | Observe VPN routes |

**Note**: Crossplane's selector mechanism (`matchLabels`) achieves data-source-like discovery. Data sources do not need to become separate managed resources. Upjet does not auto-generate resources from data sources; they are handled by Kubernetes-native reference resolution.

---

## Resources Requiring Special Handling

### `arubacloud_restore` — PARTIAL

**Problem**: `restore` is an imperative one-shot operation. It creates a new volume from a backup at a specific point in time. Crossplane's reconciliation loop would attempt to recreate the restore operation on every re-sync unless the resource reaches and stays in a terminal Ready state.

**Analysis**:
- `Create`: triggers a restore operation.
- `Read`: returns current restore state.
- `Delete`: removes the restore artifact.
- Re-creating the resource would trigger a new restore from scratch, potentially overwriting user data.

**Recommended approach**:
1. Include the resource in the provider.
2. Document that `managementPolicies: [Create, Observe]` (or `ObserveOnly` after first apply) must be used.
3. Add a validation comment in the generated CRD.
4. Consider a custom initializer that sets management policy after successful create.

**Alternative**: Exclude from v0.x and document why in `docs/limitations.md`. Include in Phase 8.

### `arubacloud_schedulejob` — PARTIAL

**Problem**: Jobs with `schedule_job_type: OneShot` execute once and terminate. Crossplane would observe them in a terminal state but might attempt to recreate them if they are deleted externally or if drift is detected.

**Analysis**:
- `Recurring` jobs are declarative and fully safe for Crossplane.
- `OneShot` jobs require `managementPolicies: [Create, Observe]` after first run.

**Recommended approach**: Include the resource. Document the OneShot limitation. Users must set appropriate management policies for OneShot jobs.

### Resources with composite IDs

Resources with 3- or 4-part Terraform import IDs require custom `GetIDFn` and `GetExternalNameFn` implementations in `config/external_name.go`.

Affected resources:
- `subnet` — 3-part: `project_id/vpc_id/subnet_id`
- `securitygroup` — 3-part: `project_id/vpc_id/sg_id`
- `securityrule` — 4-part: `project_id/vpc_id/sg_id/rule_id`
- `vpcpeering` — 3-part: `project_id/vpc_id/peering_id`
- `snapshot` — 3-part: `project_id/snap_id/billing_period` (unusual: non-ID in path)
- `database` — 3-part: `project_id/dbaas_id/db_id`
- `dbaasuser` — 3-part: `project_id/dbaas_id/user_id`

The parent IDs (`vpc_id`, `sg_id`, `dbaas_id`) are derivable from the resolved references at reconcile time.

### `arubacloud_kaas` — High Risk

- Creation can take 10–30 minutes.
- `kubeconfig` is a sensitive Computed field that must become a Crossplane connection detail.
- `security_group_name` is referenced by name rather than ID/URI — requires custom reference configuration.
- Node pool updates are in-place (supported).
- Kubernetes version upgrades are supported.

### `arubacloud_dbaasuser` — Medium Risk

- `password` is immutable (any change forces replacement).
- `password` is never returned by the API (write-only).
- In Crossplane: password should come from a Secret reference and must be treated as a connection detail output.
