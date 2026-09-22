# ApplicationEnvironment Composition

Deploy a Docker container on an ArubaCloud CloudServer with a single Crossplane claim.

The `ApplicationEnvironment` composite resource provisions a CloudServer and injects a
cloud-init script at first boot that installs Docker and runs your image. All networking
infrastructure (VPC, subnet, security group, etc.) must exist beforehand and is referenced
by name.

This guide uses Crossplane v2 native features:

| Feature | Purpose |
|---|---|
| `Pipeline` mode | Standard Crossplane v2 Composition mode |
| `function-go-templating` | Generates **all** composed resources in a single Go template step — no `function-patch-and-transform` |
| `function-auto-ready` | Automatically marks the composite Ready when all composed resources are Ready |
| Connection details | `publicIp` and `privateIp` written to a Kubernetes Secret via `CompositeConnectionDetails` |

---

## Prerequisites

### 1. Crossplane v2 with this provider installed

```bash
kubectl get provider provider-arubacloud
# INSTALLED   HEALTHY
# True        True
```

### 2. Install Functions and provider-kubernetes

```yaml
# packages.yaml
apiVersion: pkg.crossplane.io/v1
kind: Function
metadata:
  name: function-go-templating
spec:
  package: xpkg.upbound.io/crossplane-contrib/function-go-templating:v0.7.0
---
apiVersion: pkg.crossplane.io/v1
kind: Function
metadata:
  name: function-auto-ready
spec:
  package: xpkg.upbound.io/crossplane-contrib/function-auto-ready:v0.3.0
---
apiVersion: pkg.crossplane.io/v1
kind: Provider
metadata:
  name: provider-kubernetes
spec:
  package: xpkg.upbound.io/crossplane-contrib/provider-kubernetes:v0.15.0
```

```bash
kubectl apply -f packages.yaml
kubectl wait --for=condition=healthy function/function-go-templating --timeout=120s
kubectl wait --for=condition=healthy function/function-auto-ready --timeout=120s
kubectl wait --for=condition=healthy provider/provider-kubernetes --timeout=120s
```

### 3. Configure provider-kubernetes for in-cluster access

```yaml
# providerconfig-kubernetes.yaml
apiVersion: kubernetes.crossplane.io/v1alpha1
kind: ProviderConfig
metadata:
  name: kubernetes-in-cluster
spec:
  credentials:
    source: InjectedIdentity
```

```bash
kubectl apply -f providerconfig-kubernetes.yaml
```

---

## Architecture

```
ApplicationEnvironment (Claim, namespaced)
  └── XApplicationEnvironment (Composite, cluster-scoped)
        └── Composition Pipeline
              │
              ├── step: compose  (function-go-templating)
              │     ├── Object (provider-kubernetes)  ← cloud-init Secret
              │     ├── Cloudserver                   ← userDataSecretRef → Secret above
              │     └── CompositeConnectionDetails    ← publicIp + privateIp from status
              │
              └── step: auto-ready  (function-auto-ready)
                    └── sets XR Ready=True when all composed resources are Ready
```

The cloud-init Secret Object and the CloudServer are created concurrently. If the
CloudServer reconciles before the Secret exists it retries automatically — it self-heals
within a few cycles.

---

## Files

Create these two files under `apis/composition/`:

### `xapplicationenvironment.yaml` — CompositeResourceDefinition

```yaml
apiVersion: apiextensions.crossplane.io/v1
kind: CompositeResourceDefinition
metadata:
  name: xapplicationenvironments.apps.arubacloud.crossplane.io
spec:
  group: apps.arubacloud.crossplane.io
  names:
    kind: XApplicationEnvironment
    plural: xapplicationenvironments
  claimNames:
    kind: ApplicationEnvironment
    plural: applicationenvironments
  # Keys written to the claim's writeConnectionSecretToRef Secret
  connectionSecretKeys:
    - publicIp
    - privateIp
  versions:
    - name: v1alpha1
      served: true
      referenceable: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              required:
                - parameters
              properties:
                parameters:
                  type: object
                  required:
                    - image
                    - projectRef
                    - vpcRef
                    - subnetRef
                    - securityGroupRef
                    - elasticIpRef
                    - keyPairRef
                    - blockStorageRef
                  properties:
                    image:
                      type: string
                      description: "Docker image to run, e.g. nginx:latest"
                    port:
                      type: integer
                      default: 80
                      description: "Container port mapped to host port 80"
                    location:
                      type: string
                      default: ITBG-Bergamo
                    zone:
                      type: string
                      default: ITBG-1
                    flavor:
                      type: string
                      default: CSO4A8
                      description: "CloudServer flavour, e.g. CSO4A8 (4 vCPU / 8 GB)"
                    projectRef:
                      type: object
                      required: [name]
                      properties:
                        name:
                          type: string
                    vpcRef:
                      type: object
                      required: [name]
                      properties:
                        name:
                          type: string
                    subnetRef:
                      type: object
                      required: [name]
                      properties:
                        name:
                          type: string
                    securityGroupRef:
                      type: object
                      required: [name]
                      properties:
                        name:
                          type: string
                    elasticIpRef:
                      type: object
                      required: [name]
                      properties:
                        name:
                          type: string
                    keyPairRef:
                      type: object
                      required: [name]
                      properties:
                        name:
                          type: string
                    blockStorageRef:
                      type: object
                      required: [name]
                      properties:
                        name:
                          type: string
```

