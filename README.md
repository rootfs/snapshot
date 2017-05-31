# Volume Snapshot Controller

## Status

Pre-alpha

## Quick Howto

### Host Path

#### Start Snapshot Controller 

(assuming running Kubernetes local cluster):
```
_output/bin/snapshot-controller  -kubeconfig=${HOME}/.kube/config
```

####  Create a snapshot
 * Create a hostpath PV and PVC
```bash
kubectl -f example/hostpath/pv.yaml
kubectl -f example/hostpath/pvc.yaml
```
 * Create a Snapshot Third Party Resource 
```bash
kubectl -f example/hostpath/snapshot.yaml
```

#### Check VolumeSnapshot and VolumeSnapshotData are created

```bash
kubectl get volumesnapshot,volumesnapshotdata -o yaml
```

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

### Check PV and PVC are created

```bash
kubectl get pv,pvc
```
  

## TODO List

