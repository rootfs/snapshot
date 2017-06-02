# Volume Snapshot Controller

## Status

Pre-alpha

## Quick Howto

### Host Path Volume Type

#### Start Snapshot Controller 

(assuming running Kubernetes local cluster):
```
_output/bin/snapshot-controller  -kubeconfig=${HOME}/.kube/config
```

####  Create a snapshot
 * Create a hostpath PV and PVC
```bash
kubectl create namespace myns
kubectl -f example/hostpath/pv.yaml
kubectl -f example/hostpath/pvc.yaml
```
 * Create a Snapshot Third Party Resource 
```bash
kubectl -f example/hostpath/snapshot.yaml
```

#### Check VolumeSnapshot and VolumeSnapshotData are created

```bash
kubectl get volumesnapshot,volumesnapshotdata -o yaml --namespace=myns
```

## Snapshot based PV Provisioner

Unlike exiting PV provisioners that provision blank volume, Snapshot based PV provisioners create volumes based on existing snapshots. Thus new provisioners are needed.

There is a special annotation give to PVCs that request snapshot based PVs. As illustrated in [the example](examples/hostpath/claim.yaml), `snapshot.alpha.kubernetes.io` must point to an existing VolumeSnapshot Object
```yaml
metadata:
  name: 
  namespace: 
  annotations:
    snapshot.alpha.kubernetes.io/snapshot: snapshot-demo
```

## HostPath Volume Type

#### Start PV Provisioner and Storage Class to restore a snapshot to a PV

Start provisioner (assuming running Kubernetes local cluster):
```bash
_output/bin/snapshot-provisioner-hostpath  -kubeconfig=${HOME}/.kube/config
```

Create a storage class:
```bash
kubectl create -f example/hostpath/class.yaml
```

### Create a PVC that claims a PV based on an existing snapshot 

```bash
kubectl create -f example/hostpath/claim.yaml
```

#### Check PV and PVC are created

```bash
kubectl get pv,pvc
```
  

## Snapshot Volume Plugin Interface

As illustrated in example plugin [hostPath](pkg/volume/hostpath/processor.go)

### Plugin API

A Volume plugin must provide `RegisterPlugin()` to return plugin struct, `GetPluginName()` to return plugin name, and implement the following interface as illustrated in [hostPath](pkg/volume/hostpath/processor.go)

```go
type VolumePlugin interface {
	// Init inits volume plugin
	Init(interface{})
	// SnapshotCreate creates a VolumeSnapshot from a PersistentVolumeSpec
	SnapshotCreate(*v1.PersistentVolumeSpec) (*tprv1.VolumeSnapshotDataSource, error)
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