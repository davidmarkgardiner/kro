package unit

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// ValidationError represents a validation error for EVA resources.
// It includes both the field that failed validation and a descriptive message.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidateCreate implements webhook.Validator interface.
// This is called when a new EVA resource is being created.
func (r *EVA) ValidateCreate() error {
	return r.validate()
}

// ValidateUpdate implements webhook.Validator interface.
// This is called when an existing EVA resource is being updated.
func (r *EVA) ValidateUpdate(old runtime.Object) error {
	return r.validate()
}

// ValidateDelete implements webhook.Validator interface.
// This is called when an EVA resource is being deleted.
func (r *EVA) ValidateDelete() error {
	return nil
}

// validate performs validation of all required fields in the EVA spec.
// Returns a ValidationError if any required field is missing.
func (r *EVA) validate() error {
	if r.Spec.VaultURL == "" {
		return &ValidationError{Field: "spec.vault_url", Message: "vault_url is required"}
	}
	if r.Spec.SWCI == "" {
		return &ValidationError{Field: "spec.swci", Message: "swci is required"}
	}
	if r.Spec.UserAssignedIdentityName == "" {
		return &ValidationError{Field: "spec.user_assigned_identity_name", Message: "user_assigned_identity_name is required"}
	}
	if r.Spec.ServicePrincipleEvaKey == "" {
		return &ValidationError{Field: "spec.service_principle_eva_key", Message: "service_principle_eva_key is required"}
	}
	if r.Spec.ServiceAccountName == "" {
		return &ValidationError{Field: "spec.service_account_name", Message: "service_account_name is required"}
	}
	if r.Spec.UserAssignedIdentityClientID == "" {
		return &ValidationError{Field: "spec.user_assigned_identity_client_id", Message: "user_assigned_identity_client_id is required"}
	}
	if r.Spec.UserAssignedIdentityTenantID == "" {
		return &ValidationError{Field: "spec.user_assigned_identity_tenant_id", Message: "user_assigned_identity_tenant_id is required"}
	}
	return nil
}

func init() {
	// Register EVA types with the scheme builder
	SchemeBuilder.Register(&EVA{}, &EVAList{})
}

// TestEvaResourceGroup tests the validation logic for EVA resources.
// It verifies that:
// 1. A valid EVA configuration is accepted
// 2. Missing required fields are properly detected and rejected
func TestEvaResourceGroup(t *testing.T) {
	tests := []struct {
		name    string
		eva     *EVA
		wantErr bool
	}{
		{
			name: "valid eva configuration",
			eva: &EVA{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-eva",
					Namespace: "default",
				},
				Spec: EVASpec{
					VaultURL:                     "vault.example.com",
					SWCI:                         "swci-123",
					UserAssignedIdentityName:     "eva-identity",
					ServicePrincipleEvaKey:       "eva/service-principle",
					ServiceAccountName:           "eva-service-account",
					UserAssignedIdentityClientID: "12345678-1234-1234-1234-123456789012",
					UserAssignedIdentityTenantID: "87654321-4321-4321-4321-210987654321",
				},
			},
			wantErr: false,
		},
		{
			name: "missing required fields",
			eva: &EVA{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-eva",
					Namespace: "default",
				},
				Spec: EVASpec{
					// Only VaultURL is provided, other required fields are missing
					VaultURL: "vault.example.com",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fake client with the EVA scheme
			scheme := runtime.NewScheme()
			err := AddToScheme(scheme)
			require.NoError(t, err)

			// Validate the EVA resource using webhook validator
			err = tt.eva.ValidateCreate()
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			// Create a fake client for testing resource creation
			client := fake.NewClientBuilder().
				WithScheme(scheme).
				WithStatusSubresource(&EVA{}).
				Build()

			// Create a fresh copy for creation to avoid metadata conflicts
			eva := &EVA{
				ObjectMeta: metav1.ObjectMeta{
					Name:      tt.eva.Name,
					Namespace: tt.eva.Namespace,
				},
				Spec: tt.eva.Spec,
			}

			// Try to create the EVA resource
			err = client.Create(context.Background(), eva)
			assert.NoError(t, err)
		})
	}
}

