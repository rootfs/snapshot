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

package volume

import (
	"k8s.io/client-go/pkg/api/v1"

	tprv1 "github.com/rootfs/snapshot/pkg/apis/tpr/v1"
	"github.com/rootfs/snapshot/pkg/cloudprovider"
)

type VolumePlugin interface {
	// Init inits volume plugin
	// TODO supply a cloud provider interface for AWS/GCE/Azure/OpenStack volumes
	Init(cloudprovider.Interface)
	// SnapshotCreate creates a VolumeSnapshot from a PersistentVolumeSpec
	SnapshotCreate(*v1.PersistentVolume) (*tprv1.VolumeSnapshotDataSource, error)
	// SnapshotDelete deletes a VolumeSnapshot
	// PersistentVolume is provided for volume types, if any, that need PV Spec to delete snapshot
	SnapshotDelete(*tprv1.VolumeSnapshotDataSource, *v1.PersistentVolume) error
	// SnapshotRestore restores (promotes) a volume snapshot into a volume
	SnapshotRestore(*tprv1.VolumeSnapshotData, *v1.PersistentVolumeClaim, string, map[string]string) (*v1.PersistentVolumeSource, map[string]string, error)
	// VolumeDelete deletes a PV
	// TODO in the future pass kubernetes client for certain volumes (e.g. rbd) so they can access storage class to retrieve secret
	VolumeDelete(pv *v1.PersistentVolume) error
}
