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
	VolumeSnapshotResource = "snapshot"
)

type VolumeSnapshotStatus struct {
	// The time the snapshot was successfully created
	// +optional
	CreationTimestamp metav1.Time

	// Representes the lates available observations about the volume snapshot
	Conditions []VolumeSnapshotCondition
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
	Type VolumeSnapshotConditionType
	// Status of the condition, one of True, False, Unknown.
	Status core_v1.ConditionStatus
	// The last time the condition transitioned from one status to another.
	// +optional
	LastTransitionTime metav1.Time
	// The reason for the condition's last transition.
	// +optional
	Reason string
	// A human readable message indicating details about the transition.
	// +optional
	Message string
}

// The volume snapshot object accessible to the user. Upon succesful creation of the actual
// snapshot by the volume provider it is bound to the corresponding VolumeSnapshotData through
// the VolumeSnapshotSpec
type VolumeSnapshot struct {
	metav1.TypeMeta
	// +optional
	metav1.ObjectMeta

	// Spec represents the desired state of the snapshot
	// +optional
	Spec VolumeSnapshotSpec

	// Status represents the latest observer state of the snapshot
	// +optional
	Status VolumeSnapshotStatus
}

type VolumeSnapshotList struct {
	metav1.TypeMeta
	// +optional
	metav1.ListMeta
	Items []VolumeSnapshot
}

// The desired state of the volume snapshot
type VolumeSnapshotSpec struct {
	// PersistentVolumeClaimName is the name of the PVC being snapshotted
	PersistentVolumeClaimName string

	// SnapshotDataName binds the VolumeSnapshot object with the VolumeSnapshotData
	SnapshotDataName string
}

// The actual state of the volume snapshot
type VolumeSnapshotDataStatus struct {
	// The time the snapshot was successfully created
	// +optional
	CreationTimestamp metav1.Time

	// Representes the lates available observations about the volume snapshot
	Conditions []VolumeSnapshotDataCondition
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
	Type VolumeSnapshotDataConditionType
	// Status of the condition, one of True, False, Unknown.
	Status core_v1.ConditionStatus
	// The last time the condition transitioned from one status to another.
	// +optional
	LastTransitionTime metav1.Time
	// The reason for the condition's last transition.
	// +optional
	Reason string
	// A human readable message indicating details about the transition.
	// +optional
	Message string
}

// VolumeSnapshotData represents the actual "on-disk" snapshot object
type VolumeSnapshotData struct {
	metav1.TypeMeta
	// +optional
	metav1.ObjectMeta

	// Spec represents the desired state of the snapshot
	// +optional
	Spec VolumeSnapshotDataSpec

	// Status represents the latest observed state of the snapshot
	// +optional
	Status VolumeSnapshotDataStatus
}

type VolumeSnapshotDataList struct {
	metav1.TypeMeta
	// +optional
	metav1.ListMeta
	Items []VolumeSnapshotData
}

// The desired state of the volume snapshot
type VolumeSnapshotDataSpec struct {
	// Source represents the location and type of the volume snapshot
	VolumeSnapshotDataSource

	// VolumeSnapshotRef is part of bi-directional binding between VolumeSnapshot
	// and VolumeSnapshotData
	// +optional
	VolumeSnapshotRef *core_v1.ObjectReference

	// PersistentVolumeRef represents the PersistentVolume that the snapshot has been
	// taken from
	// +optional
	PersistentVolumeRef *core_v1.ObjectReference
}

// Represents the actual location and type of the snapshot. Only one of its members may be specified.
type VolumeSnapshotDataSource struct {
	//
}
