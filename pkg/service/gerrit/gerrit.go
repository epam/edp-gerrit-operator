package gerrit

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"

	"github.com/dchest/uniuri"
	"github.com/epam/edp-gerrit-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/epam/edp-gerrit-operator/v2/pkg/client/gerrit"
	"github.com/epam/edp-gerrit-operator/v2/pkg/controller/helper"
	"github.com/epam/edp-gerrit-operator/v2/pkg/service/gerrit/spec"
	"github.com/epam/edp-gerrit-operator/v2/pkg/service/helpers"
	"github.com/epam/edp-gerrit-operator/v2/pkg/service/platform"
	platformHelper "github.com/epam/edp-gerrit-operator/v2/pkg/service/platform/helper"
	jenPlatformHelper "github.com/epam/edp-jenkins-operator/v2/pkg/service/platform/helper"
	keycloakApi "github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	coreV1Api "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var log = ctrl.Log.WithName("service_gerrit")

const (
	imgFolder                        = "img"
	gerritIcon                       = "gerrit.svg"
	jenkinsDefaultScriptConfigMapKey = "context"
)

// Interface expresses behaviour of the Gerrit EDP Component
type Interface interface {
	IsDeploymentReady(instance *v1alpha1.Gerrit) (bool, error)
	Configure(instance *v1alpha1.Gerrit) (*v1alpha1.Gerrit, bool, error)
	ExposeConfiguration(instance *v1alpha1.Gerrit) (*v1alpha1.Gerrit, error)
	Integrate(instance *v1alpha1.Gerrit) (*v1alpha1.Gerrit, error)
	GetGerritSSHUrl(instance *v1alpha1.Gerrit) (string, error)
	GetServicePort(instance *v1alpha1.Gerrit) (int32, error)
	GetRestClient(gerritInstance *v1alpha1.Gerrit) (gerrit.ClientInterface, error)
}

type ErrUserNotFound string

func (e ErrUserNotFound) Error() string {
	return string(e)
}

func IsErrUserNotFound(err error) bool {
	if err == nil {
		return false
	}

	_, ok := err.(ErrUserNotFound)
	return ok
}

// ComponentService implements gerrit.Interface
type ComponentService struct {
	// Providing Gerrit EDP component implementation through the interface (platform abstract)
	PlatformService platform.PlatformService
	client          client.Client
	k8sScheme       *runtime.Scheme
	gerritClient    gerrit.Client
}

// NewComponentService returns a new instance of a gerrit.Service type
func NewComponentService(ps platform.PlatformService, kc client.Client, ks *runtime.Scheme) Interface {
	return ComponentService{PlatformService: ps, client: kc, k8sScheme: ks}
}

// IsDeploymentReady check if DC for Gerrit is ready
func (s ComponentService) IsDeploymentReady(instance *v1alpha1.Gerrit) (bool, error) {
	return s.PlatformService.IsDeploymentReady(instance)
}

