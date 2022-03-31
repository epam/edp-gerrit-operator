package gerrit

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"

	keycloakApi "github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"gopkg.in/resty.v1"
	appsv1 "k8s.io/api/apps/v1"
	coreV1Api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	pmock "github.com/epam/edp-gerrit-operator/v2/mock/platform"
	"github.com/epam/edp-gerrit-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/epam/edp-gerrit-operator/v2/pkg/client/gerrit"
	"github.com/epam/edp-gerrit-operator/v2/pkg/service/gerrit/spec"
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

func GenPkey() (error, []byte) {
	pk, err := rsa.GenerateKey(rand.Reader, 128)
	if err != nil {
		return err, nil
	}
	privkeyBytes := x509.MarshalPKCS1PrivateKey(pk)
	pkey := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: privkeyBytes,
		},
	)
	return nil, pkey
}

func CreateGerritInstance() *v1alpha1.Gerrit {
	return &v1alpha1.Gerrit{
		ObjectMeta: metav1.ObjectMeta{
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
	str := "err not found"
	tnf := ErrUserNotFound(str)
	assert.Equal(t, str, tnf.Error())
}

func TestIsErrUserNotFound(t *testing.T) {
	tnf := ErrUserNotFound("err not found")
	err := errors.Wrap(tnf, "error")
	if !IsErrUserNotFound(err) {
		t.Fatal("wrong error type")
	}
}

func TestComponentService_IsDeploymentReady(t *testing.T) {
	ps := &pmock.PlatformService{}
	CS := ComponentService{
		PlatformService: ps,
	}

	inst := &v1alpha1.Gerrit{}

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
	inst := &v1alpha1.Gerrit{}
	CS := ComponentService{}
	CS.setAnnotation(inst, key, value)
	assert.Equal(t, map[string]string{key: value}, inst.Annotations)
}

func Test_setAnnotation(t *testing.T) {
	inst := &v1alpha1.Gerrit{
		ObjectMeta: metav1.ObjectMeta{
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
	ps.On("CreateSecret", instance, instance.Name+"-admin-password").Return(errTest)
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
	ps.On("CreateSecret", instance, instance.Name+"-admin-password").Return(nil)
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
	ps.On("CreateSecret", instance, instance.Name+"-admin-password").Return(nil)
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
	ps.On("CreateSecret", instance, instance.Name+"-admin-password").Return(nil)
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
	ps.On("CreateSecret", instance, instance.Name+"-admin-password").Return(nil)
	ps.On("GetService", instance.Namespace, instance.Name).Return(service, nil)
	ps.On("GetDeploymentSSHPort", instance).Return(port, nil)
	ps.On("UpdateService", *service, port).Return(errTest)

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
	ps.On("CreateSecret", instance, instance.Name+"-admin-password").Return(nil)
	ps.On("GetService", instance.Namespace, instance.Name).Return(service, nil)
	ps.On("GetDeploymentSSHPort", instance).Return(port, nil)
	ps.On("UpdateService", *service, servicePort).Return(nil)
	ps.On("PatchDeploymentEnv", *instance, newEnv).Return(errTest)

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
	ps.On("CreateSecret", instance, instance.Name+"-admin-password").Return(nil)
	ps.On("GetService", instance.Namespace, instance.Name).Return(service, nil)
	ps.On("GetDeploymentSSHPort", instance).Return(port, nil)
	ps.On("UpdateService", *service, servicePort).Return(nil)
	ps.On("PatchDeploymentEnv", *instance, newEnv).Return(nil)

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
	ps.On("CreateSecret", instance, instance.Name+"-admin-password").Return(nil)
	ps.On("GetService", instance.Namespace, instance.Name).Return(service, nil)
	ps.On("GetDeploymentSSHPort", instance).Return(port, nil)
	ps.On("UpdateService", *service, port).Return(nil)
	ps.On("GetSecretData", instance.Namespace, secretName).Return(secretData, nil)
	ps.On("GetPods", instance.Namespace,
		metav1.ListOptions{LabelSelector: "app=" + instance.Name}).Return(podList, errTest)

	configure, b, err := CS.Configure(instance)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), errTest.Error()))
	assert.True(t, strings.Contains(err.Error(), "Unable to determine Gerrit pod name"))
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
			TypeMeta: metav1.TypeMeta{}}}}

	errTest := errors.New("test")
	ps.On("GetExternalEndpoint", instance.Namespace, instance.Name).Return("", "", nil)
	ps.On("CreateSecret", instance, instance.Name+"-admin-password").Return(nil)
	ps.On("GetService", instance.Namespace, instance.Name).Return(service, nil)
	ps.On("GetDeploymentSSHPort", instance).Return(port, nil)
	ps.On("UpdateService", *service, port).Return(nil)
	ps.On("GetSecretData", instance.Namespace, secretName).Return(secretData, nil)
	ps.On("GetSecretData", instance.Namespace, instance.Name+"-admin").Return(secretData, errTest)
	ps.On("GetPods", instance.Namespace,
		metav1.ListOptions{LabelSelector: "app=" + instance.Name}).Return(podList, nil)

	configure, b, err := CS.Configure(instance)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), errTest.Error()))
	assert.True(t, strings.Contains(err.Error(), "Failed to create Gerrit admin SSH keypair"))
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
			TypeMeta: metav1.TypeMeta{}}}}

	errTest := errors.New("test")
	ps.On("GetExternalEndpoint", instance.Namespace, instance.Name).Return("", "", nil)
	ps.On("CreateSecret", instance, instance.Name+"-admin-password").Return(nil)
	ps.On("GetService", instance.Namespace, instance.Name).Return(service, nil)
	ps.On("GetDeploymentSSHPort", instance).Return(port, nil)
	ps.On("UpdateService", *service, port).Return(nil)
	ps.On("GetSecretData", instance.Namespace, secretName).Return(secretData, nil)
	ps.On("GetSecretData", instance.Namespace, instance.Name+"-admin").Return(secretData, nil)
	ps.On("GetSecretData", instance.Namespace, instance.Name+"-project-creator").Return(secretData, errTest)
	ps.On("GetPods", instance.Namespace,
		metav1.ListOptions{LabelSelector: "app=" + instance.Name}).Return(podList, nil)

	configure, b, err := CS.Configure(instance)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), errTest.Error()))
	assert.True(t, strings.Contains(err.Error(), "Failed to create Gerrit project-creator SSH keypair"))

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
			TypeMeta: metav1.TypeMeta{}}}}

	ps.On("GetExternalEndpoint", instance.Namespace, instance.Name).Return("", "", nil)
	ps.On("CreateSecret", instance, instance.Name+"-admin-password").Return(nil)
	ps.On("GetService", instance.Namespace, instance.Name).Return(service, nil)
	ps.On("GetDeploymentSSHPort", instance).Return(port, nil)
	ps.On("UpdateService", *service, port).Return(nil)
	ps.On("GetSecretData", instance.Namespace, secretName).Return(secretData, nil)
	ps.On("GetSecretData", instance.Namespace, instance.Name+"-admin").Return(secretData, nil)
	ps.On("GetSecretData", instance.Namespace, instance.Name+"-project-creator").Return(secretData, nil)
	ps.On("GetPods", instance.Namespace,
		metav1.ListOptions{LabelSelector: "app=" + instance.Name}).Return(podList, nil)

	configure, b, err := CS.Configure(instance)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "Unable to verify Gerrit credentials"))
	assert.Equal(t, instance, configure)
	assert.False(t, b)
}

