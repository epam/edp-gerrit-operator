package v1alpha1

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:deprecatedversion

// GerritMergeRequest is the Schema for the gerrit merge request API.
type GerritMergeRequest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GerritMergeRequestSpec   `json:"spec,omitempty"`
	Status GerritMergeRequestStatus `json:"status,omitempty"`
}

// GerritMergeRequestSpec defines the desired state of GerritMergeRequest.
type GerritMergeRequestSpec struct {
	// OwnerName indicates which gerrit CR should be taken to initialize correct client.
	// +nullable
	OwnerName string `json:"ownerName"`

	// ProjectName is gerrit project name.
	ProjectName string `json:"projectName"`

	AuthorName string `json:"authorName"`

	AuthorEmail string `json:"authorEmail"`

	// TargetBranch default value is master.
	// +optional
	TargetBranch string `json:"targetBranch,omitempty"`

	// +optional
	SourceBranch string `json:"sourceBranch,omitempty"`

	// +optional
	CommitMessage string `json:"commitMessage,omitempty"`

	// +optional
	ChangesConfigMap string `json:"changesConfigMap,omitempty"`

	// AdditionalArguments contains merge command additional command line arguments.
	// +nullable
	// +optional
	AdditionalArguments []string `json:"additionalArguments,omitempty"`
}

// GerritMergeRequestStatus defines the observed state of GerritMergeRequest.
type GerritMergeRequestStatus struct {
	// +optional
	Value string `json:"value,omitempty"`
	// +optional
	ChangeURL string `json:"changeUrl,omitempty"`
	// +optional
	ChangeID string `json:"changeId,omitempty"`
}

// +kubebuilder:object:root=true

// GerritMergeRequestList contains a list of GerritMergeRequest.
type GerritMergeRequestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []GerritMergeRequest `json:"items"`
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
	if in.Spec.CommitMessage == "" && in.Spec.SourceBranch != "" {
		return fmt.Sprintf("merge %s to %s", in.Spec.SourceBranch, in.TargetBranch())
	} else if in.Spec.CommitMessage == "" && in.Spec.ChangesConfigMap != "" {
		return fmt.Sprintf("merge files contents from config map: %s", in.Spec.ChangesConfigMap)
	}

	return in.Spec.CommitMessage
}

func init() {
	SchemeBuilder.Register(&GerritMergeRequest{}, &GerritMergeRequestList{})
}
