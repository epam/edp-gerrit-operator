package gerrit

import (
	"context"
	"fmt"
	"gerrit-operator/pkg/apis/v2/v1alpha1"
	"gerrit-operator/pkg/client/gerrit"
	"gerrit-operator/pkg/helper"
	"gerrit-operator/pkg/service/gerrit/spec"
	"gerrit-operator/pkg/service/helpers"
	"gerrit-operator/pkg/service/platform"
	"github.com/dchest/uniuri"
	"github.com/google/uuid"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/pkg/errors"
	coreV1Api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"path/filepath"
	"reflect"
	"regexp"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"strconv"
)

var log = logf.Log.WithName("service_gerrit")

// Interface expresses behaviour of the Gerrit EDP Component
type Interface interface {
	IsDeploymentConfigReady(instance v1alpha1.Gerrit) (bool, error)
	Install(instance *v1alpha1.Gerrit) (*v1alpha1.Gerrit, error)
	Configure(instance *v1alpha1.Gerrit) (*v1alpha1.Gerrit, bool, error)
	ExposeConfiguration(instance *v1alpha1.Gerrit) (*v1alpha1.Gerrit, error)
	Integrate(instance *v1alpha1.Gerrit) (*v1alpha1.Gerrit, error)
	GetGerritSSHUrl(instance *v1alpha1.Gerrit) (string, error)
	GetServicePort(instance *v1alpha1.Gerrit) (int32, error)
}

// ComponentService implements gerrit.Interface
type ComponentService struct {
	// Providing Gerrit EDP component implementation through the interface (platform abstract)
	PlatformService platform.PlatformService
	k8sClient       client.Client
	gerritClient    gerrit.Client
}

// NewComponentService returns a new instance of a gerrit.Service type
func NewComponentService(ps platform.PlatformService, kc client.Client) Interface {
	return ComponentService{PlatformService: ps, k8sClient: kc}
}

// IsDeploymentConfigReady check if DC for Gerrit is ready
func (s ComponentService) IsDeploymentConfigReady(instance v1alpha1.Gerrit) (bool, error) {
	gerritIsReady := false
	GerritDc, err := s.PlatformService.GetDeploymentConfig(instance)
	if err != nil {
		return gerritIsReady, helpers.LogErrorAndReturn(err)
	}
	if GerritDc.Status.UpdatedReplicas == 1 && GerritDc.Status.AvailableReplicas == 1 {
		gerritIsReady = true
	}
	return gerritIsReady, nil
}

// Install has a minimal set of logic, required to install "vanilla" Gerrit EDP Component
func (s ComponentService) Install(instance *v1alpha1.Gerrit) (*v1alpha1.Gerrit, error) {
	sa, err := s.PlatformService.CreateServiceAccount(instance)
	if err != nil {
		return instance, errors.Wrapf(err, "[ERROR] Failed to create Service Account for %v/%v",
			instance.Namespace, instance.Name)
	}

	if err := s.PlatformService.CreateSecurityContext(instance, sa); err != nil {
		return instance, errors.Wrapf(err, "[ERROR] Failed to create Security Context for %v/%v",
			instance.Namespace, instance.Name)
	}

	if err := s.PlatformService.CreateService(instance); err != nil {
		return instance, errors.Wrapf(err, "[ERROR] Failed to create Service for %v/%v",
			instance.Namespace, instance.Name)
	}

	if err := s.PlatformService.CreateExternalEndpoint(instance); err != nil {
		return instance, errors.Wrapf(err, "[ERROR] Failed to create External Endpoint for %v/%v",
			instance.Namespace, instance.Name)
	}

	if err := s.PlatformService.CreateVolume(instance); err != nil {
		return instance, errors.Wrapf(err, "[ERROR] Failed to create Volume for %v/%v",
			instance.Namespace, instance.Name)
	}

	if err := s.PlatformService.CreateDeployConf(instance); err != nil {
		return instance, errors.Wrapf(err, "[ERROR] Failed to create Deploy Config for %v/%v",
			instance.Namespace, instance.Name)
	}

	if err := s.PlatformService.CreateSecret(instance, instance.Name+"-admin-password", map[string][]byte{
		"user":     []byte(spec.GerritDefaultAdminUser),
		"password": []byte(uniuri.New()),
	}); err != nil {
		return instance, errors.Wrapf(err, "[ERROR] Failed to create admin Secret %v for Gerrit", instance.Name+"-admin-password")
	}

	return instance, nil
}