---

### `composition.yaml` — Composition

```yaml
apiVersion: apiextensions.crossplane.io/v1
kind: Composition
metadata:
  name: applicationenvironments.apps.arubacloud.crossplane.io
spec:
  compositeTypeRef:
    apiVersion: apps.arubacloud.crossplane.io/v1alpha1
    kind: XApplicationEnvironment
  mode: Pipeline
  pipeline:

    # ── Step 1: generate all composed resources ───────────────────────────────
    # One Go template produces three outputs separated by ---:
    #   1. Object (provider-kubernetes) → creates the cloud-init Secret
    #   2. Cloudserver                 → references that Secret via userDataSecretRef
    #   3. CompositeConnectionDetails  → writes publicIp + privateIp to the claim Secret
    - step: compose
      functionRef:
        name: function-go-templating
      input:
        apiVersion: gotemplating.fn.crossplane.io/v1beta1
        kind: GoTemplate
        source: Inline
        inline:
          template: |
            {{- $xr    := .observed.composite.resource }}
            {{- $name  := $xr.metadata.name }}
            {{- $ns    := $xr.spec.claimRef.namespace | default "crossplane-system" }}
            {{- $p     := $xr.spec.parameters }}
            {{- $image := $p.image }}
            {{- $port  := $p.port | default 80 }}

            # ── 1. cloud-init Secret (created by provider-kubernetes) ─────────
            apiVersion: kubernetes.crossplane.io/v1alpha2
            kind: Object
            metadata:
              name: {{ $name }}-cloudinit
              annotations:
                gotemplating.fn.crossplane.io/composition-resource-name: cloudinit-secret
            spec:
              providerConfigRef:
                name: kubernetes-in-cluster
              forProvider:
                manifest:
                  apiVersion: v1
                  kind: Secret
                  metadata:
                    name: {{ $name }}-cloudinit
                    namespace: {{ $ns }}
                  stringData:
                    userdata: |
                      #cloud-config
                      packages:
                        - docker.io
                      runcmd:
                        - systemctl enable docker --now
                        - docker pull {{ $image }}
                        - docker run -d --restart=always -p 80:{{ $port }} --name app {{ $image }}
            ---
            # ── 2. CloudServer ────────────────────────────────────────────────
            apiVersion: arubacloud.crossplane.io/v1alpha1
            kind: Cloudserver
            metadata:
              name: {{ $name }}-server
              namespace: {{ $ns }}
              annotations:
                gotemplating.fn.crossplane.io/composition-resource-name: cloudserver
            spec:
              forProvider:
                name: {{ $name }}-server
                location: {{ $p.location | default "ITBG-Bergamo" }}
                zone: {{ $p.zone | default "ITBG-1" }}
                tags:
                  - crossplane
                  - app-environment
                projectIdRef:
                  name: {{ $p.projectRef.name }}
                network:
                  vpcUriRefRef:
                    name: {{ $p.vpcRef.name }}
                  subnetUriRefsRefs:
                    - name: {{ $p.subnetRef.name }}
                  securitygroupUriRefsRefs:
                    - name: {{ $p.securityGroupRef.name }}
                  elasticIpUriRefRef:
                    name: {{ $p.elasticIpRef.name }}
                settings:
                  flavorName: {{ $p.flavor | default "CSO4A8" }}
                  keyPairUriRefRef:
                    name: {{ $p.keyPairRef.name }}
                  userDataSecretRef:
                    name: {{ $name }}-cloudinit
                    key: userdata
                storage:
                  bootVolumeUriRefRef:
                    name: {{ $p.blockStorageRef.name }}
              writeConnectionSecretToRef:
                name: {{ $name }}-server-connection
              providerConfigRef:
                kind: ProviderConfig
                name: default
            ---
            # ── 3. Connection details → written to the claim Secret ───────────
            # Reads publicIp and privateIp from the observed CloudServer status.
            # Empty on first reconcile; populated once the server is Ready.
            {{- $cs    := dig "cloudserver" "resource" "status" "atProvider" dict .observed.resources }}
            {{- $pubIp := dig "publicIp" "" $cs }}
            {{- $prvIp := dig "privateIp" "" $cs }}
            apiVersion: meta.gotemplating.fn.crossplane.io/v1alpha1
            kind: CompositeConnectionDetails
            data:
              {{- if $pubIp }}
              publicIp: {{ $pubIp | b64enc }}
              {{- end }}
              {{- if $prvIp }}
              privateIp: {{ $prvIp | b64enc }}
              {{- end }}

    # ── Step 2: auto-ready ────────────────────────────────────────────────────
    # Marks the XR Ready=True when every composed resource is Ready.
    # No configuration needed.
    - step: auto-ready
      functionRef:
        name: function-auto-ready
```

