package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GerritReplicationConfigSpec defines the desired state of GerritReplicationConfig.
type GerritReplicationConfigSpec struct {
	SSHUrl string `json:"ssh_url"`

	// +optional
	OwnerName string `json:"owner_name,omitempty"`
}

// GerritReplicationConfigStatus defines the observed state of GerritReplicationConfig.
type GerritReplicationConfigStatus struct {
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

// GerritReplicationConfig is the Schema for the gerrit replication config API.
type GerritReplicationConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GerritReplicationConfigSpec   `json:"spec,omitempty"`
	Status GerritReplicationConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// GerritReplicationConfigList contains a list of GerritReplicationConfig.
type GerritReplicationConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []GerritReplicationConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GerritReplicationConfig{}, &GerritReplicationConfigList{})
}
