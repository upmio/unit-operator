package main

import (
	"flag"
	"os"
	"time"

	klog "k8s.io/klog/v2"
	"k8s.io/utils/ptr"

	"github.com/upmio/unit-operator/pkg/controller/project"

	upmv1alpha1 "github.com/upmio/unit-operator/api/v1alpha1"
	upmv1alpha2 "github.com/upmio/unit-operator/api/v1alpha2"
	"github.com/upmio/unit-operator/pkg/certs"
	genericClient "github.com/upmio/unit-operator/pkg/client/generic"
	"github.com/upmio/unit-operator/pkg/controller"
	"github.com/upmio/unit-operator/pkg/controller/unit"
	"github.com/upmio/unit-operator/pkg/controller/unitset"
	"github.com/upmio/unit-operator/pkg/vars"
	upmioWebhook "github.com/upmio/unit-operator/pkg/webhook/v1alpha2"

	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	//+kubebuilder:scaffold:imports

	certmanagerV1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	serviceMonitorv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	apiextensionsV1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")

	leaderElect bool

	metricsAddr   string
	probeAddr     string
	agentHostType string
	versionFlag   bool

	webhookPort int

	//secureMetrics bool
	//enableHTTP2   bool

	logDir         string
	logFileMaxSize string
	logLevel       string
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(upmv1alpha2.AddToScheme(scheme))
	utilruntime.Must(upmv1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":20154",
		"The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":20153",
		"The address the probe endpoint binds to.")

	flag.BoolVar(&versionFlag, "version", false,
		"show the version ")
	flag.StringVar(&agentHostType, "unit-agent-host-type", "",
		"The host type of unit-agent.")

	flag.IntVar(&webhookPort, "webhook-port", 9443,
		"Webhook server port")

	//flag.BoolVar(&secureMetrics, "metrics-secure", false,
	//	"If set the metrics endpoint is served securely")
	//flag.BoolVar(&enableHTTP2, "enable-http2", false,
	//	"If set, HTTP/2 will be enabled for the metrics and webhook servers")

	flag.BoolVar(&leaderElect, "leader-elect", true, ""+
		"Start a leader election client and gain leadership before "+
		"executing the main loop. Enable this when running replicated "+
		"components for high availability.")

	// Register log flags
	flag.StringVar(&logFileMaxSize, "log-file-max-size", "",
		"Maximum log file size in MB before rotation (overrides LOG_MAX_SIZE).")
	flag.StringVar(&logDir, "log-dir", "",
		"Directory to write log files (overrides LOG_PATH).")
	flag.StringVar(&logLevel, "log-level", "",
		"Log level: debug | info | error (overrides LOG_LEVEL).")
}