---

## Example Claim

```yaml
# applicationenvironment-nginx.yaml
apiVersion: apps.arubacloud.crossplane.io/v1alpha1
kind: ApplicationEnvironment
metadata:
  name: nginx-app
  namespace: crossplane-system
spec:
  parameters:
    image: nginx:latest
    port: 80
    location: ITBG-Bergamo
    zone: ITBG-1
    flavor: CSO4A8
    projectRef:
      name: my-project
    vpcRef:
      name: main-vpc
    subnetRef:
      name: private-subnet
    securityGroupRef:
      name: web-sg
    elasticIpRef:
      name: web-eip
    keyPairRef:
      name: my-keypair
    blockStorageRef:
      name: boot-vol
  # publicIp and privateIp will be written here once the server is Ready
  writeConnectionSecretToRef:
    name: nginx-app-connection
  providerConfigRef:
    kind: ProviderConfig
    name: default
```

Apply:

```bash
kubectl apply -f apis/composition/xapplicationenvironment.yaml
kubectl apply -f apis/composition/composition.yaml
kubectl apply -f applicationenvironment-nginx.yaml
```

---

## Verify

```bash
# Watch the claim become Ready (CloudServer provisioning takes ~10–15 min)
kubectl get applicationenvironment nginx-app -n crossplane-system -w

# Inspect composed resources
kubectl get cloudserver -n crossplane-system
kubectl get object    -n crossplane-system   # cloud-init Secret Object

# Describe the claim to see events and readiness
kubectl describe applicationenvironment nginx-app -n crossplane-system

# Read the public IP from the connection Secret
kubectl get secret nginx-app-connection -n crossplane-system \
  -o jsonpath='{.data.publicIp}' | base64 -d

# Test the running container (port 80 must be open in web-sg)
PUBLIC_IP=$(kubectl get secret nginx-app-connection -n crossplane-system \
  -o jsonpath='{.data.publicIp}' | base64 -d)
curl http://$PUBLIC_IP
```

---

## Updating the Image

`user_data` is immutable on ArubaCloud CloudServers — changing it forces a
destroy-and-recreate of the CloudServer. To deploy a new image version, delete
the claim and reapply:

```bash
kubectl delete applicationenvironment nginx-app -n crossplane-system
# edit applicationenvironment-nginx.yaml → update image
kubectl apply -f applicationenvironment-nginx.yaml
```

---

## Troubleshooting

| Symptom | Likely cause | Fix |
|---|---|---|
| CloudServer `Synced=False` — `userDataSecretRef not found` | cloud-init Secret not created yet | Wait 1–2 min; self-heals once `provider-kubernetes` reconciles the Object |
| Composition pipeline error — function not found | Function package not installed | `kubectl get function` and apply `packages.yaml` |
| `CompositeConnectionDetails` empty | Server not Ready yet | Wait for `Ready=True`; IPs populate on next reconcile after provisioning |
| Container not running (SSH confirms server is up) | cloud-init still executing | Check `/var/log/cloud-init-output.log` on the server |
| Port 80 unreachable from internet | Missing security rule | Add an Ingress TCP rule on port 80 to `web-sg` |
