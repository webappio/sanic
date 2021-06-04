package provisioners

import (
	"fmt"
	"github.com/layer-devops/sanic/pkg/provisioners/external"
	"github.com/layer-devops/sanic/pkg/provisioners/k3s"
	"github.com/layer-devops/sanic/pkg/provisioners/minikube"
	"github.com/layer-devops/sanic/pkg/provisioners/provisioner"
)

type provisionerBuilder func(map[string]string) provisioner.Provisioner

var provisionerBuilders = map[string]provisionerBuilder{
	"external": func(args map[string]string) provisioner.Provisioner {
		return external.Create(args)
	},
	"k3s": func(args map[string]string) provisioner.Provisioner {
		return &k3s.ProvisionerK3s{}
	},
	"minikube": func(args map[string]string) provisioner.Provisioner {
		return &minikube.ProvisionerMinikube{}
	},
}


type provisionerConfigValidator func(map[string]string) error

var provisionerValidators = map[string]provisionerConfigValidator{
	"external": func(args map[string]string) error {
		return external.ValidateConfig(args)
	},
	"k3s": func(args map[string]string) error {
		for k := range args {
			return fmt.Errorf("k3s takes no arguments, got %s", k)
		}
		return nil
	},
	"minikube": func(args map[string]string) error {
		for k := range args {
			return fmt.Errorf("minikube takes no arguments, got %s", k)
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
func GetProvisioner(provisionerName string, provisionerArgs map[string]string) provisioner.Provisioner {
	return provisionerBuilders[provisionerName](provisionerArgs)
}

func ValidateProvisionerConfig(provisionerName string, provisionerArgs map[string]string) error {
	validator, exists := provisionerValidators[provisionerName]
	if !exists {
		panic(fmt.Errorf("a provisioner did not have a validator defined: %s", provisionerName))
	}
	return validator(provisionerArgs)
}
