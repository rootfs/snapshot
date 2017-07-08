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

package cinder

import (
	"fmt"
	"time"

	"k8s.io/client-go/pkg/api/v1"

	"github.com/golang/glog"

	crdv1 "github.com/rootfs/snapshot/pkg/apis/crd/v1"
	"github.com/rootfs/snapshot/pkg/cloudprovider"
	"github.com/rootfs/snapshot/pkg/cloudprovider/providers/openstack"
	"github.com/rootfs/snapshot/pkg/volume"
)

type cinderPlugin struct {
	cloud *openstack.OpenStack
}

var _ volume.VolumePlugin = &cinderPlugin{}

// Init inits volume plugin
func (c *cinderPlugin) Init(cloud cloudprovider.Interface) {
	c.cloud = cloud.(*openstack.OpenStack)
}

func RegisterPlugin() volume.VolumePlugin {
	return &cinderPlugin{}
}

func GetPluginName() string {
	return "cinder"
}

func (c *cinderPlugin) VolumeDelete(pv *v1.PersistentVolume) error {
	if pv == nil || pv.Spec.Cinder == nil {
		return fmt.Errorf("invalid Cinder PV: %v", pv)
	}
	volumeId := pv.Spec.Cinder.VolumeID
	err := c.cloud.DeleteVolume(volumeId)
	if err != nil {
		return err
	}

	return nil
}

// SnapshotCreate creates a VolumeSnapshot from a PersistentVolumeSpec
func (c *cinderPlugin) SnapshotCreate(pv *v1.PersistentVolume) (*crdv1.VolumeSnapshotDataSource, error) {
	spec := &pv.Spec
	if spec == nil || spec.Cinder == nil {
		return nil, fmt.Errorf("invalid PV spec %v", spec)
	}
	volumeID := spec.Cinder.VolumeID
	snapshotName := string(pv.Name) + fmt.Sprintf("%d", time.Now().UnixNano())
	snapshotDescription := "kubernetes snapshot"
	tags := make(map[string]string)
	glog.Infof("issuing Cinder.CreateSnapshot - SourceVol: %s, Name: %s", volumeID, snapshotName)
	snapID, err := c.cloud.CreateSnapshot(volumeID, snapshotName, snapshotDescription, tags)
	if err != nil {
		return nil, err
	}

	return &crdv1.VolumeSnapshotDataSource{
		CinderSnapshot: &crdv1.CinderVolumeSnapshotSource{
			SnapshotID: snapID,
		},
	}, nil
}

// SnapshotDelete deletes a VolumeSnapshot
// PersistentVolume is provided for volume types, if any, that need PV Spec to delete snapshot
func (c *cinderPlugin) SnapshotDelete(src *crdv1.VolumeSnapshotDataSource, _ *v1.PersistentVolume) error {
	if src == nil || src.CinderSnapshot == nil {
		return fmt.Errorf("invalid VolumeSnapshotDataSource: %v", src)
	}
	snapshotID := src.CinderSnapshot.SnapshotID
	err := c.cloud.DeleteSnapshot(snapshotID)
	if err != nil {
		return err
	}
	return nil
}

// SnapshotRestore creates a new Volume using the data on the specified Snapshot
func (c *cinderPlugin) SnapshotRestore(*crdv1.VolumeSnapshotData, *v1.PersistentVolumeClaim, string, map[string]string) (*v1.PersistentVolumeSource, map[string]string, error) {
	var err error
	var tags = make(map[string]string)
	if snapshotData == nil || snapshotData.Spec.Cinder == nil {
		return nil, nil, fmt.Errorf("failed to retrieve Snapshot spec")
	}
	if pvc == nil {
		return nil, nil, fmt.Errorf("no pvc specified")
	}

	c.cloud.CreateVolume("foo", 1, "blah", "blah", "snapid", &tags)
	return nil, nil, nil
}

// Describe an EBS volume snapshot status for create or delete.
// return status (completed or pending or error), and error
func (c *cinderPlugin) DescribeSnapshot(snapshotData *crdv1.VolumeSnapshotData) (isCompleted bool, err error) {
	return true, nil
}
