package gerrit

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/dchest/uniuri"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	v1 "k8s.io/api/apps/v1"
	coreV1Api "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"

	gerritApi "github.com/epam/edp-gerrit-operator/v2/api/v1"
	gerritClient "github.com/epam/edp-gerrit-operator/v2/pkg/client/gerrit"
	"github.com/epam/edp-gerrit-operator/v2/pkg/client/git"
	"github.com/epam/edp-gerrit-operator/v2/pkg/service/gerrit/spec"
	"github.com/epam/edp-gerrit-operator/v2/pkg/service/helpers"
	"github.com/epam/edp-gerrit-operator/v2/pkg/service/platform"
	platformHelper "github.com/epam/edp-gerrit-operator/v2/pkg/service/platform/helper"
)

var log = ctrl.Log.WithName("service_gerrit")

const (
	user      = "user"
	password  = "password"
	rsaID     = "id_rsa"
	rsaIDFile = "id_rsa.pub"
	admin     = "-admin"

	DeploymentRestartAnnotation = "kubectl.kubernetes.io/restartedAt"
)

// Interface expresses behavior of the Gerrit EDP Component.
type Interface interface {
	IsDeploymentReady(instance *gerritApi.Gerrit) (bool, error)
	Configure(instance *gerritApi.Gerrit) (*gerritApi.Gerrit, bool, error)
	ExposeConfiguration(ctx context.Context, instance *gerritApi.Gerrit) (*gerritApi.Gerrit, error)
	Integrate(ctx context.Context, instance *gerritApi.Gerrit) (*gerritApi.Gerrit, error)
	GetGerritSSHUrl(instance *gerritApi.Gerrit) (string, error)
	GetServicePort(instance *gerritApi.Gerrit) (int32, error)
	GetRestClient(gerritInstance *gerritApi.Gerrit) (gerritClient.ClientInterface, error)
	GetGitClient(ctx context.Context, child Child, workDir string) (*git.Client, error)
}

type UserNotFoundError string

func (e UserNotFoundError) Error() string {
	return string(e)
}

func IsErrUserNotFound(err error) bool {
	var notFoundError UserNotFoundError
	return errors.As(err, &notFoundError)
}

// ComponentService implements gerrit.Interface.
type ComponentService struct {
	// Providing Gerrit EDP component implementation through the interface (platform abstract)
	PlatformService      platform.PlatformService
	client               client.Client
	k8sScheme            *runtime.Scheme
	gerritClient         gerritClient.ClientInterface
	runningInClusterFunc func() bool
}

// NewComponentService returns a new instance of a gerrit.Service type.
func NewComponentService(ps platform.PlatformService, kc client.Client, ks *runtime.Scheme) Interface {
	return ComponentService{
		PlatformService:      ps,
		client:               kc,
		k8sScheme:            ks,
		runningInClusterFunc: platformHelper.RunningInCluster,
		gerritClient:         &gerritClient.Client{},
	}
}

func (s ComponentService) runningInCluster() bool {
	if s.runningInClusterFunc != nil {
		return s.runningInClusterFunc()
	}

	return false
}

// IsDeploymentReady check if DC for Gerrit is ready.
func (s ComponentService) IsDeploymentReady(instance *gerritApi.Gerrit) (bool, error) {
	isReady, err := s.PlatformService.IsDeploymentReady(instance)
	if err != nil {
		return false, fmt.Errorf("failed to check readiness of Gerrit deployment %q: %w", instance.Name, err)
	}

	return isReady, nil
}

