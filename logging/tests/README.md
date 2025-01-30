# Logging Component Tests

This directory contains the test suite for the kro logging component. The tests are organized into three main categories: unit tests, integration tests, and end-to-end (E2E) tests.

## Test Structure

```
tests/
├── unit/              # Unit tests for schema validation
├── integration/       # Integration tests for collector functionality
├── e2e/              # End-to-end tests for complete system
└── README.md         # This file
```

## Prerequisites

- Go 1.19 or later
- Access to a Kubernetes cluster (for integration and E2E tests)
- Azure Log Analytics Workspace (for integration and E2E tests)
- kubectl configured with cluster access
- kro installed in the cluster

## Environment Setup

1. Create a test Kubernetes cluster (if not using existing one):
```bash
kind create cluster --name kro-test
```

2. Install kro in the cluster:
```bash
# Add the kro Helm repository
helm repo add kro https://kro.run/charts
helm repo update

# Install kro
helm install kro kro/kro -n kro-system --create-namespace
```

3. Configure test credentials:
```bash
# Create a secret with Log Analytics credentials for testing
kubectl create secret generic law-credentials \
  --from-literal=workspaceId=your-workspace-id \
  --from-literal=token=your-token \
  -n logging
```

## Running Tests

### Unit Tests
Unit tests validate the schema and basic functionality without requiring external dependencies.

```bash
go test ./unit/... -v
```

These tests cover:
- Schema validation
- Required field validation
- Type validation
- Default value handling

### Integration Tests
Integration tests verify the logging collector functionality in a real cluster environment.

```bash
# Run integration tests
go test ./integration/... -v
```

These tests cover:
- Log collection from pods
- Log forwarding to Log Analytics
- Configuration validation
- Error handling

### E2E Tests
End-to-end tests validate the complete logging system across multiple namespaces.

```bash
# Run E2E tests
go test ./e2e/... -v
```

These tests cover:
- Multi-namespace logging
- Log routing and filtering
- Failure scenarios and recovery
- System metrics and monitoring

## Test Configuration

### Environment Variables

The following environment variables can be used to configure the tests:

- `TEST_WORKSPACE_ID`: Log Analytics Workspace ID
- `TEST_WORKSPACE_TOKEN`: Log Analytics Workspace Token
- `TEST_K8S_CONTEXT`: Kubernetes context to use
- `TEST_NAMESPACE`: Namespace for integration tests
- `SKIP_CLEANUP`: Set to "true" to preserve test resources after completion

### Test Data

Test data and fixtures are included in the respective test directories:
- `unit/testdata/`: Sample YAML files for schema validation
- `integration/testdata/`: Test pod configurations
- `e2e/testdata/`: Complete application configurations

## Adding New Tests

### Unit Tests
1. Add test cases to `unit/schema_test.go`
2. Include both valid and invalid configurations
3. Test edge cases and error conditions

### Integration Tests
1. Create new test functions in `integration/collector_test.go`
2. Add test pod configurations in `integration/testdata/`
3. Implement verification steps

### E2E Tests
1. Add new test scenarios in `e2e/logging_test.go`
2. Create application configurations in `e2e/testdata/`
3. Implement failure scenarios

## Test Coverage

To run tests with coverage reporting:

```bash
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

## Troubleshooting

### Common Issues

1. **Test Environment Setup**
   - Ensure Kubernetes cluster is accessible
   - Verify kro installation
   - Check Log Analytics credentials

2. **Integration Tests**
   - Verify namespace permissions
   - Check pod logs for errors
   - Ensure Log Analytics connectivity

3. **E2E Tests**
   - Monitor resource creation
   - Check system logs
   - Verify network connectivity

### Debug Logging

To enable debug logging during tests:

```bash
export TEST_DEBUG=true
go test ./... -v
```

## Contributing

When contributing new tests:

1. Follow existing test patterns
2. Include documentation
3. Add test data as needed
4. Update this README if necessary

## Best Practices

1. **Test Independence**
   - Each test should be self-contained
   - Clean up resources after tests
   - Don't rely on external state

2. **Resource Management**
   - Use unique names for test resources
   - Implement proper cleanup
   - Handle timeouts appropriately

3. **Error Handling**
   - Include meaningful error messages
   - Test error conditions
   - Validate error responses

4. **Documentation**
   - Document test purpose
   - Include example usage
   - Explain test configuration 