# ArubaCloud Crossplane Provider — Architecture

## 1. Existing Terraform Provider Architecture

```
main.go  (terraform-plugin-framework providerserver)
  └─ internal/provider/provider.go  (ArubaCloudProvider)
       ├─ Schema: client_id, client_secret, resource_timeout, base_url,
       │          token_issuer_url, log_level
       ├─ Configure(): aruba.NewClient(options)  →  OAuth2 token  →  REST API
       ├─ Resources():  25 resource types
       └─ DataSources(): 24 data source types
            └─ Each resource: terraform-plugin-framework schema + CRUD
                 └─ github.com/Arubacloud/sdk-go/pkg/aruba
                      └─ ArubaCloud REST API (JWT bearer auth)
```

### Provider Configuration Fields

| Field | Env Var | Default | Sensitive |
|---|---|---|---|
| `client_id` | `ARUBACLOUD_CLIENT_ID` | — | No |
| `client_secret` | `ARUBACLOUD_CLIENT_SECRET` | — | **Yes** |
| `resource_timeout` | — | `30m` | No |
| `base_url` | — | (ArubaCloud default) | No |
| `token_issuer_url` | `ARUBACLOUD_TOKEN_ISSUER_URL` | (ArubaCloud default) | No |
| `log_level` | `ARUBACLOUD_LOG_LEVEL` | `OFF` | No |

### Authentication Mechanism

OAuth2 client credentials flow:
1. Provider sends `client_id` + `client_secret` to `token_issuer_url`.
2. Receives a JWT bearer token.
3. All API calls use `Authorization: Bearer <token>`.
4. The ArubaCloud SDK (`sdk-go`) manages token acquisition transparently.

### Async Operation Model

Every resource uses a consistent polling pattern:

```
Create → API call → WaitUntilReady() → success / failure / timeout
Read   → API call → detect transitional state → resume WaitUntilReady()
Delete → API call → WaitForResourceDeleted() → success / timeout
```

**Transitional states**: `InCreation`, `Creating`, `Updating`, `Deleting`, `Pending`, `Provisioning`  
**Terminal failure states**: `Failed`, `Error`, `Errored`, `Faulted`  
**Ready states**: `Active`, `InUse`, `Running`, `Stopped`, `NotUsed`

Poll intervals: 5 s (active state), 10 s (deletion). Default resource timeout: 30 m.

### URI-Based Resource References

ArubaCloud resources are addressed by full URIs, not bare IDs:

```
/projects/{projectID}/compute/cloudServers/{serverID}
/projects/{projectID}/network/vpcs/{vpcID}
/projects/{projectID}/network/vpcs/{vpcID}/subnets/{subnetID}
/projects/{projectID}/providers/Aruba.Container/kaas/{kaasID}
/projects/{projectID}/providers/Aruba.Database/dbaas/{dbaasID}/databases/{dbID}
```

Resources expose a `uri` (Computed) field. Dependent resources reference parent resources via `*_uri_ref` or `*_uri_refs` fields, e.g.:

- `cloudserver.network.vpc_uri_ref` → VPC
- `cloudserver.network.subnet_uri_refs[]` → list of Subnets
- `cloudserver.network.securitygroup_uri_refs[]` → list of SecurityGroups
- `cloudserver.settings.key_pair_uri_ref` → KeyPair
- `cloudserver.storage.boot_volume_uri_ref` → BlockStorage

### Terraform Plugin API

The Terraform provider uses **`terraform-plugin-framework` v1.16.1** exclusively.  
It does **not** use `terraform-plugin-sdk/v2`.

---

## 2. Upjet v2 Architecture

Upjet (v2.4.1) converts a Terraform provider into a Crossplane provider in two phases.

### Build Phase

```
Terraform provider binary
   ↓ (terraform providers schema -json)
schema.json + provider-metadata.yaml
   ↓ (upjet code generator)
Generated Go types  →  CRDs
Generated controllers
```

### Runtime Phase

```
Kubernetes API
  ↓ managed resource YAML
Crossplane Runtime
  ↓ reconcile loop
Upjet Generic Reconciler
  ↓ map spec.forProvider → Terraform HCL
Terraform Execution (one of three modes)
  ↓ result
Map Terraform state → status.atProvider
  ↓
Connection secret (sensitive outputs)
```

### Execution Modes (Upjet v2)

| Mode | Mechanism | Compatible with framework provider |
|---|---|---|
| CLI | Fork the `terraform` binary | **Yes** (universal) |
| SDK v2 | Embed `terraform-plugin-sdk/v2` directly | No — ArubaCloud uses framework |
| Framework | Embed via protov6 protocol | **Yes** |

**Recommendation for Phase 1**: Use CLI mode. It is universally compatible, proven in production, and requires no special framework integration. Migrate to Framework mode later if subprocess overhead becomes a bottleneck.

