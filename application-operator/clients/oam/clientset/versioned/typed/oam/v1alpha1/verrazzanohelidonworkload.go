// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	"time"

	v1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	scheme "github.com/verrazzano/verrazzano/application-operator/clients/oam/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// VerrazzanoHelidonWorkloadsGetter has a method to return a VerrazzanoHelidonWorkloadInterface.
// A group's client should implement this interface.
type VerrazzanoHelidonWorkloadsGetter interface {
	VerrazzanoHelidonWorkloads(namespace string) VerrazzanoHelidonWorkloadInterface
}

// VerrazzanoHelidonWorkloadInterface has methods to work with VerrazzanoHelidonWorkload resources.
type VerrazzanoHelidonWorkloadInterface interface {
	Create(ctx context.Context, verrazzanoHelidonWorkload *v1alpha1.VerrazzanoHelidonWorkload, opts v1.CreateOptions) (*v1alpha1.VerrazzanoHelidonWorkload, error)
	Update(ctx context.Context, verrazzanoHelidonWorkload *v1alpha1.VerrazzanoHelidonWorkload, opts v1.UpdateOptions) (*v1alpha1.VerrazzanoHelidonWorkload, error)
	UpdateStatus(ctx context.Context, verrazzanoHelidonWorkload *v1alpha1.VerrazzanoHelidonWorkload, opts v1.UpdateOptions) (*v1alpha1.VerrazzanoHelidonWorkload, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha1.VerrazzanoHelidonWorkload, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.VerrazzanoHelidonWorkloadList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.VerrazzanoHelidonWorkload, err error)
	VerrazzanoHelidonWorkloadExpansion
}

// verrazzanoHelidonWorkloads implements VerrazzanoHelidonWorkloadInterface
type verrazzanoHelidonWorkloads struct {
	client rest.Interface
	ns     string
}

// newVerrazzanoHelidonWorkloads returns a VerrazzanoHelidonWorkloads
func newVerrazzanoHelidonWorkloads(c *OamV1alpha1Client, namespace string) *verrazzanoHelidonWorkloads {
	return &verrazzanoHelidonWorkloads{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the verrazzanoHelidonWorkload, and returns the corresponding verrazzanoHelidonWorkload object, and an error if there is any.
func (c *verrazzanoHelidonWorkloads) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.VerrazzanoHelidonWorkload, err error) {
	result = &v1alpha1.VerrazzanoHelidonWorkload{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("verrazzanohelidonworkloads").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of VerrazzanoHelidonWorkloads that match those selectors.
func (c *verrazzanoHelidonWorkloads) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.VerrazzanoHelidonWorkloadList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.VerrazzanoHelidonWorkloadList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("verrazzanohelidonworkloads").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested verrazzanoHelidonWorkloads.
func (c *verrazzanoHelidonWorkloads) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("verrazzanohelidonworkloads").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a verrazzanoHelidonWorkload and creates it.  Returns the server's representation of the verrazzanoHelidonWorkload, and an error, if there is any.
func (c *verrazzanoHelidonWorkloads) Create(ctx context.Context, verrazzanoHelidonWorkload *v1alpha1.VerrazzanoHelidonWorkload, opts v1.CreateOptions) (result *v1alpha1.VerrazzanoHelidonWorkload, err error) {
	result = &v1alpha1.VerrazzanoHelidonWorkload{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("verrazzanohelidonworkloads").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(verrazzanoHelidonWorkload).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a verrazzanoHelidonWorkload and updates it. Returns the server's representation of the verrazzanoHelidonWorkload, and an error, if there is any.
func (c *verrazzanoHelidonWorkloads) Update(ctx context.Context, verrazzanoHelidonWorkload *v1alpha1.VerrazzanoHelidonWorkload, opts v1.UpdateOptions) (result *v1alpha1.VerrazzanoHelidonWorkload, err error) {
	result = &v1alpha1.VerrazzanoHelidonWorkload{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("verrazzanohelidonworkloads").
		Name(verrazzanoHelidonWorkload.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(verrazzanoHelidonWorkload).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *verrazzanoHelidonWorkloads) UpdateStatus(ctx context.Context, verrazzanoHelidonWorkload *v1alpha1.VerrazzanoHelidonWorkload, opts v1.UpdateOptions) (result *v1alpha1.VerrazzanoHelidonWorkload, err error) {
	result = &v1alpha1.VerrazzanoHelidonWorkload{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("verrazzanohelidonworkloads").
		Name(verrazzanoHelidonWorkload.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(verrazzanoHelidonWorkload).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the verrazzanoHelidonWorkload and deletes it. Returns an error if one occurs.
func (c *verrazzanoHelidonWorkloads) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("verrazzanohelidonworkloads").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *verrazzanoHelidonWorkloads) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("verrazzanohelidonworkloads").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched verrazzanoHelidonWorkload.
func (c *verrazzanoHelidonWorkloads) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.VerrazzanoHelidonWorkload, err error) {
	result = &v1alpha1.VerrazzanoHelidonWorkload{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("verrazzanohelidonworkloads").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
