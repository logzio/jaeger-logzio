package store

import (
	"io/ioutil"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

const (
	defaultListenerURL = "https://listener.logz.io:8071"
)

// LogzioConfig struct for logzio span store
type LogzioConfig struct {
	AccountToken string `yaml:"accountToken"`
	APIToken     string `yaml:"apiToken"`
	ListenerURL  string `yaml:"listenerURL"`
}

// Validate logzio config, return error if invalid
func (config *LogzioConfig) Validate() error {
	if config.AccountToken == "" {
		return errors.New("account token is empty, can't create span writer")
	}

	if config.ListenerURL == "" {
		config.ListenerURL = defaultListenerURL
	}

	return nil
}

//ParseConfig receives config file  path, parse it  and  return logzio span store config
func ParseConfig(filePath string) (LogzioConfig, error) {
	logzioConfig := LogzioConfig{}
	yamlFile, err := ioutil.ReadFile(filePath)
	if err != nil {
		return logzioConfig, err
	}
	err = yaml.Unmarshal(yamlFile, &logzioConfig)
	return logzioConfig, err
}

func (config *LogzioConfig) String() string {
	desc := "account token: " + config.AccountToken +
		"\n api token: " + config.APIToken
	if config.ListenerURL != "" {
		desc += "\n listener host: " + config.ListenerURL
	}
	return desc
}
