package gerrit

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gopkg.in/resty.v1"
	appsV1 "k8s.io/api/apps/v1"
	coreV1Api "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	jenkinsV1Api "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
	keycloakApi "github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1"

	pmock "github.com/epam/edp-gerrit-operator/v2/mock/platform"
	gerritApi "github.com/epam/edp-gerrit-operator/v2/pkg/apis/v2/v1"
	"github.com/epam/edp-gerrit-operator/v2/pkg/client/gerrit"
	gerritClientMocks "github.com/epam/edp-gerrit-operator/v2/pkg/client/gerrit/mocks"
	"github.com/epam/edp-gerrit-operator/v2/pkg/service/gerrit/spec"
	"github.com/epam/edp-gerrit-operator/v2/pkg/service/platform"
)

const (
	name        = "name"
	namespace   = "namespace"
	key         = "key"
	value       = "value"
	oldK        = "old key"
	oldV        = "old value"
	sc          = "sc"
	h           = "h"
	port        = int32(80)
	servicePort = int32(100)
)

func GenPkey() ([]byte, error) {
	pk, err := rsa.GenerateKey(rand.Reader, 128)
	if err != nil {
		return nil, err
	}

	privateKeyBytes := x509.MarshalPKCS1PrivateKey(pk)
	pkey := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: privateKeyBytes,
		},
	)

	return pkey, nil
}

func CreateGerritInstance() *gerritApi.Gerrit {
	return &gerritApi.Gerrit{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

func CreateService(port int32) *coreV1Api.Service {
	return &coreV1Api.Service{
		Spec: coreV1Api.ServiceSpec{
			Ports: []coreV1Api.ServicePort{
				{Name: spec.SSHPortName, NodePort: port},
			},
		},
	}
}

func TestErrUserNotFoundError(t *testing.T) {
	t.Parallel()

	str := "err not found"
	tnf := UserNotFoundError(str)

	assert.Equal(t, str, tnf.Error())
}

func TestIsErrUserNotFound(t *testing.T) {
	t.Parallel()

	t.Run("should return true for UserNotFoundError", func(t *testing.T) {
		t.Parallel()

		tnf := UserNotFoundError("err not found")
		err := errors.Wrap(tnf, "error")

		assert.True(t, IsErrUserNotFound(err), "wrong error type")
	})

	t.Run("should return false for other errors", func(t *testing.T) {
		t.Parallel()

		err := errors.New("random error")
		notExistsErr := os.ErrNotExist

		assert.False(t, IsErrUserNotFound(nil))
		assert.False(t, IsErrUserNotFound(err))
		assert.False(t, IsErrUserNotFound(notExistsErr))
	})
}

func TestComponentService_IsDeploymentReady(t *testing.T) {
	ps := &pmock.PlatformService{}
	CS := ComponentService{
		PlatformService: ps,
	}
	inst := &gerritApi.Gerrit{}

	ps.On("IsDeploymentReady", inst).Return(true, nil)

	ready, err := CS.IsDeploymentReady(inst)
	assert.True(t, ready)
	assert.NoError(t, err)
}

func TestComponentService_GetGerritSSHUrl(t *testing.T) {
	str := "url"
	instance := CreateGerritInstance()
	ps := &pmock.PlatformService{}
	CS := ComponentService{PlatformService: ps}

	ps.On("GetExternalEndpoint", instance.Namespace, instance.Name).Return(str, "", nil)

	_, err := CS.GetGerritSSHUrl(instance)
	assert.NoError(t, err)

	instance.Spec.SSHUrl = "url"
	sshURL, err := CS.GetGerritSSHUrl(instance)

	assert.NoError(t, err)
	assert.Equal(t, sshURL, instance.Spec.SSHUrl)
}

func TestComponentService_GetGerritSSHUrlErr(t *testing.T) {
	instance := CreateGerritInstance()
	errTest := errors.New("test")
	ps := &pmock.PlatformService{}
	CS := ComponentService{PlatformService: ps, runningInClusterFunc: func() bool {
		return true
	}}

	ps.On("GetExternalEndpoint", instance.Namespace, instance.Name).Return("", "", errTest)

	_, err := CS.GetGerritSSHUrl(instance)
	assert.NoError(t, err)

	CS = ComponentService{PlatformService: ps, runningInClusterFunc: func() bool {
		return false
	}}

	_, err = CS.GetGerritSSHUrl(instance)
	assert.Error(t, err)
}

func Test_setAnnotation_EmptyInstance(t *testing.T) {
	inst := &gerritApi.Gerrit{}
	CS := ComponentService{}

	CS.setAnnotation(inst, key, value)

	assert.Equal(t, map[string]string{key: value}, inst.Annotations)
}

func Test_setAnnotation(t *testing.T) {
	inst := &gerritApi.Gerrit{
		ObjectMeta: metaV1.ObjectMeta{
			Annotations: map[string]string{oldK: oldV},
		},
	}
	CS := ComponentService{}

	CS.setAnnotation(inst, key, value)

	assert.Equal(t, map[string]string{key: value, oldK: oldV}, inst.Annotations)
}

func TestComponentService_GetServicePort(t *testing.T) {
	instance := CreateGerritInstance()
	ps := &pmock.PlatformService{}
	CS := ComponentService{PlatformService: ps}
	service := CreateService(port)

	ps.On("GetService", instance.Namespace, instance.Name).Return(service, nil)

	p, err := CS.GetServicePort(instance)
	assert.NoError(t, err)
	assert.Equal(t, port, p)
}

func TestComponentService_GetServicePortErr(t *testing.T) {
	instance := CreateGerritInstance()
	ps := &pmock.PlatformService{}
	CS := ComponentService{PlatformService: ps}
	service := &coreV1Api.Service{}
	errTest := errors.New("test")

	ps.On("GetService", instance.Namespace, instance.Name).Return(service, errTest)

	_, err := CS.GetServicePort(instance)
	assert.Error(t, err)
}

func TestComponentService_GetServicePort_NoPorts(t *testing.T) {
	instance := CreateGerritInstance()
	ps := &pmock.PlatformService{}
	CS := ComponentService{PlatformService: ps}
	service := &coreV1Api.Service{}

	ps.On("GetService", instance.Namespace, instance.Name).Return(service, nil)

	p, err := CS.GetServicePort(instance)
	assert.Error(t, err)
	assert.Equal(t, int32(0), p)
}

func TestComponentService_Configure_CreateSecretErr(t *testing.T) {
	instance := CreateGerritInstance()
	ps := &pmock.PlatformService{}
	kc := fake.NewClientBuilder().Build()
	ks := &runtime.Scheme{}
	CS := ComponentService{PlatformService: ps, client: kc, k8sScheme: ks}
	errTest := errors.New("test")

	ps.On("GetExternalEndpoint", instance.Namespace, instance.Name).Return("", "", nil)
	ps.On("CreateSecret", instance, instance.Name+"-admin-password", mock.Anything).Return(errTest)

	configure, b, err := CS.Configure(instance)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), errTest.Error()))
	assert.Equal(t, instance, configure)
	assert.False(t, b)
}

