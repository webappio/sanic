package config

import (
	"errors"
	"fmt"
	"github.com/distributed-containers-inc/sanic/shell"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path/filepath"
)

//Command is a configuration structure which consists of a name (e.g., print_hello) and a command (e.g., "echo hello")
type Command struct {
	Name    string
	Command string
}

//Environment is a specific environment which can be entered with "sanic env"
type Environment struct {
	Commands []Command
	//Provisioner can be one of:
	// - localdev, a kubernetes-in-docker environment suitable for local development, using "kind"
	// - TODO prod environment docs
	ClusterProvisioner string `yaml:"clusterProvisioner"`
}

//SanicConfig is the global structure of entries in sanic.yaml
type SanicConfig struct {
	Environments map[string]Environment
}

//ReadFromPath returns a new SanicConfig from the given filesystem path to a yaml file
func ReadFromPath(configPath string) (SanicConfig, error) {
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return SanicConfig{}, errors.New("configuration file could not be read: " + err.Error())
	}

	cfg := SanicConfig{}
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return SanicConfig{}, errors.New("configuration file error: " + err.Error())
	}
	for envName, env := range cfg.Environments {
		if env.ClusterProvisioner != "localdev" && env.ClusterProvisioner != "" {
			return SanicConfig{}, errors.New(fmt.Sprintf(
				"configuration file error: environment %s's"+
					" clusterProvisioner key must be one of 'localdev' or omitted, was: '%s'",
				envName, env.ClusterProvisioner))
		}
	}

	return cfg, nil
}

//Read returns a new SanicConfig, given that the environment (e.g., sanic env) has one configured
func Read() (SanicConfig, error) {
	configPath := os.Getenv("SANIC_CONFIG") //TODO shouldn't be reading env vars here
	if configPath == "" {
		return SanicConfig{}, errors.New("enter an environment with 'sanic env'")
	}

	return ReadFromPath(configPath)
}

//HasEnvironment returns the configuration has a given environment defined
func (cfg *SanicConfig) HasEnvironment(env string) bool {
	_, exists := cfg.Environments[env]
	return exists
}

//CurrentEnvironment returns an Environment struct corresponding to the environment the user is in.
//Fails if the user is not in an environment
func (cfg *SanicConfig) CurrentEnvironment(s shell.Shell) (*Environment, error) {
	if ret, exists := cfg.Environments[s.GetSanicEnvironment()]; exists {
		return &ret, nil
	}
	return nil, errors.New("the environment " + s.GetSanicEnvironment() + " does not exist in the project '" + filepath.Base(s.GetSanicRoot()) + `'`)
}
