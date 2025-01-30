# Schema Validation Tests

This directory contains unit tests for validating the logging schema defined in the ResourceGroup. The tests ensure that the schema correctly enforces the required structure and validation rules for logging instances.

## Overview

The main test file `schema_test.go` validates that the logging schema correctly enforces:
- Required fields are present
- Field types are correct
- Nested object structure is valid

## Schema Structure

The logging schema defines the following structure:

```yaml
spec:
  name: string (required)        # Name of the logging deployment
  endpointType: string (required) # Type of logging endpoint (e.g., law)
  law:
    workspaceId: string (required) # Log Analytics Workspace ID
    token: string (required)       # Log Analytics Workspace token
    table: string (required)       # Log Analytics table name
```

## Test Cases

The test suite includes the following validation scenarios:

1. **Valid Configuration**
   - Tests a complete and valid logging instance
   - All required fields are present
   - Expected to pass validation

2. **Missing Required Fields**
   - Tests omitting required top-level fields (name, endpointType)
   - Tests omitting required nested fields (law.table)
   - Expected to fail validation

## Implementation Details

### Schema Loading

The test uses `loadLoggingSchema()` to:
1. Load the schema definition from YAML
2. Convert kro's simple schema format to Kubernetes OpenAPI schema
3. Create a validation schema that Kubernetes can use

### Schema Conversion

The `convertSimpleSchemaToOpenAPI()` function:
1. Converts kro's human-friendly schema format to Kubernetes OpenAPI format
2. Sets up field requirements and types
3. Defines the nested structure for the law configuration

### Validation

The test uses Kubernetes' built-in schema validation to:
1. Parse test cases into unstructured objects
2. Validate them against the OpenAPI schema
3. Verify that validation errors occur as expected

## Running the Tests

To run the schema validation tests:

```bash
# Run all unit tests
go test ./unit/... -v

# Run just the schema tests
go test ./unit/schema_test.go -v

# Run a specific test case
go test ./unit/... -v -run TestLoggingSchema/valid_logging_instance
```

## Adding New Test Cases

To add new test cases:

1. Add a new entry to the `tests` slice in `TestLoggingSchema`
2. Provide:
   - A descriptive name
   - The YAML to test
   - Whether validation errors are expected
3. Run the tests to verify

Example:
```go
{
    name: "new test case",
    input: `
        apiVersion: kro.run/v1alpha1
        kind: Logging
        metadata:
          name: test-logging
        spec:
          // ... test configuration ...
    `,
    wantErr: true, // or false
}
```

## Error Handling

The test framework will:
1. Report detailed validation errors when they occur
2. Fail if validation errors occur unexpectedly
3. Fail if validation errors don't occur when they should

## Dependencies

The tests use:
- `k8s.io/apiextensions-apiserver` for schema validation
- `k8s.io/apimachinery` for Kubernetes types
- `sigs.k8s.io/yaml` for YAML parsing 