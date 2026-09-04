# Deep-Dive Guide: provider-arubacloud

---

## 1. The Conceptual Stack — Why Does This Exist?

```
User writes YAML (Kubernetes manifest)
       ↓
Crossplane (Kubernetes operator framework)
       ↓
This provider (provider-arubacloud)
       ↓
Upjet v2 (generic Terraform-to-Crossplane bridge)
       ↓
Terraform binary (CLI subprocess, NOT a library)
       ↓
ArubaCloud Terraform provider (arubacloud/arubacloud v1.0.1)
       ↓
ArubaCloud REST API
```

The goal is to manage ArubaCloud infrastructure **as Kubernetes objects**. A user applies a `Cloudserver` YAML and Crossplane keeps the real cloud resource reconciled to that spec — create, update, delete, detect drift, all automatically.

The reason **Upjet** exists is that writing a Crossplane provider from scratch means hand-coding every resource's Create/Read/Update/Delete logic. Upjet shortcuts this by embedding the Terraform provider's logic: it spawns a real `terraform` binary subprocess, feeds it HCL, and reads back state. The Go code in this repo is mostly **configuration + code-generation scaffolding**, not raw API calls.

---

## 2. The Two "Modes" of Upjet and Why CLI Mode Was Chosen

Upjet has two provider runner strategies:

| Mode | How it works | Constraint |
|---|---|---|
| **Shared (gRPC)** | Provider binary runs as a persistent daemon; Upjet sends RPCs | Requires `terraform-plugin-sdk/v2` protocol |
| **CLI (fork)** | Upjet forks `terraform apply` for each reconcile | Works with any provider, including `terraform-plugin-framework` |

ArubaCloud's Terraform provider is written with **terraform-plugin-framework** (not sdk/v2), which is not compatible with gRPC shared mode. CLI mode was mandatory. You can see the commented-out gRPC option in `cmd/provider/main.go`:

```go
// use the following WorkspaceStoreOption to enable the shared gRPC mode
// terraform.WithProviderRunner(terraform.NewSharedProvider(log, ...))
WorkspaceStore: terraform.NewWorkspaceStore(log),
```

`WorkspaceStore` = CLI mode. Each reconcile creates a temporary directory (a "workspace"), writes a `main.tf`, runs `terraform apply`, and reads `terraform.tfstate`.

---

## 3. How the Code Was Generated — The Generation Pipeline

Most files with a `zz_` prefix are **machine-generated**. Here is the chain:

### Step 1: Grab the Terraform provider schema
```bash
make generate.schema  # produces config/schema.json
```
This calls `terraform providers schema -json` after installing the ArubaCloud provider, dumping the full schema of every resource and data source.

### Step 2: Run `upjet generate`
```bash
make generate  # invokes cmd/generator/main.go via go generate
```
The generator reads `config/schema.json`, `config/provider-metadata.yaml`, and your config in `config/` to produce:
- `apis/cluster/arubacloud/v1alpha1/zz_*.go` — CRD types (one per resource)
- `internal/controller/cluster/arubacloud/zz_*.go` — controller setup functions
- `package/crds/*.yaml` — actual CRD YAML files
- `examples-generated/` — example manifests

### What the generator reads from your config
The entry point is `config/provider.go`:

```go
func GetProvider() *ujconfig.Provider {
    pc := ujconfig.NewProvider([]byte(providerSchema), resourcePrefix, modulePath, []byte(providerMetadata),
        ujconfig.WithRootGroup("crossplane.io"),
        ujconfig.WithIncludeList(ExternalNameConfigured()),  // ← which resources to include
        ujconfig.WithDefaultResourceOptions(
            ExternalNameConfigurations(),                    // ← apply external name table
        ))
    for _, configure := range []func(provider *ujconfig.Provider){
        arubacloudCluster.Configure,                        // ← your hand-written customizations
    } {
        configure(pc)
    }
    pc.ConfigureResources()
    return pc
}
```

`ExternalNameConfigured()` returns a list of regex strings (e.g. `"arubacloud_cloudserver$"`) — only resources in this list get included in the provider. This is why adding a new resource requires touching `config/external_name.go` first.

---

## 4. Anatomy of a Generated Resource (CloudServer Example)

Every resource generates two files.

### `zz_cloudserver_types.go` — The CRD Type

This maps to a Kubernetes CRD with three sections:

```go
// What the user writes in spec.forProvider
type CloudserverParameters struct {
    Location  *string               `json:"location,omitempty" tf:"location,omitempty"`
    ProjectID *string               `json:"projectId,omitempty" tf:"project_id,omitempty"`
    // +crossplane:generate:reference:type=...Project
    ProjectIDRef      *v2.Reference `json:"projectIdRef,omitempty" tf:"-"`   // ← generated cross-ref
    ProjectIDSelector *v2.Selector  `json:"projectIdSelector,omitempty" tf:"-"`
    ...
}

// What the user can set once, then Crossplane locks it (immutable init fields)
type CloudserverInitParameters struct { ... }

// What Crossplane writes back from the cloud (status.atProvider)
type CloudserverObservation struct {
    ID *string `json:"id,omitempty" tf:"id,omitempty"`
    ...
}
```

The `tf:` struct tags are crucial: they map Go field names (camelCase) to Terraform attribute names (snake_case). Fields with `tf:"-"` are Crossplane-only and never sent to Terraform.

### `zz_cloudserver_terraformed.go` — The Terraform Interface

This implements the `terraform.Terraformed` interface that Upjet calls during reconciliation:

```go
func (tr *Cloudserver) GetTerraformResourceType() string { return "arubacloud_cloudserver" }
func (tr *Cloudserver) GetParameters() (map[string]any, error) { ... }  // spec.forProvider → map
func (tr *Cloudserver) SetObservation(obs map[string]any) error { ... } // tfstate → status.atProvider
func (tr *Cloudserver) LateInitialize(attrs []byte) (bool, error) { ... } // fill defaults from state
```

The `GetParameters` / `SetObservation` pair is what Upjet uses to round-trip data between the Kubernetes object and Terraform state.

---

## 5. The Reconciliation Loop — What Happens When You Apply a Manifest

```bash
kubectl apply -f cloudserver.yaml
```

1. **API Server** creates the `Cloudserver` object in etcd.
2. **Upjet controller** (running in the provider pod) gets notified via watch.
3. **Setup function** (`clients.TerraformSetupBuilder`) resolves the `ProviderConfig` reference → reads the Kubernetes Secret → extracts JSON credentials → builds a `terraform.Setup` with a `Configuration` map that becomes the `provider {}` block in the HCL.
4. **Workspace** is created: a temp directory with a generated `main.tf`:
   ```hcl
   terraform {
     required_providers {
       arubacloud = { source = "arubacloud/arubacloud" version = "1.0.1" }
     }
   }
   provider "arubacloud" { client_id = "..." client_secret = "..." }
   resource "arubacloud_cloudserver" "managed" {
     location   = "ITBG-Bergamo"
     project_id = "proj-abc123"
     ...
   }
   ```
5. **`terraform apply`** runs in the workspace. The Terraform provider binary calls ArubaCloud APIs.
6. **State** (`terraform.tfstate`) is read back. `SetObservation()` copies values into `status.atProvider`.
7. **External name** is set: the `status.atProvider.id` is written to the annotation `crossplane.io/external-name`.
8. **Connection details** (passwords, kubeconfig, IPs) are extracted by `AdditionalConnectionDetailsFn` and written to a Kubernetes Secret.

On every subsequent poll (default every 10 minutes, controlled by `--poll` flag), Upjet runs `terraform plan` to detect drift. If the plan is non-empty, it runs `terraform apply` again.

---

## 6. External Names — The Identity Bridge

