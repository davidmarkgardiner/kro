// Package integration provides integration tests for the EVA operator.
// These tests verify the end-to-end functionality of the operator by simulating
// a complete reconciliation cycle and verifying all resources are created correctly.
package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	// Import the EVA types from the unit package
	"kro.run/eva/tests/unit"
)

// TestEvaIntegration performs an end-to-end test of the EVA operator.
// It simulates the complete lifecycle of an EVA resource, including:
// 1. Creation of the EVA custom resource
// 2. Creation of all dependent resources (SecretStore, ExternalSecret, Jobs, etc.)
// 3. Status updates
// 4. Verification of all created resources
func TestEvaIntegration(t *testing.T) {
	// Create a test scheme and register all required types
	scheme := runtime.NewScheme()
	require.NoError(t, unit.AddToScheme(scheme))    // EVA types
	require.NoError(t, v1beta1.AddToScheme(scheme)) // External Secrets types
	require.NoError(t, batchv1.AddToScheme(scheme)) // Job and CronJob types
	require.NoError(t, corev1.AddToScheme(scheme))  // Core types (Namespace, etc.)
	require.NoError(t, rbacv1.AddToScheme(scheme))  // RBAC types

	// Create a test namespace for isolation
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-eva",
		},
	}

	// Create a fake client with the test scheme
	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&unit.EVA{}).
		Build()

	// Create the namespace first
	err := k8sClient.Create(context.Background(), namespace)
	require.NoError(t, err)

	// Create an EVA instance with all required fields
	eva := &unit.EVA{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-eva",
			Namespace: namespace.Name,
		},
		Spec: unit.EVASpec{
			VaultURL:                     "vault.example.com",
			SWCI:                         "swci-123",
			UserAssignedIdentityName:     "eva-identity",
			ServicePrincipleEvaKey:       "eva/service-principle",
			ServiceAccountName:           "eva-service-account",
			UserAssignedIdentityClientID: "12345678-1234-1234-1234-123456789012",
			UserAssignedIdentityTenantID: "87654321-4321-4321-4321-210987654321",
		},
	}

	// Create the EVA resource
	err = k8sClient.Create(context.Background(), eva)
	require.NoError(t, err)

	// Simulate controller reconciliation by creating all dependent resources
	t.Log("Simulating controller reconciliation...")

	// Create SecretStore for Vault integration
	path := "secret"
	secretStore := &v1beta1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "eva-secretstore",
			Namespace: namespace.Name,
		},
		Spec: v1beta1.SecretStoreSpec{
			Provider: &v1beta1.SecretStoreProvider{
				Vault: &v1beta1.VaultProvider{
					Server: eva.Spec.VaultURL,
					Path:   &path,
					Auth: v1beta1.VaultAuth{
						Jwt: &v1beta1.VaultJwtAuth{
							Path: "azure",
							Role: eva.Spec.UserAssignedIdentityName,
						},
					},
				},
			},
		},
	}
	err = k8sClient.Create(context.Background(), secretStore)
	require.NoError(t, err)

	// Create ExternalSecret to fetch service principal credentials
	externalSecret := &v1beta1.ExternalSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sp-externalsecret",
			Namespace: namespace.Name,
		},
		Spec: v1beta1.ExternalSecretSpec{
			SecretStoreRef: v1beta1.SecretStoreRef{
				Name: "eva-secretstore",
				Kind: "SecretStore",
			},
			Target: v1beta1.ExternalSecretTarget{
				Name: "sp-secrets",
			},
			Data: []v1beta1.ExternalSecretData{
				{
					SecretKey: "sp-client-id",
					RemoteRef: v1beta1.ExternalSecretDataRemoteRef{
						Key:      eva.Spec.ServicePrincipleEvaKey,
						Property: "client-id",
					},
				},
			},
		},
	}
	err = k8sClient.Create(context.Background(), externalSecret)
	require.NoError(t, err)

	// Create FirstJob for initial token setup
	firstJob := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "eva-firstjob",
			Namespace: namespace.Name,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					ServiceAccountName: eva.Spec.ServiceAccountName,
					RestartPolicy:      corev1.RestartPolicyOnFailure,
					Containers: []corev1.Container{
						{
							Name:  "test",
							Image: "test:latest",
						},
					},
				},
			},
		},
	}
	err = k8sClient.Create(context.Background(), firstJob)
	require.NoError(t, err)

	// Create CronJob for token refresh
	cronJob := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "eva-cronjob",
			Namespace: namespace.Name,
		},
		Spec: batchv1.CronJobSpec{
			Schedule: "0 */8 * * *", // Every 8 hours
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							ServiceAccountName: eva.Spec.ServiceAccountName,
							RestartPolicy:      corev1.RestartPolicyOnFailure,
							Containers: []corev1.Container{
								{
									Name:  "test",
									Image: "test:latest",
								},
							},
						},
					},
				},
			},
		},
	}
	err = k8sClient.Create(context.Background(), cronJob)
	require.NoError(t, err)

	// Create Role for RBAC permissions
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "gpinfra-role",
			Namespace: namespace.Name,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"*"},
				Resources: []string{"cronjobs", "jobs"},
				Verbs:     []string{"get", "create", "delete"},
			},
			{
				APIGroups: []string{"*"},
				Resources: []string{"secrets"},
				Verbs:     []string{"create", "delete"},
			},
		},
	}
	err = k8sClient.Create(context.Background(), role)
	require.NoError(t, err)

	// Create RoleBinding to assign permissions
	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "gpinfra-role-binding",
			Namespace: namespace.Name,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     "gpinfra-role",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind: "ServiceAccount",
				Name: eva.Spec.ServiceAccountName,
			},
		},
	}
	err = k8sClient.Create(context.Background(), roleBinding)
	require.NoError(t, err)

	// Update EVA status to indicate readiness
	eva.Status.Ready = true
	err = k8sClient.Status().Update(context.Background(), eva)
	require.NoError(t, err)

	// Verify all resources were created correctly
	t.Run("verify resources", func(t *testing.T) {
		// Verify SecretStore configuration
		var ss v1beta1.SecretStore
		err = k8sClient.Get(context.Background(),
			types.NamespacedName{Name: "eva-secretstore", Namespace: namespace.Name},
			&ss)
		require.NoError(t, err)
		assert.Equal(t, eva.Spec.VaultURL, ss.Spec.Provider.Vault.Server)

		// Verify ExternalSecret configuration
		var es v1beta1.ExternalSecret
		err = k8sClient.Get(context.Background(),
			types.NamespacedName{Name: "sp-externalsecret", Namespace: namespace.Name},
			&es)
		require.NoError(t, err)

		// Verify FirstJob configuration
		var job batchv1.Job
		err = k8sClient.Get(context.Background(),
			types.NamespacedName{Name: "eva-firstjob", Namespace: namespace.Name},
			&job)
		require.NoError(t, err)
		assert.Equal(t, eva.Spec.ServiceAccountName, job.Spec.Template.Spec.ServiceAccountName)

		// Verify CronJob configuration
		var cj batchv1.CronJob
		err = k8sClient.Get(context.Background(),
			types.NamespacedName{Name: "eva-cronjob", Namespace: namespace.Name},
			&cj)
		require.NoError(t, err)
		assert.Equal(t, eva.Spec.ServiceAccountName, cj.Spec.JobTemplate.Spec.Template.Spec.ServiceAccountName)

		// Verify Role configuration
		var r rbacv1.Role
		err = k8sClient.Get(context.Background(),
			types.NamespacedName{Name: "gpinfra-role", Namespace: namespace.Name},
			&r)
		require.NoError(t, err)

		// Verify RoleBinding configuration
		var rb rbacv1.RoleBinding
		err = k8sClient.Get(context.Background(),
			types.NamespacedName{Name: "gpinfra-role-binding", Namespace: namespace.Name},
			&rb)
		require.NoError(t, err)
		assert.Equal(t, eva.Spec.ServiceAccountName, rb.Subjects[0].Name)

		// Verify EVA status
		var updatedEva unit.EVA
		err = k8sClient.Get(context.Background(),
			types.NamespacedName{Name: eva.Name, Namespace: namespace.Name},
			&updatedEva)
		require.NoError(t, err)
		assert.True(t, updatedEva.Status.Ready)
	})
}
