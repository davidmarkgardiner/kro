# External Vault Authentication (EVA)

This KRO component manages the setup and maintenance of External Secrets with Azure Vault authentication in a Kubernetes cluster.

## Overview

The EVA component automates several key tasks:
1. Sets up a SecretStore that connects to Azure Vault using Azure Workload Identity
2. Creates an ExternalSecret to fetch service principal credentials
3. Manages JWT token refresh through a combination of initial job and recurring cronjob
4. Sets up necessary RBAC permissions

## Components

### SecretStore
- Configures connection to Azure Vault
- Uses Azure Workload Identity for authentication
- Manages JWT-based authentication

### ExternalSecret
- Fetches service principal credentials from Vault
- Creates a Kubernetes secret with client ID and secret
- Automatically refreshes based on the interval

### Jobs
1. Initial Job (`eva-firstjob`)
   - Runs once during setup
   - Fetches initial JWT token
   - Creates eva-jwt secret

2. CronJob (`eva-cronjob`)
   - Runs every 8 hours
   - Refreshes JWT token
   - Updates eva-jwt secret

### RBAC
- Role with permissions for jobs and secrets management
- RoleBinding to associate permissions with service account

## Configuration

### Required Values
```yaml
spec:
  vault_url: string                    # Azure Vault URL
  swci: string                         # SWCI identifier
  user_assigned_identity_name: string  # Azure User Assigned Identity name
  service_principle_eva_key: string    # Vault key for service principle
  service_account_name: string         # Kubernetes ServiceAccount name
  user_assigned_identity_client_id: string    # Azure Client ID
  user_assigned_identity_tenant_id: string    # Azure Tenant ID
```

## Usage

1. Create the ResourceGroup:
```bash
kubectl apply -f eva-rg.yaml
```

2. Create an Instance with your configuration:
```bash
kubectl apply -f eva-instance.yaml
```

## Security Considerations

- Uses Azure Workload Identity for secure authentication
- Implements least privilege RBAC
- Automatically rotates JWT tokens
- Securely manages service principal credentials
- Uses secure pod configurations (non-root, read-only filesystem, etc.)

## Resource Requirements

Each job/cronjob pod requires:
- CPU: 100m (limit), 100m (request)
- Memory: 128Mi (limit), 128Mi (request)

## Dependencies

- Azure Vault instance
- Azure Workload Identity setup
- External Secrets Operator
- Kubernetes cluster with RBAC enabled 