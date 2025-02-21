Yes, since KRO is written in Go, the tests would be written in Golang. Let me search through the codebase to see how KRO implements its tests to provide you with a more specific example.

Search files...


Based on the search results, I can provide a detailed example of how to write tests for your storage resource group using Go. KRO uses a combination of unit tests and integration tests with the Ginkgo testing framework. Here's how you can structure the tests:

1. First, create a test suite file in the `tests` directory:

```go
// storage/tests/storage_test.go

package storage_test

import (
    "context"
    "fmt"
    "testing"
    "time"

    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
    corev1 "k8s.io/api/core/v1"
    storagev1 "k8s.io/api/storage/v1"
    "k8s.io/apimachinery/pkg/api/errors"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/types"
    "k8s.io/apimachinery/pkg/util/rand"

    krov1alpha1 "github.com/awslabs/kro/api/v1alpha1"
    "github.com/awslabs/kro/test/integration/environment"
)

var env *environment.Environment

func TestStorage(t *testing.T) {
    RegisterFailHandler(Fail)
    BeforeSuite(func() {
        var err error
        env, err = environment.New(
            environment.ControllerConfig{
                AllowCRDDeletion: true,
                ReconcileConfig: ctrlinstance.ReconcileConfig{
                    DefaultRequeueDuration: 15 * time.Second,
                },
            },
        )
        Expect(err).NotTo(HaveOccurred())
    })
    AfterSuite(func() {
        Expect(env.Stop()).NotTo(HaveOccurred())
    })

    RunSpecs(t, "Storage Suite")
}
```

2. Add test cases for the storage configuration:

```go
var _ = Describe("Storage", func() {
    var (
        ctx       context.Context
        namespace string
    )

    BeforeEach(func() {
        ctx = context.Background()
        namespace = fmt.Sprintf("test-%s", rand.String(5))
        Expect(env.Client.Create(ctx, &corev1.Namespace{
            ObjectMeta: metav1.ObjectMeta{
                Name: namespace,
            },
        })).To(Succeed())
    })

    Context("Resource Creation", func() {
        It("should create storage resources successfully", func() {
            // Create storage secret first
            secret := &corev1.Secret{
                ObjectMeta: metav1.ObjectMeta{
                    Name:      "test-storage-secret",
                    Namespace: namespace,
                },
                StringData: map[string]string{
                    "azurestorageaccountname": "teststorage",
                    "azurestorageaccountkey":  "test-key",
                },
            }
            Expect(env.Client.Create(ctx, secret)).To(Succeed())

            // Create storage instance
            instance := &unstructured.Unstructured{
                Object: map[string]interface{}{
                    "apiVersion": "kro.run/v1alpha1",
                    "kind":       "StorageConfig",
                    "metadata": map[string]interface{}{
                        "name":      "test-storage",
                        "namespace": namespace,
                    },
                    "spec": map[string]interface{}{
                        "name":           "test-storage",
                        "namespace":      namespace,
                        "storageAccount": "teststorage",
                        "secretName":     "test-storage-secret",
                        "shareName":      "testshare",
                        "storageSize":    "5Gi",
                        "accessMode":     "ReadWriteMany",
                    },
                },
            }
            Expect(env.Client.Create(ctx, instance)).To(Succeed())

            // Verify StorageClass creation
            Eventually(func(g Gomega) {
                sc := &storagev1.StorageClass{}
                err := env.Client.Get(ctx, types.NamespacedName{
                    Name: "test-storage-sc",
                }, sc)
                g.Expect(err).NotTo(HaveOccurred())
                g.Expect(sc.Provisioner).To(Equal("file.csi.azure.com"))
                g.Expect(sc.Parameters["storageAccount"]).To(Equal("teststorage"))
            }, 20*time.Second, time.Second).Should(Succeed())

            // Verify PersistentVolume creation
            Eventually(func(g Gomega) {
                pv := &corev1.PersistentVolume{}
                err := env.Client.Get(ctx, types.NamespacedName{
                    Name: "test-storage-pv",
                }, pv)
                g.Expect(err).NotTo(HaveOccurred())
                g.Expect(pv.Spec.Capacity.Storage().String()).To(Equal("5Gi"))
                g.Expect(pv.Spec.AccessModes).To(ContainElement(corev1.ReadWriteMany))
            }, 20*time.Second, time.Second).Should(Succeed())

            // Verify PersistentVolumeClaim creation
            Eventually(func(g Gomega) {
                pvc := &corev1.PersistentVolumeClaim{}
                err := env.Client.Get(ctx, types.NamespacedName{
                    Name:      "test-storage-pvc",
                    Namespace: namespace,
                }, pvc)
                g.Expect(err).NotTo(HaveOccurred())
                g.Expect(pvc.Spec.Resources.Requests.Storage().String()).To(Equal("5Gi"))
            }, 20*time.Second, time.Second).Should(Succeed())
        })

        It("should handle invalid storage size", func() {
            instance := &unstructured.Unstructured{
                Object: map[string]interface{}{
                    "apiVersion": "kro.run/v1alpha1",
                    "kind":       "StorageConfig",
                    "metadata": map[string]interface{}{
                        "name":      "invalid-storage",
                        "namespace": namespace,
                    },
                    "spec": map[string]interface{}{
                        "name":           "invalid-storage",
                        "namespace":      namespace,
                        "storageAccount": "teststorage",
                        "secretName":     "test-storage-secret",
                        "storageSize":    "invalid",
                    },
                },
            }
            Expect(env.Client.Create(ctx, instance)).NotTo(Succeed())
        })

        It("should handle storage updates", func() {
            // Create initial storage
            // Update storage size
            // Verify PV and PVC are updated
        })

        It("should cleanup resources on deletion", func() {
            // Create storage resources
            // Delete storage instance
            // Verify cleanup
        })
    })

    Context("Multiple Share Names", func() {
        It("should handle multiple share names", func() {
            // Test with multiple share names
        })
    })

    Context("Access Modes", func() {
        It("should handle different access modes", func() {
            // Test different access modes
        })
    })
})
```

