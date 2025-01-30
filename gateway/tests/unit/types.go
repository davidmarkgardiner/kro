// Package unit provides types and unit tests for the Gateway ResourceGroup
package unit

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// SchemeGroupVersion is group version used to register these objects
var SchemeGroupVersion = schema.GroupVersion{Group: "kro.run", Version: "v1alpha1"}

// Kind takes an unqualified kind and returns back a Group qualified GroupKind
func Kind(kind string) schema.GroupKind {
	return SchemeGroupVersion.WithKind(kind).GroupKind()
}

// Resource takes an unqualified resource and returns a Group qualified GroupResource
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

var (
	// SchemeBuilder initializes a scheme builder
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	// AddToScheme is a global function that registers our types
	AddToScheme = SchemeBuilder.AddToScheme
)

// addKnownTypes adds our types to the API scheme
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&GatewayConfig{},
		&GatewayConfigList{},
	)
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}

// GatewayConfigSpec defines the desired state of a Gateway resource
type GatewayConfigSpec struct {
	// Name is the identifier for this Gateway
	Name string `json:"name"`

	// Hostname for the gateway
	Hostname string `json:"hostname"`

	// Namespace for the gateway
	Namespace string `json:"namespace,omitempty"`

	// TLS Secret Name
	TLSSecretName string `json:"tlsSecretName"`

	// TLS Secret Namespace
	TLSSecretNamespace string `json:"tlsSecretNamespace"`

	// Port for the gateway
	Port int32 `json:"port,omitempty"`

	// Protocol for the gateway
	Protocol string `json:"protocol,omitempty"`
}

// DeepCopy creates a deep copy of GatewayConfigSpec
func (in *GatewayConfigSpec) DeepCopy() *GatewayConfigSpec {
	if in == nil {
		return nil
	}
	out := new(GatewayConfigSpec)
	*out = *in
	return out
}

// GatewayConfigStatus defines the observed state of a Gateway resource
type GatewayConfigStatus struct {
	// GatewayName is the name of the created gateway
	GatewayName string `json:"gatewayName"`

	// GatewayNamespace is the namespace of the created gateway
	GatewayNamespace string `json:"gatewayNamespace"`

	// Hostname is the configured hostname
	Hostname string `json:"hostname"`
}

// DeepCopy creates a deep copy of GatewayConfigStatus
func (in *GatewayConfigStatus) DeepCopy() *GatewayConfigStatus {
	if in == nil {
		return nil
	}
	out := new(GatewayConfigStatus)
	*out = *in
	return out
}

// GatewayConfig is the Schema for the gateway API
type GatewayConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired state of Gateway
	Spec GatewayConfigSpec `json:"spec"`

	// Status defines the observed state of Gateway
	Status GatewayConfigStatus `json:"status"`
}

// DeepCopyInto copies all properties of this object into another object of the same type
func (in *GatewayConfig) DeepCopyInto(out *GatewayConfig) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = *in.Spec.DeepCopy()
	out.Status = *in.Status.DeepCopy()
}

// DeepCopy creates a new GatewayConfig with the same content as this one
func (in *GatewayConfig) DeepCopy() *GatewayConfig {
	if in == nil {
		return nil
	}
	out := new(GatewayConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject returns a generically typed copy of an object
func (in *GatewayConfig) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// GatewayConfigList contains a list of GatewayConfig resources
type GatewayConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items is the list of GatewayConfig resources
	Items []GatewayConfig `json:"items"`
}

// DeepCopyInto copies all properties of this object into another object of the same type
func (in *GatewayConfigList) DeepCopyInto(out *GatewayConfigList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]GatewayConfig, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy creates a new GatewayConfigList with the same content as this one
func (in *GatewayConfigList) DeepCopy() *GatewayConfigList {
	if in == nil {
		return nil
	}
	out := new(GatewayConfigList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject returns a generically typed copy of an object
func (in *GatewayConfigList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}
