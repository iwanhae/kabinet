# Kabinet Helm Chart

This Helm chart deploys Kabinet - a Kubernetes event filing cabinet that collects cluster events in real time, stores them efficiently, and provides a rich web interface for exploration and analytics.

## Prerequisites

- Kubernetes 1.19+
- Helm 3.0+
- PV provisioner support in the underlying infrastructure (if persistence is enabled)

## Installing the Chart

To install the chart with the release name `kabinet`:

```bash
helm install kabinet ./chart/kabinet
```

The command deploys Kabinet on the Kubernetes cluster with the default configuration. The [Parameters](#parameters) section lists the parameters that can be configured during installation.

## Uninstalling the Chart

To uninstall/delete the `kabinet` deployment:

```bash
helm uninstall kabinet
```

The command removes all the Kubernetes components associated with the chart and deletes the release.

## Parameters

### Global Parameters

| Name                      | Description                                     | Default           |
|---------------------------|-------------------------------------------------|-------------------|
| `replicaCount`            | Number of Kabinet replicas                      | `1`               |
| `image.repository`        | Kabinet image repository                        | `ghcr.io/iwanhae/kabinet` |
| `image.pullPolicy`        | Image pull policy                               | `IfNotPresent`    |
| `image.tag`               | Image tag (defaults to chart appVersion)        | `""`              |
| `imagePullSecrets`        | Image pull secrets                              | `[]`              |
| `nameOverride`            | String to partially override kabinet.fullname   | `""`              |
| `fullnameOverride`        | String to fully override kabinet.fullname       | `""`              |

### Service Account Parameters

| Name                           | Description                                        | Default |
|--------------------------------|----------------------------------------------------|---------|
| `serviceAccount.create`        | Specifies whether a ServiceAccount should be created | `true`  |
| `serviceAccount.automount`     | Automatically mount API credentials                | `true`  |
| `serviceAccount.annotations`   | Annotations to add to the service account          | `{}`    |
| `serviceAccount.name`          | The name of the ServiceAccount to use              | `""`    |

### RBAC Parameters

| Name                | Description                                     | Default              |
|---------------------|-------------------------------------------------|----------------------|
| `rbac.create`       | Specifies whether RBAC resources should be created | `true`            |
| `rbac.rules`        | Custom RBAC rules for the ClusterRole           | See `values.yaml`    |

### Pod Parameters

| Name                     | Description                              | Default |
|--------------------------|------------------------------------------|---------|
| `podAnnotations`         | Annotations for Kabinet pods             | `{}`    |
| `podLabels`              | Labels for Kabinet pods                  | `{}`    |
| `podSecurityContext`     | Security context for the pod             | `{}`    |
| `securityContext`        | Security context for the container       | `{}`    |

### Service Parameters

| Name                | Description                         | Default      |
|---------------------|-------------------------------------|--------------|
| `service.type`      | Kubernetes Service type             | `ClusterIP`  |
| `service.port`      | Service HTTP port                   | `8080`       |

### Ingress Parameters

| Name                       | Description                                      | Default           |
|----------------------------|--------------------------------------------------|-------------------|
| `ingress.enabled`          | Enable ingress controller resource               | `false`           |
| `ingress.className`        | IngressClass that will be used                   | `""`              |
| `ingress.annotations`      | Ingress annotations                              | `{}`              |
| `ingress.hosts`            | Ingress hosts configuration                      | See `values.yaml` |
| `ingress.tls`              | Ingress TLS configuration                        | `[]`              |

### Resource Parameters

| Name                | Description                         | Default |
|---------------------|-------------------------------------|---------|
| `resources.limits`  | Resource limits for the container   | `{}`    |
| `resources.requests`| Resource requests for the container | `{}`    |

### Kabinet Configuration Parameters

| Name                                    | Description                                | Default           |
|-----------------------------------------|--------------------------------------------|-------------------|
| `kabinet.storageLimitGB`                | Storage limit in gigabytes                 | `10`              |
| `kabinet.listenPort`                    | Port on which the API server will listen   | `8080`            |
| `kabinet.persistence.enabled`           | Enable persistence using PVC               | `true`            |
| `kabinet.persistence.storageClass`      | PVC Storage Class                          | `""`              |
| `kabinet.persistence.accessMode`        | PVC Access Mode                            | `ReadWriteOnce`   |
| `kabinet.persistence.size`              | PVC Storage Request                        | `20Gi`            |
| `kabinet.persistence.existingClaim`     | Name of an existing PVC to use             | `""`              |

### ServiceMonitor Parameters (Prometheus Operator)

| Name                              | Description                              | Default |
|-----------------------------------|------------------------------------------|---------|
| `serviceMonitor.enabled`          | Create ServiceMonitor resource           | `false` |
| `serviceMonitor.labels`           | Additional labels for ServiceMonitor     | `{}`    |
| `serviceMonitor.interval`         | Scrape interval                          | `30s`   |
| `serviceMonitor.scrapeTimeout`    | Scrape timeout                           | `10s`   |

### Other Parameters

| Name                | Description                              | Default |
|---------------------|------------------------------------------|---------|
| `nodeSelector`      | Node labels for pod assignment           | `{}`    |
| `tolerations`       | Tolerations for pod assignment           | `[]`    |
| `affinity`          | Affinity for pod assignment              | `{}`    |

## Configuration Examples

### Using an existing PersistentVolumeClaim

```yaml
kabinet:
  persistence:
    enabled: true
    existingClaim: my-existing-pvc
```

### Configuring storage limit

```yaml
kabinet:
  storageLimitGB: 50
```

### Enabling Ingress

```yaml
ingress:
  enabled: true
  className: "nginx"
  annotations:
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
  hosts:
    - host: kabinet.example.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: kabinet-tls
      hosts:
        - kabinet.example.com
```

### Resource Limits

```yaml
resources:
  limits:
    cpu: 1000m
    memory: 2Gi
  requests:
    cpu: 500m
    memory: 1Gi
```

### Enable Prometheus ServiceMonitor

```yaml
serviceMonitor:
  enabled: true
  interval: 30s
  scrapeTimeout: 10s
  labels:
    prometheus: kube-prometheus
```

## Accessing Kabinet

After installation, you can access Kabinet using:

1. **Port-forward** (for testing):
   ```bash
   kubectl port-forward svc/kabinet 8080:8080
   ```
   Then visit http://localhost:8080

2. **Ingress** (if enabled):
   Visit the configured hostname

3. **LoadBalancer** (if service.type=LoadBalancer):
   ```bash
   kubectl get svc kabinet
   ```
   Use the EXTERNAL-IP shown

## Web Interface

Kabinet provides two main interfaces:

- **Analytics Dashboard** (`/`): Real-time insights with event timelines, top noisy namespaces, warning analysis, and recent critical events
- **Discover Page** (`/p/discover`): Advanced SQL query builder with syntax highlighting and interactive results

## Storage Management

Kabinet automatically manages event data lifecycle:

- Recent events are stored in a fast DuckDB database
- Older events are archived to compressed Parquet files (ZSTD compression)
- When the storage limit is reached, the oldest files are automatically pruned
- The default storage limit is 10GB but can be adjusted via `kabinet.storageLimitGB`

## RBAC Permissions

The chart creates a ClusterRole with permissions to watch events across the cluster:

- `events` (core API group)
- `events.k8s.io` API group

These permissions are required for Kabinet to collect cluster events.

## Troubleshooting

### Check pod status
```bash
kubectl get pods -l app.kubernetes.io/name=kabinet
```

### View logs
```bash
kubectl logs -l app.kubernetes.io/name=kabinet
```

### Check RBAC permissions
```bash
kubectl auth can-i list events --as=system:serviceaccount:default:kabinet
```

## More Information

For more details about Kabinet, visit:
- GitHub Repository: https://github.com/iwanhae/kabinet
- Documentation: See README.md in the repository
