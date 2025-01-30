// Package integration provides integration tests for the Storage ResourceGroup
// These tests verify the StorageConfig CR functionality against a real Kubernetes cluster
package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/krolaw/kro/storage/tests/unit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// Global test variables
var (
	testEnv    *envtest.Environment  // Test environment for running the tests
	cfg        *rest.Config          // Kubernetes config for the test cluster
	k8sClient  client.Client         // Client for interacting with the cluster
	testScheme = runtime.NewScheme() // Scheme for registering resource types
)

// TestMain sets up the test environment and runs all tests
func TestMain(m *testing.M) {
	// Set up logging
	logf.SetLogger(zap.New(zap.WriteTo(os.Stdout), zap.UseDevMode(true)))

	// Configure the test environment to use the existing cluster
	useCluster := true
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "config", "crd", "bases"),
		},
		ErrorIfCRDPathMissing: true,
		UseExistingCluster:    &useCluster,
	}

	// Start the test environment
	var err error
	cfg, err = testEnv.Start()
	if err != nil {
		fmt.Printf("Failed to start testenv: %v\n", err)
		os.Exit(1)
	}

	// Run all tests
	code := m.Run()

	// Clean up the test environment
	if err := testEnv.Stop(); err != nil {
		fmt.Printf("Failed to stop testenv: %v\n", err)
	}

	os.Exit(code)
}

// setupTestEnv prepares the test environment by registering resource types
// and creating a Kubernetes client
func setupTestEnv(t *testing.T) client.Client {
	var err error

	// Register standard Kubernetes types
	require.NoError(t, scheme.AddToScheme(testScheme))
	require.NoError(t, corev1.AddToScheme(testScheme))
	require.NoError(t, storagev1.AddToScheme(testScheme))

	// Register our custom StorageConfig type
	gv := schema.GroupVersion{
		Group:   "kro.run",
		Version: "v1alpha1",
	}
	testScheme.AddKnownTypes(gv, &unit.StorageConfig{}, &unit.StorageConfigList{})
	metav1.AddToGroupVersion(testScheme, gv)

	// Create a new client for the test cluster
	k8sClient, err = client.New(cfg, client.Options{Scheme: testScheme})
	require.NoError(t, err)
	require.NotNil(t, k8sClient)

	return k8sClient
}

