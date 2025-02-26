/*
Copyright 2025 Rancher Labs, Inc.

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

// Code generated by main. DO NOT EDIT.

package fake

import (
	"context"

	v3 "github.com/rancher/rancher/pkg/apis/management.cattle.io/v3"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeAuthConfigs implements AuthConfigInterface
type FakeAuthConfigs struct {
	Fake *FakeManagementV3
}

var authconfigsResource = v3.SchemeGroupVersion.WithResource("authconfigs")

var authconfigsKind = v3.SchemeGroupVersion.WithKind("AuthConfig")

// Get takes name of the authConfig, and returns the corresponding authConfig object, and an error if there is any.
func (c *FakeAuthConfigs) Get(ctx context.Context, name string, options v1.GetOptions) (result *v3.AuthConfig, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(authconfigsResource, name), &v3.AuthConfig{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.AuthConfig), err
}

// List takes label and field selectors, and returns the list of AuthConfigs that match those selectors.
func (c *FakeAuthConfigs) List(ctx context.Context, opts v1.ListOptions) (result *v3.AuthConfigList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(authconfigsResource, authconfigsKind, opts), &v3.AuthConfigList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v3.AuthConfigList{ListMeta: obj.(*v3.AuthConfigList).ListMeta}
	for _, item := range obj.(*v3.AuthConfigList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested authConfigs.
func (c *FakeAuthConfigs) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(authconfigsResource, opts))
}

// Create takes the representation of a authConfig and creates it.  Returns the server's representation of the authConfig, and an error, if there is any.
func (c *FakeAuthConfigs) Create(ctx context.Context, authConfig *v3.AuthConfig, opts v1.CreateOptions) (result *v3.AuthConfig, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(authconfigsResource, authConfig), &v3.AuthConfig{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.AuthConfig), err
}

// Update takes the representation of a authConfig and updates it. Returns the server's representation of the authConfig, and an error, if there is any.
func (c *FakeAuthConfigs) Update(ctx context.Context, authConfig *v3.AuthConfig, opts v1.UpdateOptions) (result *v3.AuthConfig, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(authconfigsResource, authConfig), &v3.AuthConfig{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.AuthConfig), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeAuthConfigs) UpdateStatus(ctx context.Context, authConfig *v3.AuthConfig, opts v1.UpdateOptions) (*v3.AuthConfig, error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateSubresourceAction(authconfigsResource, "status", authConfig), &v3.AuthConfig{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.AuthConfig), err
}

// Delete takes name of the authConfig and deletes it. Returns an error if one occurs.
func (c *FakeAuthConfigs) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteActionWithOptions(authconfigsResource, name, opts), &v3.AuthConfig{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeAuthConfigs) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(authconfigsResource, listOpts)

	_, err := c.Fake.Invokes(action, &v3.AuthConfigList{})
	return err
}

// Patch applies the patch and returns the patched authConfig.
func (c *FakeAuthConfigs) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v3.AuthConfig, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(authconfigsResource, name, pt, data, subresources...), &v3.AuthConfig{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.AuthConfig), err
}
