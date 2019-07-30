package service

import (
	"gerrit-operator/pkg/apis/edp/v1alpha1"
	"gopkg.in/resty.v1"
	logPrint "log"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	Image                 = "openfrontier/gerrit:2.14.8"
	DbImage               = "postgres:9.6"
	Port                  = 8080
	SSHPort               = 30001
	DBPort                = 5432
	LivenessProbeDelay    = 180
	ReadinessProbeDelay   = 60
	DbLivenessProbeDelay  = 60
	DbReadinessProbeDelay = 60
	MemoryRequest         = "500Mi"

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
	// Providing gerrit service implementation through the interface (platform abstract)
	platformService PlatformService
	k8sClient       client.Client
}

func (service GerritServiceImpl) Install(instance v1alpha1.Gerrit) (*v1alpha1.Gerrit, error) {
	logPrint.Printf("[INFO] Gerrit Installation started")
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
