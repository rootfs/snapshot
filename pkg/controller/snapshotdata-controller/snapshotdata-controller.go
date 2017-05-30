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

package data_controller

import (
	"time"

	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/rest"
	kcache "k8s.io/client-go/tools/cache"

	tprv1 "github.com/rootfs/snapshot/pkg/apis/tpr/v1"
)

const (
	reconcilerLoopPeriod time.Duration = 100 * time.Millisecond
)

type SnapshotDataController interface {
	Run(stopCh <-chan struct{})
}

type snapshotDataController struct {
	snapshotClient *rest.RESTClient
	snapshotScheme *runtime.Scheme
}

func NewSnapshotDataController(client *rest.RESTClient,
	scheme *runtime.Scheme,
	syncDuration time.Duration) SnapshotDataController {
	sc := &snapshotDataController{
		snapshotClient: client,
		snapshotScheme: scheme,
	}

	return sc
}

// Run starts an Snapshot resource controller
func (c *snapshotDataController) Run(ctx <-chan struct{}) {
	glog.Infof("Watch snapshot data objects\n")

	// Watch snapshot objects
	source := kcache.NewListWatchFromClient(
		c.snapshotClient,
		tprv1.VolumeSnapshotDataResourcePlural,
		apiv1.NamespaceAll,
		fields.Everything())

	_, controller := kcache.NewInformer(
		source,

		// The object type.
		&tprv1.VolumeSnapshotData{},

		// resyncPeriod
		// Every resyncPeriod, all resources in the kcache will retrigger events.
		// Set to 0 to disable the resync.
		time.Minute*60,

		// Your custom resource event handlers.
		kcache.ResourceEventHandlerFuncs{
			AddFunc:    c.onSnapshotDataAdd,
			UpdateFunc: c.onSnapshotDataUpdate,
			DeleteFunc: c.onSnapshotDataDelete,
		})
	go controller.Run(ctx)
}

func (c *snapshotDataController) onSnapshotDataAdd(obj interface{}) {
	// Add snapshot: Add snapshot to DesiredStateOfWorld, then ask snapshotter to create
	// the actual snapshot
	snapshotdata, ok := obj.(*tprv1.VolumeSnapshotData)
	if !ok {
		glog.Warning("expecting type VolumeSnapshotData but received type %T", obj)
		return
	}
	glog.Infof("[CONTROLLER] OnAdd %s, Spec %#v", snapshotdata.Metadata.SelfLink, snapshotdata.Spec)
	/*
		if snapshotdata.Spec.HostPath != nil {
			snap, err := hostpath.Snapshot(snapshotdata.Spec.HostPath.Path)
			if err != nil {
				glog.Warningf("failed to snapshot %s, err: %v", snapshotdata.Spec.HostPath.Path, err)
			} else {
				glog.Infof("snapshot %s to snap %s", snapshotdata.Spec.HostPath.Path, snap)
			}
		}
	*/
}

func (c *snapshotDataController) onSnapshotDataUpdate(oldObj, newObj interface{}) {
	oldSnapshot := oldObj.(*tprv1.VolumeSnapshotData)
	newSnapshot := newObj.(*tprv1.VolumeSnapshotData)
	glog.Infof("[CONTROLLER] OnUpdate oldObj: %s\n", oldSnapshot.Metadata.SelfLink)
	glog.Infof("[CONTROLLER] OnUpdate newObj: %s\n", newSnapshot.Metadata.SelfLink)
}

func (c *snapshotDataController) onSnapshotDataDelete(obj interface{}) {
	// Delete snapshot: Remove the snapshot from DesiredStateOfWorld, then ask snapshotter to delete
	// the snapshot itself
	snapshot := obj.(*tprv1.VolumeSnapshotData)
	glog.Infof("[CONTROLLER] OnDelete %s\n", snapshot.Metadata.SelfLink)
}
