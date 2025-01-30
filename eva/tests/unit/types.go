package unit

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

// EVA is the Schema for the eva API
type EVA struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EVASpec   `json:"spec,omitempty"`
	Status EVAStatus `json:"status,omitempty"`
}

// EVASpec defines the desired state of EVA
type EVASpec struct {
	VaultURL                     string `json:"vault_url"`
	SWCI                         string `json:"swci"`
	UserAssignedIdentityName     string `json:"user_assigned_identity_name"`
	ServicePrincipleEvaKey       string `json:"service_principle_eva_key"`
	ServiceAccountName           string `json:"service_account_name"`
	UserAssignedIdentityClientID string `json:"user_assigned_identity_client_id"`
	UserAssignedIdentityTenantID string `json:"user_assigned_identity_tenant_id"`
}

// EVAStatus defines the observed state of EVA
type EVAStatus struct {
	Ready bool `json:"ready,omitempty"`
}

// EVAList contains a list of EVA
type EVAList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EVA `json:"items"`
}

var (
	// GroupVersion is group version used to register these objects
	GroupVersion = schema.GroupVersion{Group: "kro.run", Version: "v1alpha1"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

func init() {
	SchemeBuilder.Register(&EVA{}, &EVAList{})
}

// DeepCopyObject implements runtime.Object interface
func (in *EVA) DeepCopyObject() runtime.Object {
	out := new(EVA)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto copies all properties of this object into another object of the same type
func (in *EVA) DeepCopyInto(out *EVA) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	out.Status = in.Status
}

// DeepCopyObject implements runtime.Object interface
func (in *EVAList) DeepCopyObject() runtime.Object {
	out := new(EVAList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto copies all properties of this object into another object of the same type
func (in *EVAList) DeepCopyInto(out *EVAList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]EVA, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}