3. Add helper functions for common test operations:

```go
// Helper function to create a storage secret
func createStorageSecret(ctx context.Context, name, namespace string) error {
    secret := &corev1.Secret{
        ObjectMeta: metav1.ObjectMeta{
            Name:      name,
            Namespace: namespace,
        },
        StringData: map[string]string{
            "azurestorageaccountname": "teststorage",
            "azurestorageaccountkey":  "test-key",
        },
    }
    return env.Client.Create(ctx, secret)
}

// Helper function to verify resource deletion
func verifyResourceDeletion(ctx context.Context, name, namespace string) {
    // Verify StorageClass deletion
    Eventually(func() bool {
        sc := &storagev1.StorageClass{}
        err := env.Client.Get(ctx, types.NamespacedName{
            Name: name + "-sc",
        }, sc)
        return errors.IsNotFound(err)
    }, 20*time.Second, time.Second).Should(BeTrue())

    // Verify PV deletion
    Eventually(func() bool {
        pv := &corev1.PersistentVolume{}
        err := env.Client.Get(ctx, types.NamespacedName{
            Name: name + "-pv",
        }, pv)
        return errors.IsNotFound(err)
    }, 20*time.Second, time.Second).Should(BeTrue())

    // Verify PVC deletion
    Eventually(func() bool {
        pvc := &corev1.PersistentVolumeClaim{}
        err := env.Client.Get(ctx, types.NamespacedName{
            Name:      name + "-pvc",
            Namespace: namespace,
        }, pvc)
        return errors.IsNotFound(err)
    }, 20*time.Second, time.Second).Should(BeTrue())
}
```

This test structure:
- Uses Ginkgo/Gomega for BDD-style testing
- Tests resource creation, validation, updates, and deletion
- Verifies proper handling of edge cases
- Includes helper functions for common operations
- Uses Eventually blocks for async operations
- Follows KRO's existing testing patterns

To run the tests:
```bash
go test ./storage/tests/... -v
```

Would you like me to elaborate on any specific test cases or add more scenarios?
