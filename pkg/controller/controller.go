package controller

import (
	"fmt"
	"github.com/rs/zerolog/log"
	networkingV1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

type IngressController struct {
	clientSet *kubernetes.Clientset
	queue     workqueue.RateLimitingInterface
	indexer   cache.Indexer
	informer  cache.Controller
}

func NewIngressController(client *kubernetes.Clientset) *IngressController {
	ListWatcher := cache.NewListWatchFromClient(client.NetworkingV1().RESTClient(), "ingresses", "", fields.Everything())
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	indexer, informer := cache.NewIndexerInformer(ListWatcher, &networkingV1.Ingress{}, 0, cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				log.Info().Str("object", key).Msg("watch.add.Ingress")
				queue.Add(&Event{Type: Add, Obj: key})
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(newObj)
			if err == nil {
				log.Info().Str("object", key).Msg("watch.update.Ingress")
				queue.Add(&Event{Type: Update, Obj: key})
			}
		},
		DeleteFunc: func(obj interface{}) {
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				log.Info().Str("object", key).Msg("watch.delete.Ingress")
				queue.Add(&Event{Type: Delete, Obj: key, Tombstone: obj})
			}
		},
	}, cache.Indexers{})

	return &IngressController{
		clientSet: client,
		queue:     queue,
		indexer:   indexer,
		informer:  informer,
	}
}

func (c *IngressController) processNextItem() bool {
	item, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(item)

	event, ok := item.(*Event)
	if !ok {
		log.Error().Msgf("Expected *Event but got %T", item)
		c.queue.Forget(item)
		return true
	}

	key, ok := event.Obj.(string)
	if !ok {
		log.Error().Msgf("Expected string but got %T for object", event.Obj)
		c.queue.Forget(item)
		return true
	}

	log.Info().Str("eventType", event.Type.String()).Str("key", key).Msg("Processing event")

	err := c.syncToStdout(key)
	c.handleErr(err, item)

	return true
}

// handleErr checks if an error happened and makes sure we will retry later.
func (c *IngressController) handleErr(err error, key any) {
	if err == nil {
		// Forget about the #AddRateLimited history of the key on every successful synchronization.
		// This ensures that future processing of updates for this key is not delayed because of
		// an outdated error history.
		c.queue.Forget(key)
		return
	}

	// This controller retries 5 times if something goes wrong. After that, it stops trying.
	if c.queue.NumRequeues(key) < 5 {
		klog.Infof("Error syncing pod %v: %v", key, err)

		// Re-enqueue the key rate limited. Based on the rate limiter on the
		// queue and the re-enqueue history, the key will be processed later again.
		c.queue.AddRateLimited(key)
		return
	}

	c.queue.Forget(key)
	// Report to an external entity that, even after several retries, we could not successfully process this key
	runtime.HandleError(err)
	klog.Infof("Dropping ingress %q out of the queue: %v", key, err)
}

// syncToStdout is the business logic of the controller. In this controller it simply prints
// information about the ingress to stdout. In case an error happened, it has to simply return the error.
// The retry logic should not be part of the business logic.
func (c *IngressController) syncToStdout(key string) error {
	obj, exists, err := c.indexer.GetByKey(key)
	if err != nil {
		klog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}

	if !exists {
		fmt.Printf("Ingress %s does not exist anymore\n", key)
	} else {
		fmt.Printf("Sync/Add/Update for Ingress %s\n", obj.(*networkingV1.Ingress).GetName())
	}
	return nil
}

func (c *IngressController) RunWorker() {
	for c.processNextItem() {
	}
}

func (c *IngressController) Run(thread int, stopCh chan struct{}) {
	<-stopCh
}