func TestComponentService_Integrate_GetExternalEndpointErr(t *testing.T) {
	instance := CreateGerritInstance()

	errTest := errors.New("test")
	ps := &pmock.PlatformService{}
	CS := ComponentService{PlatformService: ps}

	ps.On("GetExternalEndpoint", instance.Namespace, instance.Name).Return("", "", errTest)

	_, err := CS.Integrate(instance)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), errTest.Error()))
}

func TestComponentService_Integrate_ParseDefaultTemplateErr(t *testing.T) {
	instance := &v1alpha1.Gerrit{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "name",
			Namespace: "namespace",
		},
		Spec: v1alpha1.GerritSpec{
			KeycloakSpec: v1alpha1.KeycloakSpec{
				Enabled: true,
			},
		},
	}
	client := &keycloakApi.KeycloakClient{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "name",
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
	scheme.AddKnownTypes(appsv1.SchemeGroupVersion, &v1alpha1.Gerrit{}, &keycloakApi.KeycloakClient{})
	cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(client).Build()
	CS := ComponentService{PlatformService: ps, client: cl}

	var envs []coreV1Api.EnvVar

	ps.On("GetExternalEndpoint", instance.Namespace, instance.Name).Return("", "", nil)
	ps.On("GenerateKeycloakSettings", instance).Return(&envs, nil)
	ps.On("PatchDeploymentEnv", *instance, envs).Return(nil)
	ps.On("GetService", instance.Namespace, instance.Name).Return(service, nil)
	ps.On("GetSecretData", instance.Namespace, ciUserCredentialsName).Return(secretData, nil)

	_, err := CS.Integrate(instance)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "Template file not found in path specificed!"))
}

