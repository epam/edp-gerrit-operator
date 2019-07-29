package service

import (
	"gerrit-operator/pkg/apis/edp/v1alpha1"
	"gopkg.in/resty.v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	StatusInstall     = "installing"
	StatusFailed      = "failed"
	StatusCreated     = "created"
	StatusConfiguring = "configuring"
	StatusConfigured  = "configured"
	StatusReady       = "ready"
	StatuseExposeConf = "exposing config"
)

type Client struct {
	client resty.Client
}

type GerritService interface {
	Install(instance v1alpha1.Gerrit) (*v1alpha1.Gerrit, error)
	Configure(instance v1alpha1.Gerrit) (*v1alpha1.Gerrit, error)
	ExposeConfiguration(instance v1alpha1.Gerrit) (*v1alpha1.Gerrit, error)
	Integrate(instance v1alpha1.Gerrit) (*v1alpha1.Gerrit, error)
}

func NewGerritService(platformService PlatformService, k8sClient client.Client) GerritService {
	return GerritServiceImpl{platformService: platformService, k8sClient: k8sClient}
}

type GerritServiceImpl struct {
	// Providing sonar service implementation through the interface (platform abstract)
	platformService PlatformService
	k8sClient       client.Client
}

func (service GerritServiceImpl) Install(instance v1alpha1.Gerrit) (*v1alpha1.Gerrit, error) {
	return &instance, nil
}

func (service GerritServiceImpl) Configure(instance v1alpha1.Gerrit) (*v1alpha1.Gerrit, error) {
	return &instance, nil
}

func (service GerritServiceImpl) ExposeConfiguration(instance v1alpha1.Gerrit) (*v1alpha1.Gerrit, error) {
	return &instance, nil
}

func (service GerritServiceImpl) Integrate(instance v1alpha1.Gerrit) (*v1alpha1.Gerrit, error) {
	return &instance, nil
}
