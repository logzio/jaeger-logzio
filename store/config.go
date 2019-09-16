package store

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
	"jaeger-logzio/_vendor-20190915113043/github.com/pkg/errors"
)

// LogzioConfig struct for logzio span store
type LogzioConfig struct {
	AccountToken string `yaml:"accountToken"`
	APIToken     string `yaml:"apiToken"`
	ListenerHost string `yaml:"listenerHost"`
}

// Validate logzio config, return error if invalid
func (config *LogzioConfig) Validate() error {
	if config.AccountToken == "" {
		return errors.New("account token is empty, can't create span writer")
	}
	return nil
}

//ParseConfig receives config file  path, parse it  and  return logzio span store config
func ParseConfig(filePath string) (LogzioConfig, error) {
	logzioConfig := LogzioConfig{}
	yamlFile, err := ioutil.ReadFile(filePath)
	if err == nil {
		err = yaml.Unmarshal(yamlFile, &logzioConfig)
	}
	return logzioConfig, err
}

func (config *LogzioConfig) String() string {
	desc := "account token: " + config.AccountToken +
		"\n api token: " + config.APIToken
	if config.ListenerHost != "" {
		desc += "\n listener host: " + config.ListenerHost
	}
	return desc
}