func TestComponentService_Integrate_getKeycloakClientErr(t *testing.T) {
	instance := &v1alpha1.Gerrit{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "name",
			Namespace: "namespace",
		},
		Spec: v1alpha1.GerritSpec{
			KeycloakSpec: v1alpha1.KeycloakSpec{
				Enabled: true,
			},
		},
	}

	ps := &pmock.PlatformService{}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(appsv1.SchemeGroupVersion, &v1alpha1.Gerrit{})
	cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects().Build()
	CS := ComponentService{PlatformService: ps, client: cl}

	ps.On("GetExternalEndpoint", instance.Namespace, instance.Name).Return("", "", nil)

	_, err := CS.Integrate(instance)
	assert.Error(t, err)

}

func TestComponentService_Integrate_GenerateKeycloakSettingsErr(t *testing.T) {
	instance := &v1alpha1.Gerrit{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "name",
			Namespace: "namespace",
		},
		Spec: v1alpha1.GerritSpec{
			KeycloakSpec: v1alpha1.KeycloakSpec{
				Enabled: true,
			},
		},
	}
	client := &keycloakApi.KeycloakClient{}

	errTest := errors.New("test")

	ps := &pmock.PlatformService{}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(appsv1.SchemeGroupVersion, &v1alpha1.Gerrit{}, &keycloakApi.KeycloakClient{})
	cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(client).Build()
	CS := ComponentService{PlatformService: ps, client: cl}

	ps.On("GetExternalEndpoint", instance.Namespace, instance.Name).Return("", "", nil)
	ps.On("GenerateKeycloakSettings", instance).Return(nil, errTest)

	_, err := CS.Integrate(instance)
	assert.Equal(t, errTest, err)

}

func TestComponentService_Integrate_PatchDeploymentEnvErr(t *testing.T) {
	instance := &v1alpha1.Gerrit{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "name",
			Namespace: "namespace",
		},
		Spec: v1alpha1.GerritSpec{
			KeycloakSpec: v1alpha1.KeycloakSpec{
				Enabled: true,
			},
		},
	}
	client := &keycloakApi.KeycloakClient{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "name",
			Namespace: "namespace",
		},
	}

	ps := &pmock.PlatformService{}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(appsv1.SchemeGroupVersion, &v1alpha1.Gerrit{}, &keycloakApi.KeycloakClient{})
	cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(client).Build()
	CS := ComponentService{PlatformService: ps, client: cl}

	var envs []coreV1Api.EnvVar

	errTest := errors.New("test")
	ps.On("GetExternalEndpoint", instance.Namespace, instance.Name).Return("", "", nil)
	ps.On("GenerateKeycloakSettings", instance).Return(&envs, nil)
	ps.On("PatchDeploymentEnv", *instance, envs).Return(errTest)

	_, err := CS.Integrate(instance)
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

	_, err := CS.getUrl(*instance)
	assert.Equal(t, errTest, err)
}

func TestComponentService_getUrl(t *testing.T) {
	instance := CreateGerritInstance()
	url := fmt.Sprintf("%v://%v", sc, h)

	ps := &pmock.PlatformService{}
	CS := ComponentService{PlatformService: ps}
	ps.On("GetExternalEndpoint", instance.Namespace, instance.Name).Return(h, sc, nil)

	u, err := CS.getUrl(*instance)
	assert.NoError(t, err)
	assert.Equal(t, url, *u)
}

