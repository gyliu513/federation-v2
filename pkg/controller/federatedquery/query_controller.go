/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package federatedquery

import (
	"fmt"
	"reflect"
	"time"

	"github.com/golang/glog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	fedv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	fedclientset "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset/versioned"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
)

const (
	minRetryDelay = 5 * time.Second
	maxRetryDelay = 300 * time.Second
	maxRetries    = 5
	numWorkers    = 2

	userAgent = "FederatedQuery"
)

type FederatedQueryController struct {
	// Client to federation api server
	client fedclientset.Interface
	// Informer Store for ServiceDNS objects
	federatedQueryObjectStore cache.Store
	// Informer controller for FederatedQuery objects
	federatedQueryObjectController cache.Controller

	queue workqueue.RateLimitingInterface

	minRetryDelay time.Duration
	maxRetryDelay time.Duration
}

func StartController(config *restclient.Config, stopChan <-chan struct{}, minimizeLatency bool) error {
	restclient.AddUserAgent(config, userAgent)
	fedClient := fedclientset.NewForConfigOrDie(config)

	controller, err := newFederatedQueryController(fedClient, minimizeLatency)
	if err != nil {
		return err
	}
	// TODO: consider making numWorkers configurable
	go controller.Run(stopChan, numWorkers)
	return nil
}

func newFederatedQueryController(client fedclientset.Interface, minimizeLatency bool) (*FederatedQueryController, error) {
	d := &FederatedQueryController{
		client: client,
	}

	// Start informer in federated API servers on FederatedQuery objects
	// TODO: Change this to shared informer
	d.federatedQueryObjectStore, d.federatedQueryObjectController = cache.NewInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (pkgruntime.Object, error) {
				return client.CoreV1alpha1().FederatedQueries(metav1.NamespaceAll).List(options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return client.CoreV1alpha1().FederatedQueries(metav1.NamespaceAll).Watch(options)
			},
		},
		&fedv1a1.FederatedQuery{},
		util.NoResyncPeriod,
		cache.ResourceEventHandlerFuncs{
			AddFunc: d.enqueueObject,
			UpdateFunc: func(old, cur interface{}) {
				oldObj, ok1 := old.(*fedv1a1.FederatedQuery)
				curObj, ok2 := cur.(*fedv1a1.FederatedQuery)
				if !ok1 || !ok2 {
					glog.Errorf("gyliu Received unknown objects: %v, %v", old, cur)
					return
				}
				if d.needsUpdate(oldObj, curObj) {
					glog.V(1).Infof("gyliu fedquery need update old %v cur %v", oldObj, curObj)
					d.enqueueObject(cur)
				}
			},
			DeleteFunc: d.enqueueObject,
		},
	)

	d.minRetryDelay = minRetryDelay
	d.maxRetryDelay = maxRetryDelay
	if minimizeLatency {
		d.minimizeLatency()
	}

	d.queue = workqueue.NewNamedRateLimitingQueue(workqueue.NewItemExponentialFailureRateLimiter(d.minRetryDelay, d.maxRetryDelay), userAgent)

	return d, nil
}

// minimizeLatency reduces delays and timeouts to make the controller more responsive (useful for testing).
func (d *FederatedQueryController) minimizeLatency() {
	d.minRetryDelay = 50 * time.Millisecond
	d.maxRetryDelay = 2 * time.Second
}

func (d *FederatedQueryController) Run(stopCh <-chan struct{}, workers int) {
	defer runtime.HandleCrash()
	defer d.queue.ShutDown()

	glog.Infof("gyliu Starting FederatedQueryObjectController")
	defer glog.Infof("gyliu Shutting down FederatedQueryObjectController")

	go d.federatedQueryObjectController.Run(stopCh)

	// wait for the caches to synchronize before starting the worker
	if !cache.WaitForCacheSync(stopCh, d.federatedQueryObjectController.HasSynced) {
		runtime.HandleError(fmt.Errorf("gyliu Timed out waiting for caches to sync"))
		return
	}

	glog.Infof("gyliu FederatedQueryObjectController synced and ready")

	for i := 0; i < workers; i++ {
		go wait.Until(d.worker, time.Second, stopCh)
	}

	<-stopCh
}