The external name is the Crossplane concept that links a Kubernetes object to a real cloud resource. It lives in the annotation `crossplane.io/external-name`.

**The problem**: Terraform's `terraform import` takes a composite ID like `project-abc/server-xyz`. Crossplane only stores a single string (the external name). So you need two functions:

| Function | Direction | Example |
|---|---|---|
| `GetExternalNameFn` | `tfstate["id"]` → single leaf name | `"project-abc/server-xyz"` → `"server-xyz"` |
| `GetIDFn` | leaf name + parameters → full import ID | `"server-xyz"` + `{project_id: "project-abc"}` → `"project-abc/server-xyz"` |

This is why `config/external_name.go` exists. Every resource with a composite ID needs both functions. The helper `leafIDFromSlash(index)` extracts by slash position:

```go
func leafIDFromSlash(index int) func(map[string]any) (string, error) {
    return func(tfstate map[string]any) (string, error) {
        id, _ := tfstate["id"].(string)
        parts := strings.Split(id, "/")
        if len(parts) > index {
            return parts[index], nil  // e.g. index=2 for "proj/vpc/subnet-id"
        }
        ...
    }
}
```

`identifierFromProviderWithProjectID()` covers the common 2-part case (`project_id/resource_id`). For 3-part and 4-part IDs, dedicated functions like `subnetExternalName()` and `securityRuleExternalName()` handle the extra segments.

---

## 7. Cross-Resource References — URI-Based References

Most Terraform providers reference resources by opaque IDs. ArubaCloud is unusual: many of its resources cross-reference each other by **full URIs** (e.g. `https://api.arubacloud.com/projects/abc/vpcs/xyz`).

Upjet's reference system normally extracts the external name of the referenced resource. For URI-based references, a custom extractor is needed:

```go
// config/cluster/arubacloud/configure.go
const uriExtractor = `github.com/crossplane/upjet/v2/pkg/resource.ExtractParamPath("uri",true)`
```

`ExtractParamPath("uri", true)` reads the field `status.atProvider.uri` from the referenced resource — not its external name. This is why, in a `Cloudserver` manifest, you write:

```yaml
network:
  vpcUriRefRef:
    name: main-vpc    # ← name of a VPC Kubernetes object
```

And Crossplane resolves it to `status.atProvider.uri` of `main-vpc`, then passes that full URI string to Terraform as `network.vpc_uri_ref`.

For KaaS, `security_group_name` is different — it references by display name, not URI:

```go
const nameExtractor = `github.com/crossplane/upjet/v2/pkg/resource.ExtractParamPath("name",false)`

r.References["network.security_group_name"] = ujconfig.Reference{
    TerraformName: tfSecurityGroup,
    Extractor:     nameExtractor,  // reads spec.forProvider.name, not URI
}
```

---

## 8. Connection Details — Secrets Written Back

When Terraform creates a resource that produces sensitive outputs (passwords, kubeconfigs), Crossplane writes them to a Kubernetes Secret. Two mechanisms:

**Automatic** — Upjet detects fields marked `sensitive = true` in the Terraform schema and maps them automatically.

**Manual** — override via `AdditionalConnectionDetailsFn` for non-sensitive fields or when auto-detection misses them:

```go
r.Sensitive.AdditionalConnectionDetailsFn = func(attr map[string]any) (map[string][]byte, error) {
    conn := map[string][]byte{}
    if kc, ok := attr["kubeconfig"].(string); ok && kc != "" {
        conn["kubeconfig"] = []byte(kc)
    }
    if ip, ok := attr["management_ip"].(string); ok && ip != "" {
        conn["management_ip"] = []byte(ip)
    }
    return conn, nil
}
```

`attr` is the full `status.atProvider` observation map. The returned map entries are written to the Secret named in `spec.writeConnectionSecretToRef`.

---

## 9. Late Initialization — Filling in Defaults From the API