// Configure contains logic related to self configuration of the Gerrit EDP Component.
func (s ComponentService) Configure(instance *gerritApi.Gerrit) (*gerritApi.Gerrit, bool, error) {
	gerritUrl, err := s.GetGerritSSHUrl(instance)
	if err != nil {
		return instance, false, errors.Wrap(err, "unable to get Gerrit SSH URL")
	}

	executableFilePath, err := platformHelper.GetExecutableFilePath()
	if err != nil {
		return instance, false, errors.Wrap(err, "unable to get executable file path")
	}

	gerritScriptsPath := platformHelper.LocalScriptsRelativePath
	if !s.runningInCluster() {
		gerritScriptsPath = filepath.FromSlash(fmt.Sprintf("%v/../%v/%v", executableFilePath, platformHelper.LocalConfigsRelativePath, platformHelper.DefaultScriptsDirectory))
	}

	err = s.PlatformService.CreateSecret(
		instance,
		instance.Name+"-admin-password",
		map[string][]byte{
			user:     []byte(spec.GerritDefaultAdminUser),
			password: []byte(uniuri.New()),
		},
		map[string]string{},
	)
	if err != nil {
		return instance, false, errors.Wrapf(err, "failed to create admin Secret %s for Gerrit", instance.Name+"-admin-password")
	}

	sshPortService, err := s.GetServicePort(instance)
	if err != nil {
		return instance, false, err
	}

	sshPort, err := s.PlatformService.GetDeploymentSSHPort(instance)
	if err != nil {
		return instance, false, fmt.Errorf("failed to get SSH port for Gerrit instance %q: %w", instance.Name, err)
	}

	service, err := s.PlatformService.GetService(instance.Namespace, instance.Name)
	if err != nil {
		return instance, false, errors.Wrap(err, "unable to get Gerrit service")
	}

	err = s.PlatformService.UpdateService(service, sshPortService)
	if err != nil {
		return instance, false, errors.Wrapf(err, "unable to update Gerrit service")
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
		return instance, false, errors.Wrapf(err, "failed to get Gerrit REST API URL %v/%v", instance.Namespace, instance.Name)
	}

	gerritAdminPassword, err := s.getGerritAdminPassword(instance)
	if err != nil {
		return instance, false, errors.Wrapf(err, "failed to get Gerrit admin password from secret for %s/%s", instance.Namespace, instance.Name)
	}

	podList, err := s.PlatformService.GetPods(instance.Namespace, &metaV1.ListOptions{LabelSelector: "app=" + instance.Name})
	if err != nil || len(podList.Items) != 1 {
		return instance, false, errors.Wrapf(err, "unable to determine Gerrit pod name: %v", len(podList.Items))
	}

	_, gerritAdminPublicKey, err := s.createSSHKeyPairs(instance, instance.Name+admin)
	if err != nil {
		return instance, false, errors.Wrapf(err, "failed to create Gerrit admin SSH keypair %v/%v", instance.Namespace, instance.Name)
	}

	err = s.gerritClient.InitNewRestClient(instance, gerritApiUrl, spec.GerritDefaultAdminUser, gerritAdminPassword)
	if err != nil {
		return instance, false, errors.Wrapf(err, "failed to initialize Gerrit REST client for %v/%v", instance.Namespace, instance.Name)
	}

	status, err := s.gerritClient.CheckCredentials()
	if (status != http.StatusUnauthorized && status >= http.StatusBadRequest && status <= http.StatusHTTPVersionNotSupported) || err != nil {
		return instance, false, errors.Wrap(err, "failed to check credentials in Gerrit")
	}

	if status == http.StatusUnauthorized {
		// apparently we are trying to retry with default password
		err = s.gerritClient.InitNewRestClient(instance, gerritApiUrl, spec.GerritDefaultAdminUser, spec.GerritDefaultAdminPassword)
		if err != nil {
			return instance, false, errors.Wrapf(err, "Failed to initialize Gerrit REST client for %v/%v", instance.Namespace, instance.Name)
		}

		status, err = s.gerritClient.CheckCredentials()
		if (status != http.StatusUnauthorized && status >= http.StatusBadRequest && status <= http.StatusHTTPVersionNotSupported) || err != nil {
			return instance, false, errors.Wrap(err, "failed to check credentials in Gerrit")
		}

		if status == http.StatusUnauthorized {
			var adminInstance *gerritApi.Gerrit

			adminInstance, err = s.gerritClient.InitAdminUser(instance, s.PlatformService, gerritScriptsPath, podList.Items[0].Name,
				string(gerritAdminPublicKey))
			if err != nil {
				return adminInstance, false, errors.Wrap(err, "failed to initialize Gerrit Admin User")
			}
		}

		err = s.setGerritAdminUserPassword(instance, gerritUrl, gerritAdminPassword, gerritApiUrl, sshPortService)
		if err != nil {
			return instance, false, err
		}
	}

	gerritSecretName := instance.Name + admin

	gerritAdminSshKeys, err := s.PlatformService.GetSecret(instance.Namespace, instance.Name+admin)
	if err != nil {
		return instance, false, fmt.Errorf("failed to get secret %q for gerrit: %w", gerritSecretName, err)
	}

	podName := podList.Items[0].Name
	command := []string{"/bin/sh", "-c", "chown -R gerrit2:gerrit2 /var/gerrit/review_site"}

	_, _, err = s.PlatformService.ExecInPod(instance.Namespace, podName, command)
	if err != nil {
		return instance, false, fmt.Errorf("failed to exec command %q in a pod %q: %w", strings.Join(command, " "), podName, err)
	}

	err = s.gerritClient.InitNewSshClient(spec.GerritDefaultAdminUser, gerritAdminSshKeys[rsaID], gerritUrl, sshPortService)
	if err != nil {
		return instance, false, fmt.Errorf("failed to init ssh client for gerrit: %w", err)
	}

	ciToolsStatus, err := s.gerritClient.CheckGroup(spec.GerritCIToolsGroupName)
	if err != nil {
		return instance, false, fmt.Errorf("failed to check Gerrit group %q: %w", spec.GerritCIToolsGroupName, err)
	}

	projectBootstrappersStatus, err := s.gerritClient.CheckGroup(spec.GerritProjectBootstrappersGroupName)
	if err != nil {
		return instance, false, fmt.Errorf("failed to check Gerrit group %q: %w", spec.GerritProjectBootstrappersGroupName, err)
	}

	if *ciToolsStatus == http.StatusNotFound || *projectBootstrappersStatus == http.StatusNotFound {
		err = s.gerritClient.InitNewSshClient(spec.GerritDefaultAdminUser, gerritAdminSshKeys[rsaID], gerritUrl, sshPortService)
		if err != nil {
			return instance, false, fmt.Errorf("failed to init ssh client for gerrit: %w", err)
		}

		const errTemplate = "failed to create Gerrit group %q: %w"

		_, err = s.gerritClient.CreateGroup(spec.GerritCIToolsGroupName, spec.GerritCIToolsGroupDescription,
			true)
		if err != nil {
			return instance, false, fmt.Errorf(errTemplate, spec.GerritCIToolsGroupName, err)
		}

		_, err = s.gerritClient.CreateGroup(spec.GerritProjectBootstrappersGroupName,
			spec.GerritProjectBootstrappersGroupDescription, true)
		if err != nil {
			return instance, false, fmt.Errorf(errTemplate, spec.GerritProjectBootstrappersGroupName, err)
		}

		_, err = s.gerritClient.CreateGroup(spec.GerritProjectDevelopersGroupName,
			spec.GerritProjectDevelopersGroupNameDescription, true)
		if err != nil {
			return instance, false, fmt.Errorf(errTemplate, spec.GerritProjectDevelopersGroupName, err)
		}

		_, err = s.gerritClient.CreateGroup(spec.GerritReadOnlyGroupName, "", true)
		if err != nil {
			return instance, false, fmt.Errorf(errTemplate, spec.GerritReadOnlyGroupName, err)
		}

		err = s.gerritClient.InitAllProjects(instance, s.PlatformService, gerritScriptsPath, podList.Items[0].Name,
			string(gerritAdminPublicKey))
		if err != nil {
			return instance, false, errors.Wrapf(err, "failed to initialize Gerrit All-Projects project")
		}
	}

	return instance, false, nil
}

