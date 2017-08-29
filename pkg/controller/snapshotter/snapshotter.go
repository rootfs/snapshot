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
	"time"

	"github.com/golang/glog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/rest"
	"k8s.io/kubernetes/pkg/util/goroutinemap"
	"k8s.io/kubernetes/pkg/util/goroutinemap/exponentialbackoff"

	crdv1 "github.com/rootfs/snapshot/pkg/apis/crd/v1"
	"github.com/rootfs/snapshot/pkg/controller/cache"
	"github.com/rootfs/snapshot/pkg/volume"
)

const (
	defaultExponentialBackOffOnError = true
)

// VolumeSnapshotter does the "heavy lifting": it spawns goroutines that talk to the
// backend to actually perform the operations on the storage devices.
// It creates and deletes the snapshots and promotes snapshots to volumes (PV). The create
// and delete operations need to be idempotent and count with the fact the API object writes
type VolumeSnapshotter interface {
	CreateVolumeSnapshot(snapshot *crdv1.VolumeSnapshot)
	DeleteVolumeSnapshot(snapshot *crdv1.VolumeSnapshot)
	PromoteVolumeSnapshotToPV(snapshot *crdv1.VolumeSnapshot)
	UpdateVolumeSnapshot(snapshotName string, status *[]crdv1.VolumeSnapshotCondition, bFoundVolumeSnapshotMetadata bool) (*map[string]string, error)
	UpdateVolumeSnapshotData(snapshotDataName string, status *[]crdv1.VolumeSnapshotDataCondition) error
}

type volumeSnapshotter struct {
	restClient         *rest.RESTClient
	coreClient         kubernetes.Interface
	scheme             *runtime.Scheme
	actualStateOfWorld cache.ActualStateOfWorld
	runningOperation   goroutinemap.GoRoutineMap
	volumePlugins      *map[string]volume.VolumePlugin
}

const (
	snapshotOpCreatePrefix  string = "create"
	snapshotOpDeletePrefix  string = "delete"
	snapshotOpPromotePrefix string = "promote"
	// Number of retries when we create a VolumeSnapshotData object.
	createVolumeSnapshotDataRetryCount = 5
	// Interval between retries when we create a VolumeSnapshotData object.
	createVolumeSnapshotDataInterval = 10 * time.Second
	// CloudSnapshotCreatedForVolumeSnapshotNamespaceTag is a name of a tag attached to a real snapshot in cloud
	// (e.g. AWS EBS or GCE PD) with namespace of a volumesnapshot used to create this snapshot.
	CloudSnapshotCreatedForVolumeSnapshotNamespaceTag = "kubernetes.io/created-for/snapshot/namespace"
	// CloudSnapshotCreatedForVolumeSnapshotNameTag is a name of a tag attached to a real snapshot in cloud
	// (e.g. AWS EBS or GCE PD) with name of a volumesnapshot used to create this snapshot.
	CloudSnapshotCreatedForVolumeSnapshotNameTag = "kubernetes.io/created-for/snapshot/name"
        // CloudSnapshotCreatedForVolumeSnapshotTimestampTag is a name of a tag attached to a real snapshot in cloud
        // (e.g. AWS EBS or GCE PD) with timestamp when the create snapshot request is issued.
        CloudSnapshotCreatedForVolumeSnapshotTimestampTag = "kubernetes.io/created-for/snapshot/timestamp"
)

func NewVolumeSnapshotter(
	restClient *rest.RESTClient,
	scheme *runtime.Scheme,
	clientset kubernetes.Interface,
	asw cache.ActualStateOfWorld,
	volumePlugins *map[string]volume.VolumePlugin) VolumeSnapshotter {
	return &volumeSnapshotter{
		restClient:         restClient,
		coreClient:         clientset,
		scheme:             scheme,
		actualStateOfWorld: asw,
		runningOperation:   goroutinemap.NewGoRoutineMap(defaultExponentialBackOffOnError),
		volumePlugins:      volumePlugins,
	}
}

