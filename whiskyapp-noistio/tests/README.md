# WhiskyApp Tests

This directory contains both unit and integration tests for the WhiskyApp custom resource and controller.

## Directory Structure

```
tests/
├── integration/
│   └── whiskyapp_integration_test.go  # Integration tests with real k8s cluster
├── unit/
│   ├── types.go                       # WhiskyApp type definitions
│   └── whiskyapp_test.go              # Unit tests for validation and resource generation
├── cleanup-ns.sh                      # Helper script to clean up test namespaces
└── README.md                          # This file
```

## Test Coverage

### Integration Tests (`integration/`)
Tests the complete functionality of the WhiskyApp controller:

1. **Resource Creation**
   - Creates WhiskyApp CRD
   - Starts controller with proper RBAC
   - Creates test namespace
   - Deploys WhiskyApp instance

2. **Security Settings**
   - Verifies pod security context
     - Non-root user (nginx:101)
     - Read-only root filesystem
     - Dropped capabilities
   - Validates volume mounts for nginx
     - `/var/cache/nginx`
     - `/var/run`
     - `/var/log/nginx`

3. **Resource Management**
   - Tests resource requests/limits
   - Validates default values
   - Verifies scaling operations

4. **Network Policies**
   - Validates ingress rules for ingressgateway
   - Checks egress rules for DNS access

### Unit Tests (`unit/`)
Tests individual components without requiring a cluster:

1. **Type Validation**
   - Required fields (name, image)
   - Resource requirements format
   - Default values

2. **Resource Generation**
   - ServiceAccount creation
   - Deployment configuration
   - Service settings
   - NetworkPolicy rules

## Running Tests

### Prerequisites
- Go 1.21+
- Access to a Kubernetes cluster
- kubectl configured with proper access

### Commands

Run all tests:
```bash
go test ./... -v
```

Run only unit tests:
```bash
go test ./unit/... -v
```

Run only integration tests:
```bash
go test ./integration/... -v
```

### Test Environment

The integration tests:
1. Create a unique namespace for each test run
2. Deploy all required resources
3. Clean up automatically after completion
4. Use optimized timeouts for faster execution

## Performance Optimizations

The tests are optimized for speed while maintaining reliability:

1. **Resource Creation**
   - 100ms polling interval
   - 30s maximum wait time
   - Parallel resource creation

2. **Container Settings**
   - Minimal probe delays
     - Readiness: 1s initial, 2s period
     - Liveness: 2s initial, 4s period
   - Resource limits for quick startup

3. **Cleanup**
   - Automatic namespace cleanup
   - Finalizer removal for quick deletion
   - 30s cleanup timeout

## Troubleshooting

Common issues and solutions:

1. **Namespace Cleanup**
   - Use `cleanup-ns.sh` to remove stuck namespaces
   - Check for remaining finalizers
   - Verify RBAC permissions

2. **Test Timeouts**
   - Increase wait times for slower environments
   - Check pod events for startup issues
   - Verify resource quotas

3. **Controller Issues**
   - Enable debug logging
   - Check CRD registration
   - Verify RBAC permissions

## Best Practices

When adding new tests:

1. **Test Organization**
   - Keep unit and integration tests separate
   - Use descriptive test names
   - Add proper error messages

2. **Resource Management**
   - Create unique resource names
   - Clean up all resources
   - Handle finalizers properly

3. **Security**
   - Test security contexts
   - Validate network policies
   - Check resource limits

4. **Documentation**
   - Update this README
   - Document test prerequisites
   - Add comments for complex logic 