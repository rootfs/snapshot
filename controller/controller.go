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

package controller

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	tprv1 "github.com/rootfs/snapshot/apis/tpr/v1"
)

type SnapshotController struct {
	SnapshotClient *rest.RESTClient
	SnapshotScheme *runtime.Scheme
}

// Run starts an Snapshot resource controller
func (c *SnapshotController) Run(ctx context.Context) error {
	fmt.Print("Watch snapshot objects\n")

	// Watch snapshot objects
	_, err := c.watchSnapshot(ctx)
	if err != nil {
		fmt.Printf("Failed to register watch for Snapshot resource: %v\n", err)
		return err
	}

	<-ctx.Done()
	return ctx.Err()
}

func (c *SnapshotController) watchSnapshot(ctx context.Context) (cache.Controller, error) {
	source := cache.NewListWatchFromClient(
		c.SnapshotClient,
		tprv1.VolumeSnapshotResource,
		apiv1.NamespaceAll,
		fields.Everything())

	_, controller := cache.NewInformer(
		source,

		// The object type.
		&tprv1.VolumeSnapshot{},

		// resyncPeriod
		// Every resyncPeriod, all resources in the cache will retrigger events.
		// Set to 0 to disable the resync.
		0,

		// Your custom resource event handlers.
		cache.ResourceEventHandlerFuncs{
			AddFunc:    c.onAdd,
			UpdateFunc: c.onUpdate,
			DeleteFunc: c.onDelete,
		})

	go controller.Run(ctx.Done())
	return controller, nil
}

func (c *SnapshotController) onAdd(obj interface{}) {
	snapshot := obj.(*tprv1.VolumeSnapshot)
	fmt.Printf("[CONTROLLER] OnAdd %s\n", snapshot.ObjectMeta.SelfLink)

	// NEVER modify objects from the store. It's a read-only, local cache.
	// You can use exampleScheme.Copy() to make a deep copy of original object and modify this copy
	// Or create a copy manually for better performance
	copyObj, err := c.SnapshotScheme.Copy(snapshot)
	if err != nil {
		fmt.Printf("ERROR creating a deep copy of snapshot object: %v\n", err)
		return
	}

	snapshotCopy := copyObj.(*tprv1.VolumeSnapshot)
	snapshotCopy.Status = tprv1.VolumeSnapshotStatus{
		Conditions: []tprv1.VolumeSnapshotCondition{
			tprv1.VolumeSnapshotCondition{
				Type: tprv1.VolumeSnapshotConditionReady,
			},
		},
	}

	err = c.SnapshotClient.Put().
		Name(snapshot.ObjectMeta.Name).
		Namespace(snapshot.ObjectMeta.Namespace).
		Resource(tprv1.VolumeSnapshotResource).
		Body(snapshotCopy).
		Do().
		Error()

	if err != nil {
		fmt.Printf("ERROR updating status: %v\n", err)
	} else {
		fmt.Printf("UPDATED status: %#v\n", snapshotCopy)
	}
}

func (c *SnapshotController) onUpdate(oldObj, newObj interface{}) {
	oldSnapshot := oldObj.(*tprv1.VolumeSnapshot)
	newSnapshot := newObj.(*tprv1.VolumeSnapshot)
	fmt.Printf("[CONTROLLER] OnUpdate oldObj: %s\n", oldSnapshot.ObjectMeta.SelfLink)
	fmt.Printf("[CONTROLLER] OnUpdate newObj: %s\n", newSnapshot.ObjectMeta.SelfLink)
}

func (c *SnapshotController) onDelete(obj interface{}) {
	snapshot := obj.(*tprv1.VolumeSnapshot)
	fmt.Printf("[CONTROLLER] OnDelete %s\n", snapshot.ObjectMeta.SelfLink)
}
