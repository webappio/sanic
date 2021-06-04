package minikube

import (
	"fmt"
	"github.com/pkg/errors"
	"os"
	"os/exec"
	"strings"
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
	cmd, err := provisioner.KubectlCommand(
		"get", "services",
		"-n", "kube-system",
		"-o", "jsonpath={.spec.clusterIP}",
		"traefik",
	)
	if err != nil {
		return nil, errors.Wrap(err, "could not get the traefik service")
	}
	out, err := cmd.Output()
	if err != nil {
		return nil, errors.Wrap(err, "could not get the traefik service")
	}
	ip := strings.TrimSpace(string(out))
	if ip == "" {
		return nil, fmt.Errorf("could not get the IP of the traefik service")
	}
	return []string{ip}, nil
}

func (provisioner *ProvisionerMinikube) InClusterDir(hostDir string) string {
	return hostDir //minikube runs the server on the computer itself
}

func (provisioner *ProvisionerMinikube) CheckRegistryInsecureOK() error {
	return nil //TODO check that /etc/docker/daemon.json is ok
}