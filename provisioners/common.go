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

	//GetKubeConfig returns where the absolute path to where the configuration file is placed for this provisioner
	//Note: it might not necessarily exist
	KubeConfigLocation() string
}

var Provisioners = map[string]Provisioner{
	"localdev": &localdev.ProvisionerLocalDev{},
}

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
			" does not have a 'clusterProvisioners' key defined in it.")
	}

	return Provisioners[env.ClusterProvisioner], nil
}
