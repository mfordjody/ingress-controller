package controller

import (
	"k8s.io/client-go/kubernetes"
)

// https://github.com/kubernetes/client-go/blob/master/examples/workqueue/main.go
type IngressController struct {
	clientSet *kubernetes.Clientset
}

func NewIngressController(client *kubernetes.Clientset) *IngressController {
	return &IngressController{
		clientSet: client,
	}
}

func (c *IngressController) Run(thread int, stopCh chan struct{}) {
	<-stopCh
}
