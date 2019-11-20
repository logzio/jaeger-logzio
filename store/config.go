package store

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/spf13/viper"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

const (
	defaultListenerURL = "https://listener.logz.io:8071"
	accountTokenParam  = "accountToken"
	apiTokenParam      = "apiToken"
	listenerURLParam   = "listenerURL"
)

// LogzioConfig struct for logzio span store
type LogzioConfig struct {
	AccountToken string `yaml:"accountToken"`
	APIToken     string `yaml:"apiToken"`
	ListenerURL  string `yaml:"listenerURL"`
}

// Validate logzio config, return error if invalid
func (config *LogzioConfig) Validate() error {
	if config.ListenerURL == "" {
		config.ListenerURL = defaultListenerURL
	}

	if config.AccountToken == "" {
		return errors.New("account token is empty, can't create span writer")
	}

	return nil
}

//ParseConfig receives config file  path, parse it  and  return logzio span store config
func ParseConfig(filePath string) (LogzioConfig, error) {
	if filePath != "" {
		logzioConfig := LogzioConfig{}
		yamlFile, err := ioutil.ReadFile(filePath)
		if err != nil {
			return logzioConfig, err
		}
		err = yaml.Unmarshal(yamlFile, &logzioConfig)
		return logzioConfig, err
	}
	v := viper.New()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.SetDefault(listenerURLParam, defaultListenerURL)
	v.SetDefault(apiTokenParam, "")
	v.AutomaticEnv()

	logzioConfig := LogzioConfig{
		ListenerURL:  v.GetString(listenerURLParam),
		AccountToken: v.GetString(accountTokenParam),
		APIToken:     v.GetString(apiTokenParam),
	}
	return logzioConfig, nil
}

func (config *LogzioConfig) String() string {
	desc := fmt.Sprintf("account token: %v \n api token: %v \n listener url: %v", config.AccountToken, config.APIToken, config.ListenerURL)
	return desc
}
