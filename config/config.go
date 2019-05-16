package config

import (
	"errors"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
)

type Command struct {
	Name string
	Command string
}

type Environment struct {
	Commands []Command
}

type SanicConfig struct {
	Environments map[string]Environment
}

func ReadFromPath(configPath string) (*SanicConfig, error) {
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, errors.New("configuration file could not be read: " + err.Error())
	}

	ret := new(SanicConfig)
	err = yaml.UnmarshalStrict(data, ret)
	if err != nil {
		return nil, errors.New("configuration file error: " + err.Error())
	}

	return ret, nil
}

func Read() (*SanicConfig, error) {
	configPath := os.Getenv("SANIC_CONFIG")
	if configPath == "" {
		return nil, errors.New("enter an environment with 'sanic env'")
	}

	return ReadFromPath(configPath)
}

func CurrentEnvironment(cfg *SanicConfig) (*Environment, error) {
	sanicEnv := os.Getenv("SANIC_ENV")
	if ret, exists := cfg.Environments[sanicEnv]; exists {
		return &ret, nil
	}
	return nil, errors.New("the environment you are you does not exist, ensure it was not removed from the configuration")
}

