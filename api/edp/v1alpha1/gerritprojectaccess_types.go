package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// GerritProjectAccessSpec defines the desired state of GerritProjectAccess.
type GerritProjectAccessSpec struct {
	// ProjectName is gerrit project name.
	ProjectName string `json:"projectName"`

	// OwnerName indicates which gerrit CR should be taken to initialize correct client.
	// +nullable
	// +optional
	OwnerName string `json:"ownerName,omitempty"`

	// Parent is parent project.
	// +optional
	Parent string `json:"parent,omitempty"`

	// References contains gerrit references.
	// +nullable
	// +optional
	References []Reference `json:"references,omitempty"`
}

type Reference struct {
	// Patter is reference pattern, example: refs/heads/*.
	// +optional
	Pattern string `json:"refPattern,omitempty"`

	// +optional
	PermissionName string `json:"permissionName,omitempty"`

	// +optional
	PermissionLabel string `json:"permissionLabel,omitempty"`

	// +optional
	GroupName string `json:"groupName,omitempty"`

	// +optional
	Action string `json:"action,omitempty"`

	// Force indicates whether the force flag is set.
	// +optional
	Force bool `json:"force,omitempty"`

	// Min is the min value of the permission range.
	// +optional
	Min int `json:"min,omitempty"`

	// Max is the max value of the permission range.
	// +optional
	Max int `json:"max,omitempty"`
}

// GerritProjectAccessStatus defines the observed state of GerritProjectAccess.
type GerritProjectAccessStatus struct {
	// +optional
	Created bool `json:"created,omitempty"`

	// +optional
	Value string `json:"value,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:deprecatedversion

// GerritProjectAccess is the Schema for the gerrit project access API.
type GerritProjectAccess struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GerritProjectAccessSpec   `json:"spec,omitempty"`
	Status GerritProjectAccessStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// GerritProjectAccessList contains a list of GerritProjectAccess.
type GerritProjectAccessList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []GerritProjectAccess `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GerritProjectAccess{}, &GerritProjectAccessList{})
}
