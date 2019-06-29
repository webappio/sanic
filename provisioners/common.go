package provisioners

import (
	"fmt"
	"github.com/distributed-containers-inc/sanic/provisioners/dummy"
	"github.com/distributed-containers-inc/sanic/provisioners/external"
	"github.com/distributed-containers-inc/sanic/provisioners/localdev"
)

//Provisioner is an interface which represents a way to deploy kubernetes services.
type Provisioner interface {
	//EnsureCluster checks if the cluster exists and is configured correctly. Otherwise, it prompts the user
	//with instructions on how to set up the cluster.
	EnsureCluster() error

	//KubeConfigLocation returns where the absolute path to where the configuration file is placed for this provisioner
	//Note: it might not necessarily exist
	KubeConfigLocation() string

	//Registry returns:
	// - registryAddr: the registry to push to, e.g., registry.example.com:3000, or "" if none is defined
	// - registryInsecure: whether the registry uses HTTP (currently only used in localdev)
	Registry() (registryAddr string, registryInsecure bool, err error)

	//EdgeNodes returns a list of hostnames or IP addresses that will expose the edge nodes (where the ingress controllers are hosted)
	EdgeNodes() ([]string, error)

	//InClusterDir is the primary mechanism for live mounting:
	//It returns where the specified host folder is synchronized in all of the kubernetes nodes
	//If a provisioner does not support live mounting, or has an error, it should return a descriptive error string
	//I.e., if your sanic project is at /home/user/project, and provisioner is localdev, this returns /hosthome/project
	InClusterDir(hostDir string) string

	//PruneWhileApplying dictates whether "sanic deploy" will prune unreferenced pods. This is dangerous in production.
	PruneWhileApplying() bool
}

type provisionerBuilder func(map[string]string) Provisioner

var provisionerBuilders = map[string]provisionerBuilder{
	"dummy": func(args map[string]string) Provisioner {
		return &dummy.ProvisionerDummy{}
	},
	"external": func(args map[string]string) Provisioner {
		return external.Create(args)
	},
	"localdev": func(args map[string]string) Provisioner {
		return &localdev.ProvisionerLocalDev{}
	},
}

type provisionerConfigValidator func(map[string]string) error

var provisionerValidators = map[string]provisionerConfigValidator{
	"dummy": func(args map[string]string) error {
		for k := range args {
			return fmt.Errorf("dummy takes no arguments, got %s", k)
		}
		return nil
	},
	"external": func(args map[string]string) error {
		return external.ValidateConfig(args)
	},
	"localdev": func(args map[string]string) error {
		for k := range args {
			return fmt.Errorf("localdev takes no arguments, got %s", k)
		}
		return nil
	},
}

//ProvisionerExists checks whether the given provisioner exists
func ProvisionerExists(name string) bool {
	_, ok := provisionerBuilders[name]
	return ok
}

//GetProvisionerNames returns a slice of all of the defined provisioner names.
func GetProvisionerNames() []string {
	var names []string
	for k := range provisionerBuilders {
		names = append(names, k)
	}
	return names
}

//GetProvisioner returns the provisioner for the current environment
func GetProvisioner(provisionerName string, provisionerArgs map[string]string) Provisioner {
	return provisionerBuilders[provisionerName](provisionerArgs)
}

func ValidateProvisionerConfig(provisionerName string, provisionerArgs map[string]string) error {
	validator, exists := provisionerValidators[provisionerName]
	if !exists {
		panic(fmt.Errorf("a provisioner did not have a validator defined: %s", provisionerName))
	}
	return validator(provisionerArgs)
}
