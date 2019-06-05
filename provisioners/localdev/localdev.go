package localdev

import (
	"context"
	"fmt"
	"github.com/distributed-containers-inc/sanic/util"
)

//ProvisionerLocalDev is a provisioner which uses "kind" to set up a local, 4-node development kubernetes cluster
//within docker itself.
//Architecture looks as follows:
//(docker container) sanic-control-plane
//  kubelet configured as master, as well as control plane components
//  containerd to run pods
//    sanic registry container, where the build daemon pushes to and the nodes pull from
//(docker container) sanic-worker
//  kubelet configured as node
//  containerd to run pods
//    (deployed apps)
//(docker container) sanic-worker2
//  kubelet configured as node
//  containerd to run pods
//(docker container) sanic-worker3
//  kubelet configured as node
//  containerd to run pods
type ProvisionerLocalDev struct{}

func (provisioner *ProvisionerLocalDev) KubeConfigLocation() string {
	return kindContext.KubeConfigPath()
}

func (provisioner *ProvisionerLocalDev) EnsureCluster() error {
	clusterError := provisioner.checkCluster()

	if clusterError == nil {
		return nil //nothing to do, cluster is healthy
	}
	fmt.Printf("Creating a new cluster, old one cannot be used: %s\n", clusterError.Error())
	fmt.Println("This will take between 1 and 10 minutes, depending on your internet connection speed.")
	err := provisioner.startCluster()
	if err != nil {
		return err
	}
	err = util.RunContextuallyInParallel(context.Background(), provisioner.startRegistry, provisioner.patchRegistryContainers)
	if err != nil {
		return fmt.Errorf("error while starting or configuring registry: %s", err.Error())
	}
	//TODO message about where the webserver is available
	return err
}

func (provisioner *ProvisionerLocalDev) Registry() (string, error) {
	masters, err := clusterMasterNodes()
	if err != nil {
		return "", err
	}
	if len(masters) != 1 {
		return "", fmt.Errorf("got %d control plane containers, expected only one", len(masters))
	}
	ip, err := masters[0].IP()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s:%d", ip, RegistryNodePort), nil
}

func (provisioner *ProvisionerLocalDev) RegistryPushDefault() bool {
	return true
}
