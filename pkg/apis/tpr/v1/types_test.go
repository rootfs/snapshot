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

package v1

import (
	"testing"

	core_v1 "k8s.io/client-go/pkg/api/v1"
)

func Test_GetSupportedVolumeFromSnapshotDataSpec(t *testing.T) {
	testSpec1 := VolumeSnapshotDataSpec{
		VolumeSnapshotDataSource: VolumeSnapshotDataSource{
			HostPath: &HostPathVolumeSnapshotSource{
				Path: "no_file",
			},
		},
	}

	pluginName := GetSupportedVolumeFromSnapshotDataSpec(&testSpec1)

	if pluginName != "hostPath" {
		t.Fatalf("Run failed with error: expected hostPath plugin name, got %v", pluginName)
	}

	testSpec2 := VolumeSnapshotDataSpec{}

	pluginName = GetSupportedVolumeFromSnapshotDataSpec(&testSpec2)
	if pluginName != "" {
		t.Fatalf("Run failed with error: expected no plugin name, got %v", pluginName)
	}
}

func Test_GetSupportedVolumeFromPVSpec(t *testing.T) {
	testSpec1 := core_v1.PersistentVolumeSpec{
		PersistentVolumeSource: core_v1.PersistentVolumeSource{
			HostPath: &core_v1.HostPathVolumeSource{
				Path: "no_file",
			},
		},
	}

	pluginName := GetSupportedVolumeFromPVSpec(&testSpec1)

	if pluginName != "hostPath" {
		t.Fatalf("Run failed with error: expected hostPath plugin name, got %v", pluginName)
	}

	testSpec2 := core_v1.PersistentVolumeSpec{}

	pluginName = GetSupportedVolumeFromPVSpec(&testSpec2)
	if pluginName != "" {
		t.Fatalf("Run failed with error: expected no plugin name, got %v", pluginName)
	}
}