func (d *FederatedQueryController) needsUpdate(oldObject, newObject *fedv1a1.FederatedQuery) bool {
	glog.V(1).Infof("gyliu needsUpdate FederatedQueryController old spec %s new spec %s", oldObject.Spec, newObject.Spec)
	if !reflect.DeepEqual(oldObject.Spec, newObject.Spec) {
		return true
	}
	if !reflect.DeepEqual(oldObject.Status, newObject.Status) {
		return true
	}

	return false
}

// obj could be an *fedv1a1.FederatedQuery, or a DeletionFinalStateUnknown marker item.
func (d *FederatedQueryController) enqueueObject(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		glog.Errorf("gyliu Couldn't get key for object %#v: %v", obj, err)
		return
	}
	glog.V(1).Infof("gyliu enqueue %s", key)
	d.queue.Add(key)
}

func (d *FederatedQueryController) worker() {
	// processNextWorkItem will automatically wait until there's work available
	glog.V(1).Infof("gyliu start to processNextItem")
	for d.processNextItem() {
		// continue looping
	}
}

func (d *FederatedQueryController) processNextItem() bool {
	key, quit := d.queue.Get()
	if quit {
		return false
	}

	defer d.queue.Done(key)

	err := d.processItem(key.(string))

	if err == nil {
		// No error, tell the queue to stop tracking history
		d.queue.Forget(key)
	} else if d.queue.NumRequeues(key) < maxRetries {
		glog.Errorf("gyliu Error processing %s (will retry): %v", key, err)
		// requeue the item to work on later
		d.queue.AddRateLimited(key)
	} else {
		// err != nil and too many retries
		glog.Errorf("gyliu Error processing %s (giving up): %v", key, err)
		d.queue.Forget(key)
		runtime.HandleError(err)
	}

	return true
}

func (d *FederatedQueryController) processItem(key string) error {
	startTime := time.Now()
	glog.V(1).Infof("gyliu Start processing change to FederatedQueryController %s", key)
	defer func() {
		glog.V(1).Infof("gyliu Finished processing FederatedQueryController %q (%v)", key, time.Since(startTime))
	}()

	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}

	obj, exists, err := d.federatedQueryObjectStore.GetByKey(key)
	if err != nil {
		return fmt.Errorf("error fetching object with key %s from store: %v", key, err)
	}

	if !exists {
		//delete corresponding FederatedQuery object
		glog.V(1).Infof("gyliu delete federated query object FederatedQueryController %q (%v)s", key)
		return d.client.CoreV1alpha1().FederatedQueries(namespace).Delete(name, &metav1.DeleteOptions{})
	}

	queryObject, ok := obj.(*fedv1a1.FederatedQuery)
	if !ok {
		return fmt.Errorf("recieved event for unknown object %v", obj)
	}

	cachedClusters := queryObject.Spec.ClusterNames

	glog.V(1).Infof("gyliu fedQueryObject in cache %v cachedClusters %s", queryObject, cachedClusters)

	fedQueryObject, err := d.client.CoreV1alpha1().FederatedQueries(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	curClusters := fedQueryObject.Spec.ClusterNames
	glog.V(1).Infof("gyliu fedQueryObject %v curClusters %s", fedQueryObject, curClusters)

	// Update only if the new endpoints are not equal to the existing ones.
	if !reflect.DeepEqual(fedQueryObject.Spec.ClusterNames, cachedClusters) {
		glog.V(1).Infof("gyliu fedQueryObject.Spec.ClusterNames %s new cachedClusters %s", fedQueryObject.Spec.ClusterNames, cachedClusters)
		fedQueryObject.Spec.ClusterNames = cachedClusters
		_, err = d.client.CoreV1alpha1().FederatedQueries(namespace).Update(fedQueryObject)
	}

	return err
}