func TestComponentService_Configure_GetServicePortErr(t *testing.T) {
	instance := CreateGerritInstance()
	ps := &pmock.PlatformService{}
	kc := fake.NewClientBuilder().Build()
	ks := &runtime.Scheme{}
	CS := ComponentService{PlatformService: ps, client: kc, k8sScheme: ks}
	service := &coreV1Api.Service{}
	errTest := errors.New("test")

	ps.On("GetExternalEndpoint", instance.Namespace, instance.Name).Return("", "", nil)
	ps.On("CreateSecret", instance, instance.Name+"-admin-password", mock.Anything).Return(nil)
	ps.On("GetService", instance.Namespace, instance.Name).Return(service, errTest)

	configure, b, err := CS.Configure(instance)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), errTest.Error()))
	assert.Equal(t, instance, configure)
	assert.False(t, b)
}

func TestComponentService_Configure_GetDeploymentSSHPortErr(t *testing.T) {
	instance := CreateGerritInstance()
	ps := &pmock.PlatformService{}
	kc := fake.NewClientBuilder().Build()
	ks := &runtime.Scheme{}
	CS := ComponentService{PlatformService: ps, client: kc, k8sScheme: ks}
	service := CreateService(port)
	errTest := errors.New("test")

	ps.On("GetExternalEndpoint", instance.Namespace, instance.Name).Return("", "", nil)
	ps.On("CreateSecret", instance, instance.Name+"-admin-password", mock.Anything).Return(nil)
	ps.On("GetService", instance.Namespace, instance.Name).Return(service, nil)
	ps.On("GetDeploymentSSHPort", instance).Return(port, errTest)

	configure, b, err := CS.Configure(instance)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), errTest.Error()))
	assert.Equal(t, instance, configure)
	assert.False(t, b)
}

func TestComponentService_Configure_GetServiceErr(t *testing.T) {
	instance := CreateGerritInstance()
	ps := &pmock.PlatformService{}
	kc := fake.NewClientBuilder().Build()
	ks := &runtime.Scheme{}
	CS := ComponentService{PlatformService: ps, client: kc, k8sScheme: ks}
	service := CreateService(port)
	errTest := errors.New("test")

	ps.On("GetExternalEndpoint", instance.Namespace, instance.Name).Return("", "", nil)
	ps.On("CreateSecret", instance, instance.Name+"-admin-password", mock.Anything).Return(nil)
	ps.On("GetService", instance.Namespace, instance.Name).Return(service, nil).Once()
	ps.On("GetService", instance.Namespace, instance.Name).Return(service, errTest)
	ps.On("GetDeploymentSSHPort", instance).Return(port, nil)

	configure, b, err := CS.Configure(instance)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), errTest.Error()))
	assert.Equal(t, instance, configure)
	assert.False(t, b)
}

func TestComponentService_Configure_UpdateServiceErr(t *testing.T) {
	instance := CreateGerritInstance()
	ps := &pmock.PlatformService{}
	kc := fake.NewClientBuilder().Build()
	ks := &runtime.Scheme{}
	CS := ComponentService{PlatformService: ps, client: kc, k8sScheme: ks}
	service := CreateService(port)
	errTest := errors.New("test")

	ps.On("GetExternalEndpoint", instance.Namespace, instance.Name).Return("", "", nil)
	ps.On("CreateSecret", instance, instance.Name+"-admin-password", mock.Anything).Return(nil)
	ps.On("GetService", instance.Namespace, instance.Name).Return(service, nil)
	ps.On("GetDeploymentSSHPort", instance).Return(port, nil)
	ps.On("UpdateService", service, port).Return(errTest)

	configure, b, err := CS.Configure(instance)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), errTest.Error()))
	assert.Equal(t, instance, configure)
	assert.False(t, b)
}

func TestComponentService_Configure_updateDeploymentConfigPortErr(t *testing.T) {
	instance := CreateGerritInstance()
	ps := &pmock.PlatformService{}
	kc := fake.NewClientBuilder().Build()
	ks := &runtime.Scheme{}
	CS := ComponentService{PlatformService: ps, client: kc, k8sScheme: ks}
	service := CreateService(servicePort)
	newEnv := []coreV1Api.EnvVar{
		{
			Name:  spec.SSHListnerEnvName,
			Value: fmt.Sprintf("*:%d", servicePort),
		},
	}
	errTest := errors.New("test")

	ps.On("GetExternalEndpoint", instance.Namespace, instance.Name).Return("", "", nil)
	ps.On("CreateSecret", instance, instance.Name+"-admin-password", mock.Anything).Return(nil)
	ps.On("GetService", instance.Namespace, instance.Name).Return(service, nil)
	ps.On("GetDeploymentSSHPort", instance).Return(port, nil)
	ps.On("UpdateService", service, servicePort).Return(nil)
	ps.On("PatchDeploymentEnv", instance, newEnv).Return(errTest)

	configure, b, err := CS.Configure(instance)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), errTest.Error()))
	assert.Equal(t, instance, configure)
	assert.False(t, b)
}

