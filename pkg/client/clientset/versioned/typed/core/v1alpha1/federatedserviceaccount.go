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
	scheme "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// FederatedServiceAccountsGetter has a method to return a FederatedServiceAccountInterface.
// A group's client should implement this interface.
type FederatedServiceAccountsGetter interface {
	FederatedServiceAccounts(namespace string) FederatedServiceAccountInterface
}

// FederatedServiceAccountInterface has methods to work with FederatedServiceAccount resources.
type FederatedServiceAccountInterface interface {
	Create(*v1alpha1.FederatedServiceAccount) (*v1alpha1.FederatedServiceAccount, error)
	Update(*v1alpha1.FederatedServiceAccount) (*v1alpha1.FederatedServiceAccount, error)
	UpdateStatus(*v1alpha1.FederatedServiceAccount) (*v1alpha1.FederatedServiceAccount, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.FederatedServiceAccount, error)
	List(opts v1.ListOptions) (*v1alpha1.FederatedServiceAccountList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.FederatedServiceAccount, err error)
	FederatedServiceAccountExpansion
}

// federatedServiceAccounts implements FederatedServiceAccountInterface
type federatedServiceAccounts struct {
	client rest.Interface
	ns     string
}

// newFederatedServiceAccounts returns a FederatedServiceAccounts
func newFederatedServiceAccounts(c *CoreV1alpha1Client, namespace string) *federatedServiceAccounts {
	return &federatedServiceAccounts{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the federatedServiceAccount, and returns the corresponding federatedServiceAccount object, and an error if there is any.
func (c *federatedServiceAccounts) Get(name string, options v1.GetOptions) (result *v1alpha1.FederatedServiceAccount, err error) {
	result = &v1alpha1.FederatedServiceAccount{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("federatedserviceaccounts").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of FederatedServiceAccounts that match those selectors.
func (c *federatedServiceAccounts) List(opts v1.ListOptions) (result *v1alpha1.FederatedServiceAccountList, err error) {
	result = &v1alpha1.FederatedServiceAccountList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("federatedserviceaccounts").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested federatedServiceAccounts.
func (c *federatedServiceAccounts) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("federatedserviceaccounts").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a federatedServiceAccount and creates it.  Returns the server's representation of the federatedServiceAccount, and an error, if there is any.
func (c *federatedServiceAccounts) Create(federatedServiceAccount *v1alpha1.FederatedServiceAccount) (result *v1alpha1.FederatedServiceAccount, err error) {
	result = &v1alpha1.FederatedServiceAccount{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("federatedserviceaccounts").
		Body(federatedServiceAccount).
		Do().
		Into(result)
	return
}

// Update takes the representation of a federatedServiceAccount and updates it. Returns the server's representation of the federatedServiceAccount, and an error, if there is any.
func (c *federatedServiceAccounts) Update(federatedServiceAccount *v1alpha1.FederatedServiceAccount) (result *v1alpha1.FederatedServiceAccount, err error) {
	result = &v1alpha1.FederatedServiceAccount{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("federatedserviceaccounts").
		Name(federatedServiceAccount.Name).
		Body(federatedServiceAccount).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *federatedServiceAccounts) UpdateStatus(federatedServiceAccount *v1alpha1.FederatedServiceAccount) (result *v1alpha1.FederatedServiceAccount, err error) {
	result = &v1alpha1.FederatedServiceAccount{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("federatedserviceaccounts").
		Name(federatedServiceAccount.Name).
		SubResource("status").
		Body(federatedServiceAccount).
		Do().
		Into(result)
	return
}

// Delete takes name of the federatedServiceAccount and deletes it. Returns an error if one occurs.
func (c *federatedServiceAccounts) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("federatedserviceaccounts").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *federatedServiceAccounts) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("federatedserviceaccounts").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched federatedServiceAccount.
func (c *federatedServiceAccounts) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.FederatedServiceAccount, err error) {
	result = &v1alpha1.FederatedServiceAccount{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("federatedserviceaccounts").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