// ExposeConfiguration describes integration points of the Gerrit EDP Component for the other Operators and Components.
func (s ComponentService) ExposeConfiguration(ctx context.Context, instance *gerritApi.Gerrit) (*gerritApi.Gerrit, error) {
	vLog := log.WithValues("gerrit", instance.Name)
	vLog.Info("start exposing configuration")

	err := s.initRestClient(instance)
	if err != nil {
		return instance, errors.Wrapf(err, "Failed to init Gerrit REST client")
	}

	err = s.initSSHClient(instance)
	if err != nil {
		return instance, errors.Wrapf(err, "Failed to init Gerrit SSH client")
	}

	ciUserSecretName := formatSecretName(instance.Name, spec.GerritDefaultCiUserSecretPostfix)
	ciUserSshSecretName := fmt.Sprintf("%s-ciuser%s", instance.Name, spec.SshKeyPostfix)

	if err = s.PlatformService.CreateSecret(
		instance,
		ciUserSecretName,
		map[string][]byte{
			user:     []byte(spec.GerritDefaultCiUserUser),
			password: []byte(uniuri.New()),
		},
		map[string]string{},
	); err != nil {
		return instance, errors.Wrapf(err, "Failed to create ci user Secret %v for Gerrit", ciUserSecretName)
	}

	ciUserAnnotationKey := helpers.GenerateAnnotationKey(spec.EdpCiUserSuffix)
	s.setAnnotation(instance, ciUserAnnotationKey, ciUserSecretName)

	ciUserCredentials, err := s.PlatformService.GetSecretData(instance.Namespace, ciUserSecretName)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get Secret %s for %s/%s", ciUserSecretName,
			instance.Namespace, instance.Name)
	}

	privateKey, publicKey, err := helpers.GenerateKeyPairs()
	if err != nil {
		return instance, errors.Wrapf(err, "Unable to generate SSH key pairs for Gerrit")
	}

	err = s.PlatformService.CreateSecret(
		instance,
		ciUserSshSecretName,
		map[string][]byte{
			"username": []byte(spec.GerritDefaultCiUserUser),
			rsaID:      privateKey,
			rsaIDFile:  publicKey,
		},
		map[string]string{
			"app.edp.epam.com/secret-type": "repository",
		},
	)
	if err != nil {
		return instance, errors.Wrapf(err, "Failed to create Secret with SSH key pairs for Gerrit")
	}

	ciUserSshKeyAnnotationKey := helpers.GenerateAnnotationKey(spec.EdpCiUSerSshKeySuffix)
	s.setAnnotation(instance, ciUserSshKeyAnnotationKey, ciUserSshSecretName)

	err = s.gerritClient.CreateUser(spec.GerritDefaultCiUserUser, string(ciUserCredentials[password]),
		"EDP CI bot", string(publicKey))
	if err != nil {
		return instance, errors.Wrapf(err, "Failed to create ci user %v in Gerrit", spec.GerritDefaultCiUserUser)
	}

	userGroups := map[string][]string{
		spec.GerritDefaultCiUserUser: {spec.GerritProjectBootstrappersGroupName, spec.GerritAdministratorsGroup},
	}

	for _, userValue := range reflect.ValueOf(userGroups).MapKeys() {
		userStrValue := userValue.String()

		err = s.gerritClient.AddUserToGroups(userStrValue, userGroups[userStrValue])
		if err != nil {
			return instance, errors.Wrapf(err, "Failed to add user %v to groups %v", userStrValue, userGroups[userStrValue])
		}
	}

	err = s.exposeArgoCDConfiguration(ctx, instance)
	if err != nil {
		return nil, err
	}

	err = s.client.Update(ctx, instance)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't update project")
	}

	if instance.Spec.KeycloakSpec.Enabled {
		var secret uuid.UUID

		secret, err = uuid.NewUUID()
		if err != nil {
			return instance, errors.Wrap(err, "Failed to generate secret for Gerrit in Keycloak")
		}

		identityServiceClientCredentials := map[string][]byte{
			"client_id":    []byte(instance.Name),
			"clientSecret": []byte(secret.String()),
		}

		identityServiceSecretName := formatSecretName(instance.Name, spec.IdentityServiceCredentialsSecretPostfix)

		err = s.PlatformService.CreateSecret(
			instance,
			identityServiceSecretName,
			identityServiceClientCredentials,
			map[string]string{},
		)
		if err != nil {
			return instance, errors.Wrapf(err, fmt.Sprintf("Failed to create secret %v", identityServiceSecretName))
		}

		annotationKey := helpers.GenerateAnnotationKey(spec.IdentityServiceCredentialsSecretPostfix)
		s.setAnnotation(instance, annotationKey, formatSecretName(instance.Name, spec.IdentityServiceCredentialsSecretPostfix))

		err = s.client.Update(ctx, instance)
		if err != nil {
			return nil, errors.Wrap(err, "couldn't update annotations")
		}
	}

	return instance, nil
}