// Configure contains logic related to self configuration of the Gerrit EDP Component
func (s ComponentService) Configure(instance *v1alpha1.Gerrit) (*v1alpha1.Gerrit, bool, error) {
	gerritUrl, err := s.GetGerritSSHUrl(instance)
	if err != nil {
		return instance, false, errors.Wrapf(err, "Unable to get Gerrit SSH URL")
	}
	executableFilePath, err := helper.GetExecutableFilePath()
	if err != nil {
		return instance, false, errors.Wrapf(err, "[ERROR] Unable to get executable file path")
	}

	GerritScriptsPath := spec.GerritDefaultScriptsPath
	if _, err = k8sutil.GetOperatorNamespace(); err != nil && err == k8sutil.ErrNoNamespace {
		GerritScriptsPath = filepath.FromSlash(fmt.Sprintf("%v/%v/scripts", executableFilePath, spec.LocalConfigsRelativePath))
	}

	sshPortService, err := s.GetServicePort(instance)
	if err != nil {
		return instance, false, err
	}

	sshPortDC, err := s.getDcSshPortNumber(instance)
	if err != nil {
		return instance, false, err
	}

	service, err := s.PlatformService.GetService(instance.Namespace, instance.Name)
	if err != nil {
		return instance, false, errors.Wrapf(err, "Unable to get Gerrit service")
	}

	err = s.PlatformService.UpdateService(*service, sshPortService)
	if err != nil {
		return instance, false, errors.Wrapf(err, "Unable to update Gerrit service")
	}

	dcUpdated, err := s.updateDeploymentConfigPort(sshPortDC, sshPortService, instance)
	if err != nil {
		return instance, false, err
	}

	if dcUpdated {
		return instance, true, nil
	}

	gerritApiUrl, err := s.getGerritRestApiUrl(instance)
	if err != nil {
		return instance, false, errors.Wrapf(err, "[ERROR] Failed to get Gerrit REST API URL %v/%v", instance.Namespace, instance.Name)
	}

	gerritAdminPassword, err := s.getGerritAdminPassword(instance)
	if err != nil {
		return instance, false, errors.Wrapf(err, "[ERROR] Failed to get Gerrit admin password from secret for %v/%v", instance.Namespace, instance.Name)
	}

	podList, err := s.PlatformService.GetPods(instance.Namespace, metav1.ListOptions{LabelSelector: "deploymentconfig=" + instance.Name})
	if err != nil || len(podList.Items) != 1 {
		return instance, false, errors.Wrapf(err, "[ERROR] Unable to determine Gerrit pod name: %v", len(podList.Items))
	}

	_, gerritAdminPublicKey, err := s.createSSHKeyPairs(instance, instance.Name+"-admin")
	if err != nil {
		return instance, false, errors.Wrapf(err, "[ERROR] Failed to create Gerrit admin SSH keypair %v/%v", instance.Namespace, instance.Name)
	}

	_, _, err = s.createSSHKeyPairs(instance, instance.Name+"-project-creator")
	if err != nil {
		return instance, false, errors.Wrapf(err, "[ERROR] Failed to create Gerrit project-creator SSH keypair %v/%v", instance.Namespace, instance.Name)
	}

	err = s.gerritClient.InitNewRestClient(instance, gerritApiUrl, spec.GerritDefaultAdminUser, gerritAdminPassword)
	if err != nil {
		return instance, false, errors.Wrapf(err, "[ERROR] Failed to initialize Gerrit REST client for %v/%v", instance.Namespace, instance.Name)
	}

	status, err := s.gerritClient.CheckCredentials()

	if status == 401 {
		err = s.gerritClient.InitNewRestClient(instance, gerritApiUrl, spec.GerritDefaultAdminUser, spec.GerritDefaultAdminPassword)
		if err != nil {
			return instance, false, errors.Wrapf(err, "Failed to initialize Gerrit REST client for %v/%v", instance.Namespace, instance.Name)
		}
		status, err = s.gerritClient.CheckCredentials()

		if status == 401 {
			instance, err := s.gerritClient.InitAdminUser(*instance, s.PlatformService, GerritScriptsPath, podList.Items[0].Name,
				string(gerritAdminPublicKey))
			if err != nil {
				return &instance, false, errors.Wrapf(err, "Failed to initialize Gerrit Admin User")
			}
		}

		err := s.setGerritAdminUserPassword(*instance, gerritUrl, gerritAdminPassword, gerritApiUrl, sshPortService)
		if err != nil {
			return instance, false, err
		}
	}

	gerritAdminSshKeys, err := s.PlatformService.GetSecret(instance.Namespace, instance.Name+"-admin")
	if err != nil {
		return instance, false, err
	}

	_, _, err = s.PlatformService.ExecInPod(instance.Namespace, podList.Items[0].Name,
		[]string{"/bin/sh", "-c", "chown -R gerrit2:gerrit2 /var/gerrit/review_site"})
	if err != nil {
		return instance, false, err
	}

	err = s.gerritClient.InitNewSshClient(spec.GerritDefaultAdminUser, gerritAdminSshKeys["id_rsa"], gerritUrl, sshPortService)
	if err != nil {
		return instance, false, err
	}

	ciToolsStatus, err := s.gerritClient.CheckGroup(spec.GerritCIToolsGroupName)
	if err != nil {
		return instance, false, err
	}
	projectBootstrappersStatus, err := s.gerritClient.CheckGroup(spec.GerritProjectBootstrappersGroupName)
	if err != nil {
		return instance, false, err
	}

	if *ciToolsStatus == 404 || *projectBootstrappersStatus == 404 {

		err = s.gerritClient.InitNewSshClient(spec.GerritDefaultAdminUser, gerritAdminSshKeys["id_rsa"], gerritUrl, sshPortService)
		if err != nil {
			return instance, false, err
		}

		err = s.gerritClient.CreateGroup(spec.GerritCIToolsGroupName, spec.GerritCIToolsGroupDescription)
		if err != nil {
			return instance, false, err
		}

		err = s.gerritClient.CreateGroup(spec.GerritProjectBootstrappersGroupName, spec.GerritProjectBootstrappersGroupDescription)
		if err != nil {
			return instance, false, err
		}

		err = s.gerritClient.InitAllProjects(*instance, s.PlatformService, GerritScriptsPath, podList.Items[0].Name,
			string(gerritAdminPublicKey))
		if err != nil {
			return instance, false, errors.Wrapf(err, "Failed to initialize Gerrit All-Projects project")
		}
	}

	for _, user := range instance.Spec.Users {
		userStatus, err := s.gerritClient.GetUser(user.Username)
		if err != nil {
			return instance, false, errors.Wrapf(err, "Getting %v user failed", user.Username)
		}

		if *userStatus == 404 {
			log.Info(fmt.Sprintf("User %v not found in Gerrit", user.Username))
		} else {
			err := s.gerritClient.AddUserToGroups(user.Username, user.Groups)
			if err != nil {
				return instance, false, errors.Wrapf(err, "Failed to add user %v to groups: %v", user.Username, user.Groups)
			}
		}
	}

	return instance, false, nil
}

