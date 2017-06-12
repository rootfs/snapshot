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

package azure_dd

import (
	"fmt"
	//	"strconv"
	//	"strings"

	"k8s.io/client-go/pkg/api/v1"
	//	kvol "k8s.io/kubernetes/pkg/volume"

	//	"github.com/golang/glog"

	tprv1 "github.com/rootfs/snapshot/pkg/apis/tpr/v1"
	"github.com/rootfs/snapshot/pkg/cloudprovider"
	"github.com/rootfs/snapshot/pkg/cloudprovider/providers/azure"
	"github.com/rootfs/snapshot/pkg/volume"
)

type azurePlugin struct {
	cloud *azure.Cloud
}

var _ volume.VolumePlugin = &azurePlugin{}

func RegisterPlugin() volume.VolumePlugin {
	return &azurePlugin{}
}

func GetPluginName() string {
	return "azure_dd"
}

func (a *azurePlugin) Init(cloud cloudprovider.Interface) {
	a.cloud = cloud.(*azure.Cloud)
}

func (a *azurePlugin) SnapshotCreate(pv *v1.PersistentVolume) (*tprv1.VolumeSnapshotDataSource, error) {
	spec := &pv.Spec
	if spec == nil || spec.AzureDisk == nil {
		return nil, fmt.Errorf("invalid PV spec %v", spec)
	}
	uri := spec.AzureDisk.DataDiskURI
	time, err := a.cloud.CreateSnapshot(uri)
	if err != nil {
		return nil, err
	}
	return &tprv1.VolumeSnapshotDataSource{
		AzureDisk: &tprv1.AzureDiskVolumeSnapshotSource{
			SnapshotTime: *time,
		},
	}, nil
}

func (a *azurePlugin) SnapshotDelete(src *tprv1.VolumeSnapshotDataSource, _ *v1.PersistentVolume) error {
	if src == nil || src.AzureDisk == nil {
		return fmt.Errorf("invalid VolumeSnapshotDataSource: %v", src)
	}

	return nil
}

func (a *azurePlugin) DescribeSnapshot(snapshotData *tprv1.VolumeSnapshotData) (isCompleted bool, err error) {
	if snapshotData == nil || snapshotData.Spec.AzureDisk == nil {
		return false, fmt.Errorf("invalid VolumeSnapshotDataSource: %v", snapshotData)
	}
	return false, nil
}

func (a *azurePlugin) SnapshotRestore(snapshotData *tprv1.VolumeSnapshotData, pvc *v1.PersistentVolumeClaim, pvName string, parameters map[string]string) (*v1.PersistentVolumeSource, map[string]string, error) {
	return nil, nil, nil

}

func (a *azurePlugin) VolumeDelete(pv *v1.PersistentVolume) error {
	if pv == nil || pv.Spec.AzureDisk == nil {
		return fmt.Errorf("invalid  PV: %v", pv)
	}

	return nil
}
