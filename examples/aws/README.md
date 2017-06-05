### AWS EBS

#### Start Snapshot Controller 

(assuming running Kubernetes local cluster):
```
_output/bin/snapshot-controller  -kubeconfig=${HOME}/.kube/config
```

####  Create a snapshot
 * Create an PVC
```bash
kubectl create namespace myns
# if no default storage class, create one
kubectl -f https://raw.githubusercontent.com/kubernetes/kubernetes/master/examples/persistent-volume-provisioning/aws-ebs.yaml
kubectl -f example/aws/pvc.yaml
```
 * Create a Snapshot Third Party Resource 
```bash
kubectl -f example/aws/snapshot.yaml
```

#### Check VolumeSnapshot and VolumeSnapshotData are created

```bash
kubectl get volumesnapshot,volumesnapshotdata -o yaml --namespace=myns
```

## Snapshot based PV Provisioner

Unlike exiting PV provisioners that provision blank volume, Snapshot based PV provisioners create volumes based on existing snapshots. Thus new provisioners are needed.

There is a special annotation give to PVCs that request snapshot based PVs. As illustrated in [the example](examples/aws/claim.yaml), `snapshot.alpha.kubernetes.io` must point to an existing VolumeSnapshot Object
```yaml
metadata:
  name: 
  namespace: 
  annotations:
    snapshot.alpha.kubernetes.io/snapshot: snapshot-demo
```