// Integrate applies actions required for the integration with the other EDP Components.
func (s ComponentService) Integrate(ctx context.Context, instance *gerritApi.Gerrit) (*gerritApi.Gerrit, error) {
	l := ctrl.LoggerFrom(ctx)

	externalUrl, err := s.getExternalUrl(instance)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get Route for %v/%v", instance.Namespace, instance.Name)
	}

	if instance.Spec.KeycloakSpec.Enabled {
		l.Info("Keycloak integration enabled")

		var keycloakClient *keycloakApi.KeycloakClient

		keycloakClient, err = s.getKeycloakClient(ctx, instance)
		if err != nil {
			return instance, err
		}

		if keycloakClient == nil {
			err = s.createKeycloakClient(ctx, instance, externalUrl)
			if err != nil {
				return instance, err
			}
		}

		var keycloakEnvironmentValue []coreV1Api.EnvVar

		keycloakEnvironmentValue, err = s.PlatformService.GenerateKeycloakSettings(instance)
		if err != nil {
			return instance, fmt.Errorf("failed to generate Keycloak %q env values: %w", instance.Name, err)
		}

		if err = s.PlatformService.PatchDeploymentEnv(instance, keycloakEnvironmentValue); err != nil {
			return instance, errors.Wrap(err, "Failed to add identity service information")
		}
	} else {
		l.Info("Keycloak integration not enabled. Restarting deployment.")

		gerritDep := &v1.Deployment{}
		if err = s.client.Get(ctx, types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, gerritDep); err != nil {
			return nil, fmt.Errorf("failed to get deployment %q: %w", instance.Name, err)
		}

		old := gerritDep.DeepCopy()

		if gerritDep.Spec.Template.Annotations == nil {
			gerritDep.Spec.Template.Annotations = map[string]string{}
		}

		gerritDep.Spec.Template.Annotations[DeploymentRestartAnnotation] = time.Now().Format(time.RFC3339)

		if err = s.client.Patch(ctx, gerritDep, client.MergeFrom(old)); err != nil {
			return nil, fmt.Errorf("failed to patch deployment: %w", err)
		}
	}

	return instance, nil
}