func main() {

	flag.Parse()

	if versionFlag {
		vars.PrintVersion()
		return
	}

	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)

	//ctrl.SetLogger(log.InitLogger(logDir, "", logFileMaxSize))
	//ctrl.SetLogger(log.InitLoggerFromFlagsAndEnv(logDir, logLevel, logFileMaxSize))
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))
	klog.SetLogger(ctrl.Log)

	cfg := ctrl.GetConfigOrDie()
	cfg.UserAgent = "unit-operator-manager"

	// if the enable-http2 flag is false (the default), http/2 should be disabled
	// due to its vulnerabilities. More specifically, disabling http/2 will
	// prevent from being vulnerable to the HTTP/2 Stream Cancelation and
	// Rapid Reset CVEs. For more information see:
	// - https://github.com/advisories/GHSA-qppj-fm5r-hxr3
	// - https://github.com/advisories/GHSA-4374-p667-p6c8

	//disableHTTP2 := func(c *tls.Config) {
	//	setupLog.Info("disabling http/2")
	//	c.NextProtos = []string{"http/1.1"}
	//}
	//
	//tlsOpts := []func(*tls.Config){}
	//if !enableHTTP2 {
	//	tlsOpts = append(tlsOpts, disableHTTP2)
	//}

	//webhookServer := webhook.NewServer(webhook.Options{
	//	TLSOpts: tlsOpts,
	//})

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: metricsAddr,
		},
		WebhookServer: webhook.NewServer(webhook.Options{
			Port:    webhookPort,
			CertDir: certs.DefaultWebhookCertDir,
		}),
		HealthProbeBindAddress: probeAddr,

		LeaderElection:          leaderElect,
		LeaderElectionID:        "unit-operator",
		LeaderElectionNamespace: vars.ManagerNamespace,
		LeaseDuration:           ptr.To(30 * time.Second),
		RenewDeadline:           ptr.To(20 * time.Second),
		RetryPeriod:             ptr.To(2 * time.Second),

		//NewCache: client.NewCache,

		// LeaderElectionReleaseOnCancel defines if the leader should step down voluntarily
		// when the Manager ends. This requires the binary to immediately end when the
		// Manager is stopped, otherwise, this setting is unsafe. Setting this significantly
		// speeds up voluntary leader transitions as the new leader don't have to wait
		// LeaseDuration time first.
		//
		// In the default scaffold provided, the program ends immediately after
		// the manager stops, so would be fine to enable this option. However,
		// if you are doing or is intended to do any operation such as perform cleanups
		// after the manager stops then its usage might be unsafe.
		LeaderElectionReleaseOnCancel: true,
	})
	if err != nil {
		setupLog.Error(err, "unable to create manager")
		os.Exit(1)
	}

	//+kubebuilder:scaffold:builder

	err = genericClient.NewRegistry(cfg)
	if err != nil {
		setupLog.Error(err, "unable to init generic clientset and informer")
		os.Exit(1)
	}

	if agentHostType != "" {
		vars.UnitAgentHostType = agentHostType
	}

	err = apiextensionsV1.AddToScheme(mgr.GetScheme())
	if err != nil {
		setupLog.Error(err, "Cannot add apiextensions APIs to scheme")
		os.Exit(1)
	}

	err = serviceMonitorv1.AddToScheme(mgr.GetScheme())
	if err != nil {
		setupLog.Error(err, "Cannot add serviceMonitor APIs to scheme")
		os.Exit(1)
	}

	err = certmanagerV1.AddToScheme(mgr.GetScheme())
	if err != nil {
		setupLog.Error(err, "Cannot add certmanager APIs to scheme")
		os.Exit(1)
	}

	//kubeClient, err := client.New(mgr.GetConfig(), client.Options{Scheme: scheme})
	//if err != nil {
	//	setupLog.Error(err, "unable to create Kubernetes client")
	//	os.Exit(1)
	//}

	//webhookServer := mgr.GetWebhookServer().(*webhook.DefaultServer)
	//if err := certs.EnsurePKI(context.TODO(), kubeClient, webhookServer.Options.CertDir); err != nil {
	//	setupLog.Error(err, "unable to ensure PKI")
	//	os.Exit(1)
	//}

	if err = (&unitset.UnitSetReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("unitset-controller"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "UnitSet")
		os.Exit(1)
	}

	if err = (&unit.UnitReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("unit-controller"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Unit")
		os.Exit(1)
	}

	if err = (&project.ProjectReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("project-controller"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Project")
		os.Exit(1)
	}

	err = controller.Setup(mgr)
	if err != nil {
		setupLog.Error(err, "unable to setup grpccall manager")
		os.Exit(1)
	}

	if os.Getenv("ENABLE_WEBHOOKS") != "false" {
		if err = upmioWebhook.SetupUnitsetWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "UnitSet")
			os.Exit(1)
		}
		setupLog.Info("setup unitset webhook ok")
	}

	if os.Getenv("ENABLE_WEBHOOKS") != "false" {
		if err = upmioWebhook.SetupUnitWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "Unit")
			os.Exit(1)
		}
	}

	if os.Getenv("ENABLE_WEBHOOKS") != "false" {
		if err = upmioWebhook.SetupProjectWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "Project")
			os.Exit(1)
		}
	}
	// +kubebuilder:scaffold:builder

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
