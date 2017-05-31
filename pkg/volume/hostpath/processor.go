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

package hostpath

import (
	"k8s.io/apimachinery/pkg/util/uuid"
	"os/exec"

	tprv1 "github.com/rootfs/snapshot/pkg/apis/tpr/v1"
)

const depot = "/tmp/"

func Snapshot(path string) (*tprv1.VolumeSnapshotDataSource, error) {
	file := depot + string(uuid.NewUUID()) + ".tgz"
	cmd := exec.Command("tar", "czvf", file, path)
	res := &tprv1.VolumeSnapshotDataSource{
		HostPath: &tprv1.HostPathVolumeSnapshotSource{
			Path: file,
		},
	}
	return res, cmd.Run()
}
