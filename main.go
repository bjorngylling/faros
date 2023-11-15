package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"
)

func main() {
	buildInfo, _ := debug.ReadBuildInfo()
	var vcsRevision, vcsTime, vcsModified string
	for _, s := range buildInfo.Settings {
		switch s.Key {
		case "vcs.revision":
			vcsRevision = s.Value
		case "vcs.time":
			vcsTime = s.Value
		case "vcs.modified":
			vcsModified = s.Value
		}
	}
	if vcsModified == "true" {
		vcsRevision = ""
	}
	logger := *slog.New(slog.Default().Handler())
	logger.LogAttrs(nil, slog.LevelInfo, "faros - a k8s ingress-controller",
		slog.String("build_revision", vcsRevision),
		slog.String("build_time", vcsTime),
		slog.String("build_version", buildInfo.Main.Version))

	cl, gwCl, err := initK8sClient()
	if err != nil {
		logger.Error(err.Error())
	}
	w := Watcher{client: cl, gatewayClient: gwCl, log: logger}

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	err = w.Run(ctx)
	if err != nil {
		logger.Error(err.Error())
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Hello world!")
	})
	http.ListenAndServe(":80", nil)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	// Block until a signal is received.
	<-c
}

func initK8sClient() (*kubernetes.Clientset, *versioned.Clientset, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		var kubeconfig *string
		if home := homedir.HomeDir(); home != "" {
			kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
		} else {
			kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
		}
		flag.Parse()

		config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
		if err != nil {
			return nil, nil, fmt.Errorf("building k8s config: %w", err)
		}
	}
	cl, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("creating kubernetes client: %w", err)
	}
	gwCl, err := versioned.NewForConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("creating kubernetes client: %w", err)
	}

	return cl, gwCl, nil
}
