package helper

import (
	"context"
	coreerrors "errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/epam/edp-gerrit-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	platformType           = "PLATFORM_TYPE"
	watchNamespaceEnvVar   = "WATCH_NAMESPACE"
	debugModeEnvVar        = "DEBUG_MODE"
	inClusterNamespacePath = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
	StatusOK               = "OK"
)

// GetWatchNamespace returns the namespace the operator should be watching for changes
func GetWatchNamespace() (string, error) {
	ns, found := os.LookupEnv(watchNamespaceEnvVar)
	if !found {
		return "", fmt.Errorf("%s must be set", watchNamespaceEnvVar)
	}
	return ns, nil
}

// GetDebugMode returns the debug mode value
func GetDebugMode() (bool, error) {
	mode, found := os.LookupEnv(debugModeEnvVar)
	if !found {
		return false, nil
	}

	b, err := strconv.ParseBool(mode)
	if err != nil {
		return false, err
	}
	return b, nil
}

// Check whether the operator is running in cluster or locally
func RunningInCluster() bool {
	_, err := os.Stat(inClusterNamespacePath)
	if os.IsNotExist(err) {
		return false
	}
	return true
}

func NewTrue() *bool {
	value := true
	return &value
}

func GetPlatformTypeEnv() string {
	platformType, found := os.LookupEnv(platformType)
	if !found {
		panic("Environment variable PLATFORM_TYPE is not defined")
	}
	return platformType
}

func GetExecutableFilePath() (string, error) {
	executableFilePath, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(executableFilePath), nil
}

func FileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func SetOwnerReference(child metav1.Object, parentType metav1.TypeMeta, parentObject metav1.ObjectMeta) {
	var listOwnReference []metav1.OwnerReference

	ownRef := metav1.OwnerReference{
		APIVersion:         parentType.APIVersion,
		Kind:               parentType.Kind,
		Name:               parentObject.Name,
		UID:                parentObject.UID,
		BlockOwnerDeletion: NewTrue(),
		Controller:         NewTrue(),
	}

	listOwnReference = append(listOwnReference, ownRef)

	child.SetOwnerReferences(listOwnReference)
}

func IsInstanceOwnerSet(config metav1.Object) bool {
	ows := config.GetOwnerReferences()
	if len(ows) == 0 {
		return false
	}

	return true
}

func FindCROwnerName(ownerName string) *string {
	if len(ownerName) == 0 {
		return nil
	}
	own := strings.ToLower(ownerName)
	return &own
}

func GetInstanceOwner(ctx context.Context, k8sClient client.Client, config metav1.Object) (*v1alpha1.Gerrit, error) {
	ows := config.GetOwnerReferences()
	gerritOwner := GetGerritOwner(ows)
	if gerritOwner == nil {
		return nil, coreerrors.New("gerrit replication config cr does not have gerrit cr owner references")
	}

	nsn := types.NamespacedName{
		Namespace: config.GetNamespace(),
		Name:      gerritOwner.Name,
	}

	ownerCr := &v1alpha1.Gerrit{}
	if err := k8sClient.Get(ctx, nsn, ownerCr); err != nil {
		return nil, errors.Wrap(err, "unable to get gerrit owner")
	}

	return ownerCr, nil
}

func GetGerritOwner(references []metav1.OwnerReference) *metav1.OwnerReference {
	for _, el := range references {
		if el.Kind == "Gerrit" {
			return &el
		}
	}
	return nil
}

func GetGerritInstance(ctx context.Context, k8sClient client.Client, ownerName *string, namespace string) (*v1alpha1.Gerrit, error) {
	var gerritInstance v1alpha1.Gerrit
	options := client.ListOptions{Namespace: namespace}
	list := &v1alpha1.GerritList{}
	if ownerName == nil {
		err := k8sClient.List(ctx, list, &options)
		if err != nil {
			if k8sErrors.IsNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		gerritInstance = list.Items[0]
	} else {
		gerritInstance = v1alpha1.Gerrit{}
		err := k8sClient.Get(ctx, client.ObjectKey{
			Namespace: namespace,
			Name:      *ownerName,
		}, &gerritInstance)
		if err != nil {
			if k8sErrors.IsNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
	}

	return &gerritInstance, nil
}

func ContainsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func RemoveString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}
