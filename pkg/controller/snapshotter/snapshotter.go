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
	"fmt"

	"github.com/golang/glog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/rest"
	"k8s.io/kubernetes/pkg/util/goroutinemap"
	"k8s.io/kubernetes/pkg/util/goroutinemap/exponentialbackoff"

	tprv1 "github.com/rootfs/snapshot/pkg/apis/tpr/v1"
	"github.com/rootfs/snapshot/pkg/controller/cache"

	"github.com/rootfs/snapshot/pkg/volume/hostpath"
)

const (
	defaultExponentialBackOffOnError = true
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
	restClient         *rest.RESTClient
	coreClient         kubernetes.Interface
	scheme             *runtime.Scheme
	actualStateOfWorld cache.ActualStateOfWorld
	runningOperation   goroutinemap.GoRoutineMap
}

const (
	snapshotOpCreatePrefix  string = "create"
	snapshotOpDeletePrefix  string = "delete"
	snapshotOpPromotePrefix string = "promote"
)

func NewVolumeSnapshotter(
	restClient *rest.RESTClient,
	scheme *runtime.Scheme,
	clientset kubernetes.Interface,
	asw cache.ActualStateOfWorld) VolumeSnapshotter {
	return &volumeSnapshotter{
		restClient:         restClient,
		coreClient:         clientset,
		scheme:             scheme,
		actualStateOfWorld: asw,
		runningOperation:   goroutinemap.NewGoRoutineMap(defaultExponentialBackOffOnError),
	}
}

// This is the function responsible for determining the correct volume plugin to use,
// asking it to make a snapshot and assignig it some name that it returns to the caller.
func (vs *volumeSnapshotter) takeSnapshot(spec *v1.PersistentVolumeSpec) (*tprv1.VolumeSnapshotDataSource, error) {
	// TODO: Find a volume snapshot plugin to use for taking the snapshot and do so
	if spec.HostPath != nil {
		snap, err := hostpath.Snapshot(spec.HostPath.Path)
		if err != nil {
			glog.Warningf("failed to snapshot %s, err: %v", spec.HostPath.Path, err)
		} else {
			glog.Infof("snapshot %#v to snap %#v", spec.HostPath, snap.HostPath)
			return snap, nil
		}
	}

	return nil, nil
}

// Below are the closures meant to build the functions for the GoRoutineMap operations.

func (vs *volumeSnapshotter) getSnapshotCreateFunc(snapshotName string, snapshotSpec *tprv1.VolumeSnapshotSpec) func() error {
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
	return func() error {
		if snapshotSpec.SnapshotDataName != "" {
			// This spec has the SnapshotDataName already set: this means importing admin-created snapshots
			// TODO: Not implemented yet
			return fmt.Errorf("Importing snapshots is not implemented yet")
		}

		pvcName := snapshotSpec.PersistentVolumeClaimName
		if pvcName == "" {
			return fmt.Errorf("The PVC name is not specified in snapshot %s", snapshotName)
		}
		snapNameSpace, snapName, err := cache.GetNameAndNameSpaceFromSnapshotName(snapshotName)
		if err != nil {
			return fmt.Errorf("Snapshot %s is malformed", snapshotName)
		}
		pvc, err := vs.coreClient.CoreV1().PersistentVolumeClaims(snapNameSpace).Get(pvcName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("Failed to retrieve PVC %s from the API server: %q", pvcName, err)
		}
		if pvc.Status.Phase != v1.ClaimBound {
			return fmt.Errorf("The PVC %s not yet bound to a PV, will not attempt to take a snapshot yet.")
		}

		pvName := pvc.Spec.VolumeName
		pv, err := vs.coreClient.CoreV1().PersistentVolumes().Get(pvName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("Failed to retrieve PV %s from the API server: %q", pvName, err)
		}

		snapshotDataSource, err := vs.takeSnapshot(&pv.Spec)
		if err != nil || snapshotDataSource == nil {
			return fmt.Errorf("Failed to take snapshot of the volume %s: %q", pvName, err)
		}
		// Snapshot has been created, made an object for it
		readyCondition := tprv1.VolumeSnapshotDataCondition{
			Type:    tprv1.VolumeSnapshotDataConditionReady,
			Status:  v1.ConditionTrue,
			Message: "Snapsot created succsessfully",
		}
		snapshotData := &tprv1.VolumeSnapshotData{
			Metadata: metav1.ObjectMeta{
				// FIXME: make a unique ID
				Name: snapName,
			},
			Spec: tprv1.VolumeSnapshotDataSpec{
				VolumeSnapshotRef: &v1.ObjectReference{
					Kind: "VolumeSnapshot",
					Name: snapshotName,
				},
				PersistentVolumeRef: &v1.ObjectReference{
					Kind: "PersistentVolume",
					Name: pvName,
				},
				VolumeSnapshotDataSource: *snapshotDataSource,
			},
			Status: tprv1.VolumeSnapshotDataStatus{
				Conditions: []tprv1.VolumeSnapshotDataCondition{
					readyCondition,
				},
			},
		}
		var result tprv1.VolumeSnapshotData
		err = vs.restClient.Post().
			Resource(tprv1.VolumeSnapshotDataResourcePlural).
			Namespace(v1.NamespaceDefault).
			Body(snapshotData).
			Do().Into(&result)

		if err != nil {
			//FIXME: Errors writing to the API server are common: this needs to be re-tried
			glog.Errorf("Error creating the VolumeSnapshotData %s: %v", snapshotName, err)
		}
		vs.actualStateOfWorld.AddSnapshot(snapshotName, snapshotSpec)
		// TODO: Update the VolumeSnapshot object too

		var snapshotObj tprv1.VolumeSnapshot
		err = vs.restClient.Get().
			Name(snapName).
			Resource(tprv1.VolumeSnapshotResourcePlural).
			Namespace(v1.NamespaceDefault).
			Do().Into(&snapshotObj)

		objCopy, err := vs.scheme.DeepCopy(&snapshotObj)
		if err != nil {
			glog.Errorf("Error copying snapshot object %s object from API server", snapName)
		}
		snapshotCopy, ok := objCopy.(*tprv1.VolumeSnapshot)
		if !ok {
			glog.Warning("expecting type VolumeSnapshot but received type %T", objCopy)
		}
		snapshotCopy.Spec.SnapshotDataName = snapName
		snapshotCopy.Status.Conditions = []tprv1.VolumeSnapshotCondition{
			{
				Type:    tprv1.VolumeSnapshotConditionReady,
				Status:  v1.ConditionTrue,
				Message: "Snapsot created succsessfully",
			},
		}
		// TODO: Make diff of the two objects and then use restClient.Patch to update it
		return nil
	}
}

func (vs *volumeSnapshotter) getSnapshotDeleteFunc(snapshotName string, snapshotSpec *tprv1.VolumeSnapshotSpec) func() error {
	// Delete a snapshot
	// 1. Find the SnapshotData corresponding to Snapshot
	//   1a: Not found => finish (it's been deleted already)
	// 2. Ask the backend to remove the snapshot device
	// 3. Delete the SnapshotData object
	// 4. Remove the Snapshot from ActualStateOfWorld
	// 5. Finish
	return func() error { return nil }
}

func (vs *volumeSnapshotter) getSnapshotPromoteFunc(snapshotName string, snapshotSpec *tprv1.VolumeSnapshotSpec) func() error {
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
	glog.Infof("Snapshotter is about to create volume snapshot operation named %s, spec %#v", operationName, snapshotSpec)

	err := vs.runningOperation.Run(operationName, vs.getSnapshotCreateFunc(snapshotName, snapshotSpec))

	if err != nil {
		switch {
		case goroutinemap.IsAlreadyExists(err):
			glog.V(4).Infof("operation %q is already running, skipping", operationName)
		case exponentialbackoff.IsExponentialBackoff(err):
			glog.V(4).Infof("operation %q postponed due to exponential backoff", operationName)
		default:
			glog.Errorf("Failed to schedule the operation %q: %v", operationName, err)
		}
	}
}

func (vs *volumeSnapshotter) DeleteVolumeSnapshot(snapshotName string, snapshotSpec *tprv1.VolumeSnapshotSpec) {
	operationName := snapshotOpDeletePrefix + snapshotName + snapshotSpec.PersistentVolumeClaimName
	glog.Infof("Snapshotter is about to create volume snapshot operation named %s", operationName)

	err := vs.runningOperation.Run(operationName, vs.getSnapshotDeleteFunc(snapshotName, snapshotSpec))

	if err != nil {
		switch {
		case goroutinemap.IsAlreadyExists(err):
			glog.V(4).Infof("operation %q is already running, skipping", operationName)
		case exponentialbackoff.IsExponentialBackoff(err):
			glog.V(4).Infof("operation %q postponed due to exponential backoff", operationName)
		default:
			glog.Errorf("Failed to schedule the operation %q: %v", operationName, err)
		}
	}
}

func (vs *volumeSnapshotter) PromoteVolumeSnapshotToPV(snapshotName string, snapshotSpec *tprv1.VolumeSnapshotSpec) {
	operationName := snapshotOpPromotePrefix + snapshotName + snapshotSpec.PersistentVolumeClaimName
	glog.Infof("Snapshotter is about to create volume snapshot operation named %s", operationName)

	err := vs.runningOperation.Run(operationName, vs.getSnapshotPromoteFunc(snapshotName, snapshotSpec))

	if err != nil {
		switch {
		case goroutinemap.IsAlreadyExists(err):
			glog.V(4).Infof("operation %q is already running, skipping", operationName)
		case exponentialbackoff.IsExponentialBackoff(err):
			glog.V(4).Infof("operation %q postponed due to exponential backoff", operationName)
		default:
			glog.Errorf("Failed to schedule the operation %q: %v", operationName, err)
		}
	}
}