// ExposeConfiguration describes integration points of the Gerrit EDP Component for the other Operators and Components
func (s ComponentService) ExposeConfiguration(instance *v1alpha1.Gerrit) (*v1alpha1.Gerrit, error) {
	if err := s.initRestClient(instance); err != nil {
		return instance, errors.Wrapf(err, "Failed to init Gerrit REST client")
	}

	if err := s.initSSHClient(instance); err != nil {
		return instance, errors.Wrapf(err, "Failed to init Gerrit SSH client")
	}

	ciUserSecretName := fmt.Sprintf("%v-%v", instance.Name, spec.GerritDefaultCiUserSecretPostfix)
	projectCreatorSecretKeyName := fmt.Sprintf("%v-%v", instance.Name, spec.GerritDefaultProjectCreatorSecretPostfix)
	projectCreatorSecretPasswordName := fmt.Sprintf("%v-%v-%v", instance.Name, spec.GerritDefaultProjectCreatorSecretPostfix, "password")

	if err := s.PlatformService.CreateSecret(instance, ciUserSecretName, map[string][]byte{
		"user":     []byte(spec.GerritDefaultCiUserUser),
		"password": []byte(uniuri.New()),
	}); err != nil {
		return instance, errors.Wrapf(err, "Failed to create ci user Secret %v for Gerrit", ciUserSecretName)
	}

	ciUserAnnotationKey := helpers.GenerateAnnotationKey(spec.EdpCiUserSuffix)
	s.setAnnotation(instance, ciUserAnnotationKey, ciUserSecretName)

	if err := s.PlatformService.CreateSecret(instance, projectCreatorSecretPasswordName, map[string][]byte{
		"user":     []byte(spec.GerritDefaultProjectCreatorUser),
		"password": []byte(uniuri.New()),
	}); err != nil {
		return instance, errors.Wrapf(err, "Failed to create project-creator Secret %v for Gerrit", projectCreatorSecretPasswordName)
	}

	projectCreatorUserAnnotationKey := helpers.GenerateAnnotationKey(spec.EdpProjectCreatorUserSuffix)
	s.setAnnotation(instance, projectCreatorUserAnnotationKey, projectCreatorSecretPasswordName)

	ciUserCredentials, err := s.PlatformService.GetSecretData(instance.Namespace, ciUserSecretName)
	if err != nil {
		return nil, errors.Wrapf(err, "[ERROR] Failed to get Secret %v for %v/%v", ciUserSecretName,
			instance.Namespace, instance.Name)
	}

	_, ciUserPublicKey, err := s.createSSHKeyPairs(instance, instance.Name+"-ciuser"+spec.SshKeyPostfix)
	if err != nil {
		return instance, errors.Wrapf(err, "Failed to create Gerrit CI User SSH keypair %v/%v", instance.Namespace, instance.Name)
	}

	ciUserSshKeyAnnotationKey := helpers.GenerateAnnotationKey(spec.EdpCiUSerSshKeySuffix)
	s.setAnnotation(instance, ciUserSshKeyAnnotationKey, instance.Name+"-ciuser"+spec.SshKeyPostfix)
	projectCreatorUserSshKeyAnnotationKey := helpers.GenerateAnnotationKey(spec.EdpProjectCreatorSshKeySuffix)
	s.setAnnotation(instance, projectCreatorUserSshKeyAnnotationKey, instance.Name+"-project-creator")

	if err := s.gerritClient.CreateUser(spec.GerritDefaultCiUserUser, string(ciUserCredentials["password"]),
		"CI Jenkins", string(ciUserPublicKey)); err != nil {
		return instance, errors.Wrapf(err, "Failed to create ci user %v in Gerrit", spec.GerritDefaultCiUserUser)
	}

	projectCreatorCredentials, err := s.PlatformService.GetSecretData(instance.Namespace, projectCreatorSecretPasswordName)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get Secret %v for %v/%v", projectCreatorSecretPasswordName,
			instance.Namespace, instance.Name)
	}

	projectCreatorKeys, err := s.PlatformService.GetSecretData(instance.Namespace, projectCreatorSecretKeyName)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get Secret %v for %v/%v", projectCreatorSecretKeyName,
			instance.Namespace, instance.Name)
	}

	if err := s.gerritClient.CreateUser(spec.GerritDefaultProjectCreatorUser, string(projectCreatorCredentials["password"]),
		"Project Creator", string(projectCreatorKeys["id_rsa.pub"])); err != nil {
		return instance, errors.Wrapf(err, "Failed to create project-creator user %v in Gerrit", spec.GerritDefaultProjectCreatorUser)
	}

	userGroups := map[string][]string{
		spec.GerritDefaultCiUserUser: {spec.GerritProjectBootstrappersGroupName, spec.GerritAdministratorsGroup,
			spec.GerritCIToolsGroupName, spec.GerritNonInteractiveUsersGroup},
		spec.GerritDefaultProjectCreatorUser: {spec.GerritProjectBootstrappersGroupName, spec.GerritAdministratorsGroup},
	}

	for _, user := range reflect.ValueOf(userGroups).MapKeys() {
		if err := s.gerritClient.AddUserToGroups(reflect.Value.String(user), userGroups[reflect.Value.String(user)]); err != nil {
			return instance, errors.Wrapf(err, "Failed to add user %v to groups %v",
				reflect.Value.String(user), userGroups[reflect.Value.String(user)])
		}

	}

	_ = s.k8sClient.Update(context.TODO(), instance)

	if instance.Spec.KeycloakSpec.Enabled {
		secret, err := uuid.NewUUID()
		if err != nil {
			return instance, errors.Wrapf(err, fmt.Sprintf("Failed to generate secret for Gerrit in Keycloack"))
		}

		identityServiceClientCredentials := map[string][]byte{
			"client_id":     []byte(instance.Name),
			"client_secret": []byte(secret.String()),
		}

		identityServiceSecretName := fmt.Sprintf("%v-%v", instance.Name, spec.IdentityServiceCredentialsSecretPostfix)
		err = s.PlatformService.CreateSecret(instance, identityServiceSecretName, identityServiceClientCredentials)
		if err != nil {
			return instance, errors.Wrapf(err, fmt.Sprintf("Failed to create secret %v", identityServiceSecretName))
		}

		annotationKey := helpers.GenerateAnnotationKey(spec.IdentityServiceCredentialsSecretPostfix)
		s.setAnnotation(instance, annotationKey, fmt.Sprintf("%v-%v", instance.Name, spec.IdentityServiceCredentialsSecretPostfix))
		_ = s.k8sClient.Update(context.TODO(), instance)
	}
	return instance, nil
}

