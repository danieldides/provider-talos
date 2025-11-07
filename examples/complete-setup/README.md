# Complete Talos Cluster Setup Example

This example demonstrates the complete workflow for setting up a Talos machine using the Crossplane provider.

## Prerequisites

- A Talos machine in maintenance mode (accessible at the specified IP)
- Crossplane installed in your Kubernetes cluster
- provider-talos installed

## Resources

This example creates three resources in sequence:

1. **Secrets** - Generates cluster secrets locally
2. **Configuration** - Generates machine configuration using the secrets
3. **ConfigurationApply** - Applies the configuration to a real Talos machine

## Workflow

### Step 1: Create ProviderConfig

```bash
kubectl apply -f providerconfig.yaml
```

### Step 2: Generate Secrets

```bash
kubectl apply -f secrets.yaml
```

Wait for SYNCED=True:
```bash
kubectl get secrets.machine.talos.crossplane.io cluster-secrets
```

### Step 3: Generate Configuration

```bash
kubectl apply -f configuration.yaml
```

Wait for READY=True:
```bash
kubectl get configurations.machine.talos.crossplane.io worker-config
```

### Step 4: Apply Configuration to Machine

**Note:** Update the `node` field in `configurationapply.yaml` with your Talos machine IP.

For a machine in maintenance mode (no certificates yet):
```bash
kubectl apply -f configurationapply-insecure.yaml
```

For a configured machine (with certificates):
```bash
# First, get the certificates from the Secrets resource
kubectl get secrets.machine.talos.crossplane.io cluster-secrets -o yaml

# Update configurationapply.yaml with the certificates
kubectl apply -f configurationapply.yaml
```

## Verification

Check the status of all resources:
```bash
kubectl get secrets.machine.talos.crossplane.io
kubectl get configurations.machine.talos.crossplane.io
kubectl get configurationapplies.machine.talos.crossplane.io
```

All resources should show SYNCED=True or READY=True.
