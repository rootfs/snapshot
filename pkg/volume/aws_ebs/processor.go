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

package aws_ebs

import (
	"fmt"

	"k8s.io/client-go/pkg/api/v1"

	tprv1 "github.com/rootfs/snapshot/pkg/apis/tpr/v1"
	"github.com/rootfs/snapshot/pkg/cloudprovider"
	"github.com/rootfs/snapshot/pkg/cloudprovider/providers/aws"
	"github.com/rootfs/snapshot/pkg/volume"
)

type awsEBSPlugin struct {
	cloud *aws.Cloud
}

var _ volume.VolumePlugin = &awsEBSPlugin{}

func RegisterPlugin() volume.VolumePlugin {
	return &awsEBSPlugin{}
}

func GetPluginName() string {
	return "aws_ebs"
}

func (a *awsEBSPlugin) Init(cloud cloudprovider.Interface) {
	a.cloud = cloud.(*aws.Cloud)
}

func (a *awsEBSPlugin) SnapshotCreate(spec *v1.PersistentVolumeSpec) (*tprv1.VolumeSnapshotDataSource, error) {
	if spec == nil || spec.AWSElasticBlockStore == nil {
		return nil, fmt.Errorf("invalid PV spec %v", spec)
	}
	volumeId := spec.AWSElasticBlockStore.VolumeID
	snapshotOpt := &aws.SnapshotOptions{
		VolumeId: volumeId,
	}
	snapshotId, err := a.cloud.CreateSnapshot(snapshotOpt)
	if err != nil {
		return nil, err
	}
	return &tprv1.VolumeSnapshotDataSource{
		AWSElasticBlockStore: &tprv1.AWSElasticBlockStoreVolumeSnapshotSource{
			SnapshotID: snapshotId,
		},
	}, nil
}

func (a *awsEBSPlugin) SnapshotDelete(src *tprv1.VolumeSnapshotDataSource, _ *v1.PersistentVolume) error {
	if src == nil || src.AWSElasticBlockStore == nil {
		return fmt.Errorf("invalid VolumeSnapshotDataSource: %v", src)
	}
	snapshotId := src.AWSElasticBlockStore.SnapshotID
	_, err := a.cloud.DeleteSnapshot(snapshotId)
	if err != nil {
		return err
	}

	return nil
}