// Helper function to get PV from VolumeSnapshot
func (vs *volumeSnapshotter) getPVFromVolumeSnapshot(snapshotName string, snapshot *crdv1.VolumeSnapshot) (*v1.PersistentVolume, error) {
	pvcName := snapshot.Spec.PersistentVolumeClaimName
	if pvcName == "" {
		return nil, fmt.Errorf("The PVC name is not specified in snapshot %s", snapshotName)
	}
	snapNameSpace, _, err := cache.GetNameAndNameSpaceFromSnapshotName(snapshotName)
	if err != nil {
		return nil, fmt.Errorf("Snapshot %s is malformed", snapshotName)
	}
	pvc, err := vs.coreClient.CoreV1().PersistentVolumeClaims(snapNameSpace).Get(pvcName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("Failed to retrieve PVC %s from the API server: %q", pvcName, err)
	}
	if pvc.Status.Phase != v1.ClaimBound {
		return nil, fmt.Errorf("The PVC %s not yet bound to a PV, will not attempt to take a snapshot yet.")
	}

	pvName := pvc.Spec.VolumeName
	pv, err := vs.coreClient.CoreV1().PersistentVolumes().Get(pvName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("Failed to retrieve PV %s from the API server: %q", pvName, err)
	}
	return pv, nil
}

// Helper function that looks up VolumeSnapshotData for a VolumeSnapshot named snapshotName
func (vs *volumeSnapshotter) getSnapshotDataFromSnapshotName(snapshotName string) *crdv1.VolumeSnapshotData {
	var snapshotDataList crdv1.VolumeSnapshotDataList
	var snapshotDataObj crdv1.VolumeSnapshotData
	var found bool

	err := vs.restClient.Get().
		Resource(crdv1.VolumeSnapshotDataResourcePlural).
		Namespace(v1.NamespaceDefault).
		Do().Into(&snapshotDataList)
	if err != nil {
		glog.Errorf("Error retrieving the VolumeSnapshotData objects from API server: %v", err)
		return nil
	}
	if len(snapshotDataList.Items) == 0 {
		glog.Errorf("Error: no VolumeSnapshotData objects found on the API server")
		return nil
	}
	for _, snapData := range snapshotDataList.Items {
		if snapData.Spec.VolumeSnapshotRef != nil {
			name := snapData.Spec.VolumeSnapshotRef.Namespace + "/" + snapData.Spec.VolumeSnapshotRef.Name
			if name == snapshotName || snapData.Spec.VolumeSnapshotRef.Name == snapshotName {
				snapshotDataObj = snapData
				found = true
				break
			}
		}
	}
	if !found {
		glog.Errorf("Error: no VolumeSnapshotData for VolumeSnapshot %s found", snapshotName)
		return nil
	}

	return &snapshotDataObj
}

