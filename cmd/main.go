package main

import (
	"dubbo-kubernetes-ingress-controller/cmd/taskgroup"
	"dubbo-kubernetes-ingress-controller/pkg/controller"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"path/filepath"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	client := getKubeconfig()
	c := controller.NewIngressController(client)

	group := &taskgroup.Group{}
	group.Go(func() {
		c.Run(5, nil)
	})
	
	group.Wait()
}

func getKubeconfig() *kubernetes.Clientset {
	config, err := rest.InClusterConfig()
	if err != nil {
		config, err = clientcmd.BuildConfigFromFlags("", filepath.Join(homeDir(), ".kube", "config"))
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to get kubernetes client")
	}
	log.Info().Msg("get kubernetes client success")
	return clientset
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return ""
}
