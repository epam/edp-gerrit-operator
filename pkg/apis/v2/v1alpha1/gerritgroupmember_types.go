package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type GerritGroupMemberStatus struct {
	Value string `json:"value"`
}

type GerritGroupMemberSpec struct {
	OwnerName string `json:"ownerName"`
	GroupID   string `json:"groupId"`
	AccountID string `json:"accountId"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
type GerritGroupMember struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              GerritGroupMemberSpec   `json:"spec,omitempty"`
	Status            GerritGroupMemberStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
type GerritGroupMemberList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GerritGroupMember `json:"items"`
}
