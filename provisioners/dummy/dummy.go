package dummy

import (
	"fmt"
	"os"
)

//ProvisionerDummy allow you to define commands on an environment and build locally, but not push or deploy
type ProvisionerDummy struct{}

func printDummyMessage() {
	_, _ = fmt.Fprintln(os.Stderr, "You can only build without pushing and use commands in provisioner type 'dummy'.")
	_, _ = fmt.Fprintln(os.Stderr, "Read the options for provisioner types at https://sanic.io/docs/provisioner-types")
}

func (ProvisionerDummy) EnsureCluster() error {
	printDummyMessage()
	os.Exit(2)
	return nil
}

func (ProvisionerDummy) KubeConfigLocation() string {
	printDummyMessage()
	os.Exit(2)
	return ""
}

func (ProvisionerDummy) Registry() (registryAddr string, registryInsecure bool, err error) {
	printDummyMessage()
	os.Exit(2)
	return
}

func (ProvisionerDummy) EdgeNodes() ([]string, error) {
	return []string{}, nil
}

func (ProvisionerDummy) InClusterDir(hostDir string) string {
	printDummyMessage()
	os.Exit(2)
	return ""
}

func (ProvisionerDummy) PruneWhileApplying() bool {
	return false
}