func TestComponentService_Configure_updateDeploymentConfigPortTrue(t *testing.T) {
	instance := CreateGerritInstance()
	ps := &pmock.PlatformService{}
	kc := fake.NewClientBuilder().Build()
	ks := &runtime.Scheme{}
	CS := ComponentService{PlatformService: ps, client: kc, k8sScheme: ks}
	service := CreateService(servicePort)
	newEnv := []coreV1Api.EnvVar{
		{
			Name:  spec.SSHListnerEnvName,
			Value: fmt.Sprintf("*:%d", servicePort),
		},
	}

	ps.On("GetExternalEndpoint", instance.Namespace, instance.Name).Return("", "", nil)
	ps.On("CreateSecret", instance, instance.Name+"-admin-password", mock.Anything).Return(nil)
	ps.On("GetService", instance.Namespace, instance.Name).Return(service, nil)
	ps.On("GetDeploymentSSHPort", instance).Return(port, nil)
	ps.On("UpdateService", service, servicePort).Return(nil)
	ps.On("PatchDeploymentEnv", instance, newEnv).Return(nil)

	configure, b, err := CS.Configure(instance)
	assert.NoError(t, err)
	assert.Equal(t, instance, configure)
	assert.True(t, b)
}

func TestComponentService_Configure_GetPodsErr(t *testing.T) {
	instance := CreateGerritInstance()
	ps := &pmock.PlatformService{}
	kc := fake.NewClientBuilder().Build()
	ks := &runtime.Scheme{}
	CS := ComponentService{PlatformService: ps, client: kc, k8sScheme: ks}
	service := CreateService(port)
	secretName := fmt.Sprintf("%v-admin-password", instance.Name)
	secretData := map[string][]byte{
		"password": {'o'},
	}
	podList := &coreV1Api.PodList{}
	errTest := errors.New("test")

	ps.On("GetExternalEndpoint", instance.Namespace, instance.Name).Return("", "", nil)
	ps.On("CreateSecret", instance, instance.Name+"-admin-password", mock.Anything).Return(nil)
	ps.On("GetService", instance.Namespace, instance.Name).Return(service, nil)
	ps.On("GetDeploymentSSHPort", instance).Return(port, nil)
	ps.On("UpdateService", service, port).Return(nil)
	ps.On("GetSecretData", instance.Namespace, secretName).Return(secretData, nil)
	ps.On("GetPods", instance.Namespace, &metaV1.ListOptions{LabelSelector: "app=" + instance.Name}).Return(podList, errTest)

	configure, b, err := CS.Configure(instance)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), errTest.Error()))
	assert.Contains(t, err.Error(), "unable to determine Gerrit pod name")
	assert.Equal(t, instance, configure)
	assert.False(t, b)
}

func TestComponentService_Configure_createSSHKeyPairsAdminErr(t *testing.T) {
	instance := CreateGerritInstance()
	ps := &pmock.PlatformService{}
	kc := fake.NewClientBuilder().Build()
	ks := &runtime.Scheme{}
	CS := ComponentService{PlatformService: ps, client: kc, k8sScheme: ks}
	service := CreateService(port)
	secretName := fmt.Sprintf("%v-admin-password", instance.Name)
	secretData := map[string][]byte{
		"password":   {'o'},
		"id_rsa":     {'a'},
		"id_rsa.pub": {'k'},
	}
	podList := &coreV1Api.PodList{
		Items: []coreV1Api.Pod{{
			TypeMeta: metaV1.TypeMeta{}}}}

	errTest := errors.New("test")

	ps.On("GetExternalEndpoint", instance.Namespace, instance.Name).Return("", "", nil)
	ps.On("CreateSecret", instance, instance.Name+"-admin-password", mock.Anything).Return(nil)
	ps.On("GetService", instance.Namespace, instance.Name).Return(service, nil)
	ps.On("GetDeploymentSSHPort", instance).Return(port, nil)
	ps.On("UpdateService", service, port).Return(nil)
	ps.On("GetSecretData", instance.Namespace, secretName).Return(secretData, nil)
	ps.On("GetSecretData", instance.Namespace, instance.Name+"-admin").Return(secretData, errTest)
	ps.On("GetPods", instance.Namespace, &metaV1.ListOptions{LabelSelector: "app=" + instance.Name}).Return(podList, nil)

	configure, b, err := CS.Configure(instance)
	assert.Error(t, err)
	assert.ErrorIs(t, err, errTest)
	assert.Contains(t, err.Error(), "failed to create Gerrit admin SSH keypair")
	assert.Equal(t, instance, configure)
	assert.False(t, b)
}

func TestComponentService_Configure_createSSHKeyPairsProjectCreatorErr(t *testing.T) {
	instance := CreateGerritInstance()
	ps := &pmock.PlatformService{}
	kc := fake.NewClientBuilder().Build()
	ks := &runtime.Scheme{}
	CS := ComponentService{PlatformService: ps, client: kc, k8sScheme: ks}
	service := CreateService(port)
	secretName := fmt.Sprintf("%v-admin-password", instance.Name)
	secretData := map[string][]byte{
		"password":   {'o'},
		"id_rsa":     {'a'},
		"id_rsa.pub": {'k'},
	}
	podList := &coreV1Api.PodList{
		Items: []coreV1Api.Pod{{
			TypeMeta: metaV1.TypeMeta{}}}}
	errTest := errors.New("test")

	ps.On("GetExternalEndpoint", instance.Namespace, instance.Name).Return("", "", nil)
	ps.On("CreateSecret", instance, instance.Name+"-admin-password", mock.Anything).Return(nil)
	ps.On("GetService", instance.Namespace, instance.Name).Return(service, nil)
	ps.On("GetDeploymentSSHPort", instance).Return(port, nil)
	ps.On("UpdateService", service, port).Return(nil)
	ps.On("GetSecretData", instance.Namespace, secretName).Return(secretData, nil)
	ps.On("GetSecretData", instance.Namespace, instance.Name+"-admin").Return(secretData, nil)
	ps.On("GetSecretData", instance.Namespace, instance.Name+"-project-creator").Return(secretData, errTest)
	ps.On("GetPods", instance.Namespace, &metaV1.ListOptions{LabelSelector: "app=" + instance.Name}).Return(podList, nil)

	configure, b, err := CS.Configure(instance)
	assert.Error(t, err)
	assert.ErrorIs(t, err, errTest)
	assert.Contains(t, err.Error(), "failed to create Gerrit project-creator SSH keypair")
	assert.Equal(t, instance, configure)
	assert.False(t, b)
}

