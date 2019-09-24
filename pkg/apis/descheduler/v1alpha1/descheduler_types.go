package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DeschedulerSpec defines the desired state of Descheduler
// +k8s:openapi-gen=true
type DeschedulerSpec struct {
	// Strategies list of strategies that should be enabled in deschdeular
	Strategies []Strategy `json:"strategies,omitempty"`
	// Schedule on which cronjob should run
	Schedule string `json:"schedule,omitempty"`
	// Flags for deschedular
	Flags []Param `json:"flags"`
	// Image of the deschedular being managed, this includes the version
	Image string `json:"image,omitempty"`
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
}

// Strategy supported by deschdular
//+k8s:openapi-gen=true
type Strategy struct {
	Name   string  `json:"name,omitempty"`
	Params []Param `json:"params"`
}

// Param is a key/value pair representing the prameter in the stratery or flags
// +k8s:openapi-gen=true
type Param struct {
	Name  string `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
}

// DeschedulerStatus defines the observed state of Descheduler
// +k8s:openapi-gen=true
type DeschedulerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	Phase string `json:"phase,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Descheduler is the Schema for the deschedulers API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type Descheduler struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DeschedulerSpec   `json:"spec,omitempty"`
	Status DeschedulerStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DeschedulerList contains a list of Descheduler
type DeschedulerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Descheduler `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Descheduler{}, &DeschedulerList{})
}