// Configure contains logic related to self configuration of the Gerrit EDP Component
func (s ComponentService) Configure(instance *v1alpha1.Gerrit) (*v1alpha1.Gerrit, bool, error) {
	gerritUrl, err := s.GetGerritSSHUrl(instance)
	if err != nil {
		return instance, false, errors.Wrapf(err, "Unable to get Gerrit SSH URL")
	}
	executableFilePath, err := helper.GetExecutableFilePath()
	if err != nil {
		return instance, false, errors.Wrapf(err, "Unable to get executable file path")
	}

	GerritScriptsPath := platformHelper.LocalScriptsRelativePath
	if !helper.RunningInCluster() {
		GerritScriptsPath = filepath.FromSlash(fmt.Sprintf("%v/../%v/%v", executableFilePath, platformHelper.LocalConfigsRelativePath, platformHelper.DefaultScriptsDirectory))
	}

	if err := s.PlatformService.CreateSecret(instance, instance.Name+"-admin-password", map[string][]byte{
		"user":     []byte(spec.GerritDefaultAdminUser),
		"password": []byte(uniuri.New()),
	}); err != nil {
		return instance, false, errors.Wrapf(err, "Failed to create admin Secret %v for Gerrit", instance.Name+"-admin-password")
	}

	sshPortService, err := s.GetServicePort(instance)
	if err != nil {
		return instance, false, err
	}

	sshPort, err := s.PlatformService.GetDeploymentSSHPort(instance)
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

	dUpdated, err := s.updateDeploymentConfigPort(sshPort, sshPortService, instance)
	if err != nil {
		return instance, false, err
	}

	if dUpdated {
		return instance, true, nil
	}

	gerritApiUrl, err := s.getGerritRestApiUrl(instance)
	if err != nil {
		return instance, false, errors.Wrapf(err, "Failed to get Gerrit REST API URL %v/%v", instance.Namespace, instance.Name)
	}

	gerritAdminPassword, err := s.getGerritAdminPassword(instance)
	if err != nil {
		return instance, false, errors.Wrapf(err, "Failed to get Gerrit admin password from secret for %v/%v", instance.Namespace, instance.Name)
	}

	podList, err := s.PlatformService.GetPods(instance.Namespace, metav1.ListOptions{LabelSelector: "app=" + instance.Name})
	if err != nil || len(podList.Items) != 1 {
		return instance, false, errors.Wrapf(err, "Unable to determine Gerrit pod name: %v", len(podList.Items))
	}

	_, gerritAdminPublicKey, err := s.createSSHKeyPairs(instance, instance.Name+"-admin")
	if err != nil {
		return instance, false, errors.Wrapf(err, "Failed to create Gerrit admin SSH keypair %v/%v", instance.Namespace, instance.Name)
	}

	_, _, err = s.createSSHKeyPairs(instance, instance.Name+"-project-creator")
	if err != nil {
		return instance, false, errors.Wrapf(err, "Failed to create Gerrit project-creator SSH keypair %v/%v", instance.Namespace, instance.Name)
	}

	err = s.gerritClient.InitNewRestClient(instance, gerritApiUrl, spec.GerritDefaultAdminUser, gerritAdminPassword)
	if err != nil {
		return instance, false, errors.Wrapf(err, "Failed to initialize Gerrit REST client for %v/%v", instance.Namespace, instance.Name)
	}

	status, err := s.gerritClient.CheckCredentials()
	if (status != 401 && status >= 400 && status <= 600) || err != nil {
		return instance, false, errors.Wrapf(err, "Failed to check credentials in Gerrit")
	}

	if status == 401 {
		err = s.gerritClient.InitNewRestClient(instance, gerritApiUrl, spec.GerritDefaultAdminUser, spec.GerritDefaultAdminPassword)
		if err != nil {
			return instance, false, errors.Wrapf(err, "Failed to initialize Gerrit REST client for %v/%v", instance.Namespace, instance.Name)
		}
		status, err = s.gerritClient.CheckCredentials()
		if (status != 401 && status >= 400 && status <= 600) || err != nil {
			return instance, false, errors.Wrapf(err, "Failed to check credentials in Gerrit")
		}

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

		_, err = s.gerritClient.CreateGroup(spec.GerritCIToolsGroupName, spec.GerritCIToolsGroupDescription,
			true)
		if err != nil {
			return instance, false, err
		}

		_, err = s.gerritClient.CreateGroup(spec.GerritProjectBootstrappersGroupName,
			spec.GerritProjectBootstrappersGroupDescription, true)
		if err != nil {
			return instance, false, err
		}

		err = s.gerritClient.InitAllProjects(*instance, s.PlatformService, GerritScriptsPath, podList.Items[0].Name,
			string(gerritAdminPublicKey))
		if err != nil {
			return instance, false, errors.Wrapf(err, "Failed to initialize Gerrit All-Projects project")
		}
	}

	return instance, false, s.processUsers(instance)
}

