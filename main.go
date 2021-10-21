package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"

	"k8s.io/client-go/rest"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

var (
	commitHash = "unknown"
	buildDate  = "unknown"
)

func main() {
	fmt.Println("faros - a k8s ingress-controller")
	fmt.Printf("        commit=%s,build_date=%s\n", commitHash, buildDate)

	cl := initK8sClient()
	w := Watcher{client: cl, onChange: func(payload *Payload) {
		marshal, _ := json.Marshal(payload)
		fmt.Printf("payload: %s\n", marshal)
	}}

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	err := w.Run(ctx)
	if err != nil {
		log.Fatalf("%s", err)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	// Block until a signal is received.
	<-c
}

func initK8sClient() *kubernetes.Clientset {
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
			log.Fatalf("failed to build k8s config: %s", err)
		}
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("failed to create kubernetes client: %s", err)
	}

	return client
}
