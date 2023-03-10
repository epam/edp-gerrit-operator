package v1

import (
	"fmt"
	"path"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/epam/edp-gerrit-operator/v2/pkg/service/gerrit/spec"
)

// GerritSpec defines the desired state of Gerrit.
type GerritSpec struct {
	KeycloakSpec KeycloakSpec `json:"keycloakSpec"`

	// RestAPIUrl gerrit http full api url
	RestAPIUrl string `json:"restAPIUrl,omitempty"`

	// SSHUrl gerrit ssh url
	SSHUrl string `json:"sshUrl,omitempty"`

	// +optional
	SshPort int32 `json:"sshPort,omitempty"`

	// BasePath gerrit http route base path
	BasePath string `json:"basePath,omitempty"`
}

type KeycloakSpec struct {
	Enabled bool `json:"enabled"`

	// +optional
	Url string `json:"url,omitempty"`

	// +optional
	Realm string `json:"realm,omitempty"`
}

// GerritStatus defines the observed state of Gerrit.
type GerritStatus struct {
	ExternalUrl string `json:"externalUrl"`

	// +optional
	Available bool `json:"available,omitempty"`

	// +optional
	LastTimeUpdated metav1.Time `json:"lastTimeUpdated,omitempty"`

	// +optional
	Status string `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion

// Gerrit is the Schema for the gerrits API.
type Gerrit struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GerritSpec   `json:"spec,omitempty"`
	Status GerritStatus `json:"status,omitempty"`
}

func (in *GerritSpec) GetBasePath() string {
	if in.BasePath == "" {
		return spec.GerritRestApiUrlPath
	}

	return fmt.Sprintf("%s/", path.Join(in.BasePath, spec.GerritRestApiUrlPath))
}

// +kubebuilder:object:root=true

// GerritList contains a list of Gerrit.
type GerritList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Gerrit `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GerritList{}, &Gerrit{})
}
