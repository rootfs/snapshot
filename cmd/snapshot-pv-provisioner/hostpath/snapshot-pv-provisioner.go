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

package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"

	"github.com/golang/glog"
	"github.com/kubernetes-incubator/external-storage/lib/controller"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	tprv1 "github.com/rootfs/snapshot/pkg/apis/tpr/v1"
	tprclient "github.com/rootfs/snapshot/pkg/client"
)

const (
	provisionerName  = "volumesnapshot.external-storage.k8s.io/hostpath"
	provisionerIDAnn = "hostPathProvisionerIdentity"
	restorePoint     = "/restore/"
)

type hostPathProvisioner struct {
	// Kubernetes Client.
	client kubernetes.Interface
	// TPR client
	tprclient *rest.RESTClient
	// Identity of this hostPathProvisioner, generated. Used to identify "this"
	// provisioner's PVs.
	identity string
}

func newHostPathProvisioner(client kubernetes.Interface, tprclient *rest.RESTClient, id string) controller.Provisioner {
	return &hostPathProvisioner{
		client:    client,
		tprclient: tprclient,
		identity:  id,
	}
}

var _ controller.Provisioner = &hostPathProvisioner{}

func (p *hostPathProvisioner) snapshotRestore(snapshotName string, snapshotData tprv1.VolumeSnapshotData) (*v1.PersistentVolumeSource, error) {
	// retrieve VolumeSnapshotDataSource
	if snapshotData.Spec.HostPath == nil {
		return nil, fmt.Errorf("failed to retrieve HostPath spec from %s: %#v", snapshotName, snapshotData)
	}

	// restore snapshot to a PV
	snapId := snapshotData.Spec.HostPath.Path
	dir := restorePoint + string(uuid.NewUUID())
	os.MkdirAll(dir, 0750)
	cmd := exec.Command("tar", "xzvf", snapId, "-C", dir)
	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to restore %s to %s: %v", snapId, dir, err)
	}
	pv := &v1.PersistentVolumeSource{
		HostPath: &v1.HostPathVolumeSource{
			Path: dir,
		},
	}
	return pv, nil
}

// Provision creates a storage asset and returns a PV object representing it.
func (p *hostPathProvisioner) Provision(options controller.VolumeOptions) (*v1.PersistentVolume, error) {
	if options.PVC.Spec.Selector != nil {
		return nil, fmt.Errorf("claim Selector is not supported")
	}
	snapshotName, ok := options.PVC.Annotations[tprclient.SnapshotPVCAnnotation]
	if !ok {
		return nil, fmt.Errorf("snapshot annotation not found on PV")
	}
	// retrieve VolumeSnapshotData
	var snapshotData tprv1.VolumeSnapshotData
	err := p.tprclient.Get().
		Resource(tprv1.VolumeSnapshotDataResourcePlural).
		Namespace(v1.NamespaceDefault).
		Name(snapshotName).
		Do().Into(&snapshotData)

	if err != nil {
		return nil, fmt.Errorf("failed to retrieve VolumeSnapshotData %s: %v", snapshotName, err)
	}
	pvSrc, err := p.snapshotRestore(snapshotName, snapshotData)
	if err != nil || pvSrc == nil {
		return nil, fmt.Errorf("failed to create a PV from snapshot %s: %v", snapshotName, err)
	}
	pv := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: options.PVName,
			Annotations: map[string]string{
				provisionerIDAnn: p.identity,
			},
		},
		Spec: v1.PersistentVolumeSpec{
			PersistentVolumeReclaimPolicy: options.PersistentVolumeReclaimPolicy,
			AccessModes:                   options.PVC.Spec.AccessModes,
			Capacity: v1.ResourceList{
				v1.ResourceName(v1.ResourceStorage): options.PVC.Spec.Resources.Requests[v1.ResourceName(v1.ResourceStorage)],
			},
			PersistentVolumeSource: *pvSrc,
		},
	}

	glog.Infof("successfully created HostPath share %+v", pv.Spec.PersistentVolumeSource.HostPath)

	return pv, nil
}

func (p *hostPathProvisioner) deleteHostPath(spec *v1.PersistentVolumeSpec) error {
	if spec.HostPath == nil {
		return fmt.Errorf("no hostpath specified")
	}
	path := spec.HostPath.Path
	return os.RemoveAll(path)
}

// Delete removes the storage asset that was created by Provision represented
// by the given PV.
func (p *hostPathProvisioner) Delete(volume *v1.PersistentVolume) error {
	ann, ok := volume.Annotations[provisionerIDAnn]
	if !ok {
		return errors.New("identity annotation not found on PV")
	}
	if ann != p.identity {
		return &controller.IgnoredError{"identity annotation on PV does not match ours"}
	}

	return p.deleteHostPath(&volume.Spec)
}

var (
	master     = flag.String("master", "", "Master URL")
	kubeconfig = flag.String("kubeconfig", "", "Absolute path to the kubeconfig")
	id         = flag.String("id", "", "Unique provisioner identity")
)

func main() {
	flag.Parse()
	flag.Set("logtostderr", "true")

	var config *rest.Config
	var err error
	if *master != "" || *kubeconfig != "" {
		config, err = clientcmd.BuildConfigFromFlags(*master, *kubeconfig)
	} else {
		config, err = rest.InClusterConfig()
	}
	prId := string(uuid.NewUUID())
	if *id != "" {
		prId = *id
	}
	if err != nil {
		glog.Fatalf("Failed to create config: %v", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		glog.Fatalf("Failed to create client: %v", err)
	}

	// The controller needs to know what the server version is because out-of-tree
	// provisioners aren't officially supported until 1.5
	serverVersion, err := clientset.Discovery().ServerVersion()
	if err != nil {
		glog.Fatalf("Error getting server version: %v", err)
	}

	// make a tpr client to list VolumeSnapshot
	snapshotClient, _, err := tprclient.NewClient(config)
	if err != nil || snapshotClient == nil {
		glog.Fatalf("Failed to make TPR client: %v", err)
	}

	// Create the provisioner: it implements the Provisioner interface expected by
	// the controller
	hostPathProvisioner := newHostPathProvisioner(clientset, snapshotClient, prId)

	// Start the provision controller which will dynamically provision hostPath
	// PVs
	pc := controller.NewProvisionController(
		clientset,
		provisionerName,
		hostPathProvisioner,
		serverVersion.GitVersion,
	)

	pc.Run(wait.NeverStop)
}
