package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +k8s:openapi-gen=true
type GerritGroupSpec struct {
	OwnerName    string `json:"gerritOwner"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	VisibleToAll bool   `json:"visibleToAll"`
}

// +k8s:openapi-gen=true
type GerritGroupStatus struct {
	ID      string `json:"id"`
	GroupID string `json:"groupId"`
	Value   string `json:"value"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
type GerritGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              GerritGroupSpec   `json:"spec,omitempty"`
	Status            GerritGroupStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type GerritGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GerritGroup `json:"items"`
}
