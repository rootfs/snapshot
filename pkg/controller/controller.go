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
	"time"

	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	//v1core "k8s.io/client-go/kubernetes/typed/core/v1"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/rest"
	kcache "k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	//"k8s.io/kubernetes/pkg/api"

	tprv1 "github.com/rootfs/snapshot/pkg/apis/tpr/v1"
	"github.com/rootfs/snapshot/pkg/controller/cache"
	"github.com/rootfs/snapshot/pkg/controller/reconciler"
	"github.com/rootfs/snapshot/pkg/controller/snapshotter"
	"github.com/rootfs/snapshot/pkg/volume/hostpath"
)

const (
	reconcilerLoopPeriod time.Duration = 100 * time.Millisecond
)

type SnapshotController interface {
	Run(stopCh <-chan struct{})
}

type snapshotController struct {
	snapshotClient *rest.RESTClient
	snapshotScheme *runtime.Scheme

	// desiredStateOfWorld is a data structure containing the desired state of
	// the world according to this controller: i.e. what VolumeSnapshots need
	// the Volumesnapshotdata to be created, what Volumesnapshotdata and their
	// representing "on-disk" snapshots to be removed.
	desiredStateOfWorld cache.DesiredStateOfWorld

	// actualStateOfWorld is a data structure containing the actual state of
	// the world according to this controller: i.e. which VolumeSnapshots and
	// VolumeSnapshot data exist and to which PV/PVCs are associated.
	actualStateOfWorld cache.ActualStateOfWorld

	// reconciler is used to run an asynchronous periodic loop to create and delete
	// Volumesnapshotdata for the user created and deleted VolumeSnapshot objects and
	// trigger the actual snapshot creation in the volume backends.
	reconciler reconciler.Reconciler

	// Volume snapshotter is responsible for talking to the backend and creating, removing
	// or promoting the snapshots.
	snapshotter snapshotter.VolumeSnapshotter
	// recorder is used to record events in the API server
	recorder record.EventRecorder
}

func NewSnapshotController(client *rest.RESTClient,
	scheme *runtime.Scheme,
	syncDuration time.Duration) SnapshotController {
	sc := &snapshotController{
		snapshotClient: client,
		snapshotScheme: scheme,
	}

	//eventBroadcaster := record.NewBroadcaster()
	//eventBroadcaster.StartLogging(glog.Infof)
	//eventBroadcaster.StartRecordingToSink(&v1core.EventSinkImpl{Interface: v1core.New(client).Events("")})
	//	sc.recorder = eventBroadcaster.NewRecorder(api.Scheme, apiv1.EventSource{Component: "volume snapshotting"})

	sc.desiredStateOfWorld = cache.NewDesiredStateOfWorld()
	sc.actualStateOfWorld = cache.NewActualStateOfWorld()

	sc.snapshotter = snapshotter.NewVolumeSnapshotter(sc.desiredStateOfWorld)

	sc.reconciler = reconciler.NewReconciler(
		reconcilerLoopPeriod,
		syncDuration,
		false, /* disableReconciliationSync */
		sc.desiredStateOfWorld,
		sc.actualStateOfWorld,
		sc.snapshotter)

	return sc
}

// Run starts an Snapshot resource controller
func (c *snapshotController) Run(ctx <-chan struct{}) {
	glog.Infof("Watch snapshot objects\n")

	// Watch snapshot objects
	source := kcache.NewListWatchFromClient(
		c.snapshotClient,
		tprv1.VolumeSnapshotDataResourcePlural,
		apiv1.NamespaceAll,
		fields.Everything())

	_, controller := kcache.NewInformer(
		source,

		// The object type.
		&tprv1.Volumesnapshotdata{},

		// resyncPeriod
		// Every resyncPeriod, all resources in the kcache will retrigger events.
		// Set to 0 to disable the resync.
		0,

		// Your custom resource event handlers.
		kcache.ResourceEventHandlerFuncs{
			AddFunc:    c.onAdd,
			UpdateFunc: c.onUpdate,
			DeleteFunc: c.onDelete,
		})
	go c.reconciler.Run(ctx)
	go controller.Run(ctx)
}

func (c *snapshotController) onAdd(obj interface{}) {
	// Add snapshot: Add snapshot to DesiredStateOfWorld, then ask snapshotter to create
	// the actual snapshot
	snapshotdata := obj.(*tprv1.Volumesnapshotdata)
	glog.Infof("[CONTROLLER] OnAdd %s, Spec %#v", snapshotdata.Metadata.SelfLink, snapshotdata.Spec)

	if snapshotdata.Spec.HostPath != nil {
		snap, err := hostpath.Snapshot(snapshotdata.Spec.HostPath.Path)
		if err != nil {
			glog.Warningf("failed to snapshot %s, err: %v", snapshotdata.Spec.HostPath.Path, err)
		} else {
			glog.Infof("snapshot %s to snap %s", snapshotdata.Spec.HostPath.Path, snap)
		}
	}

}

func (c *snapshotController) onUpdate(oldObj, newObj interface{}) {
	oldSnapshot := oldObj.(*tprv1.Volumesnapshotdata)
	newSnapshot := newObj.(*tprv1.Volumesnapshotdata)
	glog.Infof("[CONTROLLER] OnUpdate oldObj: %s\n", oldSnapshot.Metadata.SelfLink)
	glog.Infof("[CONTROLLER] OnUpdate newObj: %s\n", newSnapshot.Metadata.SelfLink)
}

func (c *snapshotController) onDelete(obj interface{}) {
	// Delete snapshot: Remove the snapshot from DesiredStateOfWorld, then ask snapshotter to delete
	// the snapshot itself
	snapshot := obj.(*tprv1.Volumesnapshotdata)
	glog.Infof("[CONTROLLER] OnDelete %s\n", snapshot.Metadata.SelfLink)
}