func (s ComponentService) processUsers(instance *v1alpha1.Gerrit) error {
	processedUsers := make(map[string]struct{})
	for _, pu := range instance.Status.ProcessedUsers {
		processedUsers[pu] = struct{}{}
	}

	var userErr error
	for _, user := range instance.Spec.Users {
		log.Info("add user to groups", "user", user, "group(s)", user.Groups)
		userStatus, err := s.gerritClient.GetUser(user.Username)
		if err != nil {
			return errors.Wrapf(err, "Getting %v user failed", user.Username)
		}

		if *userStatus == 404 {
			msg := fmt.Sprintf("User %v not found in Gerrit", user.Username)
			log.Info(msg)
			userErr = ErrUserNotFound(msg)
		} else {
			if err := s.gerritClient.AddUserToGroups(user.Username, user.Groups); err != nil {
				return errors.Wrapf(err, "Failed to add user %v to groups: %v", user.Username, user.Groups)
			}

			if _, ok := processedUsers[user.Username]; !ok {
				instance.Status.ProcessedUsers = append(instance.Status.ProcessedUsers, user.Username)
				instance.Status.Status = spec.StatusConfiguring
			}
		}
	}

	if err := s.gerritClient.RemoveUsersFromGroup(instance.Spec.Users, processedUsers); err != nil {
		return errors.Wrap(err, "unable to remove users from groups")
	}

	return userErr
}

// ExposeConfiguration describes integration points of the Gerrit EDP Component for the other Operators and Components
func (s ComponentService) ExposeConfiguration(instance *v1alpha1.Gerrit) (*v1alpha1.Gerrit, error) {
	vLog := log.WithValues("gerrit", instance.Name)
	vLog.Info("start exposing configuration")
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
		return nil, errors.Wrapf(err, "Failed to get Secret %v for %v/%v", ciUserSecretName,
			instance.Namespace, instance.Name)
	}

	privateKey, publicKey, err := helpers.GenerateKeyPairs()
	if err != nil {
		return instance, errors.Wrapf(err, "Unable to generate SSH key pairs for Gerrit")
	}

	ciUserSshSecretName := fmt.Sprintf("%s-ciuser%s", instance.Name, spec.SshKeyPostfix)
	if err := s.PlatformService.CreateSecret(instance, ciUserSshSecretName, map[string][]byte{
		"username":   []byte(ciUserSshSecretName),
		"id_rsa":     privateKey,
		"id_rsa.pub": publicKey,
	}); err != nil {
		return instance, errors.Wrapf(err, "Failed to create Secret with SSH key pairs for Gerrit")
	}

	err = s.PlatformService.CreateJenkinsServiceAccount(instance.Namespace, ciUserSshSecretName, "ssh")
	if err != nil {
		return instance, errors.Wrapf(err, "Failed to create Jenkins Service Account %s", ciUserSshSecretName)
	}

	ciUserSshKeyAnnotationKey := helpers.GenerateAnnotationKey(spec.EdpCiUSerSshKeySuffix)
	s.setAnnotation(instance, ciUserSshKeyAnnotationKey, instance.Name+"-ciuser"+spec.SshKeyPostfix)
	projectCreatorUserSshKeyAnnotationKey := helpers.GenerateAnnotationKey(spec.EdpProjectCreatorSshKeySuffix)
	s.setAnnotation(instance, projectCreatorUserSshKeyAnnotationKey, instance.Name+"-project-creator")

	if err := s.gerritClient.CreateUser(spec.GerritDefaultCiUserUser, string(ciUserCredentials["password"]),
		"CI Jenkins", string(publicKey)); err != nil {
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
			spec.GerritCIToolsGroupName, spec.GerritServiceUsersGroup},
		spec.GerritDefaultProjectCreatorUser: {spec.GerritProjectBootstrappersGroupName, spec.GerritAdministratorsGroup},
	}

	for _, user := range reflect.ValueOf(userGroups).MapKeys() {
		if err := s.gerritClient.AddUserToGroups(reflect.Value.String(user), userGroups[reflect.Value.String(user)]); err != nil {
			return instance, errors.Wrapf(err, "Failed to add user %v to groups %v",
				reflect.Value.String(user), userGroups[reflect.Value.String(user)])
		}

	}

	if err := s.client.Update(context.TODO(), instance); err != nil {
		return nil, errors.Wrap(err, "couldn't update project")
	}

	if instance.Spec.KeycloakSpec.Enabled {
		secret, err := uuid.NewUUID()
		if err != nil {
			return instance, errors.Wrapf(err, fmt.Sprintf("Failed to generate secret for Gerrit in Keycloack"))
		}

		identityServiceClientCredentials := map[string][]byte{
			"client_id":    []byte(instance.Name),
			"clientSecret": []byte(secret.String()),
		}

		identityServiceSecretName := fmt.Sprintf("%v-%v", instance.Name, spec.IdentityServiceCredentialsSecretPostfix)
		err = s.PlatformService.CreateSecret(instance, identityServiceSecretName, identityServiceClientCredentials)
		if err != nil {
			return instance, errors.Wrapf(err, fmt.Sprintf("Failed to create secret %v", identityServiceSecretName))
		}

		annotationKey := helpers.GenerateAnnotationKey(spec.IdentityServiceCredentialsSecretPostfix)
		s.setAnnotation(instance, annotationKey, fmt.Sprintf("%v-%v", instance.Name, spec.IdentityServiceCredentialsSecretPostfix))
		if err := s.client.Update(context.TODO(), instance); err != nil {
			return nil, errors.Wrap(err, "couldn't update annotations")
		}
	}
	if err := s.createEDPComponent(*instance); err != nil {
		return nil, errors.Wrap(err, "unable to create EDP component")
	}

	return instance, err
}

