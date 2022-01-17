package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type GerritProjectStatus struct {
	Value    string   `json:"value"`
	Branches []string `json:"branches"`
}

type GerritProjectSpec struct {
	OwnerName         string `json:"ownerName"`
	Name              string `json:"name"`
	Parent            string `json:"parent"`
	Description       string `json:"description"`
	PermissionsOnly   bool   `json:"permissionsOnly"`
	CreateEmptyCommit bool   `json:"createEmptyCommit"`
	SubmitType        string `json:"submitType"`
	Branches          string `json:"branches"`
	Owners            string `json:"owners"`
	RejectEmptyCommit string `json:"rejectEmptyCommit"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
type GerritProject struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              GerritProjectSpec   `json:"spec,omitempty"`
	Status            GerritProjectStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
type GerritProjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GerritProject `json:"items"`
}
