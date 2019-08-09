package client

import (
	"gerrit-operator/pkg/apis/edp/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
)

var SchemeGroupVersion = schema.GroupVersion{Group: "v2.edp.epam.com", Version: "v1alpha1"}

type EdpV1Client struct {
	crClient *rest.RESTClient
}

func NewForConfig(config *rest.Config) (*EdpV1Client, error) {
	if err := createCrdClient(config); err != nil {
		return nil, err
	}
	crClient, err := rest.RESTClientFor(config)
	if err != nil {
		return nil, err
	}
	return &EdpV1Client{crClient: crClient}, nil
}

func createCrdClient(cfg *rest.Config) error {
	scheme := runtime.NewScheme()
	SchemeBuilder := runtime.NewSchemeBuilder(addKnownTypes)
	if err := SchemeBuilder.AddToScheme(scheme); err != nil {
		return err
	}
	config := cfg
	config.GroupVersion = &SchemeGroupVersion
	config.APIPath = "/apis"
	config.ContentType = runtime.ContentTypeJSON
	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: serializer.NewCodecFactory(scheme)}

	return nil
}

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&v1alpha1.Gerrit{},
		&v1alpha1.GerritList{},
	)

	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}
