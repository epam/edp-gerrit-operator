// NOTE: Boilerplate only.  Ignore this file.

// Package v1 contains API v1 Schema definitions for the v2.edp.epam.com API group
// +kubebuilder:object:generate=true
// +groupName=v2.edp.epam.com
package v1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	// SchemeGroupVersion is group version used to register these objects.
	SchemeGroupVersion = schema.GroupVersion{Group: "v2.edp.epam.com", Version: "v1"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme.
	SchemeBuilder = &scheme.Builder{GroupVersion: SchemeGroupVersion}

	AddToScheme = SchemeBuilder.AddToScheme
)