// Integrate applies actions required for the integration with the other EDP Components
func (s ComponentService) Integrate(instance *v1alpha1.Gerrit) (*v1alpha1.Gerrit, error) {
	if instance.Spec.KeycloakSpec.Enabled {
		keycloakEnvironmentValue := s.PlatformService.GenerateKeycloakSettings(instance)

		deployConf, err := s.PlatformService.GetDeploymentConfig(*instance)
		if err != nil {
			return instance, err
		}
		if err = s.PlatformService.PatchDeployConfEnv(*instance, deployConf, keycloakEnvironmentValue); err != nil {
			return instance, errors.Wrapf(err, fmt.Sprintf("Failed to add identity service information"))
		}
		return instance, nil

	} else {
		log.V(1).Info("Keycloak integration not enabled.")
	}

	return instance, nil
}

func (s *ComponentService) initRestClient(instance *v1alpha1.Gerrit) error {
	gerritAdminPassword, err := s.getGerritAdminPassword(instance)
	if err != nil {
		return errors.Wrapf(err, "Failed to get Gerrit admin password from secret for %v/%v", instance.Namespace, instance.Name)
	}

	gerritApiUrl, err := s.getGerritRestApiUrl(instance)
	if err != nil {
		return errors.Wrapf(err, "Failed to get Gerrit REST API URL %v/%v", instance.Namespace, instance.Name)
	}

	err = s.gerritClient.InitNewRestClient(instance, gerritApiUrl, spec.GerritDefaultAdminUser, gerritAdminPassword)
	if err != nil {
		return errors.Wrapf(err, "Failed to initialize Gerrit REST client for %v/%v", instance.Namespace, instance.Name)
	}

	return nil
}

