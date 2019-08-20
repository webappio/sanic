	package provisioners

import (
	"fmt"
	"github.com/distributed-containers-inc/sanic/provisioners/external"
	"github.com/distributed-containers-inc/sanic/provisioners/localdev"
	"github.com/distributed-containers-inc/sanic/provisioners/provisioner"
)

type provisionerBuilder func(map[string]string) provisioner.Provisioner

var provisionerBuilders = map[string]provisionerBuilder{
	"external": func(args map[string]string) provisioner.Provisioner {
		return external.Create(args)
	},
	"localdev": func(args map[string]string) provisioner.Provisioner {
		return &localdev.ProvisionerLocalDev{}
	},
}

type provisionerConfigValidator func(map[string]string) error

var provisionerValidators = map[string]provisionerConfigValidator{
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