func (s ComponentService) createEDPComponent(gerrit v1alpha1.Gerrit) error {
	vLog := log.WithValues("name", gerrit.Name)
	vLog.Info("creating EDP component")
	url, err := s.getUrl(gerrit)
	if err != nil {
		return err
	}
	icon, err := getIcon()
	if err != nil {
		return err
	}
	return s.PlatformService.CreateEDPComponentIfNotExist(gerrit, *url, *icon)
}

func (s ComponentService) getUrl(gerrit v1alpha1.Gerrit) (*string, error) {
	h, sc, err := s.PlatformService.GetExternalEndpoint(gerrit.Namespace, gerrit.Name)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%v://%v", sc, h)
	return &url, nil
}

func getIcon() (*string, error) {
	p, err := jenPlatformHelper.CreatePathToTemplateDirectory(imgFolder)
	if err != nil {
		return nil, err
	}
	fp := fmt.Sprintf("%v/%v", p, gerritIcon)
	f, err := os.Open(fp)
	if err != nil {
		return nil, err
	}
	reader := bufio.NewReader(f)
	content, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	encoded := base64.StdEncoding.EncodeToString(content)
	return &encoded, nil
}

// Integrate applies actions required for the integration with the other EDP Components
func (s ComponentService) Integrate(instance *v1alpha1.Gerrit) (*v1alpha1.Gerrit, error) {
	h, sc, err := s.PlatformService.GetExternalEndpoint(instance.Namespace, instance.Name)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get Route for %v/%v", instance.Namespace, instance.Name)
	}

	externalUrl := fmt.Sprintf("%v://%v", sc, h)

	if instance.Spec.KeycloakSpec.Enabled {
		client, err := s.getKeycloakClient(instance)
		if err != nil {
			return instance, err
		}

		if client == nil {
			err = s.createKeycloakClient(*instance, externalUrl)
			if err != nil {
				return instance, err
			}
		}

		keycloakEnvironmentValue, err := s.PlatformService.GenerateKeycloakSettings(instance)
		if err != nil {
			return instance, err
		}

		if err = s.PlatformService.PatchDeploymentEnv(*instance, *keycloakEnvironmentValue); err != nil {
			return instance, errors.Wrapf(err, fmt.Sprintf("Failed to add identity service information"))
		}
	} else {
		log.V(1).Info("Keycloak integration not enabled.")
	}

	err = s.configureGerritPluginInJenkins(instance, h, sc)
	if err != nil {
		return nil, err
	}

	return instance, nil
}

