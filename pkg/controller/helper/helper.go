package helper

import (
	"context"
	coreerrors "errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	gerritApi "github.com/epam/edp-gerrit-operator/v2/pkg/apis/v2/v1"
	gerritClient "github.com/epam/edp-gerrit-operator/v2/pkg/client/gerrit"
	gerritService "github.com/epam/edp-gerrit-operator/v2/pkg/service/gerrit"
)

const (
	platformType           = "PLATFORM_TYPE"
	watchNamespaceEnvVar   = "WATCH_NAMESPACE"
	debugModeEnvVar        = "DEBUG_MODE"
	inClusterNamespacePath = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
	DefaultRequeueTime     = 30
)

// GetWatchNamespace returns the namespace the operator should be watching for changes.
func GetWatchNamespace() (string, error) {
	ns, found := os.LookupEnv(watchNamespaceEnvVar)
	if !found {
		return "", fmt.Errorf("%s must be set", watchNamespaceEnvVar)
	}

	return ns, nil
}

// GetDebugMode returns the debug mode value.
func GetDebugMode() (bool, error) {
	mode, found := os.LookupEnv(debugModeEnvVar)
	if !found {
		return false, nil
	}

	b, err := strconv.ParseBool(mode)
	if err != nil {
		return false, fmt.Errorf("failed to parse bool value %q for debug mode: %w", mode, err)
	}

	return b, nil
}

// RunningInCluster check whether the operator is running in cluster or locally.
func RunningInCluster() bool {
	_, err := os.Stat(inClusterNamespacePath)
	return !os.IsNotExist(err)
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
		return "", fmt.Errorf("failed to get path for the executable that started process: %w", err)
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

func SetOwnerReference(child metav1.Object, parentType metav1.TypeMeta, parentObject *metav1.ObjectMeta) {
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
	return len(ows) != 0
}

func FindCROwnerName(ownerName string) *string {
	if ownerName == "" {
		return nil
	}

	own := strings.ToLower(ownerName)

	return &own
}

func GetInstanceOwner(ctx context.Context, k8sClient client.Client, config metav1.Object) (*gerritApi.Gerrit, error) {
	ows := config.GetOwnerReferences()
	gerritOwner := GetGerritOwner(ows)

	if gerritOwner == nil {
		return nil, coreerrors.New("gerrit replication config cr does not have gerrit cr owner references")
	}

	nsn := types.NamespacedName{
		Namespace: config.GetNamespace(),
		Name:      gerritOwner.Name,
	}

	ownerCr := &gerritApi.Gerrit{}
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

func GetGerritInstance(ctx context.Context, k8sClient client.Client, ownerName *string,
	namespace string) (*gerritApi.Gerrit, error) {
	var list gerritApi.GerritList

	if ownerName == nil {
		err := k8sClient.List(ctx, &list, &client.ListOptions{Namespace: namespace})
		if err != nil {
			return nil, errors.Wrap(err, "unable to list gerrits")
		}

		if len(list.Items) == 0 {
			return nil, errors.New("no root gerrits found")
		}

		return &list.Items[0], nil
	}

	var gerritInstance gerritApi.Gerrit
	if err := k8sClient.Get(ctx, client.ObjectKey{
		Namespace: namespace,
		Name:      *ownerName,
	}, &gerritInstance); err != nil {
		return nil, errors.Wrap(err, "unable to get gerrit instance")
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

func GetGerritClient(ctx context.Context, cl client.Client, instance client.Object, ownerName string,
	service gerritService.Interface) (gerritClient.ClientInterface, error) {
	if !IsInstanceOwnerSet(instance) {
		ownerReference := FindCROwnerName(ownerName)

		gerritInstance, err := GetGerritInstance(ctx, cl, ownerReference, instance.GetNamespace())
		if err != nil {
			return nil, errors.Wrap(err, "unable to get gerrit instance")
		}

		SetOwnerReference(instance, gerritInstance.TypeMeta, &gerritInstance.ObjectMeta)

		if err := cl.Update(ctx, instance); err != nil {
			return nil, errors.Wrap(err, "unable to update instance owner refs")
		}
	}

	gerritInstance, err := GetInstanceOwner(ctx, cl, instance)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get instance owner")
	}

	gerritCl, err := service.GetRestClient(gerritInstance)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get rest client")
	}

	return gerritCl, nil
}

func TryToDelete(ctx context.Context, k8sClient client.Client, instance client.Object, finalizerName string,
	deleteFunc func() error) error {
	if instance.GetDeletionTimestamp().IsZero() {
		finalizers := instance.GetFinalizers()
		if !ContainsString(finalizers, finalizerName) {
			finalizers = append(finalizers, finalizerName)
			instance.SetFinalizers(finalizers)

			if err := k8sClient.Update(ctx, instance); err != nil {
				return errors.Wrap(err, "unable to update instance finalizer")
			}
		}

		return nil
	}

	if err := deleteFunc(); err != nil {
		return errors.Wrap(err, "unable to perform delete function")
	}

	finalizers := instance.GetFinalizers()
	finalizers = RemoveString(finalizers, finalizerName)
	instance.SetFinalizers(finalizers)

	if err := k8sClient.Update(ctx, instance); err != nil {
		return errors.Wrap(err, "unable to remove finalizer from instance")
	}

	return nil
}
