/*
Copyright 2016 The Kubernetes Authors.

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
	"strings"
	"sync"
	"time"

	fedv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	fedclientset "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset/versioned"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	kubeclientset "k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	crclientset "k8s.io/cluster-registry/pkg/client/clientset/versioned"

	"github.com/golang/glog"
)

const (
	Pod = "Pod"
)

type FederatedQueryController struct {
	fedClient  fedclientset.Interface
	kubeClient kubeclientset.Interface
	crClient   crclientset.Interface

	// clusterMonitorPeriod is the period for updating status of cluster
	clusterMonitorPeriod time.Duration

	mu              sync.RWMutex
	knownClusterSet sets.String
	// clusterClusterStatusMap is a mapping of clusterName and cluster status of last sampling
	clusterClusterStatusMap map[string]fedv1a1.FederatedClusterStatus

	clusterController cache.Controller

	// informer for service object from members of federation.
	podResourceInformer util.FederatedInformer

	// fedNamespace is the name of the namespace containing
	// FederatedCluster resources and their associated secrets
	fedNamespace string
}

// StartClusterController starts a new cluster controller
func StartFederatedQueryController(config *restclient.Config, fedNamespace string, stopChan <-chan struct{}, clusterMonitorPeriod time.Duration) {
	restclient.AddUserAgent(config, "fedrated-query-controller")
	fedClient := fedclientset.NewForConfigOrDie(config)
	kubeClient := kubeclientset.NewForConfigOrDie(config)
	crClient := crclientset.NewForConfigOrDie(config)

	controller := newFederatedQueryController(fedClient, kubeClient, crClient, fedNamespace, clusterMonitorPeriod)
	glog.Infof("gyliu fedquery Starting federated query controller")
	controller.Run(stopChan)
}

// newClusterController returns a new cluster controller
func newFederatedQueryController(fedClient fedclientset.Interface, kubeClient kubeclientset.Interface, crClient crclientset.Interface, fedNamespace string, clusterMonitorPeriod time.Duration) *FederatedQueryController {
	cc := &FederatedQueryController{
		knownClusterSet:         make(sets.String),
		fedClient:               fedClient,
		kubeClient:              kubeClient,
		crClient:                crClient,
		clusterMonitorPeriod:    clusterMonitorPeriod,
		clusterClusterStatusMap: make(map[string]fedv1a1.FederatedClusterStatus),
		fedNamespace:            fedNamespace,
	}
	_, cc.clusterController = cache.NewInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return cc.fedClient.CoreV1alpha1().FederatedClusters(fedNamespace).List(options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return cc.fedClient.CoreV1alpha1().FederatedClusters(fedNamespace).Watch(options)
			},
		},
		&fedv1a1.FederatedCluster{},
		util.NoResyncPeriod,
		cache.ResourceEventHandlerFuncs{
			DeleteFunc: cc.delFromClusterSet,
			AddFunc:    cc.addToClusterSet,
		},
	)

	// Federated serviceInformer for the service resource in members of federation.
	cc.podResourceInformer = util.NewFederatedInformer(
		fedClient,
		kubeClient,
		crClient,
		fedNamespace,
		&metav1.APIResource{
			Name:       strings.ToLower(Pod) + "s",
			Group:      corev1.SchemeGroupVersion.Group,
			Version:    corev1.SchemeGroupVersion.Version,
			Kind:       Pod,
			Namespaced: true},
		func(pkgruntime.Object) {},
		&util.ClusterLifecycleHandlerFuncs{},
	)

	return cc
}

// delFromClusterSet delete a cluster from clusterSet and
func (cc *FederatedQueryController) delFromClusterSet(obj interface{}) {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	cluster := obj.(*fedv1a1.FederatedCluster)
	cc.delFromClusterSetByName(cluster.Name)
}

// delFromClusterSetByName delete a cluster from clusterSet by name and
// Caller must make sure that they hold the mutex
func (cc *FederatedQueryController) delFromClusterSetByName(clusterName string) {
	glog.V(1).Infof("gyliu fedquery FederatedQueryController observed a cluster deletion: %v", clusterName)
	cc.knownClusterSet.Delete(clusterName)
	delete(cc.clusterClusterStatusMap, clusterName)
}

func (cc *FederatedQueryController) addToClusterSet(obj interface{}) {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	cluster := obj.(*fedv1a1.FederatedCluster)
	cc.addToClusterSetWithoutLock(cluster)
}

// addToClusterSetWithoutLock inserts the new cluster to clusterSet and create
// a corresponding restclient to map clusterKubeClientMap if the cluster is not
// known. Caller must make sure that they hold the mutex.
func (cc *FederatedQueryController) addToClusterSetWithoutLock(cluster *fedv1a1.FederatedCluster) {
	if cc.knownClusterSet.Has(cluster.Name) {
		return
	}
	glog.V(1).Infof("gyliu fedquery FederatedQueryController observed a new cluster: %v", cluster.Name)
	cc.knownClusterSet.Insert(cluster.Name)
}

// Run begins watching and syncing.
func (cc *FederatedQueryController) Run(stopChan <-chan struct{}) {
	defer utilruntime.HandleCrash()
	go cc.clusterController.Run(stopChan)
	cc.podResourceInformer.Start()
	// monitor cluster status periodically, in phase 1 we just get the health state from "/healthz"
	go wait.Until(func() {
		if err := cc.updateClusterStatus(); err != nil {
			glog.Errorf("Error monitoring cluster status: %v", err)
		}
	}, cc.clusterMonitorPeriod, stopChan)
}

// updateClusterStatus checks cluster status and get the metrics from cluster's restapi
func (cc *FederatedQueryController) updateClusterStatus() error {
	nodes, err := cc.kubeClient.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	glog.V(1).Infof("gyliu fedquery FederatedQueryController observed nodes: %#v.", nodes)

	clusterNames, err := cc.clusterNames()

	for _, clusterName := range clusterNames {

		client, err := cc.podResourceInformer.GetClientForCluster(clusterName)
		if err != nil {
			glog.V(1).Infof("gyliu fedquery failed to get client for cluster %#v", clusterName)
			return err
		}
		glog.V(1).Infof("gyliu fedquery Get client for cluster %#v client %#v ", clusterName, client)

		unstructuredPodList, err := client.Resources("kube-system").List(metav1.ListOptions{})
		if err != nil || unstructuredPodList == nil {
			if err != nil {
				glog.V(1).Infof("gyliu fedquery  get error when getting pods %#v ", err)
			}
			if unstructuredPodList == nil {
				glog.V(1).Infof("gyliu fedquery get emdpty pods for cluster %#v ", clusterName)
			}
			return err
		}
		glog.V(1).Infof("gyliu fedquery FederatedQueryController observed pods: %#v for cluster %#v.", unstructuredPodList, clusterName)
	}
	return nil
}

// The list of clusters could come from any target informer
func (cc *FederatedQueryController) clusterNames() ([]string, error) {
	clusters, err := cc.podResourceInformer.GetReadyClusters()
	if err != nil {
		return nil, err
	}
	clusterNames := []string{}
	for _, cluster := range clusters {
		clusterNames = append(clusterNames, cluster.Name)
	}

	return clusterNames, nil
}
