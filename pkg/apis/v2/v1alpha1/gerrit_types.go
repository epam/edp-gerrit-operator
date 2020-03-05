package v1alpha1

import (
	coreV1Api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// GerritSpec defines the desired state of Gerrit
// +k8s:openapi-gen=true

type GerritVolumes struct {
	Name         string `json:"name"`
	StorageClass string `json:"storage_class"`
	Capacity     string `json:"capacity"`
}

type GerritSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	Image            string                           `json:"image"`
	Type             string                           `json:"type"`
	Version          string                           `json:"version"`
	ImagePullSecrets []coreV1Api.LocalObjectReference `json:"imagePullSecrets,omitempty"`
	Volumes          []GerritVolumes                  `json:"volumes,omitempty"`
	KeycloakSpec     KeycloakSpec                     `json:"keycloakSpec"`
	Users            []GerritUsers                    `json:"users,omitempty"`
	SshPort          int32                            `json:"sshPort,omitempty"`
}

type GerritUsers struct {
	Username string   `json:"username"`
	Groups   []string `json:"groups, omitempty"`
}

// GerritStatus defines the observed state of Gerrit
// +k8s:openapi-gen=true
type GerritStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	Available       bool      `json:"available,omitempty"`
	LastTimeUpdated time.Time `json:"lastTimeUpdated,omitempty"`
	Status          string    `json:"status,omitempty"`
	ExternalUrl     string    `json:"externalUrl"`
}

type KeycloakSpec struct {
	Enabled bool   `json:"enabled"`
	Url     string `json:"url, omitempty"`
	Realm   string `json:"realm, omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Gerrit is the Schema for the gerrits API
// +k8s:openapi-gen=true
type Gerrit struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              GerritSpec   `json:"spec,omitempty"`
	Status            GerritStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GerritList contains a list of Gerrit
type GerritList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Gerrit `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Gerrit{}, &GerritList{})
}
