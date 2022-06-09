package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// GerritGroupMemberSpec defines the desired state of GerritGroupMember
type GerritGroupMemberSpec struct {
	GroupID   string `json:"groupId"`
	AccountID string `json:"accountId"`

	// OwnerName indicates which gerrit CR should be taken to initialize correct client.
	// +nullable
	// +optional
	OwnerName string `json:"ownerName,omitempty"`
}

// GerritGroupMemberStatus defines the observed state of GerritGroupMember
type GerritGroupMemberStatus struct {
	// +optional
	Value string `json:"value,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion

// GerritGroupMember is the Schema for the gerrit group member API
type GerritGroupMember struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GerritGroupMemberSpec   `json:"spec,omitempty"`
	Status GerritGroupMemberStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// GerritGroupMemberList contains a list of GerritGroupMember
type GerritGroupMemberList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []GerritGroupMember `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GerritGroupMember{}, &GerritGroupMemberList{})
}