### Upjet v2 Key Capabilities

- External name configuration per resource
- Cross-resource reference injection
- Sensitive field → connection secret extraction
- Late initialization of API-assigned fields
- Management policy support (Observe, Create, Update, Delete)
- Namespaced resource support (Upjet v2 feature — parallel to cluster-scoped)
- CRD version management and storage migration

---

## 3. Crossplane Provider Architecture (Proposed)

```
provider-arubacloud/
├── cmd/
│   ├── generator/            # make generate entrypoint (runs upjet code gen)
│   └── provider/             # provider binary entrypoint (main.go)
├── config/                   # hand-written Upjet configuration
│   ├── provider.go           # NewProvider(), SetupFn per resource group
│   ├── external_name.go      # ExternalNameConfigs map (all 25 resources)
│   ├── provider-metadata.yaml
│   └── schema.json           # embedded Terraform provider schema
├── apis/
│   ├── cluster/              # cluster-scoped generated types + CRDs
│   │   └── arubacloud/
│   │       └── v1alpha1/
│   └── namespaced/           # namespace-scoped variants (Upjet v2)
│       └── arubacloud/
│           └── v1alpha1/
├── internal/
│   ├── clients/              # ProviderConfig credential loading + Setup func
│   └── features/             # feature flag definitions
├── examples/                 # hand-written YAML examples
├── examples-generated/       # generated YAML examples
├── package/
│   └── crossplane.yaml       # OCI package metadata
└── cluster/                  # Helm chart / install manifests
```

### API Groups

**Decision**: Single API group for v0.x.

`arubacloud.crossplane.io` — all 25 managed resources.

Rationale: The provider is small (~25 resources). A single group reduces packaging complexity, cross-resource reference overhead, and user cognitive load. Multiple groups (compute, network, database, etc.) can be introduced in v1.x if needed.

**API version**: `v1alpha1` for all resources initially. Promote to `v1beta1` when a resource is stable and has acceptance test coverage.

### ProviderConfig Design

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

Credentials secret format (JSON):
```json
{
  "client_id": "your-client-id",
  "client_secret": "your-client-secret",
  "base_url": "",
  "token_issuer_url": ""
}
```

`base_url` and `token_issuer_url` are optional. Omit or leave empty to use ArubaCloud defaults.

### Managed Resource Shape

```yaml
apiVersion: arubacloud.crossplane.io/v1alpha1
kind: CloudServer
metadata:
  name: web-01
  annotations:
    crossplane.io/external-name: srv-abc123   # ArubaCloud resource ID
spec:
  forProvider:
    name: web-01
    location: ITBG-Bergamo
    zone: ITBG-1
    # Kubernetes-native reference instead of raw URI:
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
      keyPairRef:
        name: my-keypair
    storage:
      bootVolumeRef:
        name: boot-vol
  providerConfigRef:
    name: default
status:
  atProvider:
    id: srv-abc123
    uri: /projects/proj-xyz/compute/cloudServers/srv-abc123
  conditions:
    - type: Ready
      status: "True"
    - type: Synced
      status: "True"
```

---

## 4. Cross-Resource Dependency Graph

```
Project ──────────────────────────────────────────────┐
  │                                                    │
  ├── ElasticIP                                        │
  │                                                    │
  ├── VPC ──────────────────────────────────────────┐ │
  │    │                                             │ │
  │    ├── Subnet                                    │ │
  │    │                                             │ │
  │    ├── SecurityGroup                             │ │
  │    │    └── SecurityRule                         │ │
  │    │                                             │ │
  │    └── VPCPeering                                │ │
  │         └── VPCPeeringRoute                      │ │
  │                                                  │ │
  ├── BlockStorage                                   │ │
  │    └── Snapshot                                  │ │
  │    └── Backup                                    │ │
  │         └── Restore *                            │ │
  │                                                  │ │
  ├── KeyPair                                        │ │
  │                                                  │ │
  ├── CloudServer ←─── (VPC, Subnet, SG, EIP,       │ │
  │                      KeyPair, BlockStorage)      │ │
  │                                                  │ │
  ├── VPNTunnel ←─── (VPC, Subnet, ElasticIP) ──────┘ │
  │    └── VPNRoute                                    │
  │                                                    │
  ├── KaaS ←─── (VPC, Subnet, SecurityGroup) ─────────┘
  │                                                    
  ├── ContainerRegistry ←─── (ElasticIP, VPC,
  │                            Subnet, SG, BlockStorage)
  │
  ├── DBaaS ←─── (VPC, Subnet, SG, ElasticIP?)
  │    ├── Database
  │    │    └── DatabaseGrant ←─── (Database, DBaaSUser)
  │    │    └── DatabaseBackup
  │    └── DBaaSUser
  │
  ├── KMS
  └── ScheduleJob *
```

