# Kabinet Helm Installation Guide

This guide provides detailed instructions for deploying Kabinet using Helm.

## Prerequisites

- Kubernetes cluster (version 1.19+)
- Helm 3.0 or later installed
- `kubectl` configured to access your cluster
- Sufficient cluster permissions to create ClusterRole and ClusterRoleBinding

## Quick Start

### 1. Install Kabinet with default settings

```bash
helm install kabinet ./chart/kabinet
```

This will deploy Kabinet with:
- 20Gi persistent volume for event storage
- 10GB storage limit
- ClusterIP service
- RBAC configured to watch events cluster-wide

### 2. Access Kabinet

After installation, follow the instructions from the NOTES output, or use port-forwarding:

```bash
kubectl port-forward svc/kabinet 8080:8080
```

Then open your browser to:
- **Analytics Dashboard**: http://localhost:8080
- **Discover Page**: http://localhost:8080/p/discover

## Common Installation Scenarios

### Scenario 1: Custom Storage Configuration

Deploy Kabinet with a larger storage limit and persistent volume:

```bash
helm install kabinet ./chart/kabinet \
  --set kabinet.storageLimitGB=50 \
  --set kabinet.persistence.size=100Gi
```

### Scenario 2: Using Existing PersistentVolumeClaim

If you already have a PVC, you can use it:

```bash
helm install kabinet ./chart/kabinet \
  --set kabinet.persistence.existingClaim=my-existing-pvc
```

### Scenario 3: Expose via Ingress

Deploy Kabinet with Ingress for external access:

```bash
helm install kabinet ./chart/kabinet \
  --set ingress.enabled=true \
  --set ingress.className=nginx \
  --set ingress.hosts[0].host=kabinet.example.com \
  --set ingress.hosts[0].paths[0].path=/ \
  --set ingress.hosts[0].paths[0].pathType=Prefix
```

For HTTPS with cert-manager:

```bash
helm install kabinet ./chart/kabinet \
  --set ingress.enabled=true \
  --set ingress.className=nginx \
  --set ingress.annotations."cert-manager\.io/cluster-issuer"=letsencrypt-prod \
  --set ingress.hosts[0].host=kabinet.example.com \
  --set ingress.hosts[0].paths[0].path=/ \
  --set ingress.hosts[0].paths[0].pathType=Prefix \
  --set ingress.tls[0].secretName=kabinet-tls \
  --set ingress.tls[0].hosts[0]=kabinet.example.com
```

### Scenario 4: Resource Limits

Deploy with specific resource requests and limits:

```bash
helm install kabinet ./chart/kabinet \
  --set resources.requests.cpu=500m \
  --set resources.requests.memory=1Gi \
  --set resources.limits.cpu=2000m \
  --set resources.limits.memory=4Gi
```

### Scenario 5: Using LoadBalancer Service

Expose Kabinet via a LoadBalancer:

```bash
helm install kabinet ./chart/kabinet \
  --set service.type=LoadBalancer
```

### Scenario 6: Enable Prometheus Monitoring

Deploy with Prometheus ServiceMonitor (requires Prometheus Operator):

```bash
helm install kabinet ./chart/kabinet \
  --set serviceMonitor.enabled=true \
  --set serviceMonitor.interval=30s
```

### Scenario 7: Using a Custom Values File

Create a `custom-values.yaml` file:

```yaml
kabinet:
  storageLimitGB: 100
  persistence:
    size: 200Gi

resources:
  limits:
    cpu: 2000m
    memory: 4Gi
  requests:
    cpu: 1000m
    memory: 2Gi

ingress:
  enabled: true
  className: nginx
  hosts:
    - host: kabinet.company.com
      paths:
        - path: /
          pathType: Prefix

serviceMonitor:
  enabled: true
```

Then install:

```bash
helm install kabinet ./chart/kabinet -f custom-values.yaml
```

## Installation in Specific Namespace

To install Kabinet in a specific namespace:

```bash
# Create namespace if it doesn't exist
kubectl create namespace monitoring

# Install in the namespace
helm install kabinet ./chart/kabinet \
  --namespace monitoring \
  --create-namespace
```

## Upgrading Kabinet

To upgrade an existing Kabinet installation:

```bash
helm upgrade kabinet ./chart/kabinet
```

With custom values:

```bash
helm upgrade kabinet ./chart/kabinet \
  --set kabinet.storageLimitGB=100
```

## Uninstalling Kabinet

To completely remove Kabinet:

```bash
helm uninstall kabinet
```

**Note**: By default, the PersistentVolumeClaim is not deleted. To delete it manually:

```bash
kubectl delete pvc kabinet-data
```

## Verification

### Check Pod Status

```bash
kubectl get pods -l app.kubernetes.io/name=kabinet
```

Expected output:
```
NAME                       READY   STATUS    RESTARTS   AGE
kabinet-xxxxxxxxxx-xxxxx   1/1     Running   0          1m
```

### Check Service

```bash
kubectl get svc -l app.kubernetes.io/name=kabinet
```

### View Logs

```bash
kubectl logs -l app.kubernetes.io/name=kabinet -f
```

### Verify RBAC Permissions

Check if the ServiceAccount has the correct permissions:

```bash
kubectl auth can-i list events \
  --as=system:serviceaccount:default:kabinet
```

Should return: `yes`

## Troubleshooting

### Pod is not starting

1. Check pod events:
   ```bash
   kubectl describe pod -l app.kubernetes.io/name=kabinet
   ```

2. Check logs:
   ```bash
   kubectl logs -l app.kubernetes.io/name=kabinet
   ```

### Permission Denied Errors

Ensure RBAC is properly configured:

```bash
kubectl get clusterrole | grep kabinet
kubectl get clusterrolebinding | grep kabinet
```

### Persistent Volume Issues

Check PVC status:
```bash
kubectl get pvc
```

If PVC is pending, check if:
- StorageClass is available
- There are available PVs
- The cluster has a dynamic provisioner

### Cannot Access Web Interface

1. Verify the service is running:
   ```bash
   kubectl get svc kabinet
   ```

2. Try port-forwarding:
   ```bash
   kubectl port-forward svc/kabinet 8080:8080
   ```

3. Check if the pod is ready:
   ```bash
   kubectl get pods -l app.kubernetes.io/name=kabinet
   ```

## Advanced Configuration

### Using a Specific Storage Class

```bash
helm install kabinet ./chart/kabinet \
  --set kabinet.persistence.storageClass=fast-ssd
```

### Disable Persistence (Not Recommended)

For testing purposes only:

```bash
helm install kabinet ./chart/kabinet \
  --set kabinet.persistence.enabled=false
```

**Warning**: All event data will be lost when the pod restarts.

### Custom Node Selection

Deploy on specific nodes:

```bash
helm install kabinet ./chart/kabinet \
  --set nodeSelector.disktype=ssd
```

### Add Tolerations

```bash
helm install kabinet ./chart/kabinet \
  --set tolerations[0].key=dedicated \
  --set tolerations[0].operator=Equal \
  --set tolerations[0].value=monitoring \
  --set tolerations[0].effect=NoSchedule
```

## Next Steps

After successful installation:

1. **Explore the Analytics Dashboard** at the root path to see real-time event insights
2. **Try the Discover Page** at `/p/discover` to write custom SQL queries
3. **Set up Ingress** for easier access if not already configured
4. **Configure Prometheus** monitoring if using Prometheus Operator
5. **Adjust storage limits** based on your cluster's event volume

## Support

For issues, questions, or contributions:
- GitHub Repository: https://github.com/iwanhae/kabinet
- Issue Tracker: https://github.com/iwanhae/kabinet/issues

## Chart Values Reference

For a complete list of configurable values, see:
- [chart/kabinet/README.md](chart/kabinet/README.md)
- [chart/kabinet/values.yaml](chart/kabinet/values.yaml)
