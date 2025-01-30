# Eva Operator Tests

This directory contains comprehensive tests for validating the Eva (External Vault Authentication) operator functionality. The tests are organized into two main categories to ensure both unit-level correctness and integration-level functionality.

## Test Structure

### 1. Unit Tests (`unit/`)

Unit tests focus on testing individual components in isolation:

- `types.go`: Defines the EVA custom resource types and validation logic
  - EVA resource structure
  - Validation rules for required fields
  - Type registration with the Kubernetes scheme

- `eva_test.go`: Contains unit tests for:
  - Schema validation
    - Validates required fields are present
    - Ensures proper error messages for missing fields
  - Resource creation
    - Verifies all dependent resources are created
    - Tests resource relationships and dependencies
  - Status updates
    - Tests status field updates
    - Verifies status conditions are properly set

### 2. Integration Tests (`integration/`)

Integration tests verify the end-to-end functionality:

- `eva_integration_test.go`: Tests complete operator workflow
  - Resource creation sequence
  - Dependent resource configuration
  - Status propagation
  - Resource verification

Tests the creation and configuration of:
- SecretStore for Vault integration
- ExternalSecret for service principal credentials
- Initial setup Job
- Token refresh CronJob
- RBAC Role and RoleBinding

## Prerequisites

- Go 1.21 or later
- Access to a Kubernetes cluster (for integration tests)
- External Secrets Operator CRDs installed in the cluster
- Azure credentials configured (for integration tests)

## Running the Tests

### Unit Tests

Unit tests can be run independently and don't require a Kubernetes cluster:

```bash
cd unit
go test -v ./...
```

These tests verify:
1. EVA resource validation
2. Resource creation logic
3. Status update handling

### Integration Tests

Integration tests simulate a complete operator workflow:

```bash
cd integration
go test -v ./...
```

These tests verify:
1. Complete resource creation sequence
2. Proper configuration of all resources
3. Status updates and propagation
4. Resource relationships

## Test Coverage

To run tests with coverage reporting:

```bash
go test -v ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Test Design Principles

1. **Isolation**: Each test focuses on a specific aspect of the operator
   - Unit tests for individual components
   - Integration tests for complete workflows

2. **Completeness**: Tests cover all aspects of the operator
   - Resource validation
   - Resource creation
   - Status management
   - Error handling

3. **Maintainability**: Tests are organized and documented
   - Clear test structure
   - Descriptive test names
   - Detailed comments
   - Easy to extend

4. **Reliability**: Tests are deterministic and repeatable
   - No external dependencies in unit tests
   - Mocked interfaces where appropriate
   - Proper cleanup in integration tests

## Adding New Tests

When adding new tests:

1. Unit Tests:
   - Add to existing test files or create new ones in `unit/`
   - Focus on testing specific functionality
   - Use mocks and fakes appropriately
   - Keep tests focused and isolated

2. Integration Tests:
   - Add test cases to `eva_integration_test.go`
   - Test complete workflows
   - Verify resource relationships
   - Ensure proper cleanup

## Troubleshooting

Common issues and solutions:

1. CRD not found errors:
   - Ensure External Secrets Operator CRDs are installed
   - Check scheme registration in tests
   - Verify CRD versions match

2. Resource creation failures:
   - Check required fields are set
   - Verify resource relationships
   - Ensure proper RBAC permissions

3. Status update issues:
   - Verify status subresource is enabled
   - Check status field definitions
   - Ensure proper status update calls

4. Test environment issues:
   - Verify Go version compatibility
   - Check dependency versions
   - Ensure clean test environment 