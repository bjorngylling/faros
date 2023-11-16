package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	gatev1 "sigs.k8s.io/gateway-api/apis/v1"
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

	router := Router{table: map[string]*url.URL{}, log: logger}
	err = w.Run(ctx, router.Add)
	if err != nil {
		logger.Error(err.Error())
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		slog.LogAttrs(r.Context(), slog.LevelInfo, "received request",
			slog.String("path", r.URL.Path))

		httputil.NewSingleHostReverseProxy(router.Route(r.URL.Path)).ServeHTTP(w, r)
	})
	err = http.ListenAndServe(":"+port, nil)
	if err != nil {
		logger.Error(err.Error())
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	// Block until a signal is received.
	<-c
}

type Router struct {
	table map[string]*url.URL
	log   slog.Logger
}

func (r *Router) Add(route *gatev1.HTTPRoute) {
	for _, rule := range route.Spec.Rules {
		if len(rule.BackendRefs) > 0 {
			var err error
			backend, err := url.Parse("http://" + string(rule.BackendRefs[0].Name))
			if err != nil {
				r.log.Error(err.Error())
			}
			if len(rule.Matches) > 0 {
				for _, m := range rule.Matches {
					r.table[*m.Path.Value] = backend
				}
			}
		}
	}
}

func (r *Router) Route(path string) *url.URL {
	return r.table[path]
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