func (s ComponentService) getExternalUrl(instance *gerritApi.Gerrit) (string, error) {
	if instance.Spec.ExternalURL != "" {
		return instance.Spec.ExternalURL, nil
	}

	h, sc, err := s.PlatformService.GetExternalEndpoint(instance.Namespace, instance.Name)
	if err != nil {
		return "", errors.Wrapf(err, "Failed to get Route for %v/%v", instance.Namespace, instance.Name)
	}

	return fmt.Sprintf("%v://%v", sc, h), nil
}

func (s ComponentService) getKeycloakClient(ctx context.Context, instance *gerritApi.Gerrit) (*keycloakApi.KeycloakClient, error) {
	k8sClient := &keycloakApi.KeycloakClient{}

	err := s.client.Get(ctx, types.NamespacedName{
		Name:      instance.Name,
		Namespace: instance.Namespace,
	}, k8sClient)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to Get k8s KeycloakClient object %q: %w", instance.Name, err)
	}

	return k8sClient, nil
}

func (s ComponentService) createKeycloakClient(ctx context.Context, instance *gerritApi.Gerrit, externalUrl string) error {
	keycloakClient := &keycloakApi.KeycloakClient{
		TypeMeta: metaV1.TypeMeta{
			Kind: "KeycloakClient",
		},
		ObjectMeta: metaV1.ObjectMeta{
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
		keycloakClient.Spec.TargetRealm = instance.Spec.KeycloakSpec.Realm
	}

	err := s.client.Create(ctx, keycloakClient)
	if err != nil {
		return fmt.Errorf("failed to create k8s client for KeycloakClient: %w", err)
	}

	return nil
}

func (s ComponentService) GetRestClient(gerritInstance *gerritApi.Gerrit) (gerritClient.ClientInterface, error) {
	if s.gerritClient.Resty() != nil {
		return s.gerritClient, nil
	}

	if err := s.initRestClient(gerritInstance); err != nil {
		return nil, errors.Wrap(err, "unable to init gerrit rest client")
	}

	return s.gerritClient, nil
}

func (s *ComponentService) initRestClient(instance *gerritApi.Gerrit) error {
	vLog := log.WithValues("gerrit", instance.Name)
	vLog.Info("init rest client")

	gerritAdminPassword, err := s.getGerritAdminPassword(instance)
	if err != nil {
		return errors.Wrapf(err, "Failed to get Gerrit admin password from secret for %s/%s", instance.Namespace, instance.Name)
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

func (s *ComponentService) initSSHClient(instance *gerritApi.Gerrit) error {
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

	secretName := instance.Name + admin

	gerritAdminSshKeys, err := s.PlatformService.GetSecret(instance.Namespace, secretName)
	if err != nil {
		return fmt.Errorf("failed to fetch secret %q: %w", secretName, err)
	}

	err = s.gerritClient.InitNewSshClient(spec.GerritDefaultAdminUser, gerritAdminSshKeys[rsaID], gerritUrl, sshPortService)
	if err != nil {
		return errors.Wrapf(err, "Failed to init Gerrit SSH client %v/%v", instance.Namespace, instance.Name)
	}

	vLog.Info("ssh client has been initialized.")

	return nil
}

func (s ComponentService) getGerritRestApiUrl(instance *gerritApi.Gerrit) (string, error) {
	if instance.Spec.RestAPIUrl != "" {
		return instance.Spec.RestAPIUrl, nil
	}

	gerritApiUrl := fmt.Sprintf("http://%v.%v:%v/%v", instance.Name, instance.Namespace, spec.GerritPort,
		instance.Spec.GetBasePath())

	if !s.runningInCluster() {
		h, sc, err := s.PlatformService.GetExternalEndpoint(instance.Namespace, instance.Name)
		if err != nil {
			return "", errors.Wrapf(err, "Failed to get external endpoint for %v/%v", instance.Namespace, instance.Name)
		}

		gerritApiUrl = fmt.Sprintf("%v://%v/%v", sc, h, instance.Spec.GetBasePath())
	}

	return gerritApiUrl, nil
}

func (s ComponentService) GetGerritSSHUrl(instance *gerritApi.Gerrit) (string, error) {
	if instance.Spec.SSHUrl != "" {
		return instance.Spec.SSHUrl, nil
	}

	gerritSSHUrl := fmt.Sprintf("%v.%v", instance.Name, instance.Namespace)

	if !s.runningInCluster() {
		h, _, err := s.PlatformService.GetExternalEndpoint(instance.Namespace, instance.Name)
		if err != nil {
			return "", errors.Wrapf(err, "Failed to get Service for %v/%v", instance.Namespace, instance.Name)
		}

		gerritSSHUrl = h
	}

	return gerritSSHUrl, nil
}

func (s ComponentService) getGerritAdminPassword(instance *gerritApi.Gerrit) (string, error) {
	secretName := fmt.Sprintf("%s-admin-password", instance.Name)

	gerritAdminCredentials, err := s.PlatformService.GetSecretData(instance.Namespace, secretName)
	if err != nil {
		return "", errors.Wrapf(err, "Failed to get Secret %v for %v/%v", secretName, instance.Namespace, instance.Name)
	}

	return string(gerritAdminCredentials[password]), nil
}

func (s ComponentService) createSSHKeyPairs(instance *gerritApi.Gerrit, secretName string) (privateKey, publicKey []byte, err error) {
	secretData, err := s.PlatformService.GetSecretData(instance.Namespace, secretName)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "Unable to get data from secret %v", secretName)
	}

	if secretData != nil {
		return secretData[rsaID], secretData[rsaIDFile], nil
	}

	privateKey, publicKey, err = helpers.GenerateKeyPairs()
	if err != nil {
		return nil, nil, errors.Wrapf(err, "Unable to generate SSH key pairs for Gerrit")
	}

	if err := s.PlatformService.CreateSecret(
		instance,
		secretName,
		map[string][]byte{
			rsaID:     privateKey,
			rsaIDFile: publicKey,
		},
		map[string]string{},
	); err != nil {
		return nil, nil, errors.Wrapf(err, "Failed to create Secret with SSH key pairs for Gerrit")
	}

	return
}

func (s ComponentService) setGerritAdminUserPassword(
	instance *gerritApi.Gerrit,
	gerritUrl, gerritAdminPassword, gerritApiUrl string,
	sshPortService int32,
) error {
	gerritAdminSshKeys, err := s.PlatformService.GetSecret(instance.Namespace, instance.Name+admin)
	if err != nil {
		return errors.Wrapf(err, "Failed to get Gerrit admin secret for %s/%s", instance.Namespace, instance.Name)
	}

	err = s.gerritClient.InitNewSshClient(spec.GerritDefaultAdminUser, gerritAdminSshKeys[rsaID], gerritUrl, sshPortService)
	if err != nil {
		return errors.Wrapf(err, "Failed to initialize Gerrit SSH client for %s/%s", instance.Namespace, instance.Name)
	}

	err = s.gerritClient.ChangePassword("admin", gerritAdminPassword)
	if err != nil {
		return errors.Wrapf(err, "Failed to set Gerrit admin password for %s/%s", instance.Namespace, instance.Name)
	}

	err = s.gerritClient.InitNewRestClient(instance, gerritApiUrl, spec.GerritDefaultAdminUser, gerritAdminPassword)
	if err != nil {
		return errors.Wrapf(err, "Failed to initialize Gerrit REST client for %s/%s", instance.Namespace, instance.Name)
	}

	return nil
}

func (s ComponentService) GetServicePort(instance *gerritApi.Gerrit) (int32, error) {
	service, err := s.PlatformService.GetService(instance.Namespace, instance.Name)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch service %q: %w", instance.Name, err)
	}

	for _, port := range service.Spec.Ports {
		if port.Name == spec.SSHPortName {
			return port.NodePort, nil
		}
	}

	return 0, errors.New("Unable to determine Gerrit ssh port")
}