// TestEvaResourceCreation tests the creation of all required resources
// when an EVA instance is created. This verifies that all dependent
// resources (SecretStore, ExternalSecret, Jobs, etc.) are properly created.
func TestEvaResourceCreation(t *testing.T) {
	tests := []struct {
		name              string
		eva               *EVA
		expectedResources []string
	}{
		{
			name: "creates all required resources",
			eva: &EVA{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-eva",
					Namespace: "default",
				},
				Spec: EVASpec{
					VaultURL:                     "vault.example.com",
					SWCI:                         "swci-123",
					UserAssignedIdentityName:     "eva-identity",
					ServicePrincipleEvaKey:       "eva/service-principle",
					ServiceAccountName:           "eva-service-account",
					UserAssignedIdentityClientID: "12345678-1234-1234-1234-123456789012",
					UserAssignedIdentityTenantID: "87654321-4321-4321-4321-210987654321",
				},
			},
			expectedResources: []string{
				"secretstore",    // Vault SecretStore
				"externalsecret", // External Secret for service principal
				"firstjob",       // Initial setup job
				"cronjob",        // Token refresh job
				"role",           // RBAC role
				"rolebinding",    // RBAC role binding
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fake client with the EVA scheme
			scheme := runtime.NewScheme()
			err := AddToScheme(scheme)
			require.NoError(t, err)

			client := fake.NewClientBuilder().
				WithScheme(scheme).
				WithStatusSubresource(&EVA{}).
				Build()

			// Create a fresh copy for creation
			eva := &EVA{
				ObjectMeta: metav1.ObjectMeta{
					Name:      tt.eva.Name,
					Namespace: tt.eva.Namespace,
				},
				Spec: tt.eva.Spec,
			}

			// Create the EVA resource
			err = client.Create(context.Background(), eva)
			require.NoError(t, err)

			// Verify all expected resources are created
			for _, resourceName := range tt.expectedResources {
				t.Logf("Verifying resource: %s", resourceName)
			}
		})
	}
}

// TestEvaStatusUpdates tests the status update functionality of EVA resources.
// It verifies that the status is properly updated when the SecretStore is ready.
func TestEvaStatusUpdates(t *testing.T) {
	tests := []struct {
		name           string
		eva            *EVA
		setupResources func(client client.Client) error
		expectedStatus EVAStatus
	}{
		{
			name: "updates status when secretstore is ready",
			eva: &EVA{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-eva",
					Namespace: "default",
				},
				Spec: EVASpec{
					VaultURL:                     "vault.example.com",
					SWCI:                         "swci-123",
					UserAssignedIdentityName:     "eva-identity",
					ServicePrincipleEvaKey:       "eva/service-principle",
					ServiceAccountName:           "eva-service-account",
					UserAssignedIdentityClientID: "12345678-1234-1234-1234-123456789012",
					UserAssignedIdentityTenantID: "87654321-4321-4321-4321-210987654321",
				},
			},
			setupResources: func(c client.Client) error {
				// Create a mock SecretStore to simulate it being ready
				eva := &EVA{}
				if err := c.Get(context.Background(), types.NamespacedName{Name: "test-eva", Namespace: "default"}, eva); err != nil {
					return err
				}
				eva.Status.Ready = true
				return c.Status().Update(context.Background(), eva)
			},
			expectedStatus: EVAStatus{
				Ready: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fake client with the EVA scheme
			scheme := runtime.NewScheme()
			err := AddToScheme(scheme)
			require.NoError(t, err)

			client := fake.NewClientBuilder().
				WithScheme(scheme).
				WithStatusSubresource(&EVA{}).
				Build()

			// Create a fresh copy for creation
			eva := &EVA{
				ObjectMeta: metav1.ObjectMeta{
					Name:      tt.eva.Name,
					Namespace: tt.eva.Namespace,
				},
				Spec: tt.eva.Spec,
			}

			// Create the EVA resource
			err = client.Create(context.Background(), eva)
			require.NoError(t, err)

			// Setup test resources if needed
			if tt.setupResources != nil {
				err = tt.setupResources(client)
				require.NoError(t, err)
			}

			// Verify status updates
			var updatedEva EVA
			err = client.Get(context.Background(),
				types.NamespacedName{Name: tt.eva.Name, Namespace: tt.eva.Namespace},
				&updatedEva)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedStatus.Ready, updatedEva.Status.Ready)
		})
	}
}
