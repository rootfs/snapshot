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
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	tprv1 "github.com/rootfs/snapshot/apis/tpr/v1"

	"github.com/rootfs/snapshot/volume/hostpath"
)

type SnapshotController struct {
	SnapshotClient *rest.RESTClient
	SnapshotScheme *runtime.Scheme
}

// Run starts an Snapshot resource controller
func (c *SnapshotController) Run(ctx <-chan struct{}) error {
	glog.Infof("Watch snapshot objects\n")

	// Watch snapshot objects
	source := cache.NewListWatchFromClient(
		c.SnapshotClient,
		tprv1.VolumeSnapshotResourcePlural,
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

	go controller.Run(ctx)
	return nil
}

func (c *SnapshotController) onAdd(obj interface{}) {
	snapshot := obj.(*tprv1.VolumeSnapshot)
	glog.Infof("[CONTROLLER] OnAdd %s", snapshot.ObjectMeta.SelfLink)

	//HACK this should move to snapshot data
	if snapshot.Spec.HostPath != nil {
		snap, err := hostpath.Snapshot(snapshot.Spec.HostPath.Path)
		if err != nil {
			glog.Warningf("failed to snapshot %s, err: %v", snapshot.Spec.HostPath.Path, err)
		} else {
			glog.Infof("snapshot %s to snap %s", snapshot.Spec.HostPath.Path, snap)
		}
	}

}

func (c *SnapshotController) onUpdate(oldObj, newObj interface{}) {
	oldSnapshot := oldObj.(*tprv1.VolumeSnapshot)
	newSnapshot := newObj.(*tprv1.VolumeSnapshot)
	glog.Infof("[CONTROLLER] OnUpdate oldObj: %s\n", oldSnapshot.ObjectMeta.SelfLink)
	glog.Infof("[CONTROLLER] OnUpdate newObj: %s\n", newSnapshot.ObjectMeta.SelfLink)
}

func (c *SnapshotController) onDelete(obj interface{}) {
	snapshot := obj.(*tprv1.VolumeSnapshot)
	glog.Infof("[CONTROLLER] OnDelete %s\n", snapshot.ObjectMeta.SelfLink)
}