func (s ComponentService) updateDeploymentConfigPort(sshPort, sshPortService int32, instance *gerritApi.Gerrit) (bool, error) {
	if sshPort != sshPortService || sshPort == 0 {
		newEnv := []coreV1Api.EnvVar{
			{
				Name:  spec.SSHListnerEnvName,
				Value: fmt.Sprintf("*:%d", sshPortService),
			},
		}
		if err := s.PlatformService.PatchDeploymentEnv(instance, newEnv); err != nil {
			return false, fmt.Errorf("failed to update %q env value: %w", spec.SSHListnerEnvName, err)
		}

		return true, nil
	}

	return false, nil
}

// setAnnotation add key:value to current resource annotation.
func (ComponentService) setAnnotation(instance *gerritApi.Gerrit, key, value string) {
	if len(instance.Annotations) == 0 {
		instance.ObjectMeta.Annotations = map[string]string{
			key: value,
		}
	} else {
		instance.ObjectMeta.Annotations[key] = value
	}
}

func formatSecretName(name, postfix string) string {
	return fmt.Sprintf("%s-%s", name, postfix)
}

// exposeArgoCDConfiguration creates argocd user in Gerrit and adds this user to the ReadOnly group.
// It generates login/password and ssh private/public and stores them in secrets.
func (s *ComponentService) exposeArgoCDConfiguration(_ context.Context, gerrit *gerritApi.Gerrit) error {
	argoUserSecretName := formatSecretName(gerrit.Name, spec.GerritArgoUserSecretPostfix)
	argoUserSecretData := map[string][]byte{
		user:     []byte(spec.GerritArgoUser),
		password: []byte(uniuri.New()),
	}

	err := s.PlatformService.CreateSecret(
		gerrit,
		argoUserSecretName,
		argoUserSecretData,
		map[string]string{},
	)
	if err != nil {
		return fmt.Errorf("failed to create secret %s: %w", argoUserSecretName, err)
	}

	argoUserAnnotationKey := helpers.GenerateAnnotationKey(spec.EdpArgoUserSuffix)
	s.setAnnotation(gerrit, argoUserAnnotationKey, argoUserSecretName)

	privateKey, publicKey, err := helpers.GenerateSSHED25519KeyPairs()
	if err != nil {
		return fmt.Errorf("unable to generate SSH key pairs for Gerrit ArgoCD user: %w", err)
	}

	argoUserSshSecretName := fmt.Sprintf("%s-argocd%s", gerrit.Name, spec.SshKeyPostfix)

	err = s.PlatformService.CreateSecret(
		gerrit,
		argoUserSshSecretName,
		map[string][]byte{
			"username": []byte(spec.GerritArgoUser),
			rsaID:      privateKey,
			rsaIDFile:  publicKey,
		},
		map[string]string{},
	)
	if err != nil {
		return fmt.Errorf("unable to create secret for Gerrit ArgoCD user: %w", err)
	}

	ciUserSshKeyAnnotationKey := helpers.GenerateAnnotationKey(spec.EdpArgoUserSshKeySuffix)
	s.setAnnotation(gerrit, ciUserSshKeyAnnotationKey, argoUserSshSecretName)

	err = s.gerritClient.CreateUser(
		spec.GerritArgoUser,
		string(argoUserSecretData[password]),
		"argo cd user",
		string(publicKey),
	)
	if err != nil {
		return fmt.Errorf("unable to create ArgoCD user in Gerrit: %w", err)
	}

	err = s.gerritClient.AddUserToGroups(spec.GerritArgoUser, []string{spec.GerritReadOnlyGroupName})
	if err != nil {
		return fmt.Errorf("unable to add ArgoCD user to %s group in Gerrit: %w", spec.GerritReadOnlyGroupName, err)
	}

	return nil
}
