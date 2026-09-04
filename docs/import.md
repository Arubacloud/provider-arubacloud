# Importing Existing Resources

## Overview

You can bring existing ArubaCloud infrastructure under Crossplane management without recreating it. The workflow uses the `crossplane.io/external-name` annotation and `managementPolicies`.

## Step-by-step: importing a VPC

### 1. Find the resource ID

The ArubaCloud resource ID is the last segment of the resource URI. For example:

```
URI:  /projects/proj-abc123/network/vpcs/vpc-xyz789
ID:   vpc-xyz789
```

### 2. Create the Crossplane resource with external-name

```yaml
apiVersion: arubacloud.crossplane.io/v1alpha1
kind: VPC
metadata:
  name: existing-vpc
  annotations:
    crossplane.io/external-name: vpc-xyz789   # ArubaCloud resource ID
spec:
  managementPolicies:
    - Observe                                  # read-only initially
  forProvider:
    name: existing-vpc
    location: ITBG-Bergamo
    projectIdRef:
      name: my-project
  providerConfigRef:
    name: default
```

### 3. Apply and verify

```bash
kubectl apply -f existing-vpc.yaml
kubectl get vpc existing-vpc -o jsonpath='{.status.conditions}' | jq .
```

Wait until `status.conditions` shows `Ready: True` and `Synced: True`.

### 4. Inspect atProvider

```bash
kubectl get vpc existing-vpc -o jsonpath='{.status.atProvider}' | jq .
```

The response shows the current state from ArubaCloud.

### 5. Enable full management (optional)

Once you've verified the import is correct:

```yaml
spec:
  managementPolicies:
    - Observe
    - Create
    - Update
    - Delete
```

## Import ID formats

Each resource type has a specific Terraform import format. Crossplane uses the leaf resource ID as the `external-name` and reconstructs the full Terraform import path using references.

| Resource | external-name value | Required references |
|---|---|---|
| Project | project ID | none |
| VPC | VPC ID | projectIdRef |
| Subnet | subnet ID | projectIdRef, vpcIdRef |
| SecurityGroup | SG ID | projectIdRef, vpcIdRef |
| SecurityRule | rule ID | projectIdRef, vpcIdRef, securityGroupIdRef |
| ElasticIP | EIP ID | projectIdRef |
| KeyPair | keypair ID | projectIdRef |
| BlockStorage | volume ID | projectIdRef |
| Snapshot | snapshot ID | projectIdRef, billing_period in spec |
| Backup | backup ID | projectIdRef |
| VPCPeering | peering ID | projectIdRef, vpcIdRef |
| VPCPeeringRoute | route ID | projectIdRef, vpcIdRef, vpcPeeringIdRef |
| VPNTunnel | tunnel ID | projectIdRef |
| VPNRoute | route ID | projectIdRef |
| CloudServer | server ID | projectIdRef |
| KaaS | cluster ID | projectIdRef |
| ContainerRegistry | registry ID | projectIdRef |
| DBaaS | cluster ID | projectIdRef |
| Database | database name | projectIdRef, dbaasIdRef |
| DBaaSUser | username | projectIdRef, dbaasIdRef |
| DatabaseGrant | grant ID | projectIdRef, dbaasIdRef |
| DatabaseBackup | backup ID | projectIdRef, dbaasIdRef |
| KMS | KMS ID | projectIdRef |
| ScheduleJob | job ID | projectIdRef |

## Example: importing a CloudServer

```yaml
apiVersion: arubacloud.crossplane.io/v1alpha1
kind: CloudServer
metadata:
  name: imported-server
  annotations:
    crossplane.io/external-name: srv-abc123
spec:
  managementPolicies:
    - Observe
  forProvider:
    location: ITBG-Bergamo
    zone: ITBG-1
    projectIdRef:
      name: my-project
    network:
      vpcUriRefRef:
        name: main-vpc
      subnetUriRefsRefs:
        - name: private-subnet
      securitygroupUriRefsRefs:
        - name: web-sg
    settings:
      flavorName: CSO4A8
    storage:
      bootVolumeUriRefRef:
        name: boot-vol
  providerConfigRef:
    name: default
```

> **Note on write-only fields**: `keypair.value` and `dbaasuser.password` are never returned by the API. After import these fields will remain empty in `status.atProvider`. Preserve the original values in `spec.forProvider` if you need Crossplane to manage them.