After the first successful `terraform apply`, the API may return values that weren't in the original request (e.g. default CIDRs, auto-assigned zones). Late initialization copies these back into `spec.forProvider` so subsequent plans don't show drift.

By default, Upjet late-initializes all observed fields. You can suppress specific fields:

```go
r.LateInitializer = ujconfig.LateInitializer{
    IgnoredFields: []string{"network.pod_cidr"},
}
```

`pod_cidr` is excluded for KaaS because the ArubaCloud API never returns it (the user picks it), and allowing late-init would overwrite the user's value with nil on the next reconcile.

---

## 10. Management Policies — Imperative vs. Declarative Resources

Standard Crossplane resources are fully declarative (Create → Observe → Update → Delete). Some ArubaCloud resources are imperative one-shots:

- **`restore`** — "restore this backup" is a fire-and-forget action, not a persistent resource
- **`schedulejob`** — runs once; observing it after creation would try to re-trigger it

These use a restricted management policy:

```yaml
spec:
  managementPolicies: [Create, Observe]
```

This tells Crossplane: create it once, then only read its state — never update, never delete.

---

## 11. Credentials Flow — End to End

```bash
kubectl create secret generic arubacloud-creds \
  --from-literal=credentials='{"client_id":"...","client_secret":"..."}'
```

```yaml
apiVersion: arubacloud.crossplane.io/v1beta1
kind: ProviderConfig
metadata:
  name: default
spec:
  credentials:
    source: Secret
    secretRef:
      name: arubacloud-creds
      key: credentials
```

In `internal/clients/arubacloud.go`, `TerraformSetupBuilder` is called per-reconcile:

1. Resolves the `ProviderConfig` (cluster-scoped or namespaced via `resolveProviderConfig`)
2. Calls `resource.CommonCredentialExtractor` → reads the Secret → gets raw JSON bytes
3. Unmarshals into `map[string]string`
4. Builds `ps.Configuration` (the `provider {}` block fields):
   ```go
   ps.Configuration = map[string]any{
       "client_id":     creds["client_id"],
       "client_secret": creds["client_secret"],
   }
   // Optional overrides:
   if v := creds["base_url"]; v != "" { ps.Configuration["base_url"] = v }
   ```

This map becomes the provider configuration injected into every workspace's `main.tf`.

---

## 12. Adding a New Resource — Step-by-Step Checklist

Suppose ArubaCloud adds a new Terraform resource `arubacloud_firewall`.

### Step 1: Add to the external name table

```go
// config/external_name.go
var ExternalNameConfigs = map[string]config.ExternalName{
    ...
    "arubacloud_firewall": identifierFromProviderWithProjectID(), // or custom if needed
}
```

This is the **gate**: only resources listed here are included in generation.

### Step 2: Write a resource configurator

```go
// config/cluster/arubacloud/configure.go

func Configure(p *ujconfig.Provider) {
    ...
    configureFirewall(p)  // add this call
}

func configureFirewall(p *ujconfig.Provider) {
    p.AddResourceConfigurator("arubacloud_firewall", func(r *ujconfig.Resource) {
        r.References["project_id"] = ujconfig.Reference{TerraformName: tfProject}
        r.References["vpc_id"] = ujconfig.Reference{TerraformName: tfVPC}

        // If it has URI-based references:
        r.References["subnet_uri"] = ujconfig.Reference{
            TerraformName: tfSubnet,
            Extractor:     uriExtractor,
        }

        // If it exposes connection details:
        r.Sensitive.AdditionalConnectionDetailsFn = func(attr map[string]any) (map[string][]byte, error) {
            conn := map[string][]byte{}
            if key, ok := attr["api_key"].(string); ok {
                conn["api_key"] = []byte(key)
            }
            return conn, nil
        }

        // If a field must not be late-initialized:
        r.LateInitializer = ujconfig.LateInitializer{
            IgnoredFields: []string{"some_computed_field"},
        }
    })
}
```

