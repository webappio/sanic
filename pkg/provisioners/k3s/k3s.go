package k3s

import (
	"context"
	"fmt"
	provisionerutil "github.com/distributed-containers-inc/sanic/pkg/provisioners/util"
	"github.com/pkg/errors"
	"os"
	"os/exec"
	"strings"
)

//ProvisionerK3s starts k3s server
type ProvisionerK3s struct {
}

//EnsureCluster just ensures the registry is running
func (provisioner *ProvisionerK3s) EnsureCluster() error {
	return provisionerutil.StartRegistry(provisioner, context.Background(), map[string]string{"node-role.kubernetes.io/master": "true"})
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
	cmd, err := provisioner.KubectlCommand(
		"get", "pod",
		"--namespace", "kube-system",
		"--selector", "k8s-app=sanic-registry",
		"--output", "jsonpath={.items[0].status.podIP}",
	)
	if err != nil {
		return
	}
	out, err := cmd.Output()
	if err != nil {
		return
	}
	ip := strings.TrimSpace(string(out))
	if ip == "" {
		err = fmt.Errorf("could not connect to registry - try 'sanic deploy' and waiting 90 seconds")
		return
	}
	registryAddr = fmt.Sprintf("%s:5000", ip)
	registryInsecure = true
	return
}

func (provisioner *ProvisionerK3s) EdgeNodes() ([]string, error) {
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

func (provisioner *ProvisionerK3s) InClusterDir(hostDir string) string {
	return hostDir //k3s runs the server on the computer itself
}

func (provisioner *ProvisionerK3s) CheckRegistryInsecureOK() error {
	return nil //TODO check that /etc/docker/daemon.json is ok
}