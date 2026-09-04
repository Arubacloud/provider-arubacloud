# Crossplane Provider ArubaCloud

`provider-arubacloud` is a [Crossplane](https://crossplane.io/) provider built with [Upjet](https://github.com/crossplane/upjet) that exposes XRM-conformant managed resources for the [ArubaCloud](https://www.cloud.it/) API.

## Overview

This provider lets you manage ArubaCloud infrastructure declaratively via Kubernetes. All 25 ArubaCloud resource types are supported, including compute, networking, storage, databases, containers, and security.

| Resource group | Kinds |
|---|---|
| Core | `Project`, `VPC`, `Subnet`, `KeyPair`, `ElasticIP`, `CloudServer` |
| Storage | `BlockStorage`, `Snapshot`, `Backup`, `Restore` |
| Networking | `SecurityGroup`, `SecurityRule`, `VPCPeering`, `VPCPeeringRoute`, `VPNTunnel`, `VPNRoute` |
| Containers | `KaaS`, `ContainerRegistry` |
| Databases | `DBaaS`, `Database`, `DBaaSUser`, `DatabaseGrant`, `DatabaseBackup` |
| Security | `KMS` |
| Scheduling | `ScheduleJob` |

## Prerequisites

- Kubernetes cluster with [Crossplane](https://docs.crossplane.io/latest/software/install/) v2.x installed
- ArubaCloud account with OAuth2 client credentials (`client_id` + `client_secret`)

## Install

Install the provider using the Crossplane CLI or by applying the package directly:

```bash
kubectl apply -f examples/install.yaml
```

Or using the Crossplane CLI:

```bash
crossplane xpkg install provider ghcr.io/arubacloud/provider-arubacloud:v0.3.0
```

## Quickstart

### 1. Create the credentials secret

```bash
kubectl create secret generic arubacloud-credentials \
  --namespace crossplane-system \
  --from-literal=credentials='{
    "client_id": "YOUR_CLIENT_ID",
    "client_secret": "YOUR_CLIENT_SECRET"
  }'
```

### 2. Configure the provider

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

```bash
kubectl apply -f examples/cluster/providerconfig/providerconfig.yaml
```

### 3. Create a Project and VPC

```bash
kubectl apply -f examples/cluster/arubacloud/project.yaml
kubectl apply -f examples/cluster/arubacloud/vpc.yaml
```

Wait for resources to become ready:

```bash
kubectl get project,vpc
```

### 4. Launch a CloudServer

```bash
kubectl apply -f examples/cluster/arubacloud/keypair.yaml
kubectl apply -f examples/cluster/arubacloud/subnet.yaml
kubectl apply -f examples/cluster/arubacloud/elasticip.yaml
kubectl apply -f examples/cluster/arubacloud/blockstorage.yaml
kubectl apply -f examples/cluster/arubacloud/cloudserver.yaml
```

## Documentation

| Topic | File |
|---|---|
| Authentication & ProviderConfig | [docs/authentication.md](docs/authentication.md) |
| Cross-resource references | [docs/references.md](docs/references.md) |
| External name annotation | [docs/external-names.md](docs/external-names.md) |
| Importing existing resources | [docs/import.md](docs/import.md) |
| Async operations & timeouts | [docs/async-resources.md](docs/async-resources.md) |
| Management policies | [docs/management-policies.md](docs/management-policies.md) |
| Known limitations | [docs/limitations.md](docs/limitations.md) |
| Troubleshooting | [docs/troubleshooting.md](docs/troubleshooting.md) |
| Resource matrix | [docs/resource-matrix.md](docs/resource-matrix.md) |
| Architecture | [docs/architecture.md](docs/architecture.md) |

## Examples

Complete example manifests for all resources live in `examples/cluster/arubacloud/`.

## KaaS (Kubernetes as a Service)

KaaS cluster creation takes 10–30 minutes. Set `resource_timeout: "60m"` in your credentials secret and expose the kubeconfig via `writeConnectionSecretToRef`:

```yaml
spec:
  writeConnectionSecretToRef:
    name: my-cluster-kubeconfig
    namespace: default
```

See [examples/cluster/arubacloud/kaas.yaml](examples/cluster/arubacloud/kaas.yaml).

## Importing existing resources

Bring existing ArubaCloud infrastructure under Crossplane management without recreating it:

```yaml
metadata:
  annotations:
    crossplane.io/external-name: vpc-abc123   # existing ArubaCloud resource ID
spec:
  managementPolicies:
    - Observe
```

See [docs/import.md](docs/import.md) for the full workflow.

## Developing

Run the code generation pipeline:

```bash
make generate
```

Run against a Kubernetes cluster:

```bash
make run
```

Run unit tests:

```bash
go test ./...
```

Run the linter:

```bash
make lint
```

Build and push the provider image:

```bash
make build.all
```

## Report a Bug

Open an [issue](https://github.com/Arubacloud/provider-arubacloud/issues).
