# Volume Snapshot Controller

## Status

Pre-alpha

## Quick Howto

 - [Host Path](examples/hostpath/README.md)

 - [AWS EBS](examples/aws/README.md)


## Snapshot Volume Plugin Interface

As illustrated in example plugin [hostPath](pkg/volume/hostpath/processor.go)

### Plugin API

A Volume plugin must provide `RegisterPlugin()` to return plugin struct, `GetPluginName()` to return plugin name, and implement the following interface as illustrated in [hostPath](pkg/volume/hostpath/processor.go)

```go
import (
	"k8s.io/client-go/pkg/api/v1"

	tprv1 "github.com/rootfs/snapshot/pkg/apis/tpr/v1"
	"github.com/rootfs/snapshot/pkg/cloudprovider"
)

type VolumePlugin interface {
	// Init inits volume plugin
	Init(cloudprovider.Interface)
	// SnapshotCreate creates a VolumeSnapshot from a PersistentVolumeSpec
	SnapshotCreate(*v1.PersistentVolume) (*tprv1.VolumeSnapshotDataSource, error)
	// SnapshotDelete deletes a VolumeSnapshot
	// PersistentVolume is provided for volume types, if any, that need PV Spec to delete snapshot
	SnapshotDelete(*tprv1.VolumeSnapshotDataSource, *v1.PersistentVolume) error
	// SnapshotRestore restores (promotes) a volume snapshot into a volume
	SnapshotRestore(*tprv1.VolumeSnapshotData, *v1.PersistentVolumeClaim, string, map[string]string) (*v1.PersistentVolumeSource, map[string]string, error)
	// Describe volume snapshot status.
	// return true if the snapshot is ready
	DescribeSnapshot(snapshotData *tprv1.VolumeSnapshotData) (isCompleted bool, err error)
	// VolumeDelete deletes a PV
	// TODO in the future pass kubernetes client for certain volumes (e.g. rbd) so they can access storage class to retrieve secret
	VolumeDelete(pv *v1.PersistentVolume) error
}
```

### Volume Snapshot Data Source Spec

Each volume must also provide a Snapshot Data Source Spec and add to [VolumeSnapshotDataSource](pkg/apis/tpr/v1/types.go), then declare support in [GetSupportedVolumeFromPVC](pkg/apis/tpr/v1/types.go) by returning the exact name as returned by the plugin's `GetPluginName()`

### Invocation
The plugins are added to Snapshot controller [cmd pkg](cmd/snapshot-controller/snapshot-controller.go).

