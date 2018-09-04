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

// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	v1alpha1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	"github.com/kubernetes-sigs/federation-v2/pkg/client/clientset/versioned/scheme"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer"
	rest "k8s.io/client-go/rest"
)

type CoreV1alpha1Interface interface {
	RESTClient() rest.Interface
	FederatedClustersGetter
	FederatedConfigMapsGetter
	FederatedConfigMapOverridesGetter
	FederatedConfigMapPlacementsGetter
	FederatedDeploymentsGetter
	FederatedDeploymentOverridesGetter
	FederatedDeploymentPlacementsGetter
	FederatedIngressesGetter
	FederatedIngressPlacementsGetter
	FederatedJobsGetter
	FederatedJobOverridesGetter
	FederatedJobPlacementsGetter
	FederatedNamespacePlacementsGetter
	FederatedReplicaSetsGetter
	FederatedReplicaSetOverridesGetter
	FederatedReplicaSetPlacementsGetter
	FederatedSecretsGetter
	FederatedSecretOverridesGetter
	FederatedSecretPlacementsGetter
	FederatedServicesGetter
	FederatedServiceAccountsGetter
	FederatedServicePlacementsGetter
	FederatedTypeConfigsGetter
	PropagatedVersionsGetter
}

// CoreV1alpha1Client is used to interact with features provided by the core.federation.k8s.io group.
type CoreV1alpha1Client struct {
	restClient rest.Interface
}

func (c *CoreV1alpha1Client) FederatedClusters(namespace string) FederatedClusterInterface {
	return newFederatedClusters(c, namespace)
}

func (c *CoreV1alpha1Client) FederatedConfigMaps(namespace string) FederatedConfigMapInterface {
	return newFederatedConfigMaps(c, namespace)
}

func (c *CoreV1alpha1Client) FederatedConfigMapOverrides(namespace string) FederatedConfigMapOverrideInterface {
	return newFederatedConfigMapOverrides(c, namespace)
}

func (c *CoreV1alpha1Client) FederatedConfigMapPlacements(namespace string) FederatedConfigMapPlacementInterface {
	return newFederatedConfigMapPlacements(c, namespace)
}

func (c *CoreV1alpha1Client) FederatedDeployments(namespace string) FederatedDeploymentInterface {
	return newFederatedDeployments(c, namespace)
}

func (c *CoreV1alpha1Client) FederatedDeploymentOverrides(namespace string) FederatedDeploymentOverrideInterface {
	return newFederatedDeploymentOverrides(c, namespace)
}

func (c *CoreV1alpha1Client) FederatedDeploymentPlacements(namespace string) FederatedDeploymentPlacementInterface {
	return newFederatedDeploymentPlacements(c, namespace)
}

func (c *CoreV1alpha1Client) FederatedIngresses(namespace string) FederatedIngressInterface {
	return newFederatedIngresses(c, namespace)
}

func (c *CoreV1alpha1Client) FederatedIngressPlacements(namespace string) FederatedIngressPlacementInterface {
	return newFederatedIngressPlacements(c, namespace)
}

func (c *CoreV1alpha1Client) FederatedJobs(namespace string) FederatedJobInterface {
	return newFederatedJobs(c, namespace)
}

func (c *CoreV1alpha1Client) FederatedJobOverrides(namespace string) FederatedJobOverrideInterface {
	return newFederatedJobOverrides(c, namespace)
}

func (c *CoreV1alpha1Client) FederatedJobPlacements(namespace string) FederatedJobPlacementInterface {
	return newFederatedJobPlacements(c, namespace)
}

func (c *CoreV1alpha1Client) FederatedNamespacePlacements(namespace string) FederatedNamespacePlacementInterface {
	return newFederatedNamespacePlacements(c, namespace)
}

func (c *CoreV1alpha1Client) FederatedReplicaSets(namespace string) FederatedReplicaSetInterface {
	return newFederatedReplicaSets(c, namespace)
}

func (c *CoreV1alpha1Client) FederatedReplicaSetOverrides(namespace string) FederatedReplicaSetOverrideInterface {
	return newFederatedReplicaSetOverrides(c, namespace)
}

func (c *CoreV1alpha1Client) FederatedReplicaSetPlacements(namespace string) FederatedReplicaSetPlacementInterface {
	return newFederatedReplicaSetPlacements(c, namespace)
}

func (c *CoreV1alpha1Client) FederatedSecrets(namespace string) FederatedSecretInterface {
	return newFederatedSecrets(c, namespace)
}

func (c *CoreV1alpha1Client) FederatedSecretOverrides(namespace string) FederatedSecretOverrideInterface {
	return newFederatedSecretOverrides(c, namespace)
}

func (c *CoreV1alpha1Client) FederatedSecretPlacements(namespace string) FederatedSecretPlacementInterface {
	return newFederatedSecretPlacements(c, namespace)
}

func (c *CoreV1alpha1Client) FederatedServices(namespace string) FederatedServiceInterface {
	return newFederatedServices(c, namespace)
}

func (c *CoreV1alpha1Client) FederatedServiceAccounts(namespace string) FederatedServiceAccountInterface {
	return newFederatedServiceAccounts(c, namespace)
}

func (c *CoreV1alpha1Client) FederatedServicePlacements(namespace string) FederatedServicePlacementInterface {
	return newFederatedServicePlacements(c, namespace)
}

func (c *CoreV1alpha1Client) FederatedTypeConfigs(namespace string) FederatedTypeConfigInterface {
	return newFederatedTypeConfigs(c, namespace)
}

func (c *CoreV1alpha1Client) PropagatedVersions(namespace string) PropagatedVersionInterface {
	return newPropagatedVersions(c, namespace)
}

// NewForConfig creates a new CoreV1alpha1Client for the given config.
func NewForConfig(c *rest.Config) (*CoreV1alpha1Client, error) {
	config := *c
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}
	return &CoreV1alpha1Client{client}, nil
}

// NewForConfigOrDie creates a new CoreV1alpha1Client for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *rest.Config) *CoreV1alpha1Client {
	client, err := NewForConfig(c)
	if err != nil {
		panic(err)
	}
	return client
}

// New creates a new CoreV1alpha1Client for the given RESTClient.
func New(c rest.Interface) *CoreV1alpha1Client {
	return &CoreV1alpha1Client{c}
}

func setConfigDefaults(config *rest.Config) error {
	gv := v1alpha1.SchemeGroupVersion
	config.GroupVersion = &gv
	config.APIPath = "/apis"
	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: scheme.Codecs}

	if config.UserAgent == "" {
		config.UserAgent = rest.DefaultKubernetesUserAgent()
	}

	return nil
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *CoreV1alpha1Client) RESTClient() rest.Interface {
	if c == nil {
		return nil
	}
	return c.restClient
}
