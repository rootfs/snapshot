/*
Copyright 2017 The Kubernetes Authors.

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

package internalversion

import (
	tprv1 "github.com/rootfs/snapshot/pkg/apis/tpr/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
	scheme "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset/scheme"
)

// SnapshotGetter has a method to return a SnapshotInterface.
// A group's client should implement this interface.
type SnapshotsGetter interface {
	Snapshots() SnapshotInterface
}

// SnapshotInterface has methods to work with Snapshot resources.
type SnapshotInterface interface {
	Create(snapshot *tprv1.VolumeSnapshot) (*tprv1.VolumeSnapshot, error)
	Update(*tprv1.VolumeSnapshot) (*tprv1.VolumeSnapshot, error)
	UpdateStatus(*tprv1.VolumeSnapshot) (*tprv1.VolumeSnapshot, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*tprv1.VolumeSnapshot, error)
	List(opts v1.ListOptions) (*tprv1.VolumeSnapshotList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *tprv1.VolumeSnapshot, err error)
	PersistentVolumeExpansion
}

// snapshots implements SnapshotInterface
type snapshots struct {
	client rest.Interface
}

// newSnapshots returns a snapshot
func newSnapshots(c *CoreClient) *snapshots {
	return &snapshots{
		client: c.RESTClient(),
	}
}

// Create takes the representation of a snapshot and creates it.  Returns the server's representation of the snapshot, and an error, if there is any.
func (c *snapshots) Create(snapshot *tprv1.VolumeSnapshot) (result *tprv1.VolumeSnapshot, err error) {
	result = &tprv1.VolumeSnapshot{}
	err = c.client.Post().
		Resource("snapshots").
		Body(snapshot).
		Do().
		Into(result)
	return
}

// Update takes the representation of a snapshot and updates it. Returns the server's representation of the snapshot, and an error, if there is any.
func (c *snapshots) Update(snapshot *tprv1.VolumeSnapshot) (result *tprv1.VolumeSnapshot, err error) {
	result = &tprv1.VolumeSnapshot{}
	err = c.client.Put().
		Resource("snapshots").
		Name(snapshot.Metadata.Name).
		Body(snapshot).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclientstatus=false comment above the type to avoid generating UpdateStatus().

func (c *snapshots) UpdateStatus(snapshot *tprv1.VolumeSnapshot) (result *tprv1.VolumeSnapshot, err error) {
	result = &tprv1.VolumeSnapshot{}
	err = c.client.Put().
		Resource("snapshots").
		Name(snapshot.Metadata.Name).
		SubResource("status").
		Body(snapshot).
		Do().
		Into(result)
	return
}

// Delete takes name of the snapshot and deletes it. Returns an error if one occurs.
func (c *snapshots) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Resource("snapshots").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *snapshots) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Resource("snapshots").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Get takes name of the snapshot, and returns the corresponding snapshot object, and an error if there is any.
func (c *snapshots) Get(name string, options v1.GetOptions) (result *tprv1.VolumeSnapshot, err error) {
	result = &tprv1.VolumeSnapshot{}
	err = c.client.Get().
		Resource("snapshots").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of snapshots that match those selectors.
func (c *snapshots) List(opts v1.ListOptions) (result *tprv1.VolumeSnapshotList, err error) {
	result = &tprv1.VolumeSnapshotList{}
	err = c.client.Get().
		Resource("snapshots").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested snapshots.
func (c *snapshots) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Resource("snapshots").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Patch applies the patch and returns the patched snapshot.
func (c *snapshots) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *tprv1.VolumeSnapshot, err error) {
	result = &tprv1.VolumeSnapshot{}
	err = c.client.Patch(pt).
		Resource("snapshots").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
