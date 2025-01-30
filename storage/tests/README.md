# Storage Tests

This directory contains integration and unit tests for the Storage ResourceGroup functionality.

## Overview

The Storage ResourceGroup provides a way to manage Azure File storage resources in Kubernetes. These tests verify:
- Creation and validation of StorageConfig custom resources
- Field validation and defaults
- Resource updates
- Proper cleanup and finalizer handling

## Test Structure

### Integration Tests (`integration/`)
Tests the StorageConfig CR against a real Kubernetes cluster:
```go
TestStorageConfig_Integration
├── Create namespace with unique name
├── Create test secret (simulates Azure credentials)
├── Create StorageConfig CR
├── Verify all fields
├── Test update functionality
│   └── Update storage size
└── Cleanup with finalizer handling
```

Key features:
- Uses your current Kubernetes context
- Creates isolated test namespaces
- Handles finalizers properly
- Cleans up all resources

### Unit Tests (`unit/`)
Tests the core functionality without requiring a cluster:
```go
TestStorageConfig_Validation
├── Valid configuration
├── Missing name
├── Missing namespace
├── Missing storage account
└── Missing secret name

TestStorageClass_Generation
└── Verify StorageClass properties

TestPersistentVolume_Generation
└── Verify PV properties
```

## Running Tests

```bash
# Run all tests
go test ./...

# Run integration tests with verbose output
go test ./integration/... -v

# Run unit tests with verbose output
go test ./unit/... -v
```

## Test Environment

### Integration Tests
The integration tests:
1. Use your current Kubernetes context
2. Create isolated test namespaces with unique names
3. Install/update the StorageConfig CRD
4. Create test resources
5. Handle cleanup with finalizer removal
6. Clean up all resources after tests complete

### Unit Tests
The unit tests:
1. Validate StorageConfig fields
2. Test resource generation
3. Verify default values
4. Run without requiring a cluster

## Resource Cleanup

The tests handle cleanup carefully:
1. Remove finalizers from StorageConfig before updates/deletion
2. Remove finalizers from namespaces
3. Use timeouts to prevent hanging
4. Retry cleanup operations
5. Log cleanup status

## Adding New Tests

When adding new tests:
1. Follow the existing patterns for resource creation and cleanup
2. Use unique names for resources
3. Add proper cleanup in defer blocks
4. Include clear logging
5. Use require for critical setup steps
6. Use assert for test validations
7. Handle finalizers properly
8. Add appropriate comments

## Common Issues

### Namespace Cleanup
If namespaces get stuck in Terminating state:
1. Tests will automatically remove finalizers
2. Timeout after 30 seconds
3. Log cleanup status

### Resource Conflicts
To avoid resource conflicts:
1. Use unique names with timestamps
2. Remove finalizers before updates
3. Get latest version before updates
4. Use DeepCopy for updates

## Dependencies

Required:
- Go 1.21+
- Access to a Kubernetes cluster
- kubectl configured with proper access

## Test Files

```
storage/tests/
├── integration/
│   └── storage_integration_test.go  # Integration tests
├── unit/
│   ├── types.go                     # Type definitions
│   └── storage_test.go              # Unit tests
└── README.md                        # This file
``` 