// Query status of the snapshot from plugin and update the status of VolumeSnapshot and VolumeSnapshotData
// if needed. Finish waiting when the snapshot becomes available/ready or error.
func (vs *volumeSnapshotter) waitForPluginSnapshot(snapshotName string, snapshot *crdv1.VolumeSnapshot) error {
        var snapshotDataObj crdv1.VolumeSnapshotData
        snapshotDataName := snapshot.Spec.SnapshotDataName
        glog.Infof("In waitForPluginSnapshot: %s", snapshotName)
        err := vs.restClient.Get().
                Name(snapshotDataName).
                Resource(crdv1.VolumeSnapshotDataResourcePlural).
                Namespace(v1.NamespaceDefault).
                Do().Into(&snapshotDataObj)
        if err != nil {
                return err
        }

        pv, err := vs.getPVFromVolumeSnapshot(snapshotName, snapshot)
        if err != nil {
                return err
        }
        spec := &pv.Spec
        volumeType := crdv1.GetSupportedVolumeFromPVSpec(spec)
        if len(volumeType) == 0 {
                return fmt.Errorf("unsupported volume type found in PV %#v", spec)
        }
        plugin, ok := (*vs.volumePlugins)[volumeType]
        if !ok {
                return fmt.Errorf("%s is not supported volume for %#v", volumeType, spec)
        }

	var completed bool = false
	// Wait until the snapshot is successfully created by the plugin or an error occurs that
	// fails the snapshot creation.
	for {
		completed, err = plugin.DescribeSnapshot(&snapshotDataObj)
                if !completed {
			glog.Infof("Snapshot %s creation is not complete yet. Retrying...", snapshotName)
			if err != nil {
				// Don't bail out here. The error message could be "current snapshot status is: creating".
				glog.Infof("Message received when creating snapshot %s: %v", snapshotName, err)
			}
			time.Sleep(createVolumeSnapshotDataInterval)
		} else {
			// TODO(xyang): DescribeSnapshot should return status of the snapshot.
			// If status is Pending/Creating, update VolumeSnapshot status to Pending;
			// If status is Available/Ready, update VolumeSnapshot status to Ready;
			// If status is Error, update VolumeSnapshot status to Error.
			glog.Infof("waitForPluginSnapshot: Snapshot %s creation is complete.", snapshotName)
			break
		}
	}

	if completed {
		glog.Infof("Snapshot %s successfully created and it is ready.", snapshotName)
		// UpdateVolmeSnapshot
                snapConditions := []crdv1.VolumeSnapshotCondition{
                        {
                                Type:               crdv1.VolumeSnapshotConditionReady,
                                Status:             v1.ConditionTrue,
                                Message:            "Snapshot created succsessfully",
                                LastTransitionTime: metav1.Now(),
                        },
                }
                _, err = vs.UpdateVolumeSnapshot(snapshotName, &snapConditions, true)
                if err != nil {
                        glog.Errorf("Error updating volume snapshot %s: %v", snapshotName, err)
                        // NOTE(xyang): Return error if failed to update VolumeSnapshot after
                        // create snapshot request is sent to the plugin and VolumeSnapshotData is updated
                        return fmt.Errorf("Failed to update VolumeSnapshot for snapshot %s: %v", snapshotName, err)
                }

                snapDataConditions := []crdv1.VolumeSnapshotDataCondition{
                        {
                                Type:               crdv1.VolumeSnapshotDataConditionReady,
                                Status:             v1.ConditionTrue,
                                Message:            "Snapshot data created succsessfully",
                                LastTransitionTime: metav1.Now(),
                        },
                }

                // Update VolumeSnapshotData status
                err = vs.UpdateVolumeSnapshotData(snapshotDataName, &snapDataConditions)
                if err != nil {
                        return fmt.Errorf("Error update snapshotData object %s: %v", snapshotName, err)
                }

		glog.Infof("waitForPluginSnapshot: Snapshot %s created successfully. Adding it to Actual State of World.", snapshotName)
		vs.actualStateOfWorld.AddSnapshot(snapshot)
		return nil
        }

	return fmt.Errorf("Snapshot %s is NOT completed successfully.")
}

func (vs *volumeSnapshotter) updateSnapshotDataStatus(snapshotName string, snapshot *crdv1.VolumeSnapshot) error {
	var snapshotDataObj crdv1.VolumeSnapshotData
	snapshotDataName := snapshot.Spec.SnapshotDataName
	glog.Infof("In UpdateVolumeSnapshotData")
	err := vs.restClient.Get().
		Name(snapshotDataName).
		Resource(crdv1.VolumeSnapshotDataResourcePlural).
		Namespace(v1.NamespaceDefault).
		Do().Into(&snapshotDataObj)
	if err != nil {
		return err
	}

	if len(snapshotDataObj.Status.Conditions) < 1 ||
		snapshotDataObj.Status.Conditions[0].Type != crdv1.VolumeSnapshotDataConditionReady {
		pv, err := vs.getPVFromVolumeSnapshot(snapshotName, snapshot)
		if err != nil {
			return err
		}
		spec := &pv.Spec
		volumeType := crdv1.GetSupportedVolumeFromPVSpec(spec)
		if len(volumeType) == 0 {
			return fmt.Errorf("unsupported volume type found in PV %#v", spec)
		}
		plugin, ok := (*vs.volumePlugins)[volumeType]
		if !ok {
			return fmt.Errorf("%s is not supported volume for %#v", volumeType, spec)
		}
		completed, err := plugin.DescribeSnapshot(&snapshotDataObj)
		if !completed {
			return fmt.Errorf("snapshot is not completed yet: %v", err)
		}
		glog.Infof("snapshot successfully created, updating VolumeSnapshotData status for %s", snapshotDataName)
		status := []crdv1.VolumeSnapshotDataCondition{
			{
				Type:               crdv1.VolumeSnapshotDataConditionReady,
				Status:             v1.ConditionTrue,
				Message:            "Snapshot data created succsessfully",
				LastTransitionTime: metav1.Now(),
			},
		}

		// Update VolumeSnapshotData status
		err = vs.UpdateVolumeSnapshotData(snapshotDataName, &status)
		if err != nil {
			return fmt.Errorf("Error update snapshotData object %s: %v", snapshotName, err)
		}

	}
	vs.actualStateOfWorld.AddSnapshot(snapshot)
	return nil
}

