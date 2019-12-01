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
	accountTokenParam   = "accountToken"
	apiTokenParam       = "apiToken"
	regionParam         = "region"
	customListenerParam = "customListener"
	customAPIParam      = "customAPI"
)

// LogzioConfig struct for logzio span store
type LogzioConfig struct {
	AccountToken      string `yaml:"accountToken"`
	Region            string `yaml:"region"`
	APIToken          string `yaml:"apiToken"`
	CustomListenerURL string `yaml:"customListenerUrl"`
	CustomAPIURL      string `yaml:"customAPIUrl"`
}


// Validate logzio config, return error if invalid
func (config *LogzioConfig) Validate() error {
	if config.AccountToken == "" && config.APIToken == "" {
		return errors.New("At least one of logz.io account token or api-token has to be valid")
	}
	return nil
}

//ParseConfig receives a config file path, parse it and returns logzio span store config
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
	v.SetDefault(regionParam, "")
	v.SetDefault(customAPIParam, "")
	v.SetDefault(customListenerParam, "")
	v.AutomaticEnv()

	logzioConfig := LogzioConfig{
		Region:            v.GetString(regionParam),
		AccountToken:      v.GetString(accountTokenParam),
		APIToken:          v.GetString(apiTokenParam),
		CustomAPIURL:      v.GetString(customAPIParam),
		CustomListenerURL: v.GetString(customListenerParam),
	}
	return logzioConfig, nil
}

// ListenerURL returns the constructed listener URL to write spans to
func (config *LogzioConfig) ListenerURL() string {
	if config.CustomListenerURL != "" {
		return config.CustomListenerURL
	}
	return fmt.Sprintf("https://listener%s.logz.io:8071", config.regionCode())
}

// APIURL returns the constructed API URL to read spans from
func (config *LogzioConfig) APIURL() string {
	if config.CustomAPIURL != "" {
		return config.CustomAPIURL
	}
	return fmt.Sprintf("https://api%s.logz.io/v1/elasticsearch/_msearch", config.regionCode())
}

func (config *LogzioConfig) regionCode() string {
	regionCode := ""
	if config.Region != "" && config.Region != "na" {
		regionCode = fmt.Sprintf("-%s", config.Region)
	}
	return regionCode
}

func (config *LogzioConfig) String() string {
	desc := fmt.Sprintf("account token: %v \n api token: %v \n listener url: %v", config.AccountToken, config.APIURL(), config.ListenerURL())
	return desc
}
