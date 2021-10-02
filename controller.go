package main

import (
	"fmt"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	"time"
)

type Controller struct {
	indexer  cache.Indexer
	queue    workqueue.RateLimitingInterface
	informer cache.Controller
}

func NewController(queue workqueue.RateLimitingInterface, indexer cache.Indexer, informer cache.Controller) *Controller {
	return &Controller{
		indexer:  indexer,
		queue:    queue,
		informer: informer,
	}
}

func (c *Controller) processNextItem() bool {
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(key)

	err := c.work(key.(string))	// Trigger business logic

	c.handleErr(err, key)
	return true
}

func (c *Controller) work(key string) error {
	obj, exists, err := c.indexer.GetByKey(key)
	if err != nil {
		klog.Errorf("fetching object with key %s failed with %v", key, err)
		return err
	}

	if !exists {
		fmt.Printf("Deployment %s does not exist anymore\n", key)
	} else {
		fmt.Printf("Sync/Add/Update for Deployment %s\n", obj.(*unstructured.Unstructured).GetName())
	}
	return nil
}

func (c *Controller) handleErr(err error, key interface{}) {
	if err == nil {
		c.queue.Forget(key)
		return
	}
	if c.queue.NumRequeues(key) < 5 {
		klog.Infof("error syncing deployment %v: %v", key, err)
		c.queue.AddRateLimited(key)
		return
	}
	c.queue.Forget(key)
	runtime.HandleError(err)
	klog.Infof("dropping deployment %q out of the queue: %v", key, err)
}

func (c *Controller) Run(workers int, stopCh <-chan struct{}) {
	defer runtime.HandleCrash()

	defer c.queue.ShutDown()
	klog.Info("starting Deployment controller")

	go c.informer.Run(stopCh)

	if !cache.WaitForCacheSync(stopCh, c.informer.HasSynced) {
		runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
		return
	}

	for i := 0; i < workers; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
	klog.Infof("stopping Deployment controller")
}

func (c *Controller) runWorker() {
	for c.processNextItem() {
	}
}