package store

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/hashicorp/go-hclog"

	"github.com/spf13/viper"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

const (
	accountTokenParam   = "ACCOUNT_TOKEN"
	apiTokenParam       = "API_TOKEN"
	regionParam         = "REGION"
	customListenerParam = "CUSTOM_LISTENER_URL"
	customAPIParam      = "CUSTOM_API"
	usRegionCode        = "us"
)

// LogzioConfig struct for logzio span store
type LogzioConfig struct {
	AccountToken      string `yaml:"accountToken"`
	Region            string `yaml:"region"`
	APIToken          string `yaml:"apiToken"`
	CustomListenerURL string `yaml:"customListenerUrl"`
	CustomAPIURL      string `yaml:"customAPIUrl"`
}

// validate logzio config, return error if invalid
func (config *LogzioConfig) validate(logger hclog.Logger) error {
	if config.AccountToken == "" && config.APIToken == "" {
		return errors.New("At least one of logz.io account token or api-token has to be valid")
	}
	if config.APIToken == "" {
		logger.Warn("No api token found, can't create span reader")
	}
	if config.AccountToken == "" {
		logger.Warn("No account token found, spans will not be saved")
	}
	return nil
}

//ParseConfig receives a config file path, parse it and returns logzio span store config
func ParseConfig(filePath string, logger hclog.Logger) (*LogzioConfig, error) {
	var logzioConfig *LogzioConfig
	if filePath != "" {
		logzioConfig = &LogzioConfig{}
		yamlFile, err := ioutil.ReadFile(filePath)
		if err != nil {
			return nil, err
		}
		err = yaml.Unmarshal(yamlFile, &logzioConfig)
	} else {
		v := viper.New()
		v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
		v.SetDefault(regionParam, "")
		v.SetDefault(customAPIParam, "")
		v.SetDefault(customListenerParam, "")
		v.AutomaticEnv()

		logzioConfig = &LogzioConfig{
			Region:            v.GetString(regionParam),
			AccountToken:      v.GetString(accountTokenParam),
			APIToken:          v.GetString(apiTokenParam),
			CustomAPIURL:      v.GetString(customAPIParam),
			CustomListenerURL: v.GetString(customListenerParam),
		}
	}

	if err := logzioConfig.validate(logger); err != nil {
		return nil, err
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
	if config.Region != "" && config.Region != usRegionCode {
		regionCode = fmt.Sprintf("-%s", config.Region)
	}
	return regionCode
}

func (config *LogzioConfig) String() string {
	desc := fmt.Sprintf("account token: %v \n api token: %v \n listener url: %v \n api url: %s", censorString(config.AccountToken, 4), censorString(config.APIToken, 9), config.ListenerURL(), config.APIURL())
	return desc
}

func censorString(word string, n int) string {
	if len(word) > 2*n {
		return word[:n] + strings.Repeat("*", len(word)-(n*2)) + word[len(word)-n:]
	}
	return strings.Repeat("*", len(word))
}
