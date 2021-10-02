package main

import (
	"context"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	"os"
	"os/signal"
	"time"
)

func main() {
	var clusterConfig *rest.Config
	var err error
	kubeconfig := os.Getenv("KUBECONFIG")

	if kubeconfig != "" {
		clusterConfig, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else {
		clusterConfig, err = rest.InClusterConfig()
	}
	if err != nil {
		klog.Fatalln(err)
	}

	client, err := dynamic.NewForConfig(clusterConfig)
	if err != nil {
		klog.Fatalln(err)
	}

	gvr := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
	factory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(client, time.Minute, v1.NamespaceAll, nil)
	informer := factory.ForResource(gvr).Informer()

	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultItemBasedRateLimiter())

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(key)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(newObj)
			if err == nil {
				queue.Add(key)
			}
		},
		DeleteFunc: func(obj interface{}) {
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(key)
			}
		},
	})

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	controller := NewController(queue, informer.GetIndexer(), informer)
	go controller.Run(2, ctx.Done())

	<-ctx.Done()
}
