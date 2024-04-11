package v1

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GerritMergeRequestSpec defines the desired state of GerritMergeRequest.
type GerritMergeRequestSpec struct {
	// OwnerName is the name of Gerrit CR, which should be used to initialize the client.
	// If empty, the operator will get first Gerrit CR from the namespace.
	// +optional
	// +kubebuilder:example:=`gerrit`
	OwnerName string `json:"ownerName"`

	// ProjectName is gerrit project name.
	// +required
	// +kubebuilder:example:=`my-project`
	ProjectName string `json:"projectName"`

	// AuthorName is the name of the user who creates the merge request.
	// +required
	// +kubebuilder:example:=`John Doe`
	AuthorName string `json:"authorName"`

	// AuthorEmail is the email of the user who creates the merge request.
	// +required
	// +kubebuilder:example:=`john.foe@mail.com`
	AuthorEmail string `json:"authorEmail"`

	// TargetBranch is the name of the branch to which the changes should be merged.
	// If changesConfigMap is set, the targetBranch can be only the origin HEAD branch.
	// +optional
	// +kubebuilder:default=master
	// +kubebuilder:example:=master
	TargetBranch string `json:"targetBranch,omitempty"`

	// SourceBranch is the name of the branch from which the changes should be merged.
	// If empty, changesConfigMap should be set.
	// +optional
	// +kubebuilder:example:=new-feature
	SourceBranch string `json:"sourceBranch,omitempty"`

	// CommitMessage is the commit message for the merge request.
	// If empty, the operator will generate the commit message.
	// +optional
	// +kubebuilder:example:=`merge new-feature to master`
	CommitMessage string `json:"commitMessage,omitempty"`

	// ChangesConfigMap is the name of the ConfigMap, which contains files contents that should be merged.
	// ConfigMap should contain eny data keys with content in the json
	// format: {"path": "/controllers/user.go", "contents": "some code here"} - to add file
	// or format: {"path": "/controllers/user.go"} - to remove file.
	// If files already exist in the project, they will be overwritten.
	// If empty, sourceBranch should be set.
	// +optional
	// +kubebuilder:example:=`config-map-new-feature`
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
// +kubebuilder:subresource:status
// +kubebuilder:storageversion

// GerritMergeRequest is the Schema for the gerrit merge request API.
type GerritMergeRequest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GerritMergeRequestSpec   `json:"spec,omitempty"`
	Status GerritMergeRequestStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// GerritMergeRequestList contains a list of GerritMergeRequest.
type GerritMergeRequestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []GerritMergeRequest `json:"items"`
}

func (in *GerritMergeRequest) OwnerName() string {
	return in.Spec.OwnerName
}

func (in *GerritMergeRequest) TargetBranch() string {
	if in.Spec.TargetBranch == "" {
		return "master"
	}

	return in.Spec.TargetBranch
}

func (in *GerritMergeRequest) CommitMessage() string {
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
