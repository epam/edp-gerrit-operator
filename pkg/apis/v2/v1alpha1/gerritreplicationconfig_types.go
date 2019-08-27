package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// GerritReplicationConfigSpec defines the desired state of GerritReplicationConfig
// +k8s:openapi-gen=true
type GerritReplicationConfigSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
}

// GerritReplicationConfigStatus defines the observed state of GerritReplicationConfig
// +k8s:openapi-gen=true
type GerritReplicationConfigStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GerritReplicationConfig is the Schema for the gerritreplicationconfigs API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type GerritReplicationConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GerritReplicationConfigSpec   `json:"spec,omitempty"`
	Status GerritReplicationConfigStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GerritReplicationConfigList contains a list of GerritReplicationConfig
type GerritReplicationConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GerritReplicationConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GerritReplicationConfig{}, &GerritReplicationConfigList{})
}
