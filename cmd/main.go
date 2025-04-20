package main

import (
	"flag"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"ingress-controller/cmd/wgroup"
	"ingress-controller/pkg/controller"
	"ingress-controller/pkg/proxy"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"path/filepath"
)

const IngressClassFlagHelpStr = "Ingress class name"

var (
	ingressClass string
)

func main() {
	flag.StringVar(&ingressClass, "ingressClass", "default", IngressClassFlagHelpStr)
	flag.Parse()

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	homeDir := func() string {
		if h := os.Getenv("HOME"); h != "" {
			return h
		}
		return os.Getenv("USERPROFILE")
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		config, err = clientcmd.BuildConfigFromFlags("", filepath.Join(homeDir(), ".kube", "config"))
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to get kubernetes client")
	}
	log.Info().Msg("get kubernetes client success")

	c := controller.NewIngressController(client, ingressClass)
	group := &wgroup.Group{}
	group.Go(func() {
		c.Run(5, nil)
	})
	group.Go(func() {
		err := proxy.Run(client, ingressClass)
		if err != nil {
			return
		}
	})
	group.Wait()
}
