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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	core_v1 "k8s.io/client-go/pkg/api/v1"
)

const (
	VolumeSnapshotResourcePlural = "volumesnapshots"
)

type VolumeSnapshotStatus struct {
	// The time the snapshot was successfully created
	// +optional
	CreationTimestamp metav1.Time `json:"creationTimestamp" protobuf:"bytes,1,opt,name=creationTimestamp"`

	// Representes the lates available observations about the volume snapshot
	Conditions []VolumeSnapshotCondition `json:"conditions" protobuf:"bytes,2,rep,name=conditions"`
}

type VolumeSnapshotConditionType string

// These are valid conditions of a volume snapshot.
const (
	// VolumeSnapshotReadey is added when the snapshot has been successfully created and is ready to be used.
	VolumeSnapshotConditionReady VolumeSnapshotConditionType = "Ready"
)

// VolumeSnapshot Condition describes the state of a volume snapshot  at a certain point.
type VolumeSnapshotCondition struct {
	// Type of replication controller condition.
	Type VolumeSnapshotConditionType `json:"type" protobuf:"bytes,1,opt,name=type,casttype=VolumeSnapshotConditionType"`
	// Status of the condition, one of True, False, Unknown.
	Status core_v1.ConditionStatus `json:"status" protobuf:"bytes,2,opt,name=status,casttype=ConditionStatus"`
	// The last time the condition transitioned from one status to another.
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime" protobuf:"bytes,3,opt,name=lastTransitionTime"`
	// The reason for the condition's last transition.
	// +optional
	Reason string `json:"reason" protobuf:"bytes,4,opt,name=reason"`
	// A human readable message indicating details about the transition.
	// +optional
	Message string `json:"message" protobuf:"bytes,5,opt,name=message"`
}

// +genclient=true

// The volume snapshot object accessible to the user. Upon succesful creation of the actual
// snapshot by the volume provider it is bound to the corresponding VolumeSnapshotData through
// the VolumeSnapshotSpec
type VolumeSnapshot struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`

	// Spec represents the desired state of the snapshot
	// +optional
	Spec VolumeSnapshotSpec `json:"spec" protobuf:"bytes,2,opt,name=spec"`

	// Status represents the latest observer state of the snapshot
	// +optional
	Status VolumeSnapshotStatus `json:"status" protobuf:"bytes,3,opt,name=status"`
}

type VolumesnapshotList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata"`
	Items           []VolumeSnapshot `json:"items"`
}

// The desired state of the volume snapshot
type VolumeSnapshotSpec struct {
	// PersistentVolumeClaimName is the name of the PVC being snapshotted
	PersistentVolumeClaimName string `json:"persistentVolumeClaimName" protobuf:"bytes,1,opt,name=persistentVolumeClaimName"`

	// SnapshotDataName binds the VolumeSnapshot object with the VolumeSnapshotData
	SnapshotDataName string `json:"snapshotDataName" protobuf:"bytes,2,opt,name=snapshotDataName"`
}

// The actual state of the volume snapshot
type VolumeSnapshotDataStatus struct {
	// The time the snapshot was successfully created
	// +optional
	CreationTimestamp metav1.Time `json:"creationTimestamp" protobuf:"bytes,1,opt,name=creationTimestamp"`

	// Representes the lates available observations about the volume snapshot
	Conditions []VolumeSnapshotDataCondition `json:"conditions" protobuf:"bytes,2,rep,name=conditions"`
}

type VolumeSnapshotDataList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata"`
	Items           []VolumeSnapshot `json:"items"`
}

type VolumeSnapshotDataConditionType string

// These are valid conditions of a volume snapshot.
const (
	// VolumeSnapshotDataReady is added when the on-disk snapshot has been successfully created.
	VolumeSnapshotDataConditionReady VolumeSnapshotDataConditionType = "Ready"
)

// VolumeSnapshot Condition describes the state of a volume snapshot  at a certain point.
type VolumeSnapshotDataCondition struct {
	// Type of volume snapshot condition.
	Type VolumeSnapshotDataConditionType `json:"type" protobuf:"bytes,1,opt,name=type,casttype=VolumeSnapshotDataConditionType"`
	// Status of the condition, one of True, False, Unknown.
	Status core_v1.ConditionStatus `json:"status" protobuf:"bytes,2,opt,name=status,casttype=ConditionStatus"`
	// The last time the condition transitioned from one status to another.
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime" protobuf:"bytes,3,opt,name=lastTransitionTime"`
	// The reason for the condition's last transition.
	// +optional
	Reason string `json:"reason" protobuf:"bytes,4,opt,name=reason"`
	// A human readable message indicating details about the transition.
	// +optional
	Message string `json:"message" protobuf:"bytes,5,opt,name=message"`
}

// +genclient=true
// +nonNamespaced=true

// VolumeSnapshotData represents the actual "on-disk" snapshot object
type VolumeSnapshotData struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`

	// Spec represents the desired state of the snapshot
	// +optional
	Spec VolumeSnapshotDataSpec `json:"spec" protobuf:"bytes,2,opt,name=spec"`

	// Status represents the latest observed state of the snapshot
	// +optional
	Status VolumeSnapshotDataStatus `json:"status" protobuf:"bytes,3,opt,name=status"`
}

// The desired state of the volume snapshot
type VolumeSnapshotDataSpec struct {
	// Source represents the location and type of the volume snapshot
	VolumeSnapshotDataSource `json:",inline" protobuf:"bytes,1,opt,name=volumeSnapshotDataSource"`

	// VolumeSnapshotRef is part of bi-directional binding between VolumeSnapshot
	// and VolumeSnapshotData
	// +optional
	VolumeSnapshotRef *core_v1.ObjectReference `json:"volumeSnapshotRef" protobuf:"bytes,2,opt,name=volumeSnapshotRef"`

	// PersistentVolumeRef represents the PersistentVolume that the snapshot has been
	// taken from
	// +optional
	PersistentVolumeRef *core_v1.ObjectReference `json:"persistentVolumeRef" protobuf:"bytes,3,opt,name=persistentVolumeRef"`
}

// Represents the actual location and type of the snapshot. Only one of its members may be specified.
type VolumeSnapshotDataSource struct {
	// HostPath represents a directory on the host.
	// Provisioned by a developer or tester.
	// This is useful for single-node development and testing only!
	// On-host storage is not supported in any way and WILL NOT WORK in a multi-node cluster.
	// More info: https://kubernetes.io/docs/concepts/storage/volumes#hostpath
	// +optional
	HostPath *core_v1.HostPathVolumeSource `json:"hostPath" protobuf:"bytes,3,opt,name=hostPath"`
}