func (s *ComponentService) initSSHClient(instance *v1alpha1.Gerrit) error {
	gerritUrl, err := s.GetGerritSSHUrl(instance)
	if err != nil {
		return err
	}

	sshPortService, err := s.GetServicePort(instance)
	if err != nil {
		return err
	}

	gerritAdminSshKeys, err := s.PlatformService.GetSecret(instance.Namespace, instance.Name+"-admin")
	if err != nil {
		return err
	}

	err = s.gerritClient.InitNewSshClient(spec.GerritDefaultAdminUser, gerritAdminSshKeys["id_rsa"], gerritUrl, sshPortService)
	if err != nil {
		return errors.Wrapf(err, "Failed to init Gerrit SSH client %v/%v", instance.Namespace, instance.Name)
	}

	return nil
}

func (s ComponentService) getGerritRestApiUrl(instance *v1alpha1.Gerrit) (string, error) {
	gerritApiUrl := fmt.Sprintf("http://%v.%v:%v/%v", instance.Name, instance.Namespace, spec.GerritPort, spec.GerritRestApiUrlPath)
	if _, err := k8sutil.GetOperatorNamespace(); err != nil && err == k8sutil.ErrNoNamespace {
		gerritRoute, gerritRouteScheme, err := s.PlatformService.GetRoute(instance.Namespace, instance.Name)
		if err != nil {
			return "", errors.Wrapf(err, "[ERROR] Failed to get Route for %v/%v", instance.Namespace, instance.Name)
		}
		gerritApiUrl = fmt.Sprintf("%v://%v/%v", gerritRouteScheme, gerritRoute.Spec.Host, spec.GerritRestApiUrlPath)
	}
	return gerritApiUrl, nil
}

