package localdev

import (
	"context"
	"fmt"
	"github.com/distributed-containers-inc/sanic/util"
)

//ProvisionerLocalDev is a provisioner which uses "kind" to set up a local, 4-node development kubernetes cluster
//within docker itself.
type ProvisionerLocalDev struct{}

//KubeConfigLocation returns the path to the kubectl configuration for this provisioner
func (provisioner *ProvisionerLocalDev) KubeConfigLocation() string {
	return kindContext.KubeConfigPath()
}

//EnsureCluster for localdev is a wrapper around "kind", which sets up a 4-node kubernetes cluster in docker itself.
func (provisioner *ProvisionerLocalDev) EnsureCluster() error {
	clusterError := provisioner.checkCluster()

	if clusterError == nil {
		return nil //nothing to do, cluster is healthy
	}
	fmt.Printf("Creating a new cluster, old one cannot be used: %s\n", clusterError.Error())
	fmt.Println("This takes between 1 and 10 minutes, depending on your internet connection speed.")
	err := provisioner.startCluster()
	if err != nil {
		return err
	}
	err = util.RunContextuallyInParallel(context.Background(), provisioner.startIngressController, provisioner.startRegistry)
	if err != nil {
		return fmt.Errorf("could not start the ingress controller or registry: %s", err.Error())
	}
	//TODO message about where the webserver is available
	return err
}


