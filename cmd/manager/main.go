package main

import (
	"flag"
	"os"

	"github.com/epam/edp-gerrit-operator/v2/pkg/controller/gerritgroupmember"

	"github.com/epam/edp-gerrit-operator/v2/pkg/controller/gerritprojectaccess"

	edpCompApi "github.com/epam/edp-component-operator/pkg/apis/v1/v1alpha1"
	gerritApi "github.com/epam/edp-gerrit-operator/v2/pkg/apis/v2/v1alpha1"
	gerritContr "github.com/epam/edp-gerrit-operator/v2/pkg/controller/gerrit"
	"github.com/epam/edp-gerrit-operator/v2/pkg/controller/gerritgroup"
	"github.com/epam/edp-gerrit-operator/v2/pkg/controller/gerritreplicationconfig"
	"github.com/epam/edp-gerrit-operator/v2/pkg/controller/helper"
	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	keycloakApi "github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/rest"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	//+kubebuilder:scaffold:imports
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

const gerritOperatorLock = "edp-gerrit-operator-lock"

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(gerritApi.AddToScheme(scheme))

	utilruntime.Must(edpCompApi.AddToScheme(scheme))

	utilruntime.Must(jenkinsApi.AddToScheme(scheme))

	utilruntime.Must(keycloakApi.AddToScheme(scheme))
}

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

	ns, err := helper.GetWatchNamespace()
	if err != nil {
		setupLog.Error(err, "unable to get watch namespace")
		os.Exit(1)
	}

	cfg := ctrl.GetConfigOrDie()
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		HealthProbeBindAddress: probeAddr,
		Port:                   9443,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       gerritOperatorLock,
		MapperProvider: func(c *rest.Config) (meta.RESTMapper, error) {
			return apiutil.NewDynamicRESTMapper(cfg)
		},
		Namespace: ns,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	ctrlLog := ctrl.Log.WithName("controllers")
	gerritCtrl, err := gerritContr.NewReconcileGerrit(mgr.GetClient(), mgr.GetScheme(), ctrlLog)
	if err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "gerrit")
		os.Exit(1)
	}

	if err := gerritCtrl.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "gerrit")
		os.Exit(1)
	}

	grcCtrl, err := gerritreplicationconfig.NewReconcileGerritReplicationConfig(mgr.GetClient(), mgr.GetScheme(), ctrlLog)
	if err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "gerrit-replication-config")
		os.Exit(1)
	}

	if err := grcCtrl.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "gerrit-replication-config")
		os.Exit(1)
	}

	grGroupCtrl, err := gerritgroup.NewReconcile(mgr.GetClient(), mgr.GetScheme(), ctrlLog)
	if err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "gerrit-group")
		os.Exit(1)
	}

	if err := grGroupCtrl.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "gerrit-group")
		os.Exit(1)
	}

	grProjectAccessCtrl, err := gerritprojectaccess.NewReconcile(mgr.GetClient(), mgr.GetScheme(), ctrlLog)
	if err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "gerrit-project-access")
		os.Exit(1)
	}

	if err := grProjectAccessCtrl.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "gerrit-project-access")
		os.Exit(1)
	}

	grGroupMemberCtrl, err := gerritgroupmember.NewReconcile(mgr.GetClient(), mgr.GetScheme(), ctrlLog)
	if err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "gerrit-group-member")
		os.Exit(1)
	}
	if err := grGroupMemberCtrl.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "gerrit-group-member")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}

	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
