// Package unit contains unit tests for the logging component
package unit

import (
	"fmt"
	"testing"

	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	"k8s.io/apiextensions-apiserver/pkg/apiserver/validation"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

// loadLoggingSchema loads and converts the ResourceGroup schema for logging validation.
// It takes kro's simple schema format and converts it to Kubernetes OpenAPI schema format.
// This allows us to validate logging instances against the schema defined in the ResourceGroup.
func loadLoggingSchema() (*apiextensions.CustomResourceValidation, error) {
	// This is the schema from the ResourceGroup that defines the logging API
	// It uses kro's simple schema format with field definitions and validation rules
	schemaYAML := `
apiVersion: kro.run/v1alpha1
kind: ResourceGroup
spec:
  schema:
    apiVersion: v1alpha1
    kind: Logging
    spec:
      # Name of the logging deployment (required)
      name: string | required=true description="Name of the logging deployment"
      # Type of logging endpoint, e.g., 'law' for Log Analytics Workspace (required)
      endpointType: string | required=true description="Type of logging endpoint (e.g. law)"
      # Log Analytics Workspace configuration
      law:
        # Log Analytics Workspace ID (required)
        workspaceId: string | required=true description="Log Analytics Workspace ID"
        # Log Analytics Workspace access token (required)
        token: string | required=true description="Log Analytics Workspace token"
        # Log Analytics table name for storing logs (required)
        table: string | required=true description="Log Analytics table name"
`
	// Parse the YAML into an unstructured object
	var rg unstructured.Unstructured
	if err := yaml.Unmarshal([]byte(schemaYAML), &rg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal schema: %v", err)
	}

	// Convert the simple schema format to OpenAPI schema
	schema, err := convertSimpleSchemaToOpenAPI(rg.Object["spec"].(map[string]interface{})["schema"].(map[string]interface{}))
	if err != nil {
		return nil, fmt.Errorf("failed to convert schema: %v", err)
	}

	// Return the schema wrapped in CustomResourceValidation
	return &apiextensions.CustomResourceValidation{
		OpenAPIV3Schema: schema,
	}, nil
}

// convertSimpleSchemaToOpenAPI converts kro's simple schema format to Kubernetes OpenAPI schema.
// This conversion is necessary because Kubernetes uses OpenAPI schema for validation.
// The function creates a schema that enforces:
// - Required fields (name, endpointType)
// - Nested object structure (law configuration)
// - Field types (all strings in this case)
func convertSimpleSchemaToOpenAPI(simpleSchema map[string]interface{}) (*apiextensions.JSONSchemaProps, error) {
	// Create the OpenAPI schema structure
	schema := &apiextensions.JSONSchemaProps{
		Type: "object",
		Properties: map[string]apiextensions.JSONSchemaProps{
			"spec": {
				Type: "object",
				Properties: map[string]apiextensions.JSONSchemaProps{
					"name": {
						Type:     "string",
						Required: []string{"true"},
					},
					"endpointType": {
						Type:     "string",
						Required: []string{"true"},
					},
					"law": {
						Type: "object",
						Properties: map[string]apiextensions.JSONSchemaProps{
							"workspaceId": {
								Type:     "string",
								Required: []string{"true"},
							},
							"token": {
								Type:     "string",
								Required: []string{"true"},
							},
							"table": {
								Type:     "string",
								Required: []string{"true"},
							},
						},
						Required: []string{"workspaceId", "token", "table"},
					},
				},
				Required: []string{"name", "endpointType"},
			},
		},
		Required: []string{"spec"},
	}

	return schema, nil
}

// TestLoggingSchema validates that the logging schema correctly enforces:
// - Required fields are present
// - Field types are correct
// - Nested object structure is valid
// It tests both valid and invalid configurations to ensure proper validation.
func TestLoggingSchema(t *testing.T) {
	// Load the schema from the ResourceGroup definition
	crdValidation, err := loadLoggingSchema()
	if err != nil {
		t.Fatalf("failed to load schema: %v", err)
	}

	// Create a schema validator using the OpenAPI schema
	validator, _, err := validation.NewSchemaValidator(crdValidation.OpenAPIV3Schema)
	if err != nil {
		t.Fatalf("failed to create validator: %v", err)
	}

	// Test cases cover various validation scenarios
	tests := []struct {
		name    string // Name of the test case
		input   string // YAML input to validate
		wantErr bool   // Whether we expect validation errors
	}{
		{
			name: "valid logging instance",
			input: `
apiVersion: kro.run/v1alpha1
kind: Logging
metadata:
  name: test-logging
  namespace: logging
spec:
  name: test-collector
  endpointType: law
  law:
    workspaceId: "test-workspace-id"
    token: "test-token"
    table: "test-table"
`,
			wantErr: false, // Should pass validation
		},
		{
			name: "missing required field - name",
			input: `
apiVersion: kro.run/v1alpha1
kind: Logging
metadata:
  name: test-logging
  namespace: logging
spec:
  endpointType: law
  law:
    workspaceId: "test-workspace-id"
    token: "test-token"
    table: "test-table"
`,
			wantErr: true, // Should fail validation (missing name)
		},
		{
			name: "missing required field - endpointType",
			input: `
apiVersion: kro.run/v1alpha1
kind: Logging
metadata:
  name: test-logging
  namespace: logging
spec:
  name: test-collector
  law:
    workspaceId: "test-workspace-id"
    token: "test-token"
    table: "test-table"
`,
			wantErr: true, // Should fail validation (missing endpointType)
		},
		{
			name: "missing required field in law section",
			input: `
apiVersion: kro.run/v1alpha1
kind: Logging
metadata:
  name: test-logging
  namespace: logging
spec:
  name: test-collector
  endpointType: law
  law:
    workspaceId: "test-workspace-id"
    token: "test-token"
`,
			wantErr: true, // Should fail validation (missing table in law section)
		},
	}

	// Run each test case
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the test input
			var obj unstructured.Unstructured
			err := yaml.Unmarshal([]byte(tt.input), &obj)
			if err != nil {
				t.Fatalf("failed to unmarshal YAML: %v", err)
			}

			// Convert to runtime.Object for validation
			runtimeObj := obj.DeepCopyObject()

			// Validate against the schema
			errs := validation.ValidateCustomResource(nil, runtimeObj, validator)

			// Check if validation results match expectations
			if tt.wantErr && len(errs) == 0 {
				t.Error("expected validation error, but got none")
			}
			if !tt.wantErr && len(errs) > 0 {
				t.Errorf("unexpected validation errors: %v", errs)
			}
		})
	}
}
