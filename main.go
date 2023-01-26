package main

import (
	"flag"
	"os"

	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	buildInfo "github.com/epam/edp-common/pkg/config"
	edpCompApi "github.com/epam/edp-component-operator/api/v1"
	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"

	gerritApi "github.com/epam/edp-gerrit-operator/v2/api/v1"
	gerritAlpha "github.com/epam/edp-gerrit-operator/v2/api/v1alpha1"
	gerritContr "github.com/epam/edp-gerrit-operator/v2/controllers/gerrit"
	"github.com/epam/edp-gerrit-operator/v2/controllers/gerritgroup"
	"github.com/epam/edp-gerrit-operator/v2/controllers/gerritgroupmember"
	"github.com/epam/edp-gerrit-operator/v2/controllers/gerritproject"
	"github.com/epam/edp-gerrit-operator/v2/controllers/gerritprojectaccess"
	"github.com/epam/edp-gerrit-operator/v2/controllers/gerritreplicationconfig"
	"github.com/epam/edp-gerrit-operator/v2/controllers/helper"
	"github.com/epam/edp-gerrit-operator/v2/controllers/merge_request"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

const (
	serverPort         = 9443
	gerritOperatorLock = "edp-gerrit-operator-lock"
	gitWorkDirEnv      = "GIT_WORK_DIR"
	gitWorkDirDefault  = "/tmp/git_tmp"
)

func main() {
	var (
		metricsAddr          string
		enableLeaderElection bool
		probeAddr            string
	)

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", helper.RunningInCluster(),
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")

	mode, err := helper.GetDebugMode()
	if err != nil {
		setupLog.Error(err, "unable to get debug mode value")
		os.Exit(1)
	}

	opts := zap.Options{
		Development: mode,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	logBuildInfo(setupLog)

	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(gerritApi.AddToScheme(scheme))
	utilruntime.Must(gerritAlpha.AddToScheme(scheme))
	utilruntime.Must(edpCompApi.AddToScheme(scheme))
	utilruntime.Must(jenkinsApi.AddToScheme(scheme))
	utilruntime.Must(keycloakApi.AddToScheme(scheme))

	mgr, err := initManager(metricsAddr, probeAddr, enableLeaderElection)
	if err != nil {
		setupLog.Error(err, "unable to init manager")
		os.Exit(1)
	}

	if err = initControllers(mgr, ctrl.Log.WithName("controllers")); err != nil {
		setupLog.Error(err, "error during controllers init")
		os.Exit(1)
	}

	if err = mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}

	if err = mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")

	err = mgr.Start(ctrl.SetupSignalHandler())
	if err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func initManager(metricsAddr, probeAddr string, enableLeaderElection bool) (ctrl.Manager, error) {
	ns, err := helper.GetWatchNamespace()
	if err != nil {
		return nil, errors.Wrap(err, "unable to get watch namespace")
	}

	cfg := ctrl.GetConfigOrDie()

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		HealthProbeBindAddress: probeAddr,
		Port:                   serverPort,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       gerritOperatorLock,
		MapperProvider: func(c *rest.Config) (meta.RESTMapper, error) {
			return apiutil.NewDynamicRESTMapper(cfg)
		},
		Namespace: ns,
	})
	if err != nil {
		return nil, errors.Wrap(err, "unable to start manager")
	}

	return mgr, nil
}

func initControllers(mgr ctrl.Manager, ctrlLog logr.Logger) error {
	controllersInitFuncs := controllerConstructors()

	for _, cf := range controllersInitFuncs {
		gCtrl, err := cf.Func(mgr.GetClient(), mgr.GetScheme(), ctrlLog)
		if err != nil {
			return errors.Wrapf(err, "unable to create controller, controller: %s", cf.ControllerName)
		}

		if err = gCtrl.SetupWithManager(mgr); err != nil {
			return errors.Wrapf(err, "unable to create controller, controller: %s", cf.ControllerName)
		}
	}

	setupers := prepareControllers()
	for i := range setupers {
		setuper := setupers[i]
		if err := setuper.PrepareFn(mgr, ctrlLog); err != nil {
			return err
		}
	}

	return nil
}

func controllerConstructors() []helper.InitFunc {
	return []helper.InitFunc{
		{
			Func:           gerritContr.NewReconcileGerrit,
			ControllerName: "gerrit",
		},
		{
			Func:           gerritreplicationconfig.NewReconcileGerritReplicationConfig,
			ControllerName: "gerrit-replication-config",
		},
		{
			Func:           gerritgroup.NewReconcile,
			ControllerName: "gerrit-group",
		},
		{
			Func:           gerritprojectaccess.NewReconcile,
			ControllerName: "gerrit-project-access",
		},
		{
			Func:           gerritgroupmember.NewReconcile,
			ControllerName: "gerrit-group-member",
		},
	}
}

type prepareController struct {
	PrepareFn      func(manager ctrl.Manager, ctrlLog logr.Logger) error
	ControllerName string
}

func prepareControllers() []prepareController {
	return []prepareController{
		{
			ControllerName: "gerrit-merge-request",
			PrepareFn:      prepareMergeRequestReconciler,
		},
		{
			ControllerName: "gerrit-project",
			PrepareFn:      prepareGerritProjectReconciler,
		},
	}
}

func prepareMergeRequestReconciler(mgr ctrl.Manager, ctrlLog logr.Logger) error {
	workDirectoryOption, err := mergerequest.PrepareWorkDirectoryOption(getEnvDefault(gitWorkDirEnv, gitWorkDirDefault))
	if err != nil {
		return nil
	}

	gerritServiceOption, err := mergerequest.PrepareGerritServiceOption(mgr.GetClient(), helper.GetPlatformTypeEnv(),
		mgr.GetScheme())
	if err != nil {
		return err
	}

	mergeRequestReconcilerOpts := []mergerequest.OptionFunc{
		workDirectoryOption,
		gerritServiceOption,
	}

	mergeRequestReconciler := mergerequest.NewReconcile(mgr.GetClient(), ctrlLog, mergeRequestReconcilerOpts...)

	err = mergeRequestReconciler.SetupWithManager(mgr)
	if err != nil {
		return err
	}

	return nil
}

func prepareGerritProjectReconciler(manager ctrl.Manager, ctrlLog logr.Logger) error {
	syncInterval := gerritproject.SyncInterval()

	gerritProjectReconciler, err := gerritproject.NewReconcile(manager.GetClient(), manager.GetScheme(), ctrlLog)
	if err != nil {
		return err
	}

	err = gerritProjectReconciler.SetupWithManager(manager, syncInterval)
	if err != nil {
		return err
	}

	return nil
}

func logBuildInfo(logger logr.Logger) {
	v := buildInfo.Get()

	logger.Info("Starting the Gerrit Operator",
		"version", v.Version,
		"git-commit", v.GitCommit,
		"git-tag", v.GitTag,
		"build-date", v.BuildDate,
		"go-version", v.Go,
		"go-client", v.KubectlVersion,
		"platform", v.Platform,
	)
}

func getEnvDefault(key, _default string) string {
	val := os.Getenv(key)
	if val == "" {
		return _default
	}

	return val
}
