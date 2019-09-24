package external

import (
	"fmt"
	"github.com/distributed-containers-inc/sanic/pkg/util"
	"github.com/pkg/errors"
	"os"
	"os/exec"
	"strings"
)

//ProvisionerExternal simply wraps an existing kubernetes cluster (accessed via kubectl) and registry
type ProvisionerExternal struct{
	kubeConfigLocation string
	edgeNodes []string
	registry string
}

//EnsureCluster does nothing for the external provisioner, basically by its definition
func (provisioner *ProvisionerExternal) EnsureCluster() error {
	return nil
}

//KubectlCommand for external just returns a provisioner that has KUBECONFIG pointing to the configured directory
func (provisioner *ProvisionerExternal) KubectlCommand(args ...string) (*exec.Cmd, error) {
	if _, err := exec.LookPath("kubectl"); err != nil {
		return nil, errors.Wrap(err, "could not find kubectl executable in path - is it installed?")
	}

	cmd := exec.Command("kubectl", args...)

	kubeconfig := provisioner.kubeConfigLocation
	if _, err := os.Stat(kubeconfig); err != nil {
		return nil, errors.Wrapf(err, "could not find the kubeconfig at %s for this environment", kubeconfig)
	}

	cmd.Env = append(os.Environ(), "KUBECONFIG="+kubeconfig)
	return cmd, nil
}

func (provisioner *ProvisionerExternal) Registry() (registryAddr string, registryInsecure bool, err error) {
	registryAddr = provisioner.registry
	registryInsecure = false
	return
}

func (provisioner *ProvisionerExternal) EdgeNodes() ([]string, error) {
	return provisioner.edgeNodes, nil
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

	provisioner := &ProvisionerExternal{
		kubeConfigLocation: config,
		registry: args["registry"],
	}
	if edgeNodes, exists := args["edgeNodes"]; exists {
		provisioner.edgeNodes = strings.Split(edgeNodes, ",")
	}
	return provisioner
}

func ValidateConfig(args map[string]string) error {
	config, exists := args["kubeConfig"]
	if !exists {
		return fmt.Errorf("configuration needs to include the kubeConfig key")
	}
	config, err := util.ExpandUser(config)
	if err != nil {
		return err
	}
	if _, err = os.Stat(config); err != nil {
		return fmt.Errorf("configuration file at %s did not exist: %s", config, err.Error())
	}
	if _, exists := args["registry"]; !exists {
		return fmt.Errorf("configuration needs to include the registry to push to, i.e., https://registry.hub.docker.com/registryUserName")
	}
	return nil
}