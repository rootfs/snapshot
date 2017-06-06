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
type VolumePlugin interface {
	// Init inits volume plugin
	Init(cloudprovider.Interface)
	// SnapshotCreate creates a VolumeSnapshot from a PersistentVolumeSpec
	SnapshotCreate(*v1.PersistentVolume) (*tprv1.VolumeSnapshotDataSource, error)
	// SnapshotDelete deletes a VolumeSnapshot
	// PersistentVolume is provided for volume types, if any, that need PV Spec to delete snapshot
	SnapshotDelete(*tprv1.VolumeSnapshotDataSource, *v1.PersistentVolume) error
}
```

### Volume Snapshot Data Source Spec

Each volume must also provide a Snapshot Data Source Spec and add to [VolumeSnapshotDataSource](pkg/apis/tpr/v1/types.go), then declare support in [GetSupportedVolumeFromPVC](pkg/apis/tpr/v1/types.go) by returning the exact name as returned by the plugin's `GetPluginName()`

### Invocation
The plugins are added to Snapshot controller [cmd pkg](cmd/snapshot-controller/snapshot-controller.go).


# Snapshot based PV Provisioner Volume Plugin Interface
TBD