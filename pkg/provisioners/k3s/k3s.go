package k3s

import (
	"github.com/pkg/errors"
	"os"
	"os/exec"
)

//ProvisionerK3s starts k3s server
type ProvisionerK3s struct {
}

//EnsureCluster just ensures the registry is running
func (provisioner *ProvisionerK3s) EnsureCluster() error {
	// no-op (don't need a registry for k3s)
	return nil
}

//KubectlCommand for k3s is just a wrapper around "k3s kubectl"
func (provisioner *ProvisionerK3s) KubectlCommand(args ...string) (*exec.Cmd, error) {
	if _, err := exec.LookPath("k3s"); err != nil {
		return nil, errors.Wrap(err, "could not find k3s executable in path - is it installed?")
	}
	cmd := exec.Command("k3s", append([]string{"kubectl"}, args...)...)
	cmd.Env = os.Environ()

	return cmd, nil
}

func (provisioner *ProvisionerK3s) Registry() (registryAddr string, registryInsecure bool, err error) {
	// no-op (don't need a registry for k3s)
	registryAddr = "localhost:5000"
	registryInsecure = true
	return
}

func (provisioner *ProvisionerK3s) InClusterDir(hostDir string) string {
	return hostDir //k3s runs the server on the computer itself
}

func (provisioner *ProvisionerK3s) CheckRegistryInsecureOK() error {
	return nil //TODO check that /etc/docker/daemon.json is ok
}