// Package unit provides types and unit tests for the WhiskyApp ResourceGroup
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
		&WhiskyApp{},
		&WhiskyAppList{},
	)
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}

// ResourceRequirements defines compute resource requirements
type ResourceRequirements struct {
	// CPU request for the container
	CPU string `json:"cpu"`
	// Memory request for the container
	Memory string `json:"memory"`
}

// Resources defines the resource requirements and limits
type Resources struct {
	// Requests specifies the minimum required resources
	Requests ResourceRequirements `json:"requests"`
	// Limits specifies the maximum allowed resources
	Limits ResourceRequirements `json:"limits"`
}

// WhiskyAppSpec defines the desired state of a WhiskyApp resource
type WhiskyAppSpec struct {
	// Name is the identifier for this WhiskyApp deployment
	Name string `json:"name"`

	// Image is the container image to use for the deployment
	Image string `json:"image"`

	// Replicas is the number of desired replicas (optional, defaults to 3)
	Replicas int32 `json:"replicas,omitempty"`

	// Resources defines the resource requirements and limits for the deployment
	Resources Resources `json:"resources"`
}

// DeepCopy creates a deep copy of WhiskyAppSpec
func (in *WhiskyAppSpec) DeepCopy() *WhiskyAppSpec {
	if in == nil {
		return nil
	}
	out := new(WhiskyAppSpec)
	*out = *in
	return out
}

// WhiskyAppStatus defines the observed state of a WhiskyApp resource
type WhiskyAppStatus struct {
	// Ready indicates if the WhiskyApp deployment is ready
	Ready bool `json:"ready"`

	// ServiceEndpoint is the cluster IP of the service
	ServiceEndpoint string `json:"serviceEndpoint"`
}

// DeepCopy creates a deep copy of WhiskyAppStatus
func (in *WhiskyAppStatus) DeepCopy() *WhiskyAppStatus {
	if in == nil {
		return nil
	}
	out := new(WhiskyAppStatus)
	*out = *in
	return out
}

// WhiskyApp is the Schema for the whiskyapps API
type WhiskyApp struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired state of WhiskyApp
	Spec WhiskyAppSpec `json:"spec"`

	// Status defines the observed state of WhiskyApp
	Status WhiskyAppStatus `json:"status"`
}

// DeepCopyInto copies all properties of this object into another object of the same type
func (in *WhiskyApp) DeepCopyInto(out *WhiskyApp) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = *in.Spec.DeepCopy()
	out.Status = *in.Status.DeepCopy()
}

// DeepCopy creates a new WhiskyApp with the same content as this one
func (in *WhiskyApp) DeepCopy() *WhiskyApp {
	if in == nil {
		return nil
	}
	out := new(WhiskyApp)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject returns a generically typed copy of an object
func (in *WhiskyApp) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// WhiskyAppList contains a list of WhiskyApp resources
type WhiskyAppList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items is the list of WhiskyApp resources
	Items []WhiskyApp `json:"items"`
}

// DeepCopyInto copies all properties of this object into another object of the same type
func (in *WhiskyAppList) DeepCopyInto(out *WhiskyAppList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]WhiskyApp, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy creates a new WhiskyAppList with the same content as this one
func (in *WhiskyAppList) DeepCopy() *WhiskyAppList {
	if in == nil {
		return nil
	}
	out := new(WhiskyAppList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject returns a generically typed copy of an object
func (in *WhiskyAppList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}
