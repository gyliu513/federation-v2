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

// FederatedQueriesGetter has a method to return a FederatedQueryInterface.
// A group's client should implement this interface.
type FederatedQueriesGetter interface {
	FederatedQueries(namespace string) FederatedQueryInterface
}

// FederatedQueryInterface has methods to work with FederatedQuery resources.
type FederatedQueryInterface interface {
	Create(*v1alpha1.FederatedQuery) (*v1alpha1.FederatedQuery, error)
	Update(*v1alpha1.FederatedQuery) (*v1alpha1.FederatedQuery, error)
	UpdateStatus(*v1alpha1.FederatedQuery) (*v1alpha1.FederatedQuery, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.FederatedQuery, error)
	List(opts v1.ListOptions) (*v1alpha1.FederatedQueryList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.FederatedQuery, err error)
	FederatedQueryExpansion
}

// federatedQueries implements FederatedQueryInterface
type federatedQueries struct {
	client rest.Interface
	ns     string
}

// newFederatedQueries returns a FederatedQueries
func newFederatedQueries(c *CoreV1alpha1Client, namespace string) *federatedQueries {
	return &federatedQueries{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the federatedQuery, and returns the corresponding federatedQuery object, and an error if there is any.
func (c *federatedQueries) Get(name string, options v1.GetOptions) (result *v1alpha1.FederatedQuery, err error) {
	result = &v1alpha1.FederatedQuery{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("federatedqueries").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of FederatedQueries that match those selectors.
func (c *federatedQueries) List(opts v1.ListOptions) (result *v1alpha1.FederatedQueryList, err error) {
	result = &v1alpha1.FederatedQueryList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("federatedqueries").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested federatedQueries.
func (c *federatedQueries) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("federatedqueries").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a federatedQuery and creates it.  Returns the server's representation of the federatedQuery, and an error, if there is any.
func (c *federatedQueries) Create(federatedQuery *v1alpha1.FederatedQuery) (result *v1alpha1.FederatedQuery, err error) {
	result = &v1alpha1.FederatedQuery{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("federatedqueries").
		Body(federatedQuery).
		Do().
		Into(result)
	return
}

// Update takes the representation of a federatedQuery and updates it. Returns the server's representation of the federatedQuery, and an error, if there is any.
func (c *federatedQueries) Update(federatedQuery *v1alpha1.FederatedQuery) (result *v1alpha1.FederatedQuery, err error) {
	result = &v1alpha1.FederatedQuery{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("federatedqueries").
		Name(federatedQuery.Name).
		Body(federatedQuery).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *federatedQueries) UpdateStatus(federatedQuery *v1alpha1.FederatedQuery) (result *v1alpha1.FederatedQuery, err error) {
	result = &v1alpha1.FederatedQuery{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("federatedqueries").
		Name(federatedQuery.Name).
		SubResource("status").
		Body(federatedQuery).
		Do().
		Into(result)
	return
}

// Delete takes name of the federatedQuery and deletes it. Returns an error if one occurs.
func (c *federatedQueries) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("federatedqueries").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *federatedQueries) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("federatedqueries").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched federatedQuery.
func (c *federatedQueries) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.FederatedQuery, err error) {
	result = &v1alpha1.FederatedQuery{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("federatedqueries").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}