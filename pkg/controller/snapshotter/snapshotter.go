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

package snapshotter

import (
	"github.com/golang/glog"

	"k8s.io/kubernetes/pkg/util/goroutinemap"

	tprv1 "github.com/rootfs/snapshot/pkg/apis/tpr/v1"
	"github.com/rootfs/snapshot/pkg/controller/cache"
)

// VolumeSnapshotter does the "heavy lifting": it spawns gouroutines that talk to the
// backend to actually perform the operations on the storage devices.
// It creates and deletes the snapshots and promotes snapshots to volumes (PV). The create
// and delete operations need to be idempotent and count with the fact the API object writes
type VolumeSnapshotter interface {
	CreateVolumeSnapshot(snapshotName string, snapshotSpec *tprv1.VolumeSnapshotSpec)
	DeleteVolumeSnapshot(snapshotName string, snapshotSpec *tprv1.VolumeSnapshotSpec)
	PromoteVolumeSnapshotToPV(snapshotName string, snapshotSpec *tprv1.VolumeSnapshotSpec)
}

type volumeSnapshotter struct {
	actualStateOfWorld cache.ActualStateOfWorld
	runningOperation   goroutinemap.GoRoutineMap
}

const (
	snapshotOpCreatePrefix  string = "create"
	snapshotOpDeletePrefix  string = "delete"
	snapshotOpPromotePrefix string = "promote"
)

func NewVolumeSnapshotter(asw cache.ActualStateOfWorld) VolumeSnapshotter {
	return &volumeSnapshotter{
		actualStateOfWorld: asw,
	}
}

// Below are the closures meant to build the functions for the GoRoutineMap operations.

func (vs *volumeSnapshotter) getSnapshotCreateFunc() func() error {
	// Create a snapshot:
	// 1. If Snapshot referencs SnapshotData object, try to find it
	//   1a. If doesn't exist, log error and finish, if it exists already, check its SnapshotRef
	//   1b. If it's empty, check its Spec UID (or fins out what PV/PVC does and copyt the mechanism)
	//   1c. If it matches the user (TODO: how to find out?), bind the two objects and finish
	//   1d. If it doesn't match, log error and finish.
	// 2. Create the SnapshotData object
	// 3. Ask the backend to create the snapshot (device)
	// 4. If OK, update the SnapshotData and Snapshot objects
	// 5. Add the Snapshot to the ActualStateOfWorld
	// 6. Finish (we have created snapshot for an user)

	return func() error { return nil }
}

func (vs *volumeSnapshotter) getSnapshotDeleteFunc() func() error {
	// Delete a snapshot
	// 1. Find the SnapshotData corresponding to Snapshot
	//   1a: Not found => finish (it's been deleted already)
	// 2. Ask the backend to remove the snapshot device
	// 3. Delete the SnapshotData object
	// 4. Remove the Snapshot from ActualStateOfWorld
	// 5. Finish
	return func() error { return nil }
}

func (vs *volumeSnapshotter) getSnapshotPromoteFunc() func() error {
	// Promote snapshot to a PVC
	// 1. We have a PVC referencing a Snapshot object
	// 2. Find the SnapshotData corresponding to tha Snapshot
	// 3. Ask the backend to give us a device (PV) made from the snapshot device
	// 4. Bind it to the PVC
	// 5. Finish
	return func() error { return nil }
}

func (vs *volumeSnapshotter) CreateVolumeSnapshot(snapshotName string, snapshotSpec *tprv1.VolumeSnapshotSpec) {
	operationName := snapshotOpCreatePrefix + snapshotName + snapshotSpec.PersistentVolumeClaimName
	glog.Infof("Snapshotter is about to create volume snapshot operation named %s", operationName)
}

func (vs *volumeSnapshotter) DeleteVolumeSnapshot(snapshotName string, snapshotSpec *tprv1.VolumeSnapshotSpec) {
	operationName := snapshotOpDeletePrefix + snapshotName + snapshotSpec.PersistentVolumeClaimName
	glog.Infof("Snapshotter is about to create volume snapshot operation named %s", operationName)

}

func (vs *volumeSnapshotter) PromoteVolumeSnapshotToPV(snapshotName string, snapshotSpec *tprv1.VolumeSnapshotSpec) {
	operationName := snapshotOpPromotePrefix + snapshotName + snapshotSpec.PersistentVolumeClaimName
	glog.Infof("Snapshotter is about to create volume snapshot operation named %s", operationName)

}