func TestComponentService_Configure_CheckCredentialsErr(t *testing.T) {
	instance := CreateGerritInstance()
	ps := &pmock.PlatformService{}
	kc := fake.NewClientBuilder().Build()
	ks := &runtime.Scheme{}
	gc := gerrit.NewClient(nil, resty.New(), nil)
	CS := ComponentService{PlatformService: ps, client: kc, k8sScheme: ks, gerritClient: &gc}
	service := CreateService(port)
	secretName := fmt.Sprintf("%v-admin-password", instance.Name)
	secretData := map[string][]byte{
		"password":   {'o'},
		"id_rsa":     {'a'},
		"id_rsa.pub": {'k'},
	}
	podList := &coreV1Api.PodList{
		Items: []coreV1Api.Pod{{
			TypeMeta: metaV1.TypeMeta{}}}}

	ps.On("GetExternalEndpoint", instance.Namespace, instance.Name).Return("", "", nil)
	ps.On("CreateSecret", instance, instance.Name+"-admin-password", mock.Anything).Return(nil)
	ps.On("GetService", instance.Namespace, instance.Name).Return(service, nil)
	ps.On("GetDeploymentSSHPort", instance).Return(port, nil)
	ps.On("UpdateService", service, port).Return(nil)
	ps.On("GetSecretData", instance.Namespace, secretName).Return(secretData, nil)
	ps.On("GetSecretData", instance.Namespace, instance.Name+"-admin").Return(secretData, nil)
	ps.On("GetSecretData", instance.Namespace, instance.Name+"-project-creator").Return(secretData, nil)
	ps.On("GetPods", instance.Namespace, &metaV1.ListOptions{LabelSelector: "app=" + instance.Name}).Return(podList, nil)

	configure, b, err := CS.Configure(instance)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Unable to verify Gerrit credentials")
	assert.Equal(t, instance, configure)
	assert.False(t, b)
}

func TestComponentService_Integrate_GetExternalEndpointErr(t *testing.T) {
	instance := CreateGerritInstance()
	errTest := errors.New("test")
	ps := &pmock.PlatformService{}
	CS := ComponentService{PlatformService: ps}

	ps.On("GetExternalEndpoint", instance.Namespace, instance.Name).Return("", "", errTest)

	_, err := CS.Integrate(context.Background(), instance)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), errTest.Error()))
}

func TestComponentService_Integrate_ParseDefaultTemplateErr(t *testing.T) {
	instance := &gerritApi.Gerrit{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "name",
			Namespace: "namespace",
		},
		Spec: gerritApi.GerritSpec{
			KeycloakSpec: gerritApi.KeycloakSpec{
				Enabled: true,
			},
		},
	}
	kc := &keycloakApi.KeycloakClient{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "name",
			Namespace: "namespace",
		},
	}
	jenkins := &jenkinsV1Api.Jenkins{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "jenkins",
			Namespace: "namespace",
		},
	}

	service := CreateService(servicePort)
	ciUserCredentialsName := fmt.Sprintf("%v-%v", instance.Name, spec.GerritDefaultCiUserSecretPostfix)
	secretData := map[string][]byte{
		"password": {'o'},
		"user":     {'a'},
	}

	ps := &pmock.PlatformService{}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(appsV1.SchemeGroupVersion, &gerritApi.Gerrit{},
		&keycloakApi.KeycloakClient{}, &jenkinsV1Api.Jenkins{}, &jenkinsV1Api.JenkinsList{})

	k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(kc, jenkins).Build()
	CS := ComponentService{PlatformService: ps, client: k8sClient}

	var envs []coreV1Api.EnvVar

	ps.On("GetExternalEndpoint", instance.Namespace, instance.Name).Return("", "", nil)
	ps.On("GenerateKeycloakSettings", instance).Return(envs, nil)
	ps.On("PatchDeploymentEnv", instance, envs).Return(nil)
	ps.On("GetService", instance.Namespace, instance.Name).Return(service, nil)
	ps.On("GetSecretData", instance.Namespace, ciUserCredentialsName).Return(secretData, nil)

	_, err := CS.Integrate(context.Background(), instance)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "Template file not found in path specified!"))
}

func TestComponentService_Integrate_getKeycloakClientErr(t *testing.T) {
	instance := &gerritApi.Gerrit{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "name",
			Namespace: "namespace",
		},
		Spec: gerritApi.GerritSpec{
			KeycloakSpec: gerritApi.KeycloakSpec{
				Enabled: true,
			},
		},
	}

	ps := &pmock.PlatformService{}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(appsV1.SchemeGroupVersion, &gerritApi.Gerrit{})

	cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects().Build()
	CS := ComponentService{PlatformService: ps, client: cl}

	ps.On("GetExternalEndpoint", instance.Namespace, instance.Name).Return("", "", nil)

	_, err := CS.Integrate(context.Background(), instance)
	assert.Error(t, err)
}

func TestComponentService_Integrate_GenerateKeycloakSettingsErr(t *testing.T) {
	instance := &gerritApi.Gerrit{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "name",
			Namespace: "namespace",
		},
		Spec: gerritApi.GerritSpec{
			KeycloakSpec: gerritApi.KeycloakSpec{
				Enabled: true,
			},
		},
	}
	client := &keycloakApi.KeycloakClient{}
	errTest := errors.New("test")
	ps := &pmock.PlatformService{}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(appsV1.SchemeGroupVersion, &gerritApi.Gerrit{}, &keycloakApi.KeycloakClient{})

	cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(client).Build()
	CS := ComponentService{PlatformService: ps, client: cl}

	ps.On("GetExternalEndpoint", instance.Namespace, instance.Name).Return("", "", nil)
	ps.On("GenerateKeycloakSettings", instance).Return(nil, errTest)

	_, err := CS.Integrate(context.Background(), instance)
	assert.ErrorIs(t, err, errTest)
}

