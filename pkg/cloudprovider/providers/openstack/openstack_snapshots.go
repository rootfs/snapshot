/*
Copyright 2016 The Kubernetes Authors.

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

package openstack

import (
	"github.com/gophercloud/gophercloud"
	snapshots_v2 "github.com/gophercloud/gophercloud/openstack/blockstorage/v2/snapshots"

	"github.com/golang/glog"
)

type SnapshotsV2 struct {
	blockstorage *gophercloud.ServiceClient
	opts         BlockStorageOpts
}

type Snapshot struct {
	ID             string
	Name           string
	Status         string
	SourceVolumeID string
}

type SnapshotCreateOpts struct {
	VolumeID    string
	Name        string
	Description string
	Force       bool
	Metadata    map[string]string
}

type snapshotService interface {
	createSnapshot(opts SnapshotCreateOpts) (string, error)
	deleteSnapshot(snapshotName string) error
}

func (snapshots *SnapshotsV2) createSnapshot(opts SnapshotCreateOpts) (string, error) {

	create_opts := snapshots_v2.CreateOpts{
		VolumeID:    opts.VolumeID,
		Force:       false,
		Name:        opts.Name,
		Description: opts.Description,
		Metadata:    opts.Metadata,
	}

	snap, err := snapshots_v2.Create(snapshots.blockstorage, create_opts).Extract()
	if err != nil {
		return "", err
	}
	return snap.ID, nil
}

func (snapshots *SnapshotsV2) deleteSnapshot(snapshotID string) error {
	err := snapshots_v2.Delete(snapshots.blockstorage, snapshotID).ExtractErr()
	if err != nil {
		glog.Errorf("Cannot delete snapshot %s: %v", snapshotID, err)
	}

	return err
}

// Create a snapshot from the specified volume
func (os *OpenStack) CreateSnapshot(sourceVolumeID, name, description string, tags *map[string]string) (string, error) {
	snapshots, err := os.snapshotService()
	if err != nil || snapshots == nil {
		glog.Errorf("Unable to initialize cinder client for region: %s", os.region)
		return "", err
	}

	opts := SnapshotCreateOpts{
		VolumeID:    sourceVolumeID,
		Name:        name,
		Description: description,
	}
	if tags != nil {
		opts.Metadata = *tags
	}

	snapshotID, err := snapshots.createSnapshot(opts)

	if err != nil {
		glog.Errorf("Failed to snapshot volume %s : %v", sourceVolumeID, err)
		return "", err
	}

	glog.Infof("Created snapshot %v from volume: %v", snapshotID, sourceVolumeID)
	return snapshotID, nil
}

// Delete the specified snapshot
func (os *OpenStack) DeleteSnapshot(snapshotID string) error {
	snapshots, err := os.snapshotService()
	if err != nil || snapshots == nil {
		glog.Errorf("Unable to initialize cinder client for region: %s", os.region)
		return err
	}

	err = snapshots.deleteSnapshot(snapshotID)
	if err != nil {
		glog.Errorf("Cannot delete snapshot %s: %v", snapshotID, err)
	}
	return nil
}

/*
// Retrieve list of Snapshots
func (os *OpenStack) ListSnapshots() ([]Snapshot, error) {
    snapshots, err := os.snapshotService("")
	if err != nil || snapshots == nil {
		glog.Errorf("Unable to initialize cinder client for region: %s", os.region)
		return []Snapshot, err
	}
	snapshots_v2.List().Extract()
}
*/