// TestStorageConfig_Integration tests the complete lifecycle of a StorageConfig resource
// including creation, validation, updates, and cleanup
func TestStorageConfig_Integration(t *testing.T) {
	// Setup test environment and get Kubernetes client
	ctx := context.Background()
	k8sClient := setupTestEnv(t)

	// Create test namespace with unique name to avoid conflicts
	namespaceName := fmt.Sprintf("storage-test-%d", time.Now().UnixNano())
	t.Logf("Creating test namespace: %s", namespaceName)

	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespaceName,
		},
	}

	err := k8sClient.Create(ctx, namespace)
	require.NoError(t, err)

	// Setup namespace cleanup to run after test completion
	defer func() {
		t.Logf("Cleaning up namespace: %s", namespaceName)

		// Keep trying to remove finalizers and delete namespace for up to 30 seconds
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		for {
			select {
			case <-ctx.Done():
				t.Logf("Timed out cleaning up namespace: %s", namespaceName)
				return
			default:
				// Get latest namespace state
				ns := &corev1.Namespace{}
				if err := k8sClient.Get(ctx, types.NamespacedName{Name: namespaceName}, ns); err != nil {
					if client.IgnoreNotFound(err) == nil {
						// Namespace is gone, which is what we want
						return
					}
					t.Logf("Error getting namespace: %v", err)
					time.Sleep(time.Second)
					continue
				}

				// If namespace exists but isn't being deleted yet, delete it
				if ns.DeletionTimestamp == nil {
					if err := k8sClient.Delete(ctx, ns); err != nil {
						t.Logf("Error deleting namespace: %v", err)
					}
				}

				// Remove finalizers if they exist
				if len(ns.Finalizers) > 0 {
					ns.Finalizers = nil
					if err := k8sClient.Update(ctx, ns); err != nil {
						if client.IgnoreNotFound(err) == nil {
							return
						}
						t.Logf("Error removing finalizers: %v", err)
					}
				}

				time.Sleep(time.Second)
			}
		}
	}()

	// Create a test secret to simulate Azure storage credentials
	t.Log("Creating test secret")
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-storage-secret",
			Namespace: namespace.Name,
		},
		StringData: map[string]string{
			"azurestorageaccountname": "teststorage",
			"azurestorageaccountkey":  "testkey",
		},
	}
	err = k8sClient.Create(ctx, secret)
	require.NoError(t, err)

	// Create a StorageConfig instance with test configuration
	t.Log("Creating StorageConfig instance")
	storageConfig := &unit.StorageConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kro.run/v1alpha1",
			Kind:       "StorageConfig",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-storage",
			Namespace: namespace.Name,
		},
		Spec: unit.StorageConfigSpec{
			Name:           "test-storage",
			Namespace:      namespace.Name,
			StorageAccount: "teststorage",
			SecretName:     secret.Name,
			ShareName:      "testshare",
			StorageSize:    "10Gi",
			AccessMode:     "ReadWriteMany",
		},
	}
	err = k8sClient.Create(ctx, storageConfig)
	require.NoError(t, err)

	// Verify the StorageConfig was created with correct fields
	t.Log("Verifying StorageConfig creation")
	createdConfig := &unit.StorageConfig{}
	err = k8sClient.Get(ctx, types.NamespacedName{
		Name:      storageConfig.Name,
		Namespace: storageConfig.Namespace,
	}, createdConfig)
	require.NoError(t, err)

	// Validate all spec fields match our configuration
	assert.Equal(t, "test-storage", createdConfig.Spec.Name)
	assert.Equal(t, namespace.Name, createdConfig.Spec.Namespace)
	assert.Equal(t, "teststorage", createdConfig.Spec.StorageAccount)
	assert.Equal(t, secret.Name, createdConfig.Spec.SecretName)
	assert.Equal(t, "testshare", createdConfig.Spec.ShareName)
	assert.Equal(t, "10Gi", createdConfig.Spec.StorageSize)
	assert.Equal(t, "ReadWriteMany", createdConfig.Spec.AccessMode)

	// Test updating the StorageConfig's storage size
	t.Log("Testing StorageConfig update")
	latestConfig := &unit.StorageConfig{}
	err = k8sClient.Get(ctx, types.NamespacedName{
		Name:      storageConfig.Name,
		Namespace: storageConfig.Namespace,
	}, latestConfig)
	require.NoError(t, err)

	// Remove any finalizers before update
	if len(latestConfig.Finalizers) > 0 {
		latestConfig.Finalizers = nil
		err = k8sClient.Update(ctx, latestConfig)
		require.NoError(t, err)

		// Get the latest version again after removing finalizers
		err = k8sClient.Get(ctx, types.NamespacedName{
			Name:      storageConfig.Name,
			Namespace: storageConfig.Namespace,
		}, latestConfig)
		require.NoError(t, err)
	}

	// Update the storage size while preserving all other fields
	updatedConfig := latestConfig.DeepCopy()
	updatedConfig.Spec.StorageSize = "20Gi"
	err = k8sClient.Update(ctx, updatedConfig)
	require.NoError(t, err)

	// Verify the storage size was updated
	verifyConfig := &unit.StorageConfig{}
	err = k8sClient.Get(ctx, types.NamespacedName{
		Name:      storageConfig.Name,
		Namespace: storageConfig.Namespace,
	}, verifyConfig)
	require.NoError(t, err)
	assert.Equal(t, "20Gi", verifyConfig.Spec.StorageSize)

	// Clean up the StorageConfig
	t.Log("Cleaning up StorageConfig")
	// First get the latest version
	deleteConfig := &unit.StorageConfig{}
	err = k8sClient.Get(ctx, types.NamespacedName{
		Name:      storageConfig.Name,
		Namespace: storageConfig.Namespace,
	}, deleteConfig)
	require.NoError(t, err)

	// Remove finalizers if any exist
	if len(deleteConfig.Finalizers) > 0 {
		deleteConfig.Finalizers = nil
		err = k8sClient.Update(ctx, deleteConfig)
		require.NoError(t, err)
	}

	// Now delete the resource
	err = k8sClient.Delete(ctx, deleteConfig)
	require.NoError(t, err)
}

// waitForNamespaceDeletion waits for a namespace to be fully deleted
// and handles removal of finalizers if necessary
func waitForNamespaceDeletion(ctx context.Context, c client.Client, name string) error {
	// Try to get the namespace
	ns := &corev1.Namespace{}
	err := c.Get(ctx, types.NamespacedName{Name: name}, ns)
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			// Namespace doesn't exist, which is what we want
			return nil
		}
		return err
	}

	// If namespace exists and is being deleted, wait for deletion
	if ns.DeletionTimestamp != nil {
		for i := 0; i < 60; i++ { // wait up to 60 seconds
			// First try to remove all finalizers if any exist
			if len(ns.Finalizers) > 0 {
				ns.Finalizers = nil
				if err := c.Update(ctx, ns); err != nil {
					if client.IgnoreNotFound(err) == nil {
						// If namespace is gone, that's what we want
						return nil
					}
					// If error is not NotFound, log and continue waiting
					fmt.Printf("Failed to remove finalizers: %v\n", err)
				}
			}

			// Check if namespace is gone
			err := c.Get(ctx, types.NamespacedName{Name: name}, ns)
			if err != nil && client.IgnoreNotFound(err) == nil {
				return nil
			}
			time.Sleep(time.Second)
		}
		return fmt.Errorf("timed out waiting for namespace %s to be deleted", name)
	}

	return nil
}
