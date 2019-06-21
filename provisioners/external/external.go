package external

import (
	"github.com/distributed-containers-inc/sanic/util"
	"strings"
)

//ProvisionerExternal simply wraps an existing kubernetes cluster (accessed via kubectl) and registry
type ProvisionerExternal struct{
	kubeConfigLocation string
	masters []string
	registry string
}

//EnsureCluster does nothing for the external provisioner, basically by its definition
func (provisioner *ProvisionerExternal) EnsureCluster() error {
	return nil
}

func (provisioner *ProvisionerExternal) KubeConfigLocation() string {
	return provisioner.kubeConfigLocation
}

func (provisioner *ProvisionerExternal) Registry() (string, error) {
	return provisioner.registry, nil
}

func (provisioner *ProvisionerExternal) EdgeNodes() ([]string, error) {
	return provisioner.masters, nil
}

func (provisioner *ProvisionerExternal) InClusterDir(hostDir string) string {
	return "<ERROR_IS_EXTERNAL_DO_NOT_LIVEMOUNT>"
}

//Create returns a new ProvisionerLocalDev from the given arguments
//noinspection GoUnusedParameter
func Create(args map[string]string) *ProvisionerExternal {
	//TODO error handling
	config, err := util.ExpandUser(args["kubeConfig"])
	if err != nil {
		panic(err)
	}

	return &ProvisionerExternal{
		kubeConfigLocation: config,
		masters: strings.Split(args["masters"], ","),
		registry: args["registry"],
	}
}