`*` Resources with special handling requirements (see implementation plan).

---

## 5. Sensitive Data Inventory

| Resource | Field | In Schema | Connection Detail |
|---|---|---|---|
| ProviderConfig | `client_secret` | credential | N/A |
| `keypair` | `value` | write-only Sensitive | No (public key only) |
| `dbaasuser` | `password` | Sensitive, immutable | **Yes** |
| `kaas` | `kubeconfig` | Sensitive, Computed | **Yes** — kubeconfig |
| `kaas` | `management_ip` | Computed | **Yes** — endpoint |
| `vpntunnel` | `vpn_client_settings.pre_shared_key` | REQUIRES VERIFICATION | **Yes** |
| `elasticip` | `address` | Computed (public IP) | **Yes** |

---

## 6. ID and Import Format Summary

| Resource | Terraform Import ID | Parts |
|---|---|---|
| `project` | `<project_id>` | 1 |
| `cloudserver` | `<project_id>/<server_id>` | 2 |
| `keypair` | `<project_id>/<keypair_id>` | 2 |
| `elasticip` | `<project_id>/<eip_id>` | 2 |
| `blockstorage` | `<project_id>/<vol_id>` | 2 |
| `snapshot` | `<project_id>/<snap_id>/<billing_period>` | 3 |
| `vpc` | `<project_id>/<vpc_id>` | 2 |
| `subnet` | `<project_id>/<vpc_id>/<subnet_id>` | 3 |
| `securitygroup` | `<project_id>/<vpc_id>/<sg_id>` | 3 |
| `securityrule` | `<project_id>/<vpc_id>/<sg_id>/<rule_id>` | 4 |
| `vpcpeering` | `<project_id>/<vpc_id>/<peering_id>` | 3 |
| `vpcpeeringroute` | REQUIRES VERIFICATION | — |
| `vpntunnel` | `<project_id>/<tunnel_id>` | 2 |
| `vpnroute` | REQUIRES VERIFICATION | — |
| `kaas` | `<project_id>/<kaas_id>` | 2 |
| `containerregistry` | `<project_id>/<reg_id>` | 2 |
| `backup` | `<project_id>/<backup_id>` | 2 |
| `restore` | `<project_id>/<restore_id>` | 2 |
| `dbaas` | `<project_id>/<dbaas_id>` | 2 |
| `database` | `<project_id>/<dbaas_id>/<db_id>` | 3 |
| `databasegrant` | REQUIRES VERIFICATION | — |
| `databasebackup` | REQUIRES VERIFICATION | — |
| `dbaasuser` | `<project_id>/<dbaas_id>/<user_id>` | 3 |
| `schedulejob` | `<project_id>/<job_id>` | 2 |
| `kms` | `<project_id>/<kms_id>` | 2 |

---

## 7. Key Architectural Decisions

### Decision 1: Terraform Plugin API compatibility

**Decision**: Use Upjet CLI execution mode (fork-based) in Phase 1.

**Reason**: The ArubaCloud Terraform provider uses `terraform-plugin-framework` exclusively. Upjet v2 supports framework providers via protov6, but CLI mode works universally with any provider and is the most battle-tested path.

**Evidence**: Upjet README documents three modes; CLI mode is described first and works regardless of plugin API. The DigitalOcean reference provider explicitly omits framework resources from direct embedding.

**Alternatives**: Upjet Framework mode (protov6) — possible but increases initial risk.

**Trade-offs**: CLI mode has higher subprocess overhead per reconcile. Acceptable for v0.x.

---

### Decision 2: Single API group

**Decision**: All resources under `arubacloud.crossplane.io/v1alpha1`.

**Reason**: ~25 resources is well within the range where a single group is manageable. Multiple groups add packaging, documentation, and reference complexity with no immediate benefit.

**Alternatives**: Per-service groups (compute, network, database). Better for very large providers.

---

### Decision 3: URI references become Kubernetes references

**Decision**: ArubaCloud's `*_uri_ref` fields become typed Kubernetes references (`VPCRef`, `SubnetRef`, etc.) in `spec.forProvider`. The URI is derived at reconcile time from the referenced resource's `status.atProvider.uri`.

**Reason**: Users should not need to know ArubaCloud URI construction rules. Kubernetes references enable composition and selector-based discovery.

**Evidence**: Upjet supports `config.Reference{TerraformName: "arubacloud_vpc"}` pattern.

---

### Decision 4: External name = leaf resource ID

**Decision**: `crossplane.io/external-name` annotation holds only the leaf resource ID (e.g., `srv-abc123`), not the composite Terraform import ID.

**Reason**: Parent IDs (project_id, vpc_id, etc.) are derivable from references. Storing them in the external name duplicates information and creates synchronization problems.

**Trade-off**: Custom `GetIDFn` must reconstruct the composite ID for Terraform import. Required for every composite-ID resource.
