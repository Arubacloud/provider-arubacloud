# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [0.1.0] - 2026-09-22

### Added

#### Provider

- **`provider-arubacloud`** — namespace-scoped Crossplane provider for ArubaCloud built with [Upjet](https://github.com/crossplane/upjet), wrapping the [ArubaCloud Terraform provider](https://github.com/Arubacloud/terraform-provider-arubacloud). Requires Crossplane v2.x.
- **25 managed resource kinds** across six groups:

  | Group | Kinds |
  |---|---|
  | Core | `Project`, `VPC`, `Subnet`, `KeyPair`, `ElasticIP`, `CloudServer` |
  | Storage | `BlockStorage`, `Snapshot`, `Backup`, `Restore` |
  | Networking | `SecurityGroup`, `SecurityRule`, `VPCPeering`, `VPCPeeringRoute`, `VPNTunnel`, `VPNRoute` |
  | Containers | `KaaS`, `ContainerRegistry` |
  | Databases | `DBaaS`, `Database`, `DBaaSUser`, `DatabaseGrant`, `DatabaseBackup` |
  | Security | `KMS` |
  | Scheduling | `ScheduleJob` |

- **`ProviderConfig`** (`arubacloud.crossplane.io/v1beta1`) — namespace-scoped, accepts OAuth2 `client_id` / `client_secret` from a Kubernetes Secret.
- **`ClusterProviderConfig`** (`arubacloud.crossplane.io/v1beta1`) — cluster-scoped variant for compatibility with Crossplane v1 compositions.
- Full **cross-resource reference resolution** — VPC → Subnet → SecurityGroup → CloudServer references resolved automatically by Crossplane.
- **Connection secrets** — sensitive outputs (public IP, kubeconfig, database credentials) written to Kubernetes Secrets via `writeConnectionSecretToRef`.
- **Management policies** — supports `Observe`, `Create`, `Update`, `Delete` and combinations for importing existing infrastructure.

#### CloudServer connection details

- Exposes `public_ip` and `private_ip` as connection detail keys written to the connection secret.
- Bumped underlying `terraform-provider-arubacloud` dependency to v1.1.1 to surface `public_ip`.

#### Package / distribution

- Published to [Upbound Marketplace](https://marketplace.upbound.io) as `xpkg.upbound.io/arubacloud/provider-arubacloud`.
- Published to GitHub Container Registry as `ghcr.io/arubacloud/provider-arubacloud`.
- Official ArubaCloud logo (wordmark + cloud symbol SVG) embedded as `meta.crossplane.io/iconData` in `crossplane.yaml`.
- `SafeList` and `SafeStart` capabilities declared in `crossplane.yaml`.

#### CI/CD

- GitHub Actions publish workflow: builds the provider xpkg and pushes to both GHCR and Upbound Marketplace on `v*` tags.
- `make generate` pipeline: runs Upjet code generation, updates CRDs, provider-metadata, and Go types; `check-diff` step enforces a clean working tree after generation.
- Linter (`golangci-lint`) integrated via `make lint`.

#### Examples

- Complete YAML manifests for all 25 resource kinds under `examples/namespaced/arubacloud/`.
- `ProviderConfig` example under `examples/namespaced/providerconfig/`.

#### Documentation

- `docs/authentication.md` — OAuth2 credentials setup and secret format.
- `docs/references.md` — cross-resource reference patterns.
- `docs/external-names.md` — `crossplane.io/external-name` annotation usage.
- `docs/import.md` — importing existing ArubaCloud resources under Crossplane management.
- `docs/async-resources.md` — async operations, KaaS timeouts, and `resource_timeout` configuration.
- `docs/management-policies.md` — `Observe`-only and partial-management patterns.
- `docs/limitations.md` — known limitations.
- `docs/troubleshooting.md` — debugging guide.
- `docs/resource-matrix.md` — full resource compatibility matrix.
- `docs/architecture.md` — provider architecture overview.

---

[0.1.0]: https://github.com/Arubacloud/provider-arubacloud/releases/tag/v0.1.0
