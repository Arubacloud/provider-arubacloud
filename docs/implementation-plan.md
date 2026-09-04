# ArubaCloud Crossplane Provider — Implementation Plan

> Stage 1 deliverable. Do not modify implementation code until this document is approved.

---

## Table of Contents

1. [Analyzed Sources](#1-analyzed-sources)
2. [Architecture Summary](#2-architecture-summary)
3. [Resource Inventory](#3-resource-inventory)
4. [Data Source Inventory](#4-data-source-inventory)
5. [Terraform → Crossplane Mapping](#5-terraform--crossplane-mapping)
6. [External Name Strategy](#6-external-name-strategy)
7. [Reference Strategy](#7-reference-strategy)
8. [Selector Strategy](#8-selector-strategy)
9. [ProviderConfig Strategy](#9-providerconfig-strategy)
10. [Secret Strategy](#10-secret-strategy)
11. [Async Strategy](#11-async-strategy)
12. [Management Policy Strategy](#12-management-policy-strategy)
13. [Import Strategy](#13-import-strategy)
14. [KaaS Strategy](#14-kaas-strategy)
15. [Database Strategy](#15-database-strategy)
16. [Backup / Restore / Snapshot Strategy](#16-backup--restore--snapshot-strategy)
17. [Testing Strategy](#17-testing-strategy)
18. [CI/CD Strategy](#18-cicd-strategy)
19. [Documentation Strategy](#19-documentation-strategy)
20. [Implementation Phases](#20-implementation-phases)
21. [Phase Dependencies](#21-phase-dependencies)
22. [Technical Risk Assessment](#22-technical-risk-assessment)
23. [Known Limitations](#23-known-limitations)
24. [Definition of Done](#24-definition-of-done)

---

## 1. Analyzed Sources

| Source | Version / Ref | Date Analyzed |
|---|---|---|
| `github.com/Arubacloud/terraform-provider-arubacloud` | `master` | 2026-08-11 |
| `github.com/Arubacloud/sdk-go` | `main` | 2026-08-11 |
| `github.com/crossplane/upjet` | `v2.4.1` (latest) | 2026-08-11 |
| `github.com/crossplane-contrib/provider-upjet-digitalocean` | `main` | 2026-08-11 |
| `github.com/upbound/provider-azuread/v2` (upjet-azuread) | `main` | 2026-08-11 |
| Upjet documentation: `generating-a-provider.md` | `main` | 2026-08-11 |
| Upjet documentation: `configuring-a-resource.md` | `main` | 2026-08-11 |
| Upjet documentation: `upjet-v2-upgrade.md` | `main` | 2026-08-11 |

---

## 2. Architecture Summary

### ArubaCloud Terraform Provider

- **Go version**: 1.24.0
- **Module**: `github.com/Arubacloud/terraform-provider-arubacloud`
- **Plugin API**: `terraform-plugin-framework` v1.16.1 (**not** `terraform-plugin-sdk/v2`)
- **Resources**: 25 managed resources
- **Data sources**: 24 data sources (one resource has no data source counterpart)
- **Authentication**: OAuth2 client credentials (client_id + client_secret → JWT bearer token)
- **SDK**: `github.com/Arubacloud/sdk-go` v1.0.9
- **All resources**: fully async (WaitUntilReady / WaitForResourceDeleted pattern)
- **Addressing**: URI-based (`/projects/{projectID}/...`) not bare-ID-based

### Upjet v2

- **Version**: v2.4.1 (latest stable)
- **Go version**: 1.26.5
- **Module**: `github.com/crossplane/upjet/v2`
- **Crossplane-runtime**: v2.3.3
- **Plugin API support**: ALL three — CLI (fork), SDK v2 (direct), Framework (protov6)
- **Namespaced resources**: Yes (Upjet v2 feature, parallel to cluster-scoped)
- **Management policies**: Yes (Observe, Create, Update, Delete)
- **Provider template**: `https://github.com/crossplane/upjet-provider-template`

### Critical Finding: Plugin Framework Compatibility

The ArubaCloud Terraform provider uses `terraform-plugin-framework` exclusively.  
Upjet v2 supports framework providers via its "protov6" mode.

**ASSUMPTION**: Upjet's CLI execution mode is universally compatible regardless of whether the wrapped provider uses SDK or Framework internally, because at runtime Upjet forks the `terraform` binary, which handles the plugin protocol internally.

**REQUIRES VERIFICATION**: Confirm in Phase 1, Step 3 that `make generate` (which calls `terraform providers schema -json`) correctly extracts schemas from the ArubaCloud framework provider.

---

## 3. Resource Inventory

### 3.1 Management

#### `arubacloud_project`

| Property | Value |
|---|---|
| Schema fields | `name` (Required), `description` (Optional), `tags` (Optional), `id` (Computed), `timeout` (Optional) |
| Import format | `<project_id>` |
| Create | Sync |
| Read | Sync |
| Update | name, description, tags |
| Delete | Async (polling) |
| Async behavior | Delete: polling until 404 |
| ForceNew fields | None |
| Computed fields | `id` |
| Sensitive fields | None |
| Crossplane issues | None |

---

### 3.2 Compute

#### `arubacloud_cloudserver`

| Property | Value |
|---|---|
| Schema fields | `name` (Required, immutable), `location` (Required, ForceNew), `project_id` (Required, ForceNew), `zone` (Required, ForceNew), `tags` (Optional, ForceNew), `timeout` (Optional); `network` block: `vpc_uri_ref` (Required, ForceNew), `elastic_ip_uri_ref` (Optional, ForceNew), `subnet_uri_refs` (list, Required, ForceNew), `securitygroup_uri_refs` (list, Required, ForceNew); `settings` block: `flavor_name` (Required, ForceNew), `key_pair_uri_ref` (Optional, ForceNew), `user_data` (Optional, Sensitive, write-only, ForceNew); `storage` block: `boot_volume_uri_ref` (Required, ForceNew) |
| Import format | `<project_id>/<cloudserver_id>` |
| Create | Async — WaitUntilReady() |
| Read | Sync; resumes wait if in transitional state |
| Update | Only `timeout` field is mutable via API |
| Delete | Async — WaitForResourceDeleted() |
| Async behavior | Create + Delete |
| ForceNew fields | Almost all (location, zone, project_id, tags, all network/settings/storage fields) |
| Computed fields | `id`, `uri` |
| Sensitive fields | `user_data` (write-only, never returned by API) |
| Crossplane issues | Nearly fully immutable after creation; most spec changes → replacement |

#### `arubacloud_keypair`

| Property | Value |
|---|---|
| Schema fields | `name` (Required, ForceNew), `location` (Required, ForceNew), `project_id` (Required, ForceNew), `value` (Required, ForceNew, Sensitive — public key), `tags` (Optional), `id` (Computed), `uri` (Computed), `timeout` (Optional) |
| Import format | `<project_id>/<keypair_id>` |
| Create | Async (short) |
| Update | Only `tags` mutable |
| Delete | Async (polling) |
| ForceNew fields | name, location, project_id, value |
| Sensitive fields | `value` (write-only, public key — API never returns it) |
| Crossplane issues | `value` must be preserved from state after import (never readable from API) |

---

### 3.3 Networking

#### `arubacloud_vpc`

| Property | Value |
|---|---|
| Schema fields | `name` (Required), `location` (Required, ForceNew), `project_id` (Required, ForceNew), `tags` (Optional), `id` (Computed), `uri` (Computed), `timeout` (Optional) |
| Import format | `<project_id>/<vpc_id>` |
| Create | Async — WaitUntilReady() |
| Update | name, tags |
| Delete | Async — WaitForResourceDeleted() |
| Crossplane issues | None |

#### `arubacloud_subnet`

| Property | Value |
|---|---|
| Schema fields | `name` (Required), `location` (Required, ForceNew), `project_id` (Required, ForceNew), `vpc_id` (Required, ForceNew), `type` (Required, ForceNew — "Basic" or "Advanced"), `tags` (Optional), `network` block (Required for Advanced): `address` (CIDR), `dhcp` (enabled, range, routes, dns) |
| Import format | `<project_id>/<vpc_id>/<subnet_id>` |
| Create | Async |
| Update | name, tags, DHCP settings |
| Delete | Async |
| ForceNew fields | location, project_id, vpc_id, type |
| Crossplane issues | **3-part import ID requires custom GetIDFn** |

#### `arubacloud_securitygroup`

| Property | Value |
|---|---|
| Schema fields | `name` (Required), `location` (Required, ForceNew), `project_id` (Required, ForceNew), `vpc_id` (Required, ForceNew), `tags` (Optional), `id` (Computed), `uri` (Computed), `timeout` (Optional) |
| Import format | `<project_id>/<vpc_id>/<sg_id>` |
| Create | Async |
| Update | name, tags |
| Delete | Async |
| Crossplane issues | **3-part import ID requires custom GetIDFn** |

#### `arubacloud_securityrule`

| Property | Value |
|---|---|
| Schema fields | `name` (Required), `location` (Required, ForceNew), `project_id` (Required, ForceNew), `vpc_id` (Required, ForceNew), `security_group_id` (Required, ForceNew), `tags` (Optional), `properties` block: `direction` (Ingress/Egress), `protocol` (TCP/UDP/ICMP/ANY), `port` (Optional), `target` (kind: IP/SecurityGroup, value: CIDR or URI) |
| Import format | `<project_id>/<vpc_id>/<sg_id>/<rule_id>` (or 5-part with location) |
| Create | Async |
| Update | name, tags only |
| Delete | Async |
| ForceNew fields | location, project_id, vpc_id, security_group_id, properties |
| Crossplane issues | **4-part import ID requires custom GetIDFn** |

#### `arubacloud_elasticip`

| Property | Value |
|---|---|
| Schema fields | `name` (Required), `location` (Required, ForceNew), `project_id` (Required, ForceNew), `billing_period` (Optional, Computed — Hour/Month/Year), `address` (Computed — assigned public IP), `tags` (Optional), `id` (Computed), `uri` (Computed), `timeout` (Optional) |
| Import format | `<project_id>/<eip_id>` |
| Create | Async |
| Update | name, tags, billing_period |
| Delete | Async |
| Computed fields | `id`, `uri`, `address`, `billing_period` |
| Connection detail | **`address` → expose as connection detail (public IP)** |

#### `arubacloud_vpcpeering`

| Property | Value |
|---|---|
| Schema fields | `name` (Required), `location` (Required, ForceNew), `project_id` (Required, ForceNew), `vpc_id` (Required, ForceNew), `peer_vpc` (Required, ForceNew — ID or URI), `tags` (Optional), `id` (Computed), `uri` (Computed), `timeout` (Optional) |
| Import format | `<project_id>/<vpc_id>/<peering_id>` |
| Crossplane issues | **3-part import ID; peer_vpc can be ID or URI — normalization required** |

#### `arubacloud_vpcpeeringroute`

| Property | Value |
|---|---|
| Schema fields | REQUIRES VERIFICATION (not directly inspected) |
| Import format | REQUIRES VERIFICATION |
| Crossplane issues | Likely 3-4 part composite ID |

#### `arubacloud_vpntunnel`

| Property | Value |
|---|---|
| Schema fields | `name`, `location` (ForceNew), `project_id` (ForceNew), `tags`, `properties` block: `vpn_type` (Site-To-Site), `vpn_client_protocol` (ikev2), `billing_period`, `ip_configurations` (vpc_uri_ref, subnet_uri_ref, public_ip_uri_ref), `vpn_client_settings` (encryption, hashing, DH group, `pre_shared_key`) |
| Import format | `<project_id>/<tunnel_id>` |
| Sensitive fields | `vpn_client_settings.pre_shared_key` — REQUIRES VERIFICATION on whether marked Sensitive |
| Update | name, tags only (properties immutable) |

#### `arubacloud_vpnroute`

| Property | Value |
|---|---|
| Schema fields | REQUIRES VERIFICATION |
| Import format | REQUIRES VERIFICATION |

---

### 3.4 Storage

#### `arubacloud_blockstorage`

| Property | Value |
|---|---|
| Schema fields | `name`, `project_id` (ForceNew), `location` (ForceNew), `size_gb`, `billing_period`, `zone` (Optional), `type` (ForceNew — Standard/Performance), `bootable` (Optional), `image` (Optional), `tags`, `id` (Computed), `uri` (Computed), `timeout` |
| Import format | `<project_id>/<vol_id>` |
| Create | Async |
| Update | name, tags, size_gb (only when status = Used or NotUsed), billing_period |
| Delete | Async; pre-deletes snapshots as API workaround |
| ForceNew fields | project_id, location, type |
| Crossplane issues | Volume must be in specific states to resize; Crossplane must handle update errors |

#### `arubacloud_snapshot`

| Property | Value |
|---|---|
| Schema fields | `name`, `project_id` (ForceNew), `location` (ForceNew), `volume_uri` (ForceNew), `billing_period` (ForceNew), `tags`, `id` (Computed), `uri` (Computed), `timeout` |
| Import format | `<project_id>/<snapshot_id>/<billing_period>` |
| Crossplane issues | **Unusual 3-part import ID where billing_period is part of Terraform ID; custom GetIDFn required** |

#### `arubacloud_backup`

| Property | Value |
|---|---|
| Schema fields | `name`, `project_id` (ForceNew), `location` (ForceNew), `type` (ForceNew), `volume_id` (ForceNew), `retention_days` (ForceNew), `billing_period` (ForceNew), `tags`, `id` (Computed), `uri` (Computed), `timeout` |
| Import format | `<project_id>/<backup_id>` |
| Classification | **Declarative — manages persistent backup configuration** |
| Create | Async (polls for volume visibility first) |
| Update | name, tags, billing_period |
| Delete | Async |

---

### 3.5 Containers

#### `arubacloud_kaas`

| Property | Value |
|---|---|
| Schema fields | `name`, `location` (ForceNew), `zone` (ForceNew? — REQUIRES VERIFICATION), `project_id` (ForceNew), `tags`, `billing_period`, `network` block (all ForceNew): `vpc_uri_ref`, `subnet_uri_ref`, `node_cidr`, `security_group_name` (by name, not URI!), `pod_cidr`, `settings` block: `kubernetes_version`, `node_pools` (list, with autoscaling), `ha` (bool); Computed: `management_ip`, `kubeconfig` (Sensitive) |
| Import format | `<project_id>/<kaas_id>` |
| Create | **Very long async** — can take 10–30 minutes |
| Update | name, tags, kubernetes_version, node_pools, billing_period |
| Delete | Async |
| Sensitive fields | `kubeconfig` (Computed) |
| Connection details | `kubeconfig`, `management_ip` |
| Crossplane issues | Long creation time requires extended provider timeout; `security_group_name` is by name (unusual) |

#### `arubacloud_containerregistry`

| Property | Value |
|---|---|
| Schema fields | `name`, `location` (ForceNew), `project_id` (ForceNew), `tags`, `billing_period`, `network` block: `public_ip_uri_ref`, `vpc_uri_ref`, `subnet_uri_ref`, `security_group_uri_ref`; `storage` block: `block_storage_uri_ref`; `settings` block: `admin_user`, `concurrent_users_flavor` |
| Import format | `<project_id>/<reg_id>` |
| Create | Async |
| Update | name, tags, billing_period, settings |
| Crossplane issues | Many network references; high reference configuration work |

---

### 3.6 Databases

#### `arubacloud_dbaas`

| Property | Value |
|---|---|
| Schema fields | `name`, `location` (ForceNew), `zone` (ForceNew), `project_id` (ForceNew), `engine_id` (ForceNew), `flavor` (mutable), `storage` block (size mutable, autoscaling configurable), `network` block (all ForceNew): `vpc_uri_ref`, `subnet_uri_ref`, `security_group_uri_ref`, `elastic_ip_uri_ref` (Optional), `billing_period`, `tags`, `id` (Computed), `uri` (Computed) |
| Import format | `<project_id>/<dbaas_id>` |
| Create | Async — WaitUntilReady() |
| Update | name, tags, flavor, storage.size, billing_period |
| Delete | Async |
| Crossplane issues | Many network references |

#### `arubacloud_database`

| Property | Value |
|---|---|
| Schema fields | `id` (Computed — equals database name), `uri` (Computed), `project_id` (ForceNew), `dbaas_id` (ForceNew), `name` (ForceNew — but Update code applies name changes; schema vs. code inconsistency) |
| Import format | `<project_id>/<dbaas_id>/<db_id>` |
| Create | Polls `Get()` until accessible (no status field in API) |
| Delete | Async |
| Crossplane issues | **3-part import ID; no status field (custom polling); schema/code inconsistency on name mutability** |

#### `arubacloud_dbaasuser`

| Property | Value |
|---|---|
| Schema fields | `id` (Computed), `uri` (Computed), `project_id` (ForceNew), `dbaas_id` (ForceNew), `username` (ForceNew), `password` (ForceNew, Sensitive) |
| Import format | `<project_id>/<dbaas_id>/<user_id>` |
| Create | Polls API readiness with 30 retries, 5 s intervals; password base64-encoded before sending |
| Update | None — all fields ForceNew |
| Sensitive fields | `password` (write-only, never returned by API) |
| Crossplane issues | **3-part import ID; password must come from Secret; never returned by API — must store in connection secret after create** |

#### `arubacloud_databasegrant`

| Property | Value |
|---|---|
| Schema fields | REQUIRES VERIFICATION |
| Import format | REQUIRES VERIFICATION |

#### `arubacloud_databasebackup`

| Property | Value |
|---|---|
| Schema fields | REQUIRES VERIFICATION |
| Import format | REQUIRES VERIFICATION |
| Classification | Likely declarative backup schedule/configuration |

---

### 3.7 Security

#### `arubacloud_kms`

| Property | Value |
|---|---|
| Schema fields | `name` (Required), `project_id` (Required), `location` (Optional, Computed), `tags` (Optional), `billing_period` (Optional, Computed), `id` (Computed), `uri` (Computed), `timeout` (Optional) |
| Import format | `<project_id>/<kms_id>` |
| Create | Async |
| Update | name, tags, billing_period |
| Delete | Async |
| Sensitive fields | None |
| Crossplane issues | None |

---

### 3.8 Scheduling

#### `arubacloud_schedulejob`

| Property | Value |
|---|---|
| Schema fields | `name`, `project_id`, `location`, `tags`, `timeout`, `properties` block: `enabled` (bool), `schedule_job_type` (OneShot or Recurring), `schedule_at` (ISO 8601), `execute_until` (ISO 8601), `cron` (5-field), `steps` (list: name, resource_uri, action_uri, http_verb, body) |
| Import format | `<project_id>/<job_id>` |
| Update | name, tags, enabled only (properties immutable) |
| Crossplane issues | **OneShot jobs are imperative; after execution Crossplane must not recreate them. Recurring jobs are safe.** |

---

### 3.9 Imperative Resources

#### `arubacloud_restore`

| Property | Value |
|---|---|
| Schema fields | `name`, `location` (ForceNew), `project_id` (ForceNew), `backup_id` (ForceNew), `volume_id` (ForceNew), `tags`, `id` (Computed), `uri` (Computed), `timeout` |
| Import format | `<project_id>/<restore_id>` |
| Classification | **Imperative one-shot operation — triggers data restore** |
| Crossplane issues | **Repeated reconciliation is NOT safe; re-creation overwrites data** |

---

## 4. Data Source Inventory

See `docs/resource-matrix.md` for the full data source analysis.

**Summary**: No data source should be converted to a managed resource. Crossplane's selector mechanism (`matchLabels`) replaces data source functionality for resource discovery.

---

## 5. Terraform → Crossplane Mapping

| Terraform Resource | Crossplane Kind | API Group | API Version |
|---|---|---|---|
| `arubacloud_project` | `Project` | `arubacloud.crossplane.io` | `v1alpha1` |
| `arubacloud_cloudserver` | `CloudServer` | `arubacloud.crossplane.io` | `v1alpha1` |
| `arubacloud_keypair` | `KeyPair` | `arubacloud.crossplane.io` | `v1alpha1` |
| `arubacloud_elasticip` | `ElasticIP` | `arubacloud.crossplane.io` | `v1alpha1` |
| `arubacloud_blockstorage` | `BlockStorage` | `arubacloud.crossplane.io` | `v1alpha1` |
| `arubacloud_snapshot` | `Snapshot` | `arubacloud.crossplane.io` | `v1alpha1` |
| `arubacloud_backup` | `Backup` | `arubacloud.crossplane.io` | `v1alpha1` |
| `arubacloud_restore` | `Restore` | `arubacloud.crossplane.io` | `v1alpha1` |
| `arubacloud_vpc` | `VPC` | `arubacloud.crossplane.io` | `v1alpha1` |
| `arubacloud_subnet` | `Subnet` | `arubacloud.crossplane.io` | `v1alpha1` |
| `arubacloud_securitygroup` | `SecurityGroup` | `arubacloud.crossplane.io` | `v1alpha1` |
| `arubacloud_securityrule` | `SecurityRule` | `arubacloud.crossplane.io` | `v1alpha1` |
| `arubacloud_vpcpeering` | `VPCPeering` | `arubacloud.crossplane.io` | `v1alpha1` |
| `arubacloud_vpcpeeringroute` | `VPCPeeringRoute` | `arubacloud.crossplane.io` | `v1alpha1` |
| `arubacloud_vpntunnel` | `VPNTunnel` | `arubacloud.crossplane.io` | `v1alpha1` |
| `arubacloud_vpnroute` | `VPNRoute` | `arubacloud.crossplane.io` | `v1alpha1` |
| `arubacloud_kaas` | `KaaS` | `arubacloud.crossplane.io` | `v1alpha1` |
| `arubacloud_containerregistry` | `ContainerRegistry` | `arubacloud.crossplane.io` | `v1alpha1` |
| `arubacloud_dbaas` | `DBaaS` | `arubacloud.crossplane.io` | `v1alpha1` |
| `arubacloud_database` | `Database` | `arubacloud.crossplane.io` | `v1alpha1` |
| `arubacloud_databasegrant` | `DatabaseGrant` | `arubacloud.crossplane.io` | `v1alpha1` |
| `arubacloud_databasebackup` | `DatabaseBackup` | `arubacloud.crossplane.io` | `v1alpha1` |
| `arubacloud_dbaasuser` | `DBaaSUser` | `arubacloud.crossplane.io` | `v1alpha1` |
| `arubacloud_schedulejob` | `ScheduleJob` | `arubacloud.crossplane.io` | `v1alpha1` |
| `arubacloud_kms` | `KMS` | `arubacloud.crossplane.io` | `v1alpha1` |

---

## 6. External Name Strategy

**Global rule**: All ArubaCloud resource IDs are provider-generated (not user-assigned). All resources use `config.IdentifierFromProvider`.

The `crossplane.io/external-name` annotation holds only the **leaf resource ID** (not the composite Terraform import path).

Parent IDs needed for Terraform's import are reconstructed at reconcile time from the resolved references in `spec.forProvider`.

### Simple 2-Part Resources

```go
// config/external_name.go
ExternalNameConfigs = map[string]config.ExternalName{
    "arubacloud_project":          config.IdentifierFromProvider,
    "arubacloud_cloudserver":      config.IdentifierFromProvider,
    "arubacloud_vpc":              config.IdentifierFromProvider,
    "arubacloud_keypair":          config.IdentifierFromProvider,
    "arubacloud_elasticip":        config.IdentifierFromProvider,
    "arubacloud_blockstorage":     config.IdentifierFromProvider,
    "arubacloud_backup":           config.IdentifierFromProvider,
    "arubacloud_restore":          config.IdentifierFromProvider,
    "arubacloud_kaas":             config.IdentifierFromProvider,
    "arubacloud_containerregistry":config.IdentifierFromProvider,
    "arubacloud_dbaas":            config.IdentifierFromProvider,
    "arubacloud_vpntunnel":        config.IdentifierFromProvider,
    "arubacloud_vpnroute":         config.IdentifierFromProvider,
    "arubacloud_schedulejob":      config.IdentifierFromProvider,
    "arubacloud_kms":              config.IdentifierFromProvider,
}
```

### Composite-ID Resources

These require custom `GetExternalNameFn` (extract leaf ID from composite) and `GetIDFn` (reconstruct composite from context):

| Resource | Composite Format | Parent IDs (from references) |
|---|---|---|
| `arubacloud_subnet` | `project_id/vpc_id/subnet_id` | vpc_id from VPCRef |
| `arubacloud_securitygroup` | `project_id/vpc_id/sg_id` | vpc_id from VPCRef |
| `arubacloud_securityrule` | `project_id/vpc_id/sg_id/rule_id` | vpc_id + sg_id from SG Ref |
| `arubacloud_vpcpeering` | `project_id/vpc_id/peering_id` | vpc_id from VPCRef |
| `arubacloud_vpcpeeringroute` | REQUIRES VERIFICATION | — |
| `arubacloud_snapshot` | `project_id/snapshot_id/billing_period` | billing_period from spec |
| `arubacloud_database` | `project_id/dbaas_id/db_id` | dbaas_id from DBaaSRef |
| `arubacloud_dbaasuser` | `project_id/dbaas_id/user_id` | dbaas_id from DBaaSRef |
| `arubacloud_databasegrant` | REQUIRES VERIFICATION | — |
| `arubacloud_databasebackup` | REQUIRES VERIFICATION | — |

**Example custom implementation (subnet)**:

```go
"arubacloud_subnet": {
    GetExternalNameFn: func(tfstate map[string]any) (string, error) {
        // Terraform ID format: project_id/vpc_id/subnet_id
        // Extract only the subnet_id (last segment)
        id, _ := tfstate["id"].(string)
        parts := strings.Split(id, "/")
        if len(parts) != 3 {
            return "", fmt.Errorf("unexpected subnet ID format: %s", id)
        }
        return parts[2], nil
    },
    GetIDFn: func(ctx context.Context, externalName string, parameters map[string]any, terraformProviderConfig map[string]any) (string, error) {
        projectID, _ := parameters["project_id"].(string)
        vpcID, _ := parameters["vpc_id"].(string)
        return projectID + "/" + vpcID + "/" + externalName, nil
    },
    DisableNameInitializer: true,
},
```

---

## 7. Reference Strategy

ArubaCloud resources use URI references (`*_uri_ref`) between resources. In Crossplane these should be Kubernetes-native references that resolve to URIs at reconcile time.

**Pattern**: Reference resolves the external name of the referenced resource, then uses `status.atProvider.uri` to get the ArubaCloud URI.

### Reference Configuration

Using Upjet's reference system:

```go
// config/provider.go
pc.AddResourceConfigurator("arubacloud_cloudserver", func(r *config.Resource) {
    r.References["project_id"] = config.Reference{
        TerraformName: "arubacloud_project",
    }
    r.References["network.vpc_uri_ref"] = config.Reference{
        TerraformName: "arubacloud_vpc",
        Extractor:     `github.com/crossplane/upjet/pkg/resource.ExtractParamPath("uri", true)`,
    }
    r.References["network.subnet_uri_refs"] = config.Reference{
        TerraformName: "arubacloud_subnet",
        Extractor:     `github.com/crossplane/upjet/pkg/resource.ExtractParamPath("uri", true)`,
    }
    r.References["network.securitygroup_uri_refs"] = config.Reference{
        TerraformName: "arubacloud_securitygroup",
        Extractor:     `github.com/crossplane/upjet/pkg/resource.ExtractParamPath("uri", true)`,
    }
    r.References["network.elastic_ip_uri_ref"] = config.Reference{
        TerraformName: "arubacloud_elasticip",
        Extractor:     `github.com/crossplane/upjet/pkg/resource.ExtractParamPath("uri", true)`,
    }
    r.References["settings.key_pair_uri_ref"] = config.Reference{
        TerraformName: "arubacloud_keypair",
        Extractor:     `github.com/crossplane/upjet/pkg/resource.ExtractParamPath("uri", true)`,
    }
    r.References["storage.boot_volume_uri_ref"] = config.Reference{
        TerraformName: "arubacloud_blockstorage",
        Extractor:     `github.com/crossplane/upjet/pkg/resource.ExtractParamPath("uri", true)`,
    }
})
```

**Note on URI extraction**: Upjet's `ExtractParamPath` reads from `status.atProvider`. For fields that represent URIs (not IDs), the extractor should read `atProvider.uri`. If Upjet's built-in extractors are insufficient, a custom extractor function will be needed.

**REQUIRES VERIFICATION**: Confirm that `status.atProvider.uri` is populated for all resources and available for cross-resource extraction during reconcile.

### Full Reference Map

```
Project:
  (no incoming references in schema)

VPC:
  project_id → Project

Subnet:
  project_id → Project
  vpc_id → VPC

SecurityGroup:
  project_id → Project
  vpc_id → VPC

SecurityRule:
  project_id → Project
  vpc_id → VPC
  security_group_id → SecurityGroup

ElasticIP:
  project_id → Project

VPCPeering:
  project_id → Project
  vpc_id → VPC
  peer_vpc → VPC (optional cross-project reference)

VPNTunnel:
  project_id → Project
  properties.ip_configurations.vpc_uri_ref → VPC
  properties.ip_configurations.subnet_uri_ref → Subnet
  properties.ip_configurations.public_ip_uri_ref → ElasticIP

CloudServer:
  project_id → Project
  network.vpc_uri_ref → VPC
  network.subnet_uri_refs[] → Subnet (list)
  network.securitygroup_uri_refs[] → SecurityGroup (list)
  network.elastic_ip_uri_ref → ElasticIP (optional)
  settings.key_pair_uri_ref → KeyPair (optional)
  storage.boot_volume_uri_ref → BlockStorage

BlockStorage:
  project_id → Project

Snapshot:
  project_id → Project
  volume_uri → BlockStorage

Backup:
  project_id → Project
  volume_id → BlockStorage

Restore:
  project_id → Project
  backup_id → Backup
  volume_id → BlockStorage

KaaS:
  project_id → Project
  network.vpc_uri_ref → VPC
  network.subnet_uri_ref → Subnet
  network.security_group_name → SecurityGroup (by name — CUSTOM)

ContainerRegistry:
  project_id → Project
  network.public_ip_uri_ref → ElasticIP
  network.vpc_uri_ref → VPC
  network.subnet_uri_ref → Subnet
  network.security_group_uri_ref → SecurityGroup
  storage.block_storage_uri_ref → BlockStorage

DBaaS:
  project_id → Project
  network.vpc_uri_ref → VPC
  network.subnet_uri_ref → Subnet
  network.security_group_uri_ref → SecurityGroup
  network.elastic_ip_uri_ref → ElasticIP (optional)

Database:
  project_id → Project
  dbaas_id → DBaaS

DBaaSUser:
  project_id → Project
  dbaas_id → DBaaS

DatabaseGrant:
  project_id → Project
  dbaas_id → DBaaS
  (Database and User references — REQUIRES VERIFICATION)

DatabaseBackup:
  project_id → Project
  dbaas_id → DBaaS
  (Database reference — REQUIRES VERIFICATION)

KMS:
  project_id → Project

ScheduleJob:
  project_id → Project
```

### Special Case: KaaS `security_group_name`

KaaS references the SecurityGroup by **name** (`security_group_name: string`), not by URI. This is the only resource with a name-based reference.

**Strategy**:
1. Add a custom reference: `network.security_group_name` → SecurityGroup, using name-based extraction instead of URI extraction.
2. Custom extractor reads `spec.forProvider.name` of the referenced SecurityGroup.

---

## 8. Selector Strategy

Every reference field will have a corresponding selector field generated by Upjet:

```yaml
spec:
  forProvider:
    # Option 1: direct name reference
    vpcRef:
      name: main-vpc

    # Option 2: label selector (discovers any VPC matching labels)
    vpcSelector:
      matchLabels:
        environment: production
        team: platform
```

Selectors are automatically generated by Upjet when references are configured. No additional configuration is needed beyond the reference configuration in section 7.

**Recommended label convention** for ArubaCloud resources:

```yaml
metadata:
  labels:
    arubacloud.crossplane.io/project: my-project
    arubacloud.crossplane.io/location: ITBG-Bergamo
    arubacloud.crossplane.io/environment: production
```

---

## 9. ProviderConfig Strategy

### Cluster-Scoped ProviderConfig

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

### Credentials Secret

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: arubacloud-credentials
  namespace: crossplane-system
type: Opaque
stringData:
  credentials: |
    {
      "client_id": "your-client-id",
      "client_secret": "your-client-secret",
      "base_url": "",
      "token_issuer_url": "",
      "resource_timeout": "30m"
    }
```

Fields `base_url`, `token_issuer_url`, `resource_timeout` are optional. Leave empty to use defaults.

### internal/clients/arubacloud.go

This file implements the `TerraformSetupBuilder` function that reads credentials from the ProviderConfig secret and builds the Terraform provider configuration map:

```go
func TerraformSetupBuilder(version, providerSource, providerVersion string) terraform.SetupFn {
    return func(ctx context.Context, client client.Client, mg resource.Managed) (terraform.Setup, error) {
        pc := &v1beta1.ProviderConfig{}
        // ... load ProviderConfig
        // ... read Secret
        // ... parse JSON credentials
        // ... return Setup with ps.Configuration map:
        // {
        //   "client_id": ...,
        //   "client_secret": ...,
        //   "base_url": ...,
        //   "token_issuer_url": ...,
        //   "resource_timeout": ...,
        // }
    }
}
```

### Multiple ProviderConfigs

Each managed resource can reference a different ProviderConfig via `spec.providerConfigRef.name`. This enables multi-project setups:

```yaml
# Resource in project A
spec:
  providerConfigRef:
    name: project-a-config

# Resource in project B
spec:
  providerConfigRef:
    name: project-b-config
```

However, `project_id` is a field in every managed resource schema. The ProviderConfig does **not** carry a project-level default. Users must specify `project_id` (or a `projectRef`) in every resource.

---

## 10. Secret Strategy

### Sensitive Input Fields

| Resource | Field | Secret Strategy |
|---|---|---|
| ProviderConfig | `client_secret` | JSON credential in Kubernetes Secret |
| `keypair` | `value` (public key) | Provided inline in spec (not sensitive, but write-only) |
| `dbaasuser` | `password` | **Secret reference in spec**; store as connection detail |
| `vpntunnel` | `pre_shared_key` | Inline in spec (Sensitive field); consider Secret reference |

### Connection Details (Sensitive Outputs)

| Resource | Connection Detail Keys |
|---|---|
| `elasticip` | `address` (public IP) |
| `kaas` | `kubeconfig`, `management_ip` |
| `dbaasuser` | `password` (if surfaced after create) |

**KaaS kubeconfig implementation**:

```go
pc.AddResourceConfigurator("arubacloud_kaas", func(r *config.Resource) {
    r.Sensitive.AdditionalConnectionDetailsFn = func(attr map[string]any) (map[string][]byte, error) {
        conn := map[string][]byte{}
        if kubeconfig, ok := attr["kubeconfig"].(string); ok && kubeconfig != "" {
            conn["kubeconfig"] = []byte(kubeconfig)
        }
        if mgmtIP, ok := attr["management_ip"].(string); ok && mgmtIP != "" {
            conn["management_ip"] = []byte(mgmtIP)
        }
        return conn, nil
    }
})
```

Users consume the kubeconfig:

```yaml
apiVersion: arubacloud.crossplane.io/v1alpha1
kind: KaaS
spec:
  writeConnectionSecretToRef:
    name: my-cluster-kubeconfig
    namespace: default
```

---

## 11. Async Strategy

All ArubaCloud resources are async. Since Upjet delegates to the Terraform provider at runtime, and the Terraform provider's `Create`/`Delete` functions internally call `WaitUntilReady()` / `WaitForResourceDeleted()` before returning, **Upjet's reconciler sees synchronous operations**.

There is no need to configure Upjet-level async handling for these resources. The polling is handled inside the Terraform provider binary.

**Timeout configuration**:

The Terraform provider's `resource_timeout` (default: 30 minutes) controls how long it waits. For KaaS, this must be increased.

Options:
1. Provider-level: set `resource_timeout: 60m` in the ProviderConfig credentials JSON.
2. Per-resource: `timeout` field is Optional in all schemas — users can set it per resource.

**Upjet reconcile timeout**: Set `pollInterval` conservatively (10 minutes default in reference providers) to avoid premature timeout for long-running operations.

---

## 12. Management Policy Strategy

Upjet v2 supports the following management policies:

| Policy | Description |
|---|---|
| `Observe` | Read and report state; no create/update/delete |
| `Create` | Allow creation only |
| `Update` | Allow updates to existing resources |
| `Delete` | Allow deletion |
| `LateInitialize` | Initialize spec fields from observed state |
| `FullControl` (default) | All of the above |

### Policy Recommendations by Resource Type

| Resource Type | Recommended Default | Notes |
|---|---|---|
| Standard resources | `FullControl` | Default |
| `restore` | `Create` + `Observe` | After first apply, user should switch to `Observe` |
| `schedulejob` (OneShot) | `Create` + `Observe` | After job fires, should not be recreated |
| `schedulejob` (Recurring) | `FullControl` | Safe for continuous reconciliation |
| Externally managed | `Observe` | For adoption of existing ArubaCloud resources |

### Import / Adoption Workflow

```yaml
# Step 1: Create the Crossplane resource with external name annotation
apiVersion: arubacloud.crossplane.io/v1alpha1
kind: VPC
metadata:
  name: existing-vpc
  annotations:
    crossplane.io/external-name: vpc-abc123   # existing ArubaCloud VPC ID
spec:
  managementPolicies:
    - Observe                                 # read-only initially
  forProvider:
    name: existing-vpc
    location: ITBG-Bergamo
    projectRef:
      name: my-project
  providerConfigRef:
    name: default

# Step 2: Verify status.atProvider is populated
# Step 3: Optionally enable full management:
#   managementPolicies: [Observe, Create, Update, Delete]
```

---

## 13. Import Strategy

Crossplane uses `crossplane.io/external-name` annotation to identify existing resources.

**Workflow for every resource**:

1. User creates Crossplane managed resource YAML.
2. User sets `crossplane.io/external-name: <leaf-resource-id>`.
3. User sets `managementPolicies: [Observe]` initially.
4. User applies the resource.
5. Crossplane calls the Upjet reconciler.
6. Upjet constructs the Terraform import ID using the configured `GetIDFn`.
7. Terraform imports the resource and populates state.
8. `status.atProvider` is populated.
9. User verifies and optionally enables full management.

**Example: importing an existing CloudServer**:

```yaml
apiVersion: arubacloud.crossplane.io/v1alpha1
kind: CloudServer
metadata:
  name: imported-server
  annotations:
    crossplane.io/external-name: srv-xyz123
spec:
  managementPolicies:
    - Observe
  forProvider:
    location: ITBG-Bergamo
    projectRef:
      name: my-project
    network:
      vpcRef:
        name: main-vpc
      subnetRefs:
        - name: private-subnet
      securityGroupRefs:
        - name: web-sg
    settings:
      flavorName: CSO4A8
    storage:
      bootVolumeRef:
        name: boot-vol
  providerConfigRef:
    name: default
```

---

## 14. KaaS Strategy

KaaS is the most complex resource in the provider.

### Key Design Decisions

1. **Timeout**: KaaS creation can take 10–30 minutes. Provider `resource_timeout` must be set to at least 60 minutes for KaaS resources.

2. **kubeconfig**: Exposed as a Crossplane connection detail stored in a Kubernetes Secret. Users reference it via `spec.writeConnectionSecretToRef`.

3. **security_group_name**: This field references a SecurityGroup by name, not URI. Custom reference configuration needed. The recommended approach:
   - Add `SecurityGroupRef` / `SecurityGroupSelector` to the spec.
   - Custom extractor reads the `.metadata.name` of the referenced SecurityGroup (not the URI).
   - **ASSUMPTION**: The ArubaCloud API uses the security group name directly (not URI). REQUIRES VERIFICATION.

4. **Node pools**: Mutable after creation. Upjet will detect spec changes and trigger updates.

5. **Kubernetes version upgrades**: Supported. Controller-triggered via spec change.

6. **pod_cidr**: The Terraform provider explicitly preserves this as user-controlled (never overwritten by API). This means late initialization must NOT initialize `pod_cidr` from API response.

### Connection Details

```yaml
# KaaS connection secret contains:
kubeconfig: <base64-decoded kubeconfig YAML>
management_ip: <management IP string>
```

### Deletion

KaaS deletion is async. The provider polls until the cluster is gone. For production clusters, users should verify the cluster is terminated before the `Delete` policy applies, or use `managementPolicies: [Create, Observe, Update]` to protect against accidental deletion.

---

## 15. Database Strategy

### Dependency Chain

```
Project
  └── DBaaS (database cluster)
        ├── Database (individual database within cluster)
        │     └── DatabaseGrant (grants user access to database)
        │     └── DatabaseBackup (backup schedule for database)
        └── DBaaSUser (database user for the cluster)
```

### Reference Chain

```yaml
kind: DBaaS
metadata:
  name: my-cluster
spec:
  forProvider:
    projectRef: {name: my-project}
    ...
---
kind: Database
metadata:
  name: my-db
spec:
  forProvider:
    projectRef: {name: my-project}
    dbaaSRef: {name: my-cluster}
---
kind: DBaaSUser
metadata:
  name: my-user
spec:
  forProvider:
    projectRef: {name: my-project}
    dbaaSRef: {name: my-cluster}
    username: appuser
    # password from secret:
    passwordSecretRef:
      name: db-credentials
      key: password
---
kind: DatabaseGrant
metadata:
  name: my-user-on-my-db
spec:
  forProvider:
    projectRef: {name: my-project}
    dbaaSRef: {name: my-cluster}
    databaseRef: {name: my-db}
    dbaaSUserRef: {name: my-user}
```

### Special Cases

- **Database**: Has no status field in API. `Read` polls `Get()` for success. Custom async configuration may be needed if Upjet expects a status field.
- **DBaaSUser.password**: Immutable, write-only, Sensitive. Must be stored as connection detail on creation. When user changes password field, Terraform replaces the resource (ForceNew).

---

## 16. Backup / Restore / Snapshot Strategy

### BlockStorage Backup (`arubacloud_backup`)

**Classification**: Declarative. Manages a persistent backup configuration for a volume.

**Crossplane treatment**: Standard managed resource. Safe for continuous reconciliation.

### Snapshot (`arubacloud_snapshot`)

**Classification**: Declarative. Creates and manages a persistent snapshot object.

**Crossplane treatment**: Standard managed resource. Snapshot ID + billing_period form the Terraform import ID — custom `GetIDFn` required.

### Restore (`arubacloud_restore`)

**Classification**: Imperative one-shot operation.

**Crossplane treatment**: Included in the provider, with the following constraints:
- Document prominently that `restore` is not safe for continuous reconciliation.
- After the restore operation completes, users **must** set `managementPolicies: [Observe]` to prevent re-execution.
- Alternatively: exclude from v0.x and add to `docs/limitations.md`.

**Recommended v0.x decision**: Include the resource. Add a `+kubebuilder:validation:XValidation` or webhook that warns when management policy includes `Create` without `Delete` (to discourage re-creation).

### Database Backup (`arubacloud_databasebackup`)

**Classification**: REQUIRES VERIFICATION — likely a declarative backup schedule/configuration.

---

## 17. Testing Strategy

### Unit Tests

Location: `config/*_test.go`, `internal/clients/*_test.go`

Cover:
- External name `GetExternalNameFn` and `GetIDFn` for every composite-ID resource
- Credentials parsing in `internal/clients`
- Sensitive field extraction functions
- Reference configuration correctness

```go
func TestSubnetExternalName(t *testing.T) {
    // Terraform state: project_id/vpc_id/subnet_id
    state := map[string]any{"id": "proj-123/vpc-456/sub-789"}
    name, err := GetExternalNameFn(state)
    require.NoError(t, err)
    require.Equal(t, "sub-789", name)
}

func TestSubnetGetID(t *testing.T) {
    params := map[string]any{
        "project_id": "proj-123",
        "vpc_id": "vpc-456",
    }
    id, err := GetIDFn(ctx, "sub-789", params, nil)
    require.NoError(t, err)
    require.Equal(t, "proj-123/vpc-456/sub-789", id)
}
```

### Generated Code Verification

Run after every `make generate`:

```bash
git diff --exit-code  # fails if generated files changed without being committed
```

Verify:
- CRDs are valid Kubernetes objects (`kubectl apply --dry-run=client`)
- Example YAML files are valid
- All 25 resource types appear in generated CRD list

### Integration Tests (Uptest)

Upjet provides `uptest` for end-to-end testing against real APIs.

Prerequisites: ArubaCloud credentials as environment variables or Kubernetes Secret.

Test scenarios per resource:
1. Create resource
2. Verify `status.conditions[Ready]` becomes `True`
3. Verify `status.atProvider` is populated
4. Update a mutable field
5. Verify update reflected in `status.atProvider`
6. Delete resource
7. Verify resource no longer exists in ArubaCloud

Prioritized order: Project → VPC → Subnet → ElasticIP → KeyPair → BlockStorage → CloudServer

### Acceptance Tests

Run against ArubaCloud API. Require `ARUBACLOUD_CLIENT_ID` and `ARUBACLOUD_CLIENT_SECRET`.

```
make e2e
```

Tests in `cluster/test/` using uptest manifests.

---

## 18. CI/CD Strategy

### GitHub Actions Workflows

#### `.github/workflows/ci.yml`

```yaml
jobs:
  test:
    steps:
      - go test ./...
      - golangci-lint run
      
  generate:
    steps:
      - make generate
      - git diff --exit-code  # fail if stale generated files
      
  build:
    steps:
      - go build ./cmd/provider/...
      - go build ./cmd/generator/...
      
  package:
    steps:
      - make package  # builds .xpkg OCI artifact
      - docker build  # builds provider container image
```

#### `.github/workflows/e2e.yml` (manual or scheduled)

```yaml
on:
  workflow_dispatch:
  schedule:
    - cron: '0 3 * * 1'  # Monday 3am UTC
jobs:
  e2e:
    steps:
      - make e2e
    env:
      ARUBACLOUD_CLIENT_ID: ${{ secrets.ARUBACLOUD_CLIENT_ID }}
      ARUBACLOUD_CLIENT_SECRET: ${{ secrets.ARUBACLOUD_CLIENT_SECRET }}
```

### Stale Generated File Detection

The CI pipeline must fail if generated files are modified without being committed. Pattern from reference providers:

```bash
make generate
git diff --exit-code
```

If this fails, the developer must re-run `make generate` and commit the results.

---

## 19. Documentation Strategy

```
docs/
├── implementation-plan.md       # THIS FILE — Stage 1 deliverable
├── resource-matrix.md           # Resource coverage table
├── architecture.md              # Architecture overview
├── authentication.md            # ProviderConfig + credential setup
├── references.md                # How cross-resource references work
├── external-names.md            # External name configuration details
├── import.md                    # Importing existing resources
├── async-resources.md           # Timeout configuration
├── management-policies.md       # When to use each policy
├── limitations.md               # Known limitations and unsupported resources
└── troubleshooting.md           # Common errors and solutions
```

---

## 20. Implementation Phases

### Phase 0 — Planning (CURRENT) ✓

**Goal**: Understand the problem completely.  
**Output**: `docs/implementation-plan.md`, `docs/resource-matrix.md`, `docs/architecture.md`  
**Status**: COMPLETE (pending approval)

---

### Phase 1 — Bootstrap

**Goal**: Working Upjet provider skeleton that compiles, generates, and authenticates.

**Prerequisites**: Phase 0 approved.

**Steps**:
1. Clone `upjet-provider-template` → new repo `provider-arubacloud`
2. Run `./hack/prepare.sh` with provider name `arubacloud` and module `github.com/arubacloud/provider-arubacloud`
3. Update Makefile variables:
   - `TERRAFORM_PROVIDER_SOURCE = arubacloud/arubacloud`
   - `TERRAFORM_PROVIDER_REPO = github.com/Arubacloud/terraform-provider-arubacloud`
   - `TERRAFORM_PROVIDER_VERSION` = latest tag
   - `TERRAFORM_NATIVE_PROVIDER_BINARY = terraform-provider-arubacloud`
4. Implement `internal/clients/arubacloud.go` (credential loading + TerraformSetupBuilder)
5. Add `config/external_name.go` with `IdentifierFromProvider` for all 25 resources
6. Add `config/provider.go` with `NewProvider()` and no resource-specific config yet
7. Run `make generate` — **VERIFY** schema extraction works with framework provider
8. Fix any generation issues
9. Run `go build ./...`
10. Run `go test ./...`

**Acceptance criteria**:
- `make generate` completes without error
- `go build ./cmd/provider` produces a binary
- CRDs for all 25 resources are generated
- `kubectl apply --dry-run=client -f package/crds/` passes

**Risk**: Terraform framework provider may require special handling for schema extraction. If `make generate` fails at step 7, investigate CLI mode configuration in Upjet.

---

### Phase 2 — Core Infrastructure

**Goal**: First vertical slice. Proves authentication, references, external names, async, CRUD.

**Prerequisites**: Phase 1 complete.

**Resources**: `project`, `vpc`, `subnet`, `keypair`, `elasticip`, `cloudserver`

**Steps**:
1. Implement reference configuration for all Phase 2 resources in `config/provider.go`
2. Implement composite ID `GetIDFn` / `GetExternalNameFn` for `subnet` (3-part)
3. Add connection detail for `elasticip` (`address`)
4. Run `make generate`
5. Write unit tests for external name functions
6. Write example YAMLs: project, vpc, subnet, keypair, elasticip, cloudserver
7. Deploy to test cluster with Crossplane installed
8. Test: create project, VPC, subnet, key pair, elastic IP
9. Test: create CloudServer with all references
10. Test: read, update (tags), delete lifecycle
11. Test: import existing CloudServer using `crossplane.io/external-name`

**Acceptance criteria**:
- CloudServer lifecycle (create → ready → update → delete) works end-to-end
- References resolve correctly (VPCRef → vpc_uri_ref in Terraform config)
- External name annotation correctly identifies the resource
- `status.atProvider` is populated with `id` and `uri`
- Connection details not applicable here (no sensitive outputs in Phase 2)

---

### Phase 3 — Storage

**Goal**: Block storage resources.

**Prerequisites**: Phase 2 complete.

**Resources**: `blockstorage`, `snapshot`, `backup`

**Steps**:
1. Reference config: `blockstorage` → Project
2. Reference config: `snapshot` → Project, BlockStorage
3. Reference config: `backup` → Project, BlockStorage
4. Custom `GetIDFn` for `snapshot` (3-part with billing_period)
5. Generate, test, examples

**Acceptance criteria**:
- Volume create, resize (update), delete
- Snapshot create, delete
- Backup create, delete

---

### Phase 4 — Networking

**Goal**: Security and VPC connectivity resources.

**Prerequisites**: Phase 2 complete.

**Resources**: `securitygroup`, `securityrule`, `vpcpeering`, `vpcpeeringroute`, `vpntunnel`, `vpnroute`

**Steps**:
1. Reference config for all Phase 4 resources
2. Custom `GetIDFn` for `securitygroup` (3-part), `securityrule` (4-part), `vpcpeering` (3-part)
3. Custom `GetIDFn` for `vpcpeeringroute` (REQUIRES VERIFICATION of import format)
4. Custom `GetIDFn` for `vpnroute` (REQUIRES VERIFICATION)
5. Handle `vpntunnel.pre_shared_key` sensitive field
6. Generate, test, examples

**Acceptance criteria**:
- SecurityGroup + SecurityRule create/delete
- VPCPeering create/delete
- VPNTunnel create/delete

---

### Phase 5 — Containers

**Goal**: KaaS and ContainerRegistry.

**Prerequisites**: Phases 2 + 3 complete.

**Resources**: `kaas`, `containerregistry`

**Steps**:
1. KaaS reference config (VPC, Subnet, custom security_group_name)
2. KaaS connection details (kubeconfig, management_ip)
3. KaaS timeout configuration (extend to 60+ minutes)
4. ContainerRegistry reference config (6 references)
5. Generate, test, examples
6. Test: KaaS create (accept long wait), verify kubeconfig in Secret

**Acceptance criteria**:
- KaaS becomes Ready after creation (may take 30 minutes)
- `status.atProvider.managementIp` populated
- Connection secret contains valid kubeconfig
- ContainerRegistry create/delete with all references

---

### Phase 6 — Databases

**Goal**: Full database stack.

**Prerequisites**: Phase 2 complete.

**Resources**: `dbaas`, `database`, `dbaasuser`, `databasegrant`, `databasebackup`

**Steps**:
1. DBaaS reference config (4 network references)
2. Database reference config: Project, DBaaS (3-part ID)
3. DBaaSUser: Project, DBaaS (3-part ID); connection detail for password
4. DatabaseGrant: REQUIRES VERIFICATION of schema/import
5. DatabaseBackup: REQUIRES VERIFICATION of schema/import
6. Generate, test, examples

**Acceptance criteria**:
- Full chain: DBaaS → Database → DBaaSUser → DatabaseGrant
- DBaaSUser password in connection secret
- References resolve in correct order

---

### Phase 7 — Security and Scheduling

**Goal**: KMS and ScheduleJob.

**Prerequisites**: Phase 2 complete.

**Resources**: `kms`, `schedulejob`

**Steps**:
1. KMS reference config (Project only)
2. ScheduleJob reference config (Project only)
3. Document OneShot job limitation in generated CRD description
4. Generate, test, examples

---

### Phase 8 — Imperative Resources

**Goal**: Handle `restore` with appropriate management policy guidance.

**Prerequisites**: Phase 3 complete.

**Resources**: `restore`

**Steps**:
1. Reference config: Project, Backup, BlockStorage
2. Add prominent documentation in CRD description about OneShot semantics
3. Generate, test
4. Write example with `managementPolicies: [Create, Observe]`

---

### Phase 9 — Advanced Features

**Goal**: Polish, management policies, late initialization, advanced import scenarios.

**Steps**:
1. Audit all resources for late initialization candidates
2. Test import workflow for each resource group
3. Test Observe-only mode for each resource group
4. Test management policy transitions
5. Verify connection secrets for all sensitive resources

---

### Phase 10 — Quality

**Goal**: Production-ready provider.

**Steps**:
1. Complete unit test coverage (external names, references)
2. Complete uptest e2e tests for all resources
3. Documentation: all `docs/` files
4. Example YAMLs: all resources
5. golangci-lint clean
6. Make generate produces clean diff
7. Package `.xpkg` and test install in fresh Crossplane cluster
8. CI workflows: all checks pass

---

## 21. Phase Dependencies

```
Phase 0 (Planning)
  └── Phase 1 (Bootstrap)
        ├── Phase 2 (Core: Project + VPC + CloudServer)
        │     ├── Phase 3 (Storage)
        │     │     └── Phase 8 (Restore)
        │     ├── Phase 4 (Networking)
        │     ├── Phase 5 (Containers)
        │     │     (needs Phase 3 for BlockStorage ref in ContainerRegistry)
        │     ├── Phase 6 (Databases)
        │     ├── Phase 7 (KMS + ScheduleJob)
        │     └── Phase 9 (Advanced)
        │           └── Phase 10 (Quality)
```

Phases 3–7 are independent of each other after Phase 2 and can be parallelized.

---

## 22. Technical Risk Assessment

| Risk | Probability | Impact | Mitigation |
|---|---|---|---|
| Framework provider schema extraction fails | Medium | **High** | Verify in Phase 1 Step 7; fallback: generate schema.json manually via `terraform providers schema` |
| URI reference extraction from `atProvider.uri` unavailable | Medium | **High** | Verify early in Phase 2; fallback: use `atProvider.id` + custom URI constructor |
| Composite ID `GetIDFn` parent ID lookup fails | High | Medium | Implement for every composite-ID resource; unit test thoroughly |
| KaaS 30-minute creation timeout | High | Medium | Set `resource_timeout: 60m`; document in KaaS example |
| `restore` resource unsafe reconciliation | **High** | **High** | Exclude from `FullControl` default; document prominently |
| `dbaasuser` password never returned by API | High | Medium | Store in connection secret on create; write-only field preservation |
| `snapshot` billing_period in import ID | High | Medium | Custom `GetIDFn` includes billing_period from spec |
| `securityrule` 4-part ID complexity | High | Medium | Custom `GetIDFn`; unit test for all ID permutations |
| `database` has no status field | High | Low | Use availability polling in `Read`; document in resource spec |
| OneShot `schedulejob` re-creation | Medium | **High** | Document; recommend `managementPolicies: [Create, Observe]` |
| `kaas.security_group_name` name-vs-URI mismatch | Medium | Medium | Custom reference extractor; verify with ArubaCloud API |
| Crossplane API breaking changes in runtime v2.x | Low | High | Pin crossplane-runtime version; monitor upstream |
| ArubaCloud API rate limiting | Unknown | Medium | Implement exponential backoff (SDK handles this — REQUIRES VERIFICATION) |
| VPCPeeringRoute / VPNRoute import format unknown | Medium | Low | Verify in Phase 4 before implementing |

---

## 23. Known Limitations

1. **`restore` is imperative**: Continuous reconciliation will NOT recreate a completed restore. Users must use `managementPolicies: [Create, Observe]`. See `docs/limitations.md`.

2. **`schedulejob` with OneShot type**: After a OneShot job executes, it should not be recreated. Users must set `managementPolicies: [Create, Observe]` before the job fires.

3. **`keypair.value` is write-only**: The public key is never returned by the ArubaCloud API. After import, Crossplane cannot verify the key value. Users must preserve the original key value in the spec.

4. **`dbaasuser.password` is immutable**: Any password change triggers resource replacement (ForceNew). Plan accordingly.

5. **`cloudserver` is nearly fully immutable**: Almost all fields are ForceNew. Any change except `timeout` replaces the server. Plan your CloudServer spec carefully before creation.

6. **CLI execution mode overhead**: Each reconcile forks a Terraform subprocess. Under high resource count or high reconcile frequency, this creates significant system load. Consider increasing `pollInterval` (default: 10 minutes) for non-critical resources.

7. **`vpcpeeringroute` and `vpnroute` schema**: Not directly inspected during planning. May contain surprises in Phase 4.

8. **`databasegrant` and `databasebackup` schema**: Not directly inspected. Verify in Phase 6.

9. **No provider-level project default**: `project_id` must be specified (or referenced via `projectRef`) in every resource. There is no global project shortcut in ProviderConfig.

10. **`database` name is immutable**: The database `name` is also its `id`. Despite the Update code applying name changes, the schema marks it ForceNew. The schema is authoritative — name changes replace the database.

---

## 24. Definition of Done

The provider is production-ready when:

- [ ] `go build ./...` — clean compile
- [ ] `go test ./...` — all unit tests pass
- [ ] `golangci-lint run` — no lint errors
- [ ] `make generate && git diff --exit-code` — no stale generated files
- [ ] `kubectl apply --dry-run=client -f package/crds/` — all 25 CRDs valid
- [ ] Provider image builds and pushes
- [ ] `.xpkg` builds and installs into Crossplane
- [ ] Provider becomes `Healthy` in Kubernetes
- [ ] `ProviderConfig` with valid credentials becomes `Ready`
- [ ] All Phase 2 resources: full CRUD lifecycle verified against ArubaCloud API
- [ ] All references resolve correctly (`VPCRef` → `vpc_uri_ref` in Terraform config)
- [ ] `crossplane.io/external-name` correctly identifies existing resources
- [ ] Import workflow works for at least one resource per Phase
- [ ] `managementPolicies: [Observe]` works for at least 5 representative resources
- [ ] KaaS kubeconfig appears in connection secret
- [ ] ElasticIP `address` appears in connection secret
- [ ] DBaaSUser password appears in connection secret
- [ ] Async operations complete (no premature timeout for standard resources)
- [ ] KaaS creation succeeds with extended timeout
- [ ] All example YAMLs in `examples/` are valid and apply successfully
- [ ] `docs/` is complete and accurate
- [ ] CI pipeline passes on all PRs

---

## Versioning

| Component | Version |
|---|---|
| This provider | `v0.1.0-alpha.1` (initial) |
| Upjet | `v2.4.1` |
| crossplane-runtime | `v2.3.3` |
| Crossplane | v2.x (compatible with crossplane-runtime v2) |
| ArubaCloud Terraform provider | current `master` |
| ArubaCloud SDK | `v1.0.9` |
| Go | `1.26.5` (matching Upjet requirement) |
| Terraform | latest stable (for `providers schema` extraction) |

---

*End of Stage 1 planning document. STATUS: WAITING FOR APPROVAL.*
