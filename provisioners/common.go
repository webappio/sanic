package provisioners

import (
	"github.com/distributed-containers-inc/sanic/config"
	"github.com/distributed-containers-inc/sanic/shell"
	"github.com/pkg/errors"
)

//Provisioner is an interface which represents a way to deploy kubernetes services.
type Provisioner interface {
	//EnsureCluster checks if the cluster exists and is configured correctly. Otherwise, it prompts the user
	//with instructions on how to set up the cluster.
	EnsureCluster() error
}

var Provisioners = map[string]Provisioner{
	"localdev": &ProvisionerLocalDev{},
}

func EnsureCluster() error {
	s, err := shell.Current()
	if err != nil {
		return err
	}

	cfg, err := config.Read()
	if err != nil {
		return err
	}

	env, err := cfg.CurrentEnvironment(s)
	if err != nil {
		return err
	}

	if env.ClusterProvisioner == "" {
		return errors.New("the environment " + s.GetSanicEnvironment() +
			" does not have a 'clusterProvisioners' key defined in it.")
	}

	return Provisioners[env.ClusterProvisioner].EnsureCluster()
}
