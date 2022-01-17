package v1alpha1

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type GerritMergeRequest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              GerritMergeRequestSpec   `json:"spec,omitempty"`
	Status            GerritMergeRequestStatus `json:"status,omitempty"`
}

type GerritMergeRequestSpec struct {
	OwnerName     string `json:"ownerName"`
	ProjectName   string `json:"projectName"`
	TargetBranch  string `json:"targetBranch"`
	SourceBranch  string `json:"sourceBranch"`
	CommitMessage string `json:"commitMessage"`
}

type GerritMergeRequestStatus struct {
	Value     string `json:"value"`
	ChangeURL string `json:"changeUrl"`
	ChangeID  string `json:"changeId"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type GerritMergeRequestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GerritMergeRequest `json:"items"`
}

func (in GerritMergeRequest) OwnerName() string {
	return in.Spec.OwnerName
}

func (in GerritMergeRequest) TargetBranch() string {
	if in.Spec.TargetBranch == "" {
		return "master"
	}

	return in.Spec.TargetBranch
}

func (in GerritMergeRequest) CommitMessage() string {
	if in.Spec.CommitMessage == "" {
		return fmt.Sprintf("merge %s to %s", in.Spec.SourceBranch, in.TargetBranch())
	}

	return in.Spec.CommitMessage
}