func TestComponentService_ExposeConfiguration_CreateUserErr(t *testing.T) {
	instance := CreateGerritInstance()
	ciUserSecretName := fmt.Sprintf("%v-%v", instance.Name, spec.GerritDefaultCiUserSecretPostfix)
	projectCreatorSecretPasswordName := fmt.Sprintf("%v-%v-%v", instance.Name, spec.GerritDefaultProjectCreatorSecretPostfix, "password")
	ciUserSshSecretName := fmt.Sprintf("%s-ciuser%s", instance.Name, spec.SshKeyPostfix)

	secretName := fmt.Sprintf("%v-admin-password", instance.Name)
	err, pkey := GenPkey()
	assert.NoError(t, err)
	secretData := map[string][]byte{
		"password": {'o'},
		"id_rsa":   pkey,
	}

	service := CreateService(servicePort)

	ps := &pmock.PlatformService{}
	CS := ComponentService{PlatformService: ps, gerritClient: &gerrit.Client{}}
	ps.On("GetSecretData", instance.Namespace, secretName).Return(secretData, nil)
	ps.On("GetExternalEndpoint", instance.Namespace, instance.Name).Return(h, sc, nil)
	ps.On("GetService", instance.Namespace, instance.Name).Return(service, nil)
	ps.On("GetSecret", instance.Namespace, instance.Name+"-admin").Return(secretData, nil)
	ps.On("CreateSecret", instance, ciUserSecretName).Return(nil)
	ps.On("CreateSecret", instance, ciUserSshSecretName).Return(nil)
	ps.On("CreateSecret", instance, projectCreatorSecretPasswordName).Return(nil)
	ps.On("GetSecretData", instance.Namespace, ciUserSecretName).Return(secretData, nil)
	ps.On("CreateJenkinsServiceAccount", instance.Namespace, ciUserSshSecretName, "ssh").Return(nil)
	_, err = CS.ExposeConfiguration(instance)
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

	_, err := CS.ExposeConfiguration(instance)
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
	_, err := CS.ExposeConfiguration(instance)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "Failed to init Gerrit SSH client"))

}

func TestComponentService_ExposeConfiguration_FirstCreateSecretErr(t *testing.T) {
	instance := CreateGerritInstance()
	ciUserSecretName := fmt.Sprintf("%v-%v", instance.Name, spec.GerritDefaultCiUserSecretPostfix)

	secretName := fmt.Sprintf("%v-admin-password", instance.Name)

	err, pkey := GenPkey()
	assert.NoError(t, err)
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
	ps.On("CreateSecret", instance, ciUserSecretName).Return(errTest)

	_, err = CS.ExposeConfiguration(instance)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "Failed to create ci user Secret"))
}

func TestComponentService_ExposeConfiguration_SecondCreateSecretErr(t *testing.T) {
	instance := CreateGerritInstance()
	ciUserSecretName := fmt.Sprintf("%v-%v", instance.Name, spec.GerritDefaultCiUserSecretPostfix)
	projectCreatorSecretPasswordName := fmt.Sprintf("%v-%v-%v", instance.Name, spec.GerritDefaultProjectCreatorSecretPostfix, "password")

	secretName := fmt.Sprintf("%v-admin-password", instance.Name)
	err, pkey := GenPkey()
	assert.NoError(t, err)
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
	ps.On("CreateSecret", instance, ciUserSecretName).Return(nil)
	ps.On("CreateSecret", instance, projectCreatorSecretPasswordName).Return(errTest)

	_, err = CS.ExposeConfiguration(instance)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "Failed to create project-creator Secret"))

}

func TestComponentService_ExposeConfiguration_GetSecretErr(t *testing.T) {
	instance := CreateGerritInstance()

	ciUserSecretName := fmt.Sprintf("%v-%v", instance.Name, spec.GerritDefaultCiUserSecretPostfix)
	projectCreatorSecretPasswordName := fmt.Sprintf("%v-%v-%v", instance.Name, spec.GerritDefaultProjectCreatorSecretPostfix, "password")

	secretName := fmt.Sprintf("%v-admin-password", instance.Name)
	err, pkey := GenPkey()
	assert.NoError(t, err)
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
	ps.On("CreateSecret", instance, ciUserSecretName).Return(nil)
	ps.On("CreateSecret", instance, projectCreatorSecretPasswordName).Return(nil)
	ps.On("GetSecretData", instance.Namespace, ciUserSecretName).Return(secretData, errTest)

	_, err = CS.ExposeConfiguration(instance)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "Failed to get Secret"))
}