### Step 3: Regenerate

```bash
make generate        # regenerates zz_*.go and CRD YAMLs
make generate.schema # only if the Terraform provider was updated
```

### Step 4: Write the example manifest

```yaml
# examples/cluster/arubacloud/firewall.yaml
apiVersion: arubacloud.crossplane.io/v1alpha1
kind: Firewall
metadata:
  name: my-firewall
spec:
  forProvider:
    projectIdRef:
      name: my-project
    vpcIdRef:
      name: main-vpc
  providerConfigRef:
    name: default
```

### Step 5: Verify

```bash
make lint
make test
```

---

## 13. Project Layout Reference

```
provider-arubacloud/
├── cmd/
│   └── provider/main.go        # Binary entry point: starts controller-runtime manager
├── config/
│   ├── provider.go             # Provider bootstrap: schema + configurators
│   ├── external_name.go        # ID mapping table for all 25 resources
│   ├── schema.json             # Terraform provider schema (embedded at build time)
│   ├── provider-metadata.yaml  # Resource categorization (embedded at build time)
│   └── cluster/arubacloud/
│       └── configure.go        # Hand-written: references, connection details, late-init
├── apis/
│   └── cluster/arubacloud/v1alpha1/
│       ├── zz_*_types.go       # Generated CRD type structs
│       └── zz_*_terraformed.go # Generated Terraform interface implementations
├── internal/
│   ├── clients/arubacloud.go   # Credential extraction & Terraform setup
│   └── controller/cluster/     # Generated: controller registration
├── examples/
│   └── cluster/arubacloud/     # Hand-written example manifests for all resources
├── package/crds/               # Generated CRD YAMLs (deployed to cluster)
└── Makefile                    # Wraps build/makelib/ from Upjet's build system
```

---

## 14. Key Concepts Cheat-Sheet

| Concept | Where | What it does |
|---|---|---|
| `ExternalNameConfigs` map | `config/external_name.go` | Gates which TF resources become CRDs |
| `GetIDFn` | same file | Reassembles composite TF import ID from leaf + parameters |
| `GetExternalNameFn` | same file | Extracts leaf ID from composite TF state ID |
| `AddResourceConfigurator` | `configure.go` | Attaches references, connection details, late-init to a resource |
| `uriExtractor` | `configure.go` | Reads `status.atProvider.uri` of referenced resource (ArubaCloud-specific) |
| `AdditionalConnectionDetailsFn` | `configure.go` | Copies fields from TF state into a Kubernetes Secret |
| `LateInitializer.IgnoredFields` | `configure.go` | Prevents Crossplane from overwriting user fields with API defaults |
| `TerraformSetupBuilder` | `internal/clients/` | Per-reconcile credential injection into Terraform provider config |
| `WorkspaceStore` | `cmd/provider/main.go` | Selects CLI mode (fork `terraform apply`) |
| `managementPolicies: [Create, Observe]` | end-user YAML | Makes imperative one-shot resources safe |

---

## 15. What Code You Own vs What Is Generated

| File pattern | Who owns it | Change when... |
|---|---|---|
| `config/external_name.go` | **You** | Adding/removing resources, fixing import IDs |
| `config/cluster/arubacloud/configure.go` | **You** | Adding references, connection details, late-init overrides |
| `config/provider.go` | **You** | Adding new configurator packages |
| `internal/clients/arubacloud.go` | **You** | Credential format changes |
| `cmd/provider/main.go` | **You** (rarely) | Provider-wide flags, new feature gates |
| `apis/cluster/arubacloud/v1alpha1/zz_*.go` | **Generator** | Never edit; run `make generate` |
| `internal/controller/cluster/arubacloud/zz_*.go` | **Generator** | Never edit; run `make generate` |
| `package/crds/*.yaml` | **Generator** | Never edit; run `make generate` |

The `zz_` prefix is the convention for generated files — treat them as compiler output.
