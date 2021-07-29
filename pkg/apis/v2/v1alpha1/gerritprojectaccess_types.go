package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +k8s:openapi-gen=true
type GerritProjectAccessSpec struct {
	OwnerName   string      `json:"ownerName"`
	ProjectName string      `json:"projectName"`
	Parent      string      `json:"parent"`
	References  []Reference `json:"references"`
}

// +k8s:openapi-gen=true
type Reference struct {
	Pattern         string `json:"refPattern"`
	PermissionName  string `json:"permissionName"`
	PermissionLabel string `json:"permissionLabel"`
	GroupName       string `json:"groupName"`
	Action          string `json:"action"`
	Force           bool   `json:"force"`
	Min             int    `json:"min"`
	Max             int    `json:"max"`
}

// +k8s:openapi-gen=true
type GerritProjectAccessStatus struct {
	Created bool   `json:"created"`
	Value   string `json:"value"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
type GerritProjectAccess struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              GerritProjectAccessSpec   `json:"spec,omitempty"`
	Status            GerritProjectAccessStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
type GerritProjectAccessList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GerritProjectAccess `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GerritProjectAccess{}, &GerritProjectAccessList{})
}
