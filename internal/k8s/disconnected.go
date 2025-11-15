package k8s

import (
	"context"
	"fmt"

	openapi_v2 "github.com/google/gnostic-models/openapiv2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/openapi"
	"k8s.io/client-go/rest"
)

var errNotConnected = fmt.Errorf("not connected to cluster")

// disconnectedDiscovery implements discovery.DiscoveryInterface
// It returns appropriate errors for all operations when the client is disconnected.
type disconnectedDiscovery struct{}

func (d *disconnectedDiscovery) RESTClient() rest.Interface {
	return nil
}

func (d *disconnectedDiscovery) ServerGroups() (*metav1.APIGroupList, error) {
	return nil, errNotConnected
}

func (d *disconnectedDiscovery) ServerResourcesForGroupVersion(groupVersion string) (*metav1.APIResourceList, error) {
	return nil, errNotConnected
}

func (d *disconnectedDiscovery) ServerGroupsAndResources() ([]*metav1.APIGroup, []*metav1.APIResourceList, error) {
	return nil, nil, errNotConnected
}

func (d *disconnectedDiscovery) ServerPreferredResources() ([]*metav1.APIResourceList, error) {
	return nil, errNotConnected
}

func (d *disconnectedDiscovery) ServerPreferredNamespacedResources() ([]*metav1.APIResourceList, error) {
	return nil, errNotConnected
}

func (d *disconnectedDiscovery) ServerVersion() (*version.Info, error) {
	return nil, errNotConnected
}

func (d *disconnectedDiscovery) OpenAPISchema() (*openapi_v2.Document, error) {
	return nil, errNotConnected
}

func (d *disconnectedDiscovery) OpenAPIV3() openapi.Client {
	return nil
}

func (d *disconnectedDiscovery) WithLegacy() discovery.DiscoveryInterface {
	return d
}

// disconnectedDynamic implements dynamic.Interface
// It returns appropriate errors for all operations when the client is disconnected.
type disconnectedDynamic struct{}

func (d *disconnectedDynamic) Resource(resource schema.GroupVersionResource) dynamic.NamespaceableResourceInterface {
	return &disconnectedNamespaceableResource{}
}

// disconnectedNamespaceableResource implements dynamic.NamespaceableResourceInterface
type disconnectedNamespaceableResource struct{}

func (d *disconnectedNamespaceableResource) Namespace(ns string) dynamic.ResourceInterface {
	return &disconnectedResource{}
}

func (d *disconnectedNamespaceableResource) Create(ctx context.Context, obj *unstructured.Unstructured, options metav1.CreateOptions, subresources ...string) (*unstructured.Unstructured, error) {
	return nil, errNotConnected
}

func (d *disconnectedNamespaceableResource) Update(ctx context.Context, obj *unstructured.Unstructured, options metav1.UpdateOptions, subresources ...string) (*unstructured.Unstructured, error) {
	return nil, errNotConnected
}

func (d *disconnectedNamespaceableResource) UpdateStatus(ctx context.Context, obj *unstructured.Unstructured, options metav1.UpdateOptions) (*unstructured.Unstructured, error) {
	return nil, errNotConnected
}

func (d *disconnectedNamespaceableResource) Delete(ctx context.Context, name string, options metav1.DeleteOptions, subresources ...string) error {
	return errNotConnected
}

func (d *disconnectedNamespaceableResource) DeleteCollection(ctx context.Context, options metav1.DeleteOptions, listOptions metav1.ListOptions) error {
	return errNotConnected
}

func (d *disconnectedNamespaceableResource) Get(ctx context.Context, name string, options metav1.GetOptions, subresources ...string) (*unstructured.Unstructured, error) {
	return nil, errNotConnected
}

func (d *disconnectedNamespaceableResource) List(ctx context.Context, opts metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	return nil, errNotConnected
}

func (d *disconnectedNamespaceableResource) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	return nil, errNotConnected
}

func (d *disconnectedNamespaceableResource) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, options metav1.PatchOptions, subresources ...string) (*unstructured.Unstructured, error) {
	return nil, errNotConnected
}

func (d *disconnectedNamespaceableResource) Apply(ctx context.Context, name string, obj *unstructured.Unstructured, options metav1.ApplyOptions, subresources ...string) (*unstructured.Unstructured, error) {
	return nil, errNotConnected
}

func (d *disconnectedNamespaceableResource) ApplyStatus(ctx context.Context, name string, obj *unstructured.Unstructured, options metav1.ApplyOptions) (*unstructured.Unstructured, error) {
	return nil, errNotConnected
}

// disconnectedResource implements dynamic.ResourceInterface
type disconnectedResource struct{}

func (d *disconnectedResource) Create(ctx context.Context, obj *unstructured.Unstructured, options metav1.CreateOptions, subresources ...string) (*unstructured.Unstructured, error) {
	return nil, errNotConnected
}

func (d *disconnectedResource) Update(ctx context.Context, obj *unstructured.Unstructured, options metav1.UpdateOptions, subresources ...string) (*unstructured.Unstructured, error) {
	return nil, errNotConnected
}

func (d *disconnectedResource) UpdateStatus(ctx context.Context, obj *unstructured.Unstructured, options metav1.UpdateOptions) (*unstructured.Unstructured, error) {
	return nil, errNotConnected
}

func (d *disconnectedResource) Delete(ctx context.Context, name string, options metav1.DeleteOptions, subresources ...string) error {
	return errNotConnected
}

func (d *disconnectedResource) DeleteCollection(ctx context.Context, options metav1.DeleteOptions, listOptions metav1.ListOptions) error {
	return errNotConnected
}

func (d *disconnectedResource) Get(ctx context.Context, name string, options metav1.GetOptions, subresources ...string) (*unstructured.Unstructured, error) {
	return nil, errNotConnected
}

func (d *disconnectedResource) List(ctx context.Context, opts metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	return nil, errNotConnected
}

func (d *disconnectedResource) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	return nil, errNotConnected
}

func (d *disconnectedResource) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, options metav1.PatchOptions, subresources ...string) (*unstructured.Unstructured, error) {
	return nil, errNotConnected
}

func (d *disconnectedResource) Apply(ctx context.Context, name string, obj *unstructured.Unstructured, options metav1.ApplyOptions, subresources ...string) (*unstructured.Unstructured, error) {
	return nil, errNotConnected
}

func (d *disconnectedResource) ApplyStatus(ctx context.Context, name string, obj *unstructured.Unstructured, options metav1.ApplyOptions) (*unstructured.Unstructured, error) {
	return nil, errNotConnected
}

// Compile-time interface checks
var _ discovery.DiscoveryInterface = (*disconnectedDiscovery)(nil)
var _ dynamic.Interface = (*disconnectedDynamic)(nil)
var _ dynamic.NamespaceableResourceInterface = (*disconnectedNamespaceableResource)(nil)
var _ dynamic.ResourceInterface = (*disconnectedResource)(nil)
