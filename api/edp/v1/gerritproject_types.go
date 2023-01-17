package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// GerritProjectSpec defines the desired state of GerritProject.
type GerritProjectSpec struct {
	Name string `json:"name"`

	// +optional
	OwnerName string `json:"ownerName,omitempty"`

	// +optional
	Parent string `json:"parent,omitempty"`

	// +optional
	Description string `json:"description,omitempty"`

	// +optional
	PermissionsOnly bool `json:"permissionsOnly,omitempty"`

	// +optional
	CreateEmptyCommit bool `json:"createEmptyCommit,omitempty"`

	// +optional
	SubmitType string `json:"submitType,omitempty"`

	// +optional
	Branches string `json:"branches,omitempty"`

	// +optional
	Owners string `json:"owners,omitempty"`

	// +optional
	RejectEmptyCommit string `json:"rejectEmptyCommit,omitempty"`
}

// GerritProjectStatus defines the observed state of GerritProject.
type GerritProjectStatus struct {
	// +optional
	Value string `json:"value,omitempty"`

	// +nullable
	// +optional
	Branches []string `json:"branches,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion

// GerritProject is the Schema for the gerrit project API.
type GerritProject struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GerritProjectSpec   `json:"spec,omitempty"`
	Status GerritProjectStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// GerritProjectList contains a list of GerritProject.
type GerritProjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []GerritProject `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GerritProject{}, &GerritProjectList{})
}
