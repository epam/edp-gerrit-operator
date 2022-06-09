package v1

import (
	coreV1Api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GerritSpec defines the desired state of Gerrit
type GerritSpec struct {
	Image        string       `json:"image"`
	Type         string       `json:"type"`
	Version      string       `json:"version"`
	KeycloakSpec KeycloakSpec `json:"keycloakSpec"`

	// +nullable
	// +optional
	ImagePullSecrets []coreV1Api.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	// +nullable
	// +optional
	Volumes []GerritVolumes `json:"volumes,omitempty"`

	// +optional
	SshPort int32 `json:"sshPort,omitempty"`
}

type GerritVolumes struct {
	Name     string `json:"name"`
	Capacity string `json:"capacity"`

	// +optional
	StorageClass string `json:"storage_class,omitempty"`
}

type KeycloakSpec struct {
	Enabled bool `json:"enabled"`

	// +optional
	Url string `json:"url,omitempty"`

	// +optional
	Realm string `json:"realm,omitempty"`
}

// GerritStatus defines the observed state of Gerrit
type GerritStatus struct {
	ExternalUrl string `json:"externalUrl"`

	// +optional
	Available bool `json:"available,omitempty"`

	// +optional
	LastTimeUpdated metav1.Time `json:"lastTimeUpdated,omitempty"`

	// +optional
	Status string `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion

// Gerrit is the Schema for the gerrits API
type Gerrit struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GerritSpec   `json:"spec,omitempty"`
	Status GerritStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// GerritList contains a list of Gerrit
type GerritList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Gerrit `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GerritList{}, &Gerrit{})
}
