package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// GerritGroupSpec defines the desired state of GerritGroup.
type GerritGroupSpec struct {
	Name string `json:"name"`

	// +optional
	OwnerName string `json:"gerritOwner,omitempty"`

	// +optional
	Description string `json:"description,omitempty"`

	// +optional
	VisibleToAll bool `json:"visibleToAll,omitempty"`
}

// GerritGroupStatus defines the observed state of GerritGroup.
type GerritGroupStatus struct {
	// +optional
	ID string `json:"id,omitempty"`

	// +optional
	GroupID string `json:"groupId,omitempty"`

	// +optional
	Value string `json:"value,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion

// GerritGroup is the Schema for the gerrit group API.
type GerritGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GerritGroupSpec   `json:"spec,omitempty"`
	Status GerritGroupStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// GerritGroupList contains a list of GerritGroup.
type GerritGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []GerritGroup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GerritGroup{}, &GerritGroupList{})
}
