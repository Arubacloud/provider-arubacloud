# Cross-Resource References

## How references work

ArubaCloud resources address each other using full URIs (e.g. `/projects/proj-xyz/network/vpcs/vpc-abc`). In Crossplane, these URIs are resolved at reconcile time from Kubernetes-native references.

Instead of copying raw URIs into your manifests, you write:

```yaml
# Without references (error-prone, not Kubernetes-native)
spec:
  forProvider:
    network:
      vpcUriRef: "/projects/proj-xyz/network/vpcs/vpc-abc"

# With references (Kubernetes-native)
spec:
  forProvider:
    network:
      vpcUriRefRef:
        name: main-vpc
```

Crossplane resolves `main-vpc` to the VPC's `status.atProvider.uri` and injects it into the Terraform field automatically.

## Reference field naming

Upjet generates reference fields by appending `Ref` to the original field name. For URI-based fields this produces slightly verbose but consistent names:

| Terraform field | Crossplane reference field | Crossplane selector field |
|---|---|---|
| `project_id` | `projectIdRef` | `projectIdSelector` |
| `vpc_id` | `vpcIdRef` | `vpcIdSelector` |
| `network.vpc_uri_ref` | `network.vpcUriRefRef` | `network.vpcUriRefSelector` |
| `network.subnet_uri_refs[]` | `network.subnetUriRefsRefs[]` | `network.subnetUriRefsSelector` |
| `dbaas_id` | `dbaasIdRef` | `dbaasIdSelector` |

## Using label selectors

Instead of naming a specific resource, you can select by label:

```yaml
spec:
  forProvider:
    network:
      vpcUriRefSelector:
        matchLabels:
          environment: production
          region: ITBG-Bergamo
```

This picks any VPC with those labels. Useful for Compositions.

## Full reference map

```
Project (root — no incoming refs)

VPC
  └── projectIdRef -> Project

Subnet
  ├── projectIdRef -> Project
  └── vpcIdRef -> VPC

SecurityGroup
  ├── projectIdRef -> Project
  └── vpcIdRef -> VPC

SecurityRule
  ├── projectIdRef -> Project
  ├── vpcIdRef -> VPC
  └── securityGroupIdRef -> SecurityGroup

VPCPeering
  ├── projectIdRef -> Project
  ├── vpcIdRef -> VPC
  └── peerVpcRef -> VPC (optional)

VPCPeeringRoute
  ├── projectIdRef -> Project
  ├── vpcIdRef -> VPC
  └── vpcPeeringIdRef -> VPCPeering

ElasticIP
  └── projectIdRef -> Project

KeyPair
  └── projectIdRef -> Project

BlockStorage
  └── projectIdRef -> Project

Snapshot
  ├── projectIdRef -> Project
  └── volumeUriRef -> BlockStorage (URI)

Backup
  ├── projectIdRef -> Project
  └── volumeIdRef -> BlockStorage

Restore
  ├── projectIdRef -> Project
  ├── backupIdRef -> Backup
  └── volumeIdRef -> BlockStorage

VPNTunnel
  ├── projectIdRef -> Project
  ├── properties.ipConfigurations.vpc.idRef -> VPC
  ├── properties.ipConfigurations.subnet.idRef -> Subnet
  └── properties.ipConfigurations.publicIp.idRef -> ElasticIP

VPNRoute
  ├── projectIdRef -> Project
  └── vpnTunnelIdRef -> VPNTunnel

CloudServer
  ├── projectIdRef -> Project
  ├── network.vpcUriRefRef -> VPC (URI)
  ├── network.subnetUriRefsRefs[] -> Subnet (URI list)
  ├── network.securitygroupUriRefsRefs[] -> SecurityGroup (URI list)
  ├── network.elasticIpUriRefRef -> ElasticIP (URI, optional)
  ├── settings.keyPairUriRefRef -> KeyPair (URI, optional)
  └── storage.bootVolumeUriRefRef -> BlockStorage (URI)

KaaS
  ├── projectIdRef -> Project
  ├── network.vpcUriRefRef -> VPC (URI)
  ├── network.subnetUriRefRef -> Subnet (URI)
  └── network.securityGroupNameRef -> SecurityGroup (by name)

ContainerRegistry
  ├── projectIdRef -> Project
  ├── network.publicIpUriRefRef -> ElasticIP (URI)
  ├── network.vpcUriRefRef -> VPC (URI)
  ├── network.subnetUriRefRef -> Subnet (URI)
  ├── network.securityGroupUriRefRef -> SecurityGroup (URI)
  └── storage.blockStorageUriRefRef -> BlockStorage (URI)

DBaaS
  ├── projectIdRef -> Project
  ├── network.vpcUriRefRef -> VPC (URI)
  ├── network.subnetUriRefRef -> Subnet (URI)
  ├── network.securityGroupUriRefRef -> SecurityGroup (URI)
  └── network.elasticIpUriRefRef -> ElasticIP (URI, optional)

Database
  ├── projectIdRef -> Project
  └── dbaasIdRef -> DBaaS

DBaaSUser
  ├── projectIdRef -> Project
  └── dbaasIdRef -> DBaaS

DatabaseGrant
  ├── projectIdRef -> Project
  ├── dbaasIdRef -> DBaaS
  ├── databaseRef -> Database
  └── userIdRef -> DBaaSUser

DatabaseBackup
  ├── projectIdRef -> Project
  ├── dbaasIdRef -> DBaaS
  └── databaseRef -> Database

KMS
  └── projectIdRef -> Project

ScheduleJob
  └── projectIdRef -> Project
```

## KaaS — name-based security group reference

KaaS is the only resource that references a SecurityGroup by display name rather than URI. The generated field is `network.securityGroupNameRef`:

```yaml
network:
  securityGroupNameRef:
    name: my-sg   # resolves to spec.forProvider.name of the SecurityGroup
```