func TestComponentService_ExposeConfiguration_CreateSecretErr(t *testing.T) {
	instance := CreateGerritInstance()
	ciUserSecretName := fmt.Sprintf("%v-%v", instance.Name, spec.GerritDefaultCiUserSecretPostfix)
	projectCreatorSecretPasswordName := fmt.Sprintf("%v-%v-%v", instance.Name, spec.GerritDefaultProjectCreatorSecretPostfix, "password")
	ciUserSshSecretName := fmt.Sprintf("%s-ciuser%s", instance.Name, spec.SshKeyPostfix)

	secretName := fmt.Sprintf("%v-admin-password", instance.Name)
	err, pkey := GenPkey()
	assert.NoError(t, err)
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
	ps.On("CreateSecret", instance, ciUserSecretName).Return(nil)
	ps.On("CreateSecret", instance, projectCreatorSecretPasswordName).Return(nil)
	ps.On("GetSecretData", instance.Namespace, ciUserSecretName).Return(secretData, nil)
	ps.On("CreateSecret", instance, ciUserSshSecretName).Return(errTest)
	_, err = CS.ExposeConfiguration(instance)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "Failed to create Secret with SSH key pairs for Gerrit"))

}

func TestComponentService_ExposeConfiguration_CreateJenkinsServiceAccountErr(t *testing.T) {
	instance := CreateGerritInstance()
	ciUserSecretName := fmt.Sprintf("%v-%v", instance.Name, spec.GerritDefaultCiUserSecretPostfix)
	projectCreatorSecretPasswordName := fmt.Sprintf("%v-%v-%v", instance.Name, spec.GerritDefaultProjectCreatorSecretPostfix, "password")
	ciUserSshSecretName := fmt.Sprintf("%s-ciuser%s", instance.Name, spec.SshKeyPostfix)

	secretName := fmt.Sprintf("%v-admin-password", instance.Name)
	err, pkey := GenPkey()
	assert.NoError(t, err)
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
	ps.On("CreateSecret", instance, ciUserSecretName).Return(nil)
	ps.On("CreateSecret", instance, ciUserSshSecretName).Return(nil)
	ps.On("CreateSecret", instance, projectCreatorSecretPasswordName).Return(nil)
	ps.On("GetSecretData", instance.Namespace, ciUserSecretName).Return(secretData, nil)
	ps.On("CreateJenkinsServiceAccount", instance.Namespace, ciUserSshSecretName, "ssh").Return(errTest)
	_, err = CS.ExposeConfiguration(instance)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "Failed to create Jenkins Service Account"))
}

func TestComponentService_Configure_CreateGroups(t *testing.T) {
	instance := CreateGerritInstance()
	gerritClient := gerrit.ClientInterfaceMock{}
	ps := &pmock.PlatformService{}
	kc := fake.NewClientBuilder().Build()
	ks := &runtime.Scheme{}
	//gerritClient := gerrit.NewClient(nil, resty.New(), nil)
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
			TypeMeta: metav1.TypeMeta{}}}}

	ps.On("GetExternalEndpoint", instance.Namespace, instance.Name).Return("", "", nil)
	ps.On("CreateSecret", instance, instance.Name+"-admin-password").Return(nil)
	ps.On("GetService", instance.Namespace, instance.Name).Return(service, nil)
	ps.On("GetDeploymentSSHPort", instance).Return(port, nil)
	ps.On("UpdateService", *service, port).Return(nil)
	ps.On("GetSecretData", instance.Namespace, secretName).Return(secretData, nil)
	ps.On("GetSecretData", instance.Namespace, instance.Name+"-admin").Return(secretData, nil)
	ps.On("GetSecretData", instance.Namespace, instance.Name+"-project-creator").Return(secretData, nil)
	ps.On("GetPods", instance.Namespace,
		metav1.ListOptions{LabelSelector: "app=" + instance.Name}).Return(podList, nil)
	ps.On("GetSecret", instance.Namespace, "name-admin").Return(map[string][]byte{}, nil)
	ps.On("ExecInPod", instance.Namespace, "",
		[]string{"/bin/sh", "-c", "chown -R gerrit2:gerrit2 /var/gerrit/review_site"}).
		Return("", "", nil)

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
