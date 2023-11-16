package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
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
		vcsRevision += "-modified"
	}
	logger := *slog.New(slog.Default().Handler())
	logger.LogAttrs(nil, slog.LevelInfo, "faros - a k8s ingress-controller",
		slog.String("build_revision", vcsRevision),
		slog.String("build_time", vcsTime))

	var kubeconfig string
	if home := homedir.HomeDir(); home != "" {
		flag.StringVar(&kubeconfig, "kubeconfig", filepath.Join(home, ".kube", "config"),
			"(optional) absolute path to the kubeconfig file")
	} else {
		flag.StringVar(&kubeconfig, "kubeconfig", "",
			"absolute path to the kubeconfig file")
	}
	var port string
	flag.StringVar(&port, "port", "80", "(optional) listen port")
	flag.Parse()

	cl, gwCl, err := initK8sClient(kubeconfig)
	if err != nil {
		logger.Error(err.Error())
	}
	w := Watcher{client: cl, gatewayClient: gwCl, log: logger}

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	router := Router{log: logger}
	err = w.Run(ctx, router.Add)
	if err != nil {
		logger.Error(err.Error())
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log := logger.With(slog.String("path", r.URL.Path),
			slog.String("method", r.Method),
			slog.String("host", r.Host))
		u := router.Route(r.Host, r.URL.Path)
		if u == nil {
			http.NotFound(w, r)
			log.LogAttrs(r.Context(), slog.LevelInfo, "route not found")
			return
		}
		log.LogAttrs(r.Context(), slog.LevelInfo, "routed successfully",
			slog.String("target", u.String()))
		httputil.NewSingleHostReverseProxy(u).ServeHTTP(w, r)
	})
	err = http.ListenAndServe(":"+port, nil)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error(err.Error())
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	// Block until a signal is received.
	<-c
}

func initK8sClient(kubeconfig string) (*kubernetes.Clientset, *versioned.Clientset, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
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
