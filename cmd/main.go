package main

import (
	"fmt"
	"math"
	"os"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"

	kubeinformers "k8s.io/client-go/informers"

	foo_clientset "github.com/vietanhduong/xcontroller/pkg/client/clientset/versioned"
	foo_informers "github.com/vietanhduong/xcontroller/pkg/client/informers/externalversions"
	"github.com/vietanhduong/xcontroller/pkg/controller"
	"github.com/vietanhduong/xcontroller/pkg/util/env"
	"github.com/vietanhduong/xcontroller/pkg/util/log"
)

var (
	GitCommit = "unknown"
	BuildDate = "unknown"
	Version   = "unreleased"
)

func showVersion() {
	fmt.Printf("Version:\t %s\n", Version)
	fmt.Printf("Git commit:\t %s\n", GitCommit)
	fmt.Printf("Date:\t\t %s\n", BuildDate)
}

func newCommand() *cobra.Command {
	var (
		kubeconfig string
		logLevel   string
		worker     int

		cfg        *rest.Config
		kubeClient *kubernetes.Clientset
		fooClient  *foo_clientset.Clientset
	)
	var cmd = &cobra.Command{
		Use:          "xcontroller",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			if cmd.Flag("version").Value.String() == "true" {
				showVersion()
				return nil
			}
			logger := log.NewLogger(logLevel)
			zap.ReplaceGlobals(logger)
			klog.SetLogger(log.NewK8sLogger(logLevel))

			if cfg, err = clientcmd.BuildConfigFromFlags("", kubeconfig); err != nil {
				return err
			}

			if kubeClient, err = getKubeConfig(kubeconfig); err != nil {
				return err
			}

			if fooClient, err = foo_clientset.NewForConfig(cfg); err != nil {
				return err
			}

			eventBroadcaster := record.NewBroadcaster()
			eventBroadcaster.StartStructuredLogging(0)
			eventBroadcaster.StartRecordingToSink(&typev1.EventSinkImpl{Interface: kubeClient.CoreV1().Events(metav1.NamespaceAll)})
			recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "xcontroller"})

			fooInformerFactor := foo_informers.NewSharedInformerFactory(fooClient, time.Hour)
			kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, time.Hour)

			ctrl := controller.NewController(cmd.Context(), fooInformerFactor, kubeInformerFactory, recorder)
			fooInformerFactor.Start(cmd.Context().Done())
			kubeInformerFactory.Start(cmd.Context().Done())
			return ctrl.Run(worker)
		},
	}
	cmd.PersistentFlags().BoolP("version", "v", false, "Print version information and exit. This flag is only available at the global level.")
	cmd.Flags().StringVar(&kubeconfig, "kubeconfig", env.StringFromEnv("KUBECONFIG", ""), "Full path to kubernetes client configuration, i.e. ~/.kube/config")
	cmd.Flags().StringVar(&logLevel, "log-level", env.StringFromEnv("LOG_LEVEL", "info"), "Log level")
	cmd.Flags().IntVar(&worker, "private-ingress-workers", env.ParseNumFromEnv("WORKERS", 10, 1, math.MaxInt32), "Number of workers")

	return cmd
}

func main() {
	cmd := newCommand()
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
