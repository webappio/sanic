package provisioner

import "os/exec"

//Provisioner is an interface which represents a way to deploy kubernetes services.
type Provisioner interface {
	//EnsureCluster checks if the cluster exists and is configured correctly. Otherwise, it prompts the user
	//with instructions on how to set up the cluster.
	EnsureCluster() error

	//KubectlCommand constructs a new exec.Cmd object which represents [ "kubectl" (args ...) ]
	KubectlCommand(args ...string) (*exec.Cmd, error)

	//Registry returns:
	// - registryAddr: the registry to push to, e.g., registry.example.com:3000, or "" if none is defined
	// - registryInsecure: whether the registry uses HTTP (currently only used in localdev)
	Registry() (registryAddr string, registryInsecure bool, err error)

	//EnsureRegistryInsecureOK ensures that if the registry for this provisioner is insecure, that the user can push to it
	CheckRegistryInsecureOK() error

	//EdgeNodes returns a list of hostnames or IP addresses that will expose the edge nodes (where the ingress controllers are hosted)
	EdgeNodes() ([]string, error)

	//InClusterDir is the primary mechanism for live mounting:
	//It returns where the specified host folder is synchronized in all of the kubernetes nodes
	//If a provisioner does not support live mounting, or has an error, it should return a descriptive error string
	//I.e., if your sanic project is at /home/user/project, and provisioner is localdev, this returns /hosthome/project
	InClusterDir(hostDir string) string
}