func TestComponentService_Integrate_PatchDeploymentEnvErr(t *testing.T) {
	instance := &gerritApi.Gerrit{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "name",
			Namespace: "namespace",
		},
		Spec: gerritApi.GerritSpec{
			KeycloakSpec: gerritApi.KeycloakSpec{
				Enabled: true,
			},
		},
	}
	client := &keycloakApi.KeycloakClient{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "name",
			Namespace: "namespace",
		},
	}
	ps := &pmock.PlatformService{}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(appsV1.SchemeGroupVersion, &gerritApi.Gerrit{}, &keycloakApi.KeycloakClient{})

	cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(client).Build()
	CS := ComponentService{PlatformService: ps, client: cl}

	var envs []coreV1Api.EnvVar

	errTest := errors.New("test")

	ps.On("GetExternalEndpoint", instance.Namespace, instance.Name).Return("", "", nil)
	ps.On("GenerateKeycloakSettings", instance).Return(envs, nil)
	ps.On("PatchDeploymentEnv", instance, envs).Return(errTest)

	_, err := CS.Integrate(context.Background(), instance)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "Failed to add identity service information"))
}

func TestComponentService_GetRestClient(t *testing.T) {
	instance := CreateGerritInstance()
	secretData := map[string][]byte{
		"password": {'o'},
	}
	secretName := fmt.Sprintf("%v-admin-password", instance.Name)

	ps := &pmock.PlatformService{}
	CS := ComponentService{gerritClient: &gerrit.Client{}, PlatformService: ps}

	ps.On("GetSecretData", instance.Namespace, secretName).Return(secretData, nil)
	ps.On("GetExternalEndpoint", instance.Namespace, instance.Name).Return("", "", nil)

	_, err := CS.GetRestClient(instance)
	assert.NoError(t, err)
}

func TestComponentService_GetRestClientNotNilResty(t *testing.T) {
	instance := CreateGerritInstance()

	gc := gerrit.NewClient(nil, resty.New(), nil)

	ps := &pmock.PlatformService{}
	CS := ComponentService{gerritClient: &gc, PlatformService: ps}

	_, err := CS.GetRestClient(instance)
	assert.NoError(t, err)
}

func TestComponentService_getUrlErr(t *testing.T) {
	instance := CreateGerritInstance()

	errTest := errors.New("test")
	ps := &pmock.PlatformService{}
	CS := ComponentService{PlatformService: ps}

	ps.On("GetExternalEndpoint", instance.Namespace, instance.Name).Return("", "", errTest)

	_, err := CS.getUrl(instance)
	assert.ErrorIs(t, err, errTest)
}

func TestComponentService_getUrl(t *testing.T) {
	instance := CreateGerritInstance()
	url := fmt.Sprintf("%v://%v", sc, h)

	ps := &pmock.PlatformService{}
	CS := ComponentService{PlatformService: ps}

	ps.On("GetExternalEndpoint", instance.Namespace, instance.Name).Return(h, sc, nil)

	u, err := CS.getUrl(instance)
	assert.NoError(t, err)
	assert.Equal(t, url, *u)
}

func TestComponentService_ExposeConfiguration_CreateUserErr(t *testing.T) {
	instance := CreateGerritInstance()
	ciUserSecretName := fmt.Sprintf("%v-%v", instance.Name, spec.GerritDefaultCiUserSecretPostfix)
	projectCreatorSecretPasswordName := fmt.Sprintf("%v-%v-%v", instance.Name, spec.GerritDefaultProjectCreatorSecretPostfix, "password")
	ciUserSshSecretName := fmt.Sprintf("%s-ciuser%s", instance.Name, spec.SshKeyPostfix)
	secretName := fmt.Sprintf("%v-admin-password", instance.Name)

	pkey, err := GenPkey()
	require.NoError(t, err)

	secretData := map[string][]byte{
		"password": {'o'},
		"id_rsa":   pkey,
	}
	service := CreateService(servicePort)

	scheme := runtime.NewScheme()
	err = jenkinsV1Api.AddToScheme(scheme)
	require.NoError(t, err)

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(
			&jenkinsV1Api.Jenkins{
				ObjectMeta: metaV1.ObjectMeta{
					Namespace: instance.Namespace,
					Name:      "jenkins",
				},
			},
		).
		Build()

	ps := &pmock.PlatformService{}
	CS := ComponentService{PlatformService: ps, gerritClient: &gerrit.Client{}, client: k8sClient}

	ps.On("GetSecretData", instance.Namespace, secretName).Return(secretData, nil)
	ps.On("GetExternalEndpoint", instance.Namespace, instance.Name).Return(h, sc, nil)
	ps.On("GetService", instance.Namespace, instance.Name).Return(service, nil)
	ps.On("GetSecret", instance.Namespace, instance.Name+"-admin").Return(secretData, nil)
	ps.On("CreateSecret", instance, ciUserSecretName, mock.Anything).Return(nil)
	ps.On("CreateSecret", instance, ciUserSshSecretName, mock.Anything).Return(nil)
	ps.On("CreateSecret", instance, projectCreatorSecretPasswordName, mock.Anything).Return(nil)
	ps.On("GetSecretData", instance.Namespace, ciUserSecretName).Return(secretData, nil)
	ps.On("CreateJenkinsServiceAccount", instance.Namespace, ciUserSshSecretName, "ssh").Return(nil)

	_, err = CS.ExposeConfiguration(context.Background(), instance)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "Failed to create ci user"))
}

func TestComponentService_ExposeConfiguration_initRestClientErr(t *testing.T) {
	instance := CreateGerritInstance()
	secretName := fmt.Sprintf("%v-admin-password", instance.Name)
	secretData := map[string][]byte{
		"password": {'o'},
		"id_rsa":   {'a'},
	}
	errTest := errors.New("test")
	ps := &pmock.PlatformService{}
	CS := ComponentService{PlatformService: ps}

	ps.On("GetSecretData", instance.Namespace, secretName).Return(secretData, errTest)

	_, err := CS.ExposeConfiguration(context.Background(), instance)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "Failed to init Gerrit REST client"))
}

func TestComponentService_ExposeConfiguration_initSSHClient(t *testing.T) {
	instance := CreateGerritInstance()
	secretName := fmt.Sprintf("%v-admin-password", instance.Name)
	secretData := map[string][]byte{
		"password": {'o'},
		"id_rsa":   {'a'},
	}
	service := CreateService(servicePort)

	ps := &pmock.PlatformService{}
	CS := ComponentService{PlatformService: ps, gerritClient: &gerrit.Client{}}

	ps.On("GetSecretData", instance.Namespace, secretName).Return(secretData, nil)
	ps.On("GetExternalEndpoint", instance.Namespace, instance.Name).Return(h, sc, nil)
	ps.On("GetService", instance.Namespace, instance.Name).Return(service, nil)
	ps.On("GetSecret", instance.Namespace, instance.Name+"-admin").Return(secretData, nil)

	_, err := CS.ExposeConfiguration(context.Background(), instance)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "Failed to init Gerrit SSH client"))
}

func TestComponentService_ExposeConfiguration_FirstCreateSecretErr(t *testing.T) {
	instance := CreateGerritInstance()
	ciUserSecretName := fmt.Sprintf("%v-%v", instance.Name, spec.GerritDefaultCiUserSecretPostfix)
	secretName := fmt.Sprintf("%v-admin-password", instance.Name)

	pkey, err := GenPkey()
	require.NoError(t, err)

	secretData := map[string][]byte{
		"password": {'o'},
		"id_rsa":   pkey,
	}
	service := CreateService(servicePort)
	errTest := errors.New("test")

	ps := &pmock.PlatformService{}
	CS := ComponentService{PlatformService: ps, gerritClient: &gerrit.Client{}}

	ps.On("GetSecretData", instance.Namespace, secretName).Return(secretData, nil)
	ps.On("GetExternalEndpoint", instance.Namespace, instance.Name).Return(h, sc, nil)
	ps.On("GetService", instance.Namespace, instance.Name).Return(service, nil)
	ps.On("GetSecret", instance.Namespace, instance.Name+"-admin").Return(secretData, nil)
	ps.On("CreateSecret", instance, ciUserSecretName, mock.Anything).Return(errTest)

	_, err = CS.ExposeConfiguration(context.Background(), instance)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "Failed to create ci user Secret"))
}

func TestComponentService_ExposeConfiguration_SecondCreateSecretErr(t *testing.T) {
	instance := CreateGerritInstance()
	ciUserSecretName := fmt.Sprintf("%v-%v", instance.Name, spec.GerritDefaultCiUserSecretPostfix)
	projectCreatorSecretPasswordName := fmt.Sprintf("%v-%v-%v", instance.Name, spec.GerritDefaultProjectCreatorSecretPostfix, "password")
	secretName := fmt.Sprintf("%v-admin-password", instance.Name)

	pkey, err := GenPkey()
	require.NoError(t, err)

	secretData := map[string][]byte{
		"password": {'o'},
		"id_rsa":   pkey,
	}
	service := CreateService(servicePort)
	errTest := errors.New("test")

	ps := &pmock.PlatformService{}
	CS := ComponentService{PlatformService: ps, gerritClient: &gerrit.Client{}}

	ps.On("GetSecretData", instance.Namespace, secretName).Return(secretData, nil)
	ps.On("GetExternalEndpoint", instance.Namespace, instance.Name).Return(h, sc, nil)
	ps.On("GetService", instance.Namespace, instance.Name).Return(service, nil)
	ps.On("GetSecret", instance.Namespace, instance.Name+"-admin").Return(secretData, nil)
	ps.On("CreateSecret", instance, ciUserSecretName, mock.Anything).Return(nil)
	ps.On("CreateSecret", instance, projectCreatorSecretPasswordName, mock.Anything).Return(errTest)

	_, err = CS.ExposeConfiguration(context.Background(), instance)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "Failed to create project-creator Secret"))
}

func TestComponentService_ExposeConfiguration_GetSecretErr(t *testing.T) {
	instance := CreateGerritInstance()
	ciUserSecretName := fmt.Sprintf("%v-%v", instance.Name, spec.GerritDefaultCiUserSecretPostfix)
	projectCreatorSecretPasswordName := fmt.Sprintf("%v-%v-%v", instance.Name, spec.GerritDefaultProjectCreatorSecretPostfix, "password")
	secretName := fmt.Sprintf("%v-admin-password", instance.Name)

	pkey, err := GenPkey()
	require.NoError(t, err)

	secretData := map[string][]byte{
		"password": {'o'},
		"id_rsa":   pkey,
	}
	service := CreateService(servicePort)
	errTest := errors.New("test")

	ps := &pmock.PlatformService{}
	CS := ComponentService{PlatformService: ps, gerritClient: &gerrit.Client{}}

	ps.On("GetSecretData", instance.Namespace, secretName).Return(secretData, nil)
	ps.On("GetExternalEndpoint", instance.Namespace, instance.Name).Return(h, sc, nil)
	ps.On("GetService", instance.Namespace, instance.Name).Return(service, nil)
	ps.On("GetSecret", instance.Namespace, instance.Name+"-admin").Return(secretData, nil)
	ps.On("CreateSecret", instance, ciUserSecretName, mock.Anything).Return(nil)
	ps.On("CreateSecret", instance, projectCreatorSecretPasswordName, mock.Anything).Return(nil)
	ps.On("GetSecretData", instance.Namespace, ciUserSecretName).Return(secretData, errTest)

	_, err = CS.ExposeConfiguration(context.Background(), instance)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "Failed to get Secret"))
}

func TestComponentService_ExposeConfiguration_CreateSecretErr(t *testing.T) {
	instance := CreateGerritInstance()
	ciUserSecretName := fmt.Sprintf("%v-%v", instance.Name, spec.GerritDefaultCiUserSecretPostfix)
	projectCreatorSecretPasswordName := fmt.Sprintf("%v-%v-%v", instance.Name, spec.GerritDefaultProjectCreatorSecretPostfix, "password")
	ciUserSshSecretName := fmt.Sprintf("%s-ciuser%s", instance.Name, spec.SshKeyPostfix)
	secretName := fmt.Sprintf("%v-admin-password", instance.Name)

	pkey, err := GenPkey()
	require.NoError(t, err)

	secretData := map[string][]byte{
		"password": {'o'},
		"id_rsa":   pkey,
	}
	service := CreateService(servicePort)
	errTest := errors.New("test")

	ps := &pmock.PlatformService{}
	CS := ComponentService{PlatformService: ps, gerritClient: &gerrit.Client{}}

	ps.On("GetSecretData", instance.Namespace, secretName).Return(secretData, nil)
	ps.On("GetExternalEndpoint", instance.Namespace, instance.Name).Return(h, sc, nil)
	ps.On("GetService", instance.Namespace, instance.Name).Return(service, nil)
	ps.On("GetSecret", instance.Namespace, instance.Name+"-admin").Return(secretData, nil)
	ps.On("CreateSecret", instance, ciUserSecretName, mock.Anything).Return(nil)
	ps.On("CreateSecret", instance, projectCreatorSecretPasswordName, mock.Anything).Return(nil)
	ps.On("GetSecretData", instance.Namespace, ciUserSecretName).Return(secretData, nil)
	ps.On("CreateSecret", instance, ciUserSshSecretName, mock.Anything).Return(errTest)

	_, err = CS.ExposeConfiguration(context.Background(), instance)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "Failed to create Secret with SSH key pairs for Gerrit"))
}

func TestComponentService_ExposeConfiguration_CreateJenkinsServiceAccountErr(t *testing.T) {
	instance := CreateGerritInstance()
	ciUserSecretName := fmt.Sprintf("%v-%v", instance.Name, spec.GerritDefaultCiUserSecretPostfix)
	projectCreatorSecretPasswordName := fmt.Sprintf("%v-%v-%v", instance.Name, spec.GerritDefaultProjectCreatorSecretPostfix, "password")
	ciUserSshSecretName := fmt.Sprintf("%s-ciuser%s", instance.Name, spec.SshKeyPostfix)

	secretName := fmt.Sprintf("%v-admin-password", instance.Name)
	pkey, err := GenPkey()
	require.NoError(t, err)

	secretData := map[string][]byte{
		"password": {'o'},
		"id_rsa":   pkey,
	}

	service := CreateService(servicePort)
	errTest := errors.New("test")

	scheme := runtime.NewScheme()
	err = jenkinsV1Api.AddToScheme(scheme)
	require.NoError(t, err)

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(
			&jenkinsV1Api.Jenkins{
				ObjectMeta: metaV1.ObjectMeta{
					Namespace: instance.Namespace,
					Name:      "jenkins",
				},
			},
		).
		Build()

	ps := &pmock.PlatformService{}
	CS := ComponentService{PlatformService: ps, gerritClient: &gerrit.Client{}, client: k8sClient}

	ps.On("GetSecretData", instance.Namespace, secretName).Return(secretData, nil)
	ps.On("GetExternalEndpoint", instance.Namespace, instance.Name).Return(h, sc, nil)
	ps.On("GetService", instance.Namespace, instance.Name).Return(service, nil)
	ps.On("GetSecret", instance.Namespace, instance.Name+"-admin").Return(secretData, nil)
	ps.On("CreateSecret", instance, ciUserSecretName, mock.Anything).Return(nil)
	ps.On("CreateSecret", instance, ciUserSshSecretName, mock.Anything).Return(nil)
	ps.On("CreateSecret", instance, projectCreatorSecretPasswordName, mock.Anything).Return(nil)
	ps.On("GetSecretData", instance.Namespace, ciUserSecretName).Return(secretData, nil)
	ps.On("CreateJenkinsServiceAccount", instance.Namespace, ciUserSshSecretName, "ssh").Return(errTest)

	_, err = CS.ExposeConfiguration(context.Background(), instance)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "Failed to create Jenkins Service Account"))
}

func TestComponentService_Configure_CreateGroups(t *testing.T) {
	instance := CreateGerritInstance()
	gerritClient := gerritClientMocks.ClientInterface{}
	ps := &pmock.PlatformService{}
	kc := fake.NewClientBuilder().Build()
	ks := &runtime.Scheme{}
	CS := ComponentService{PlatformService: ps, client: kc, k8sScheme: ks, gerritClient: &gerritClient}
	service := CreateService(port)
	secretName := fmt.Sprintf("%v-admin-password", instance.Name)
	secretData := map[string][]byte{
		"password":   {'o'},
		"id_rsa":     {'a'},
		"id_rsa.pub": {'k'},
	}

	podList := &coreV1Api.PodList{
		Items: []coreV1Api.Pod{{
			TypeMeta: metaV1.TypeMeta{}}}}

	ps.On("GetExternalEndpoint", instance.Namespace, instance.Name).Return("", "", nil)
	ps.On("CreateSecret", instance, instance.Name+"-admin-password", mock.Anything).Return(nil)
	ps.On("GetService", instance.Namespace, instance.Name).Return(service, nil)
	ps.On("GetDeploymentSSHPort", instance).Return(port, nil)
	ps.On("UpdateService", service, port).Return(nil)
	ps.On("GetSecretData", instance.Namespace, secretName).Return(secretData, nil)
	ps.On("GetSecretData", instance.Namespace, instance.Name+"-admin").Return(secretData, nil)
	ps.On("GetSecretData", instance.Namespace, instance.Name+"-project-creator").Return(secretData, nil)
	ps.On("GetPods", instance.Namespace, &metaV1.ListOptions{LabelSelector: "app=" + instance.Name}).Return(podList, nil)
	ps.On("GetSecret", instance.Namespace, "name-admin").Return(map[string][]byte{}, nil)
	ps.On("ExecInPod", instance.Namespace, "",
		[]string{"/bin/sh", "-c", "chown -R gerrit2:gerrit2 /var/gerrit/review_site"}).
		Return(nil, nil, nil)

	var emptyByte []byte

	statusOk := 404

	gerritClient.On("InitNewRestClient", instance, ":///a/", "admin", "o").Return(nil)
	gerritClient.On("CheckCredentials").Return(200, nil)
	gerritClient.On("InitNewSshClient", "admin", emptyByte, "", int32(80)).Return(nil)
	gerritClient.On("CheckGroup", mock.Anything).Return(&statusOk, nil)
	gerritClient.On("CreateGroup", mock.Anything, mock.Anything, mock.Anything).Return(&gerrit.Group{}, nil)
	gerritClient.On("InitAllProjects",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	_, _, err := CS.Configure(instance)
	assert.NoError(t, err)

	ps.AssertExpectations(t)
	gerritClient.AssertExpectations(t)
}

func TestNewComponentService(t *testing.T) {
	svc := NewComponentService(nil, nil, nil)
	_, ok := svc.(ComponentService)
	assert.True(t, ok)
}

func TestComponentService_GetGerritRestApiUrl(t *testing.T) {
	ps := pmock.PlatformService{}
	s := ComponentService{
		PlatformService: &ps,
	}
	basePath := "kong"
	g := gerritApi.Gerrit{Spec: gerritApi.GerritSpec{BasePath: basePath}, ObjectMeta: metaV1.ObjectMeta{
		Name:      "g1",
		Namespace: "ns1",
	}}

	ps.On("GetExternalEndpoint", g.Namespace, g.Name).Return("gerrit.host", "http", nil)
	url, err := s.getGerritRestApiUrl(&g)

	assert.NoError(t, err)
	assert.Contains(t, url, basePath)

	g.Spec.RestAPIUrl = "url"
	url, err = s.getGerritRestApiUrl(&g)
	assert.NoError(t, err)
	assert.Equal(t, g.Spec.RestAPIUrl, url)

	ps.AssertExpectations(t)
}

func TestComponentService_exposeArgoCDConfiguration(t *testing.T) {
	gerritInstance := &gerritApi.Gerrit{
		TypeMeta: metaV1.TypeMeta{},
		ObjectMeta: metaV1.ObjectMeta{
			Namespace: "test-ns",
			Name:      "test-name",
		},
	}

	tests := []struct {
		name            string
		PlatformService func(t *testing.T) platform.PlatformService
		gerritClient    func(t *testing.T) gerrit.ClientInterface
		gerrit          *gerritApi.Gerrit
		wantErr         require.ErrorAssertionFunc
		errContains     string
	}{
		{
			name: "success expose argocd configuration",
			PlatformService: func(t *testing.T) platform.PlatformService {
				serviceMock := &pmock.PlatformService{}
				serviceMock.On("CreateSecret", mock.Anything, mock.Anything, mock.Anything).
					Return(nil)

				return serviceMock
			},
			gerritClient: func(t *testing.T) gerrit.ClientInterface {
				clientMock := &gerritClientMocks.ClientInterface{}
				clientMock.On("CreateUser", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				clientMock.On("AddUserToGroups", mock.Anything, mock.Anything).
					Return(nil)

				return clientMock
			},
			wantErr: require.NoError,
		},
		{
			name: "failed to create user secret",
			PlatformService: func(t *testing.T) platform.PlatformService {
				serviceMock := &pmock.PlatformService{}
				serviceMock.On("CreateSecret", mock.Anything, mock.Anything, mock.Anything).
					Return(errors.New("failed to create secret"))

				return serviceMock
			},
			gerritClient: func(t *testing.T) gerrit.ClientInterface {
				clientMock := &gerritClientMocks.ClientInterface{}

				return clientMock
			},
			wantErr:     require.Error,
			errContains: "failed to create secret",
		},
		{
			name: "failed to create ssh secret",
			PlatformService: func(t *testing.T) platform.PlatformService {
				serviceMock := &pmock.PlatformService{}
				serviceMock.On("CreateSecret", mock.Anything, mock.Anything, mock.Anything).
					Once().
					Return(nil).
					On("CreateSecret", mock.Anything, mock.Anything, mock.Anything).
					Once().
					Return(errors.New("failed to create ssh secret"))

				return serviceMock
			},
			gerritClient: func(t *testing.T) gerrit.ClientInterface {
				clientMock := &gerritClientMocks.ClientInterface{}

				return clientMock
			},
			wantErr:     require.Error,
			errContains: "failed to create ssh secret",
		},
		{
			name: "failed to create user",
			PlatformService: func(t *testing.T) platform.PlatformService {
				serviceMock := &pmock.PlatformService{}
				serviceMock.On("CreateSecret", mock.Anything, mock.Anything, mock.Anything).
					Return(nil)

				return serviceMock
			},
			gerritClient: func(t *testing.T) gerrit.ClientInterface {
				clientMock := &gerritClientMocks.ClientInterface{}
				clientMock.On("CreateUser", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(errors.New("failed to create user"))

				return clientMock
			},
			wantErr:     require.Error,
			errContains: "failed to create user",
		},
		{
			name: "failed to add user to groups",
			PlatformService: func(t *testing.T) platform.PlatformService {
				serviceMock := &pmock.PlatformService{}
				serviceMock.On("CreateSecret", mock.Anything, mock.Anything, mock.Anything).
					Return(nil)

				return serviceMock
			},
			gerritClient: func(t *testing.T) gerrit.ClientInterface {
				clientMock := &gerritClientMocks.ClientInterface{}
				clientMock.On("CreateUser", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				clientMock.On("AddUserToGroups", mock.Anything, mock.Anything).
					Return(errors.New("failed to add user to groups"))

				return clientMock
			},
			wantErr:     require.Error,
			errContains: "failed to add user to groups",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &ComponentService{
				PlatformService: tt.PlatformService(t),
				gerritClient:    tt.gerritClient(t),
			}

			err := s.exposeArgoCDConfiguration(context.Background(), gerritInstance)

			tt.wantErr(t, err)
			if tt.errContains != "" {
				assert.Contains(t, err.Error(), tt.errContains)
			}
		})
	}
}
