package provisioners

import (
	"github.com/distributed-containers-inc/sanic/config"
	"github.com/distributed-containers-inc/sanic/provisioners/localdev"
	"github.com/distributed-containers-inc/sanic/shell"
	"github.com/pkg/errors"
)

//Provisioner is an interface which represents a way to deploy kubernetes services.
type Provisioner interface {
	//EnsureCluster checks if the cluster exists and is configured correctly. Otherwise, it prompts the user
	//with instructions on how to set up the cluster.
	EnsureCluster() error

	//KubeConfigLocation returns where the absolute path to where the configuration file is placed for this provisioner
	//Note: it might not necessarily exist
	KubeConfigLocation() string

	//Registry returns the registry to push to, e.g., http://registry.example.com:3000, or "" if none is defined
	//Note: it must start with http:// or https://
	Registry() (string, error)

	//EdgeNodes returns a list of hostnames or IP addresses that will expose the edge nodes (where the ingress controllers are hosted)
	EdgeNodes() ([]string, error)

	//InClusterDir is the primary mechanism for live mounting:
	//It returns where the specified host folder is synchronized in all of the kubernetes nodes
	//If a provisioner does not support live mounting, or has an error, it should return a descriptive error string
	//I.e., if your sanic project is at /home/user/project, and provisioner is localdev, this returns /hosthome/project
	InClusterDir(hostDir string) string
}

var provisioners = map[string]Provisioner{
	"localdev": &localdev.ProvisionerLocalDev{},
}

//GetProvisioner returns the provisioner for the current environment
func GetProvisioner() (Provisioner, error) {
	s, err := shell.Current()
	if err != nil {
		return nil, err
	}

	cfg, err := config.Read()
	if err != nil {
		return nil, err
	}

	env, err := cfg.CurrentEnvironment(s)
	if err != nil {
		return nil, err
	}

	if env.ClusterProvisioner == "" {
		return nil, errors.New("the environment " + s.GetSanicEnvironment() +
			" does not have a 'clusterProvisioner' key defined in it. Try clusterProvisioner: localdev to start.")
	}

	return provisioners[env.ClusterProvisioner], nil
}
