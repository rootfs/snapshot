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

// This file was automatically generated by lister-gen

package internalversion

import (
	tprv1 "github.com/rootfs/snapshot/pkg/apis/tpr/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// SnapshotLister helps list Snapshots.
type SnapshotLister interface {
	// List lists all Snapshots in the indexer.
	List(selector labels.Selector) (ret []*tprv1.VolumeSnapshot, err error)
	// Get retrieves the Snapshot from the index for a given name.
	Get(name string) (*tprv1.VolumeSnapshot, error)
	SnapshotListerExpansion
}

// snapshotLister implements the SnapshotLister interface.
type snapshotLister struct {
	indexer cache.Indexer
}

// NewSnapshotLister returns a new SnapshotLister.
func NewSnapshotLister(indexer cache.Indexer) SnapshotLister {
	return &snapshotLister{indexer: indexer}
}

// List lists all Snapshots in the indexer.
func (s *snapshotLister) List(selector labels.Selector) (ret []*tprv1.VolumeSnapshot, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*tprv1.VolumeSnapshot))
	})
	return ret, err
}

// Get retrieves the Snapshot from the index for a given name.
func (s *snapshotLister) Get(name string) (*tprv1.VolumeSnapshot, error) {
	key := &tprv1.VolumeSnapshot{Metadata: v1.ObjectMeta{Name: name}}
	obj, exists, err := s.indexer.Get(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(tprv1.Resource("snapshot"), name)
	}
	return obj.(*tprv1.VolumeSnapshot), nil
}