func (s ComponentService) GetGerritSSHUrl(instance *v1alpha1.Gerrit) (string, error) {
	gerritSSHUrl := fmt.Sprintf("%v.%v", instance.Name, instance.Namespace)
	if _, err := k8sutil.GetOperatorNamespace(); err != nil && err == k8sutil.ErrNoNamespace {
		gerritRoute, _, err := s.PlatformService.GetRoute(instance.Namespace, instance.Name)
		if err != nil {
			return "", errors.Wrapf(err, "Failed to get Service for %v/%v", instance.Namespace, instance.Name)
		}
		gerritSSHUrl = gerritRoute.Spec.Host
	}
	return gerritSSHUrl, nil
}

func (s ComponentService) getGerritAdminPassword(instance *v1alpha1.Gerrit) (string, error) {
	secretName := fmt.Sprintf("%v-admin-password", instance.Name)
	gerritAdminCredentials, err := s.PlatformService.GetSecretData(instance.Namespace, secretName)
	if err != nil {
		return "", errors.Wrapf(err, "[ERROR] Failed to get Secret %v for %v/%v", secretName, instance.Namespace, instance.Name)
	}
	return string(gerritAdminCredentials["password"]), nil
}

func (s ComponentService) createSSHKeyPairs(instance *v1alpha1.Gerrit, secretName string) ([]byte, []byte, error) {
	privateKey, publicKey, err := helpers.GenerateKeyPairs()
	if err != nil {
		return nil, nil, errors.Wrapf(err, "[ERROR] Unable to generate SSH key pairs for Gerrit")
	}

	if err := s.PlatformService.CreateSecret(instance, secretName, map[string][]byte{
		"id_rsa":     privateKey,
		"id_rsa.pub": publicKey,
	}); err != nil {
		return nil, nil, errors.Wrapf(err, "[ERROR] Failed to create Secret with SSH key pairs for Gerrit")
	}

	return privateKey, publicKey, nil
}

func (s ComponentService) setGerritAdminUserPassword(instance v1alpha1.Gerrit, gerritUrl, gerritAdminPassword,
	gerritApiUrl string, sshPortService int32) error {
	gerritAdminSshKeys, err := s.PlatformService.GetSecret(instance.Namespace, instance.Name+"-admin")
	if err != nil {
		return errors.Wrapf(err, "Failed to get Gerrit admin secret for %v/%v", instance.Namespace, instance.Name)
	}

	err = s.gerritClient.InitNewSshClient(spec.GerritDefaultAdminUser, gerritAdminSshKeys["id_rsa"], gerritUrl, sshPortService)
	if err != nil {
		return errors.Wrapf(err, "Failed to initialize Gerrit SSH client for %v/%v", instance.Namespace, instance.Name)
	}

	err = s.gerritClient.ChangePassword("admin", gerritAdminPassword)
	if err != nil {
		return errors.Wrapf(err, "Failed to set Gerrit admin password for %v/%v", instance.Namespace, instance.Name)
	}

	err = s.gerritClient.InitNewRestClient(&instance, gerritApiUrl, spec.GerritDefaultAdminUser, gerritAdminPassword)
	if err != nil {
		return errors.Wrapf(err, "Failed to initialize Gerrit REST client for %v/%v", instance.Namespace, instance.Name)
	}

	return nil
}

func (s ComponentService) GetServicePort(instance *v1alpha1.Gerrit) (int32, error) {
	service, err := s.PlatformService.GetService(instance.Namespace, instance.Name)
	if err != nil {
		return 0, err
	}

	for _, port := range service.Spec.Ports {
		if port.Name == spec.SSHPortName {
			return port.NodePort, nil
		}
	}

	return 0, errors.Wrapf(err, "[ERROR] Unable to determine Gerrit ssh port")
}

func (s ComponentService) getDcSshPortNumber(instance *v1alpha1.Gerrit) (int32, error) {
	dc, err := s.PlatformService.GetDeploymentConfig(*instance)
	if err != nil {
		return 0, err
	}

	for _, env := range dc.Spec.Template.Spec.Containers[0].Env {
		if env.Name == spec.SSHListnerEnvName {
			re := regexp.MustCompile(`[0-9]+`)
			if re.MatchString(env.Value) {
				ports := re.FindStringSubmatch(env.Value)
				if len(ports) != 1 {
					return 0, nil
				}
				portNumber, err := strconv.ParseInt(ports[0], 10, 32)
				if err != nil {
					return 0, err
				}
				return int32(portNumber), nil
			}
		}
	}

	return 0, nil
}

func (s ComponentService) updateDeploymentConfigPort(sshPortDC, sshPortService int32, instance *v1alpha1.Gerrit) (bool, error) {
	if sshPortDC != sshPortService || sshPortDC == 0 {
		newEnv := []coreV1Api.EnvVar{
			{
				Name:  spec.SSHListnerEnvName,
				Value: fmt.Sprintf("*:%d", sshPortService),
			},
		}
		deployConf, err := s.PlatformService.GetDeploymentConfig(*instance)
		if err != nil {
			return false, err
		}
		if err = s.PlatformService.PatchDeployConfEnv(*instance, deployConf, newEnv); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

//setAnnotation add key:value to current resource annotation
func (s ComponentService) setAnnotation(instance *v1alpha1.Gerrit, key string, value string) {
	if len(instance.Annotations) == 0 {
		instance.ObjectMeta.Annotations = map[string]string{
			key: value,
		}
	} else {
		instance.ObjectMeta.Annotations[key] = value
	}
}
