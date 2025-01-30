package unit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestStorageConfig_Validation(t *testing.T) {
	tests := []struct {
		name    string
		config  StorageConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: StorageConfig{
				Spec: StorageConfigSpec{
					Name:           "test-storage",
					Namespace:      "default",
					StorageAccount: "teststorage",
					SecretName:     "storage-secret",
				},
			},
			wantErr: false,
		},
		{
			name: "missing name",
			config: StorageConfig{
				Spec: StorageConfigSpec{
					Namespace:      "default",
					StorageAccount: "teststorage",
					SecretName:     "storage-secret",
				},
			},
			wantErr: true,
		},
		{
			name: "missing namespace",
			config: StorageConfig{
				Spec: StorageConfigSpec{
					Name:           "test-storage",
					StorageAccount: "teststorage",
					SecretName:     "storage-secret",
				},
			},
			wantErr: true,
		},
		{
			name: "missing storage account",
			config: StorageConfig{
				Spec: StorageConfigSpec{
					Name:       "test-storage",
					Namespace:  "default",
					SecretName: "storage-secret",
				},
			},
			wantErr: true,
		},
		{
			name: "missing secret name",
			config: StorageConfig{
				Spec: StorageConfigSpec{
					Name:           "test-storage",
					Namespace:      "default",
					StorageAccount: "teststorage",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateStorageConfig(&tt.config)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestStorageClass_Generation(t *testing.T) {
	config := &StorageConfig{
		Spec: StorageConfigSpec{
			Name:           "test-storage",
			Namespace:      "default",
			StorageAccount: "teststorage",
			SecretName:     "storage-secret",
		},
	}

	sc := generateStorageClass(config)
	assert.Equal(t, "test-storage-sc", sc.Name)
	assert.Equal(t, "file.csi.azure.com", sc.Provisioner)
	assert.True(t, *sc.AllowVolumeExpansion)
	assert.Equal(t, "teststorage", sc.Parameters["storageAccount"])
	assert.Equal(t, "storage-secret", sc.Parameters["csi.storage.k8s.io/provisioner-secret-name"])
}

func TestPersistentVolume_Generation(t *testing.T) {
	config := &StorageConfig{
		Spec: StorageConfigSpec{
			Name:           "test-storage",
			Namespace:      "default",
			StorageAccount: "teststorage",
			SecretName:     "storage-secret",
			StorageSize:    "10Gi",
			ShareName:      "testshare",
			AccessMode:     "ReadWriteMany",
		},
	}

	pv := generatePersistentVolume(config)
	assert.Equal(t, "test-storage-pv", pv.Name)
	assert.Equal(t, "10Gi", pv.Spec.Capacity.Storage().String())
	assert.Equal(t, corev1.PersistentVolumeReclaimPolicy("Retain"), pv.Spec.PersistentVolumeReclaimPolicy)
	assert.Equal(t, "testshare", pv.Spec.AzureFile.ShareName)
}

func validateStorageConfig(config *StorageConfig) error {
	if config.Spec.Name == "" {
		return assert.AnError
	}
	if config.Spec.Namespace == "" {
		return assert.AnError
	}
	if config.Spec.StorageAccount == "" {
		return assert.AnError
	}
	if config.Spec.SecretName == "" {
		return assert.AnError
	}
	return nil
}

func generateStorageClass(config *StorageConfig) *storagev1.StorageClass {
	allowVolumeExpansion := true
	return &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: config.Spec.Name + "-sc",
		},
		Provisioner:          "file.csi.azure.com",
		AllowVolumeExpansion: &allowVolumeExpansion,
		Parameters: map[string]string{
			"storageAccount": config.Spec.StorageAccount,
			"csi.storage.k8s.io/provisioner-secret-name": config.Spec.SecretName,
		},
	}
}

func generatePersistentVolume(config *StorageConfig) *corev1.PersistentVolume {
	quantity := resource.MustParse(config.Spec.StorageSize)
	return &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: config.Spec.Name + "-pv",
		},
		Spec: corev1.PersistentVolumeSpec{
			Capacity: corev1.ResourceList{
				corev1.ResourceStorage: quantity,
			},
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.PersistentVolumeAccessMode(config.Spec.AccessMode),
			},
			PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimPolicy("Retain"),
			StorageClassName:              config.Spec.Name + "-sc",
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				AzureFile: &corev1.AzureFilePersistentVolumeSource{
					SecretName: config.Spec.SecretName,
					ShareName:  config.Spec.ShareName,
					ReadOnly:   false,
				},
			},
		},
	}
}
