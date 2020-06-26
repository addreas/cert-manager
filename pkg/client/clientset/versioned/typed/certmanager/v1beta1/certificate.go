/*
Copyright 2020 The Jetstack cert-manager contributors.

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

package v1beta1

import (
	"context"
	"time"

	v1beta1 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1beta1"
	scheme "github.com/jetstack/cert-manager/pkg/client/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// CertificatesGetter has a method to return a CertificateInterface.
// A group's client should implement this interface.
type CertificatesGetter interface {
	Certificates(namespace string) CertificateInterface
}

// CertificateInterface has methods to work with Certificate resources.
type CertificateInterface interface {
	Create(ctx context.Context, certificate *v1beta1.Certificate, opts v1.CreateOptions) (*v1beta1.Certificate, error)
	Update(ctx context.Context, certificate *v1beta1.Certificate, opts v1.UpdateOptions) (*v1beta1.Certificate, error)
	UpdateStatus(ctx context.Context, certificate *v1beta1.Certificate, opts v1.UpdateOptions) (*v1beta1.Certificate, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1beta1.Certificate, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1beta1.CertificateList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1beta1.Certificate, err error)
	CertificateExpansion
}

// certificates implements CertificateInterface
type certificates struct {
	client rest.Interface
	ns     string
}

// newCertificates returns a Certificates
func newCertificates(c *CertmanagerV1beta1Client, namespace string) *certificates {
	return &certificates{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the certificate, and returns the corresponding certificate object, and an error if there is any.
func (c *certificates) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1beta1.Certificate, err error) {
	result = &v1beta1.Certificate{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("certificates").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of Certificates that match those selectors.
func (c *certificates) List(ctx context.Context, opts v1.ListOptions) (result *v1beta1.CertificateList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1beta1.CertificateList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("certificates").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested certificates.
func (c *certificates) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("certificates").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a certificate and creates it.  Returns the server's representation of the certificate, and an error, if there is any.
func (c *certificates) Create(ctx context.Context, certificate *v1beta1.Certificate, opts v1.CreateOptions) (result *v1beta1.Certificate, err error) {
	result = &v1beta1.Certificate{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("certificates").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(certificate).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a certificate and updates it. Returns the server's representation of the certificate, and an error, if there is any.
func (c *certificates) Update(ctx context.Context, certificate *v1beta1.Certificate, opts v1.UpdateOptions) (result *v1beta1.Certificate, err error) {
	result = &v1beta1.Certificate{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("certificates").
		Name(certificate.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(certificate).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *certificates) UpdateStatus(ctx context.Context, certificate *v1beta1.Certificate, opts v1.UpdateOptions) (result *v1beta1.Certificate, err error) {
	result = &v1beta1.Certificate{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("certificates").
		Name(certificate.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(certificate).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the certificate and deletes it. Returns an error if one occurs.
func (c *certificates) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("certificates").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *certificates) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("certificates").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched certificate.
func (c *certificates) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1beta1.Certificate, err error) {
	result = &v1beta1.Certificate{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("certificates").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
