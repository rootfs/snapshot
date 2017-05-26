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
	"flag"
	"os"
	"os/signal"

	"github.com/golang/glog"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"k8s.io/client-go/kubernetes"

	"github.com/rootfs/snapshot/client"
	"github.com/rootfs/snapshot/controller"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	kubeconfig := flag.String("kubeconfig", "", "Path to a kube config. Only required if out-of-cluster.")
	flag.Parse()
	flag.Set("logtostderr", "true")
	// Create the client config. Use kubeconfig if given, otherwise assume in-cluster.
	config, err := buildConfig(*kubeconfig)
	if err != nil {
		panic(err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	// initialize third party resource if it does not exist
	err = client.CreateTPR(clientset)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		panic(err)
	}

	// make a new config for our extension's API group, using the first config as a baseline
	snapshotClient, snapshotScheme, err := client.NewClient(config)
	if err != nil {
		panic(err)
	}

	// wait until TPR gets processed
	err = client.WaitForSnapshotResource(snapshotClient)
	if err != nil {
		panic(err)
	}

	// start a controller on instances of our TPR
	controller := controller.SnapshotController{
		SnapshotClient: snapshotClient,
		SnapshotScheme: snapshotScheme,
	}
	glog.Infof("starting snapshot controller")
	stopCh := make(chan struct{})
	go controller.Run(stopCh)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
	close(stopCh)

}

func buildConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	return rest.InClusterConfig()
}