func (s ComponentService) getKeycloakClient(instance *v1alpha1.Gerrit) (*keycloakApi.KeycloakClient, error) {
	client := &keycloakApi.KeycloakClient{}
	err := s.client.Get(context.TODO(), types.NamespacedName{
		Name:      instance.Name,
		Namespace: instance.Namespace,
	}, client)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	return client, nil
}

func (s ComponentService) createKeycloakClient(instance v1alpha1.Gerrit, externalUrl string) error {
	client := &keycloakApi.KeycloakClient{
		TypeMeta: metav1.TypeMeta{
			Kind: "KeycloakClient",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
		},
		Spec: keycloakApi.KeycloakClientSpec{
			ClientId:                instance.Name,
			Public:                  true,
			WebUrl:                  externalUrl,
			AdvancedProtocolMappers: false,
			RealmRoles: &[]keycloakApi.RealmRole{
				{
					Name:      "gerrit-administrators",
					Composite: "administrator",
				},
				{
					Name:      "gerrit-users",
					Composite: "developer",
				},
			},
		},
	}

	if instance.Spec.KeycloakSpec.Realm != "" {
		client.Spec.TargetRealm = instance.Spec.KeycloakSpec.Realm
	}

	return s.client.Create(context.TODO(), client)
}

func (s ComponentService) GetRestClient(gerritInstance *v1alpha1.Gerrit) (gerrit.ClientInterface, error) {
	if s.gerritClient.GetResty() != nil {
		return &s.gerritClient, nil
	}

	if err := s.initRestClient(gerritInstance); err != nil {
		return nil, errors.Wrap(err, "unable to init gerrit rest client")
	}

	return &s.gerritClient, nil
}

func (s *ComponentService) initRestClient(instance *v1alpha1.Gerrit) error {
	vLog := log.WithValues("gerrit", instance.Name)
	vLog.Info("init rest client")
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
	vLog.Info("rest client has been initialized.")
	return nil
}

func (s *ComponentService) initSSHClient(instance *v1alpha1.Gerrit) error {
	vLog := log.WithValues("gerrit", instance.Name)
	vLog.Info("init ssh client")
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
	vLog.Info("ssh client has been initialized.")
	return nil
}

func (s ComponentService) getGerritRestApiUrl(instance *v1alpha1.Gerrit) (string, error) {
	gerritApiUrl := fmt.Sprintf("http://%v.%v:%v/%v", instance.Name, instance.Namespace, spec.GerritPort, spec.GerritRestApiUrlPath)
	if !helper.RunningInCluster() {
		h, sc, err := s.PlatformService.GetExternalEndpoint(instance.Namespace, instance.Name)
		if err != nil {
			return "", errors.Wrapf(err, "Failed to get external endpoint for %v/%v", instance.Namespace, instance.Name)
		}
		gerritApiUrl = fmt.Sprintf("%v://%v/%v", sc, h, spec.GerritRestApiUrlPath)
	}
	return gerritApiUrl, nil
}

func (s ComponentService) GetGerritSSHUrl(instance *v1alpha1.Gerrit) (string, error) {
	gerritSSHUrl := fmt.Sprintf("%v.%v", instance.Name, instance.Namespace)
	if !helper.RunningInCluster() {
		h, _, err := s.PlatformService.GetExternalEndpoint(instance.Namespace, instance.Name)
		if err != nil {
			return "", errors.Wrapf(err, "Failed to get Service for %v/%v", instance.Namespace, instance.Name)
		}
		gerritSSHUrl = h
	}
	return gerritSSHUrl, nil
}

func (s ComponentService) getGerritAdminPassword(instance *v1alpha1.Gerrit) (string, error) {
	secretName := fmt.Sprintf("%v-admin-password", instance.Name)
	gerritAdminCredentials, err := s.PlatformService.GetSecretData(instance.Namespace, secretName)
	if err != nil {
		return "", errors.Wrapf(err, "Failed to get Secret %v for %v/%v", secretName, instance.Namespace, instance.Name)
	}
	return string(gerritAdminCredentials["password"]), nil
}

func (s ComponentService) createSSHKeyPairs(instance *v1alpha1.Gerrit, secretName string) ([]byte, []byte, error) {
	secretData, err := s.PlatformService.GetSecretData(instance.Namespace, secretName)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "Unable to get data from secret %v", secretName)
	}

	if secretData == nil {
		privateKey, publicKey, err := helpers.GenerateKeyPairs()
		if err != nil {
			return nil, nil, errors.Wrapf(err, "Unable to generate SSH key pairs for Gerrit")
		}

		if err := s.PlatformService.CreateSecret(instance, secretName, map[string][]byte{
			"id_rsa":     privateKey,
			"id_rsa.pub": publicKey,
		}); err != nil {
			return nil, nil, errors.Wrapf(err, "Failed to create Secret with SSH key pairs for Gerrit")
		}
		return privateKey, publicKey, nil
	}

	return secretData["id_rsa"], secretData["id_rsa.pub"], nil
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

	return 0, errors.Wrapf(err, "Unable to determine Gerrit ssh port")
}

func (s ComponentService) updateDeploymentConfigPort(sshPort, sshPortService int32, instance *v1alpha1.Gerrit) (bool, error) {
	if sshPort != sshPortService || sshPort == 0 {
		newEnv := []coreV1Api.EnvVar{
			{
				Name:  spec.SSHListnerEnvName,
				Value: fmt.Sprintf("*:%d", sshPortService),
			},
		}
		if err := s.PlatformService.PatchDeploymentEnv(*instance, newEnv); err != nil {
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

func (s ComponentService) configureGerritPluginInJenkins(instance *v1alpha1.Gerrit, host string, scheme string) error {
	sshPort, err := s.GetServicePort(instance)
	if err != nil {
		return errors.Wrapf(err, "Failed to get SSH port for %v/%v", instance.Namespace, instance.Name)
	}

	ciUserCredentialsName := fmt.Sprintf("%v-%v", instance.Name, spec.GerritDefaultCiUserSecretPostfix)
	ciUserCredentials, err := s.PlatformService.GetSecretData(instance.Namespace, ciUserCredentialsName)
	if err != nil {
		return errors.Wrapf(err, "Failed to get Secret for CI user for %v/%v", instance.Namespace, instance.Name)
	}

	externalUrl := fmt.Sprintf("%v://%v", scheme, host)
	jenkinsPluginInfo := platformHelper.InitNewJenkinsPluginInfo()
	jenkinsPluginInfo.ServerName = instance.Name
	jenkinsPluginInfo.ExternalUrl = externalUrl
	jenkinsPluginInfo.SshPort = sshPort
	jenkinsPluginInfo.UserName = string(ciUserCredentials["user"])
	jenkinsPluginInfo.HttpPassword = string(ciUserCredentials["password"])

	jenkinsScriptContext, err := platformHelper.ParseDefaultTemplate(jenkinsPluginInfo)
	if err != nil {
		return err
	}

	jenkinsPluginConfigurationName := fmt.Sprintf("%v-%v", instance.Name, spec.JenkinsPluginConfigPostfix)
	configMapData := map[string]string{
		jenkinsDefaultScriptConfigMapKey: jenkinsScriptContext.String(),
	}
	err = s.PlatformService.CreateConfigMap(instance, jenkinsPluginConfigurationName, configMapData)
	if err != nil {
		return err
	}

	err = s.PlatformService.CreateJenkinsScript(instance.Namespace, jenkinsPluginConfigurationName)

	return nil
}
