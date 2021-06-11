package minikube

import (
	"github.com/pkg/errors"
	"os"
	"os/exec"
)

//ProvisionerMinikube starts minikube server
type ProvisionerMinikube struct {
}

//EnsureCluster just ensures the registry is running
func (provisioner *ProvisionerMinikube) EnsureCluster() error {
	// no-op (don't need a registry for minikube)
	return nil
}

//KubectlCommand for minikube is just a wrapper around "minikube kubectl"
func (provisioner *ProvisionerMinikube) KubectlCommand(args ...string) (*exec.Cmd, error) {
	if _, err := exec.LookPath("minikube"); err != nil {
		return nil, errors.Wrap(err, "could not find minikube executable in path - is it installed?")
	}
	cmd := exec.Command("minikube", append([]string{"kubectl", "--"}, args...)...)
	cmd.Env = os.Environ()

	return cmd, nil
}

func (provisioner *ProvisionerMinikube) Registry() (registryAddr string, registryInsecure bool, err error) {
	// no-op (don't need a registry for minikube)
	registryAddr = "localhost:5000"
	registryInsecure = true
	return
}

func (provisioner *ProvisionerMinikube) EdgeNodes() ([]string, error) {
	return []string{}, nil
}

func (provisioner *ProvisionerMinikube) InClusterDir(hostDir string) string {
	return hostDir
}

func (provisioner *ProvisionerMinikube) CheckRegistryInsecureOK() error {
	return nil //TODO check that /etc/docker/daemon.json is ok
}