// This is the function responsible for determining the correct volume plugin to use,
// asking it to make a snapshot and assigning it some name that it returns to the caller.
func (vs *volumeSnapshotter) takeSnapshot(pv *v1.PersistentVolume, tags *map[string]string) (*crdv1.VolumeSnapshotDataSource, error) {
	spec := &pv.Spec
	volumeType := crdv1.GetSupportedVolumeFromPVSpec(spec)
	if len(volumeType) == 0 {
		return nil, fmt.Errorf("unsupported volume type found in PV %#v", spec)
	}
	plugin, ok := (*vs.volumePlugins)[volumeType]
	if !ok {
		return nil, fmt.Errorf("%s is not supported volume for %#v", volumeType, spec)
	}

	snap, err := plugin.SnapshotCreate(pv, tags)
	if err != nil {
		glog.Warningf("failed to snapshot %#v, err: %v", spec, err)
	} else {
		glog.Infof("snapshot created: %v", snap)
		return snap, nil
	}
	return nil, nil
}

// This is the function responsible for determining the correct volume plugin to use,
// asking it to make a snapshot and assigning it some name that it returns to the caller.
func (vs *volumeSnapshotter) deleteSnapshot(spec *v1.PersistentVolumeSpec, source *crdv1.VolumeSnapshotDataSource) error {
	volumeType := crdv1.GetSupportedVolumeFromPVSpec(spec)
	if len(volumeType) == 0 {
		return fmt.Errorf("unsupported volume type found in PV %#v", spec)
	}
	plugin, ok := (*vs.volumePlugins)[volumeType]
	if !ok {
		return fmt.Errorf("%s is not supported volume for %#v", volumeType, spec)
	}
	err := plugin.SnapshotDelete(source, nil /* *v1.PersistentVolume */)
	if err != nil {
		return fmt.Errorf("failed to delete snapshot %#v, err: %v", source, err)
	}
	glog.Infof("snapshot %#v deleted", source)

	return nil
}

// Below are the closures meant to build the functions for the GoRoutineMap operations.

func (vs *volumeSnapshotter) getSnapshotCreateFunc(snapshotName string, snapshot *crdv1.VolumeSnapshot) func() error {
	// Create a snapshot:
	// 1. Check the status of VolumeSnapshot object
	//	a. Condition Ready or Error -> return nil
	//	b. Condition Pending (Status True or Unknown) -> step 4
	//	c. Else -> step 2
	// 2. Retrieve metadata from VolumeSnapshot object if existing;
	//	otherwise create metadata and update VolumeSnapshot object.
	//	If metadata pre-existing, find snapshot through volume plugin;
	//	If metadata newly created, take snapshot through volume plugin.
	// 3. Create VolumeSnapshotData object with a reference to the snapshot by the plugin.
	// 	Update VolumeSnapshot object status and also bind VolumeSnapshotData to VolumeSnapshot.
	//	Set status to Pending.
	//	TODO(xyang): Set condition status to True or Unknown based on status from
	//	querying the snapshot through the volume plugin.
	// 4. Query the volume plugin for the status of the snapshot with snapshot id
	//	from VolumeSnapshotData object until snapshot is Ready.
	//	Update VolumeSnapshot and VolumeSnapshotData, setting status to Ready (True).
	//	Add the snapshot to ActualStateOfWorld.
	return func() error {
		glog.Infof("Enter getSnapshotCreateFunc: snapshotName %s snapshot [%#v]", snapshotName, snapshot)
		//if snapshot.Spec.SnapshotDataName != "" {
		//	// update snapshot data status and if complete, adds to asw
		//	return vs.updateSnapshotDataStatus(snapshotName, snapshot)
		//}

		// Check VolumeSnapshot object status
		//volSnapshotObj, err := vs.getVolumeSnapshot(snapshotName)
		index := len(snapshot.Status.Conditions) - 1
		if len(snapshot.Status.Conditions) > 0 &&
			((snapshot.Status.Conditions[index].Type == crdv1.VolumeSnapshotConditionReady &&
			snapshot.Status.Conditions[index].Status == v1.ConditionTrue) ||
			snapshot.Status.Conditions[index].Type == crdv1.VolumeSnapshotConditionError) {
			glog.Infof("getSnapshotCreateFunc: Snapshot %s is Ready.", snapshotName)
			return nil
		} else if len(snapshot.Status.Conditions) > 0 &&
			snapshot.Status.Conditions[index].Type == crdv1.VolumeSnapshotConditionPending &&
			(snapshot.Status.Conditions[index].Status == v1.ConditionTrue ||
			snapshot.Status.Conditions[index].Status == v1.ConditionUnknown) {
                        glog.Infof("getSnapshotCreateFunc: Snapshot %s is Pending.", snapshotName)
			// Query the volume plugin for the status of the snapshot with snapshot id
			// from VolumeSnapshotData object.
			// Add snapshot to Actual State of World when snapshot is Ready.
			err := vs.waitForPluginSnapshot(snapshotName, snapshot)
			if err != nil {
				return fmt.Errorf("Failed to create snapshot %s.", snapshotName)
			}
                        glog.Infof("getSnapshotCreateFunc: Snapshot %s created successfully.", snapshotName)
                        return nil
                } else {
			glog.Infof("getSnapshotCreateFunc: Creating snapshot %s ...", snapshotName)
			pv, err := vs.getPVFromVolumeSnapshot(snapshotName, snapshot)
			if err != nil {
				return err
			}
			pvName := pv.Name

			// Retrieve metadata from VolumeSnapshot object if existing,
			// otherwise create metadata and update VolumeSnapshot object.

			// Look for VolumeSnapshot metadata ...
			var tags *map[string]string
			var bFoundVolumeSnapshotMetadata bool = false
			if snapshot.Metadata.Name != "" && snapshot.Metadata.Namespace != "" {
				if snapshot.Metadata.Labels != nil {
					_, ok := snapshot.Metadata.Labels["Timestamp"]
					if ok {
						glog.Infof("Found Metadata Labels in VolumeSnapshot object: [%#v].", snapshot.Metadata.Labels)
						bFoundVolumeSnapshotMetadata = true
					}
				}
			}

			// If metadata not found, create it in VolumeSnapshot now
			if !bFoundVolumeSnapshotMetadata {
				tags, err = vs.UpdateVolumeSnapshot(snapshotName, nil, bFoundVolumeSnapshotMetadata)
			}

			var snapshotDataSource *crdv1.VolumeSnapshotDataSource = nil
			// Look for snapshot by the plugin if metadata already existed in VolumeSnapshot
			if bFoundVolumeSnapshotMetadata {
				spec := &pv.Spec
				volumeType := crdv1.GetSupportedVolumeFromPVSpec(spec)
				if len(volumeType) == 0 {
					return fmt.Errorf("Unsupported volume type found in PV %#v", spec)
				}
				plugin, ok := (*vs.volumePlugins)[volumeType]
				if !ok {
					return fmt.Errorf("%s is not supported volume for %#v", volumeType, spec)
				}

				// Check whether the real snapshot is already created by the plugin
				glog.Infof("getSnapshotCreateFunc: FindSnapshot %s ...", snapshotName)
				// TODO(xyang): FindSnapshot should return status of the snapshot through the plugin.
				snapshotDataSource, err = plugin.FindSnapshot(tags)
				if err != nil {
					// Don't bail out because the error could be "error requesting snapshot : Resource not found".
					glog.Infof("Error occurred when looking for snapshot by metadata: %v", err)
				}
			}

			// If snapshot not created yet, create it now; otherwise use the existing snapshot
			if snapshotDataSource == nil {
				glog.Infof("getSnapshotCreateFunc: takeSnapshot %s ...", snapshotName)
				 // TODO(xyang): takeSnapshot should return status of the snapshot through the plugin.
				snapshotDataSource, err = vs.takeSnapshot(pv, tags)
				if err != nil || snapshotDataSource == nil {
					return fmt.Errorf("Failed to take snapshot of the volume %s: %q", pvName, err)
				}
			}

			// Check whether the VolumeSnapshotData object is already created
			snapshotDataObj := vs.getSnapshotDataFromSnapshotName(snapshotName)

			// VolumeSnapshotData object not created yet. Create it now as snapshot has already been created.
			if snapshotDataObj == nil {
				readyCondition := crdv1.VolumeSnapshotDataCondition{
					Type:    crdv1.VolumeSnapshotDataConditionPending,
					Status:  v1.ConditionTrue,
					Message: "Snapshot data is being created",
				}
				snapName := "k8s-volume-snapshot-" + string(uuid.NewUUID())

				snapshotData := &crdv1.VolumeSnapshotData{
					Metadata: metav1.ObjectMeta{
						Name: snapName,
					},
					Spec: crdv1.VolumeSnapshotDataSpec{
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
					Status: crdv1.VolumeSnapshotDataStatus{
						Conditions: []crdv1.VolumeSnapshotDataCondition{
							readyCondition,
						},
					},
				}
				var result crdv1.VolumeSnapshotData
				for i := 0; i < createVolumeSnapshotDataRetryCount; i++ {
					err = vs.restClient.Post().
						Resource(crdv1.VolumeSnapshotDataResourcePlural).
						Namespace(v1.NamespaceDefault).
						Body(snapshotData).
						Do().Into(&result)
					if err != nil {
						// Re-Try it as errors writing to the API server are common
						time.Sleep(createVolumeSnapshotDataInterval)
					} else {
						break
					}
				}

				if err != nil {
					glog.Errorf("Error creating the VolumeSnapshotData %s: %v", snapshotName, err)
					// Don't proceed to create snapshot using the plugin due to error creating
					// VolumeSnapshotData
					return fmt.Errorf("Failed to create the VolumeSnapshotData %s for snapshot %s", snapName, snapshotName)
				}
			}

			// Update the VolumeSnapshot object with status. This should bind VolumeSnapshotData to VolumeSnapshot object
			snapConditions := []crdv1.VolumeSnapshotCondition{
				{
					Type:               crdv1.VolumeSnapshotConditionPending,
					Status:             v1.ConditionTrue,
					Message:            "Snapshot created succsessfully",
					LastTransitionTime: metav1.Now(),
				},
			}
			_, err = vs.UpdateVolumeSnapshot(snapshotName, &snapConditions, true)
			if err != nil {
				glog.Errorf("Error updating volume snapshot %s: %v", snapshotName, err)
				// NOTE(xyang): Return error if failed to update VolumeSnapshot after
				// create snapshot request is sent to the plugin and VolumeSnapshotData is updated
				return fmt.Errorf("Failed to update VolumeSnapshot for snapshot %s", snapshotName)
			}

			return nil
		}
	}
}

func (vs *volumeSnapshotter) getSnapshotDeleteFunc(snapshotName string, snapshot *crdv1.VolumeSnapshot) func() error {
	// Delete a snapshot
	// 1. Find the SnapshotData corresponding to Snapshot
	//   1a: Not found => finish (it's been deleted already)
	// 2. Ask the backend to remove the snapshot device
	// 3. Delete the SnapshotData object
	// 4. Remove the Snapshot from ActualStateOfWorld
	// 5. Finish
	return func() error {
		// TODO: get VolumeSnapshotDataSource from associated VolumeSnapshotData
		// then call volume delete snapshot method to delete the ot
		snapshotDataObj := vs.getSnapshotDataFromSnapshotName(snapshotName)
		if snapshotDataObj == nil {
			return fmt.Errorf("Error getting VolumeSnapshotData for VolumeSnapshot %s", snapshotName)
		}

		pv, err := vs.getPVFromVolumeSnapshot(snapshotName, snapshot)
		if err != nil {
			return err
		}

		err = vs.deleteSnapshot(&pv.Spec, &snapshotDataObj.Spec.VolumeSnapshotDataSource)
		if err != nil {
			return fmt.Errorf("Failed to delete snapshot %s: %q", snapshotName, err)
		}

		snapshotDataName := snapshotDataObj.Metadata.Name
		var result metav1.Status
		err = vs.restClient.Delete().
			Name(snapshotDataName).
			Resource(crdv1.VolumeSnapshotDataResourcePlural).
			Namespace(v1.NamespaceDefault).
			Do().Into(&result)
		if err != nil {
			return fmt.Errorf("Failed to delete VolumeSnapshotData %s from API server: %q", snapshotDataName, err)
		}

		vs.actualStateOfWorld.DeleteSnapshot(snapshotName)

		return nil
	}
}

func (vs *volumeSnapshotter) getSnapshotPromoteFunc(snapshotName string, snapshot *crdv1.VolumeSnapshot) func() error {
	// Promote snapshot to a PVC
	// 1. We have a PVC referencing a Snapshot object
	// 2. Find the SnapshotData corresponding to tha Snapshot
	// 3. Ask the backend to give us a device (PV) made from the snapshot device
	// 4. Bind it to the PVC
	// 5. Finish
	return func() error { return nil }
}

func (vs *volumeSnapshotter) CreateVolumeSnapshot(snapshot *crdv1.VolumeSnapshot) {
	snapshotName := cache.MakeSnapshotName(snapshot.Metadata.Namespace, snapshot.Metadata.Name)
	operationName := snapshotOpCreatePrefix + snapshotName + snapshot.Spec.PersistentVolumeClaimName
	//glog.Infof("Snapshotter is about to create volume snapshot operation named %s, spec %#v", operationName, snapshot.Spec)

	err := vs.runningOperation.Run(operationName, vs.getSnapshotCreateFunc(snapshotName, snapshot))

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

func (vs *volumeSnapshotter) DeleteVolumeSnapshot(snapshot *crdv1.VolumeSnapshot) {
	snapshotName := cache.MakeSnapshotName(snapshot.Metadata.Namespace, snapshot.Metadata.Name)
	operationName := snapshotOpDeletePrefix + snapshotName + snapshot.Spec.PersistentVolumeClaimName
	glog.Infof("Snapshotter is about to create volume snapshot operation named %s", operationName)

	err := vs.runningOperation.Run(operationName, vs.getSnapshotDeleteFunc(snapshotName, snapshot))

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

func (vs *volumeSnapshotter) PromoteVolumeSnapshotToPV(snapshot *crdv1.VolumeSnapshot) {
	snapshotName := cache.MakeSnapshotName(snapshot.Metadata.Namespace, snapshot.Metadata.Name)
	operationName := snapshotOpPromotePrefix + snapshotName + snapshot.Spec.PersistentVolumeClaimName
	glog.Infof("Snapshotter is about to create volume snapshot operation named %s", operationName)

	err := vs.runningOperation.Run(operationName, vs.getSnapshotPromoteFunc(snapshotName, snapshot))

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

func (vs *volumeSnapshotter) getVolumeSnapshot(snapshotName string) (*crdv1.VolumeSnapshot, error) {
        var snapshotObj crdv1.VolumeSnapshot

        glog.Infof("In getVolumeSnapshot %s", snapshotName)
        // Get a fresh copy of the VolumeSnapshotData from the API server
        // Get a fresh copy of the VolumeSnapshot from the API server
        snapNameSpace, snapName, err := cache.GetNameAndNameSpaceFromSnapshotName(snapshotName)
        if err != nil {
                return nil, fmt.Errorf("Error getting namespace and name from VolumeSnapshot name %s: %v", snapshotName, err)
        }
        glog.Infof("getVolumeSnapshot: Namespace %s Name %s", snapNameSpace, snapName)
        err = vs.restClient.Get().
                Name(snapName).
                Resource(crdv1.VolumeSnapshotResourcePlural).
                Namespace(snapNameSpace).
                Do().Into(&snapshotObj)
        if err != nil {
                return nil, fmt.Errorf("Error getting VolumeSnapshot object %s. Error: %v", snapshotName, err)
        }

	return &snapshotObj, nil
}

func (vs *volumeSnapshotter) UpdateVolumeSnapshot(snapshotName string, status *[]crdv1.VolumeSnapshotCondition, bFoundVolumeSnapshotMetadata bool) (*map[string]string, error) {
	var snapshotObj crdv1.VolumeSnapshot

	glog.Infof("In UpdateVolumeSnapshot")
	// Get a fresh copy of the VolumeSnapshot from the API server
	snapNameSpace, snapName, err := cache.GetNameAndNameSpaceFromSnapshotName(snapshotName)
	if err != nil {
		return nil, fmt.Errorf("Error getting namespace and name from VolumeSnapshot name %s: %v", snapshotName, err)
	}
	glog.Infof("UpdateVolumeSnapshot: Namespace %s Name %s", snapNameSpace, snapName)
	err = vs.restClient.Get().
		Name(snapName).
		Resource(crdv1.VolumeSnapshotResourcePlural).
		Namespace(snapNameSpace).
		Do().Into(&snapshotObj)

	objCopy, err := vs.scheme.DeepCopy(&snapshotObj)
	if err != nil {
		return nil, fmt.Errorf("Error copying snapshot object %s object from API server: %v", snapshotName, err)
	}
	snapshotCopy, ok := objCopy.(*crdv1.VolumeSnapshot)
	if !ok {
		return nil, fmt.Errorf("Error: expecting type VolumeSnapshot but received type %T", objCopy)
	}

	// VolumeSnapshot Metadata Labels not found. Update VolumeSnapshot by creating the Metadata Lables...
	if bFoundVolumeSnapshotMetadata == false {
                tags := make(map[string]string)
		tags["Timestamp"] = fmt.Sprintf("%d", time.Now().UnixNano())
		snapshotCopy.Metadata.Labels = tags
                glog.Infof("UpdateVolumeSnapshot [%s]: Metadata Name: %s Metadata Namespace: %s Setting tags in Metadata Labels: [%#v].", snapshotName, snapshotCopy.Metadata.Name, snapshotCopy.Metadata.Namespace, snapshotCopy.Metadata.Labels)
	} else {
		glog.Infof("UpdateVolumeSnapshot: Setting VolumeSnapshotData name in VolumeSnapshotSpec of VolumeSnapshot object")
                snapshotDataObj := vs.getSnapshotDataFromSnapshotName(snapshotName)
                if snapshotDataObj == nil {
			//TODO(xyang): Should we return error here?
                        return nil, fmt.Errorf("Error getting VolumeSnapshotData for VolumeSnapshot %s", snapshotName)
                }

		snapshotCopy.Spec.SnapshotDataName = snapshotDataObj.Metadata.Name
		/*snapshotCopy.Status.Conditions = []crdv1.VolumeSnapshotCondition{
			{
				Type:               crdv1.VolumeSnapshotConditionPending,//crdv1.VolumeSnapshotConditionReady,
				Status:             v1.ConditionTrue,
				Message:            "Snapshot created succsessfully",
				LastTransitionTime: metav1.Now(),
			},
		}*/
		if status != nil {
			glog.Infof("UpdateVolumeSnapshot: Setting Status in VolumeSnapshot object")
			snapshotCopy.Status.Conditions = *status
		}
	}
	glog.Infof("Updating VolumeSnapshot object [%#v]", snapshotCopy)
	// TODO: Make diff of the two objects and then use restClient.Patch to update it
	var result crdv1.VolumeSnapshot
	err = vs.restClient.Put().
		Name(snapName).
		Resource(crdv1.VolumeSnapshotResourcePlural).
		Namespace(snapNameSpace).
		Body(snapshotCopy).
		Do().Into(&result)
	if err != nil {
		return nil, fmt.Errorf("Error updating snapshot object %s on the API server: %v", snapshotName, err)
	}

	cloudTags := make(map[string]string)
        cloudTags[CloudSnapshotCreatedForVolumeSnapshotNamespaceTag] = result.Metadata.Namespace
        cloudTags[CloudSnapshotCreatedForVolumeSnapshotNameTag] = result.Metadata.Name
        cloudTags[CloudSnapshotCreatedForVolumeSnapshotTimestampTag] = result.Metadata.Labels["Timestamp"]

        glog.Infof("UpdateVolumeSnapshot: returning cloudTags [%#v]", cloudTags)
	return &cloudTags, nil
}

func (vs *volumeSnapshotter) UpdateVolumeSnapshotData(snapshotDataName string, status *[]crdv1.VolumeSnapshotDataCondition) error {
	var snapshotDataObj crdv1.VolumeSnapshotData

	glog.Infof("In UpdateVolumeSnapshotData")
	err := vs.restClient.Get().
		Name(snapshotDataName).
		Resource(crdv1.VolumeSnapshotDataResourcePlural).
		Namespace(v1.NamespaceDefault).
		Do().Into(&snapshotDataObj)

	objCopy, err := vs.scheme.DeepCopy(&snapshotDataObj)
	if err != nil {
		return fmt.Errorf("Error copying snapshot data object %s object from API server: %v", snapshotDataName, err)
	}
	snapshotDataCopy, ok := objCopy.(*crdv1.VolumeSnapshotData)
	if !ok {
		return fmt.Errorf("Error: expecting type VolumeSnapshotData but received type %T", objCopy)
	}

	snapshotDataCopy.Status.Conditions = *status
	glog.Infof("Updating VolumeSnapshotData object")
	// TODO: Make diff of the two objects and then use restClient.Patch to update it
	var result crdv1.VolumeSnapshotData
	err = vs.restClient.Put().
		Name(snapshotDataName).
		Resource(crdv1.VolumeSnapshotDataResourcePlural).
		Namespace(v1.NamespaceDefault).
		Body(snapshotDataCopy).
		Do().Into(&result)
	if err != nil {
		return fmt.Errorf("Error updating snapshotdata object %s on the API server: %v", snapshotDataName, err)
	}
	return nil
}
