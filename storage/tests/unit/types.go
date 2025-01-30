// Package unit provides types and unit tests for the Storage ResourceGroup
package unit

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// StorageConfigSpec defines the desired state of a StorageConfig resource
// It contains all the configuration needed to create Azure File storage resources
type StorageConfigSpec struct {
	// Name is the identifier for this storage configuration
	Name string `json:"name"`

	// Namespace where the storage resources will be created
	Namespace string `json:"namespace"`

	// StorageAccount is the Azure storage account name
	StorageAccount string `json:"storageAccount"`

	// SecretName references the secret containing Azure storage credentials
	SecretName string `json:"secretName"`

	// ShareName is the name of the Azure File share (optional, defaults to "app")
	ShareName string `json:"shareName,omitempty"`

	// StorageSize specifies the size of the storage (optional, defaults to "5Ti")
	StorageSize string `json:"storageSize,omitempty"`

	// AccessMode defines how the storage can be mounted (optional, defaults to "ReadWriteMany")
	AccessMode string `json:"accessMode,omitempty"`
}

// DeepCopy creates a deep copy of StorageConfigSpec
func (in *StorageConfigSpec) DeepCopy() *StorageConfigSpec {
	if in == nil {
		return nil
	}
	out := new(StorageConfigSpec)
	*out = *in
	return out
}

// StorageConfigStatus defines the observed state of a StorageConfig resource
// It contains references to the created Kubernetes resources
type StorageConfigStatus struct {
	// StorageClassName is the name of the created StorageClass
	StorageClassName string `json:"storageClassName"`

	// PersistentVolumeName is the name of the created PersistentVolume
	PersistentVolumeName string `json:"persistentVolumeName"`

	// ClaimName is the name of the created PersistentVolumeClaim
	ClaimName string `json:"claimName"`
}

// DeepCopy creates a deep copy of StorageConfigStatus
func (in *StorageConfigStatus) DeepCopy() *StorageConfigStatus {
	if in == nil {
		return nil
	}
	out := new(StorageConfigStatus)
	*out = *in
	return out
}

// StorageConfig is the Schema for the storageconfigs API
// It represents a complete storage configuration including Azure File storage resources
type StorageConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired state of StorageConfig
	Spec StorageConfigSpec `json:"spec"`

	// Status defines the observed state of StorageConfig
	Status StorageConfigStatus `json:"status"`
}

// DeepCopyInto copies all properties of this object into another object of the same type
func (in *StorageConfig) DeepCopyInto(out *StorageConfig) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = *in.Spec.DeepCopy()
	out.Status = *in.Status.DeepCopy()
}

// DeepCopy creates a new StorageConfig with the same content as this one
func (in *StorageConfig) DeepCopy() *StorageConfig {
	if in == nil {
		return nil
	}
	out := new(StorageConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject returns a generically typed copy of an object
func (in *StorageConfig) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// StorageConfigList contains a list of StorageConfig resources
type StorageConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items is the list of StorageConfig resources
	Items []StorageConfig `json:"items"`
}

// DeepCopyInto copies all properties of this object into another object of the same type
func (in *StorageConfigList) DeepCopyInto(out *StorageConfigList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]StorageConfig, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy creates a new StorageConfigList with the same content as this one
func (in *StorageConfigList) DeepCopy() *StorageConfigList {
	if in == nil {
		return nil
	}
	out := new(StorageConfigList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject returns a generically typed copy of an object
func (in *StorageConfigList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}
