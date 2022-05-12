package store

import (
	"fmt"
	"github.com/hashicorp/go-hclog"
	"github.com/spf13/viper"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

const (
	accountTokenParam     = "ACCOUNT_TOKEN"
	apiTokenParam         = "API_TOKEN"
	regionParam           = "REGION"
	customListenerParam   = "CUSTOM_LISTENER_URL"
	customAPIParam        = "CUSTOM_API"
	usRegionCode          = "us"
	customQueueDirParam   = "CUSTOM_QUEUE_DIR"
	inMemoryQueueParam    = "IN_MEMORY_QUEUE"
	CompressParam         = "COMPRESS"
	InMemoryCapacityParam = "IN_MEMORY_CAPACITY"
	LogCountLimitParam    = "LOG_COUNT_LIMIT"
	DrainIntervalParam    = "DRAIN_INTERVAL"
	// default values for in memory queue config
	defaultInMemoryCapacity = uint64(20 * 1024 * 1024)
	defaultLogCountLimit    = 500000
	defaultDrainInterval    = 3
)

// LogzioConfig struct for logzio span store
type LogzioConfig struct {
	AccountToken      string `yaml:"accountToken"`
	Region            string `yaml:"region"`
	APIToken          string `yaml:"apiToken"`
	CustomListenerURL string `yaml:"customListenerUrl"`
	CustomAPIURL      string `yaml:"customAPIUrl"`
	CustomQueueDir    string `yaml:"customQueueDir"`
	InMemoryQueue     bool   `yaml:"inMemoryQueue"`
	Compress          bool   `yaml:"compress"`
	InMemoryCapacity  uint64 `yaml:"inMemoryCapacity"`
	LogCountLimit     int    `yaml:"logCountLimit"`
	DrainInterval     int    `yaml:"drainInterval"`
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
	if config.CustomQueueDir != "" {
		if _, err := os.Stat(config.CustomQueueDir); os.IsNotExist(err) {
			errMessage := fmt.Sprintf("%s directory does not exist", config.CustomQueueDir)
			return errors.New(errMessage)
		}
	}
	config.Region = strings.ToLower(config.Region)
	validRegionCodes := [8]string{"", "us", "eu", "nl", "ca", "wa", "uk", "au"}
	regionIsValid := false
	for _, region := range validRegionCodes {
		if config.Region == region {
			regionIsValid = true
		}
	}
	if !regionIsValid {
		warnMessage := fmt.Sprintf("%s region is not supported yet", config.Region)
		config.Region = ""
		logger.Warn(warnMessage)
	}
	logger.Log(hclog.Info, config.String())
	return nil
}

//ParseConfig receives a config file path, parse it and returns logzio span store config
func ParseConfig(filePath string, logger hclog.Logger) (*LogzioConfig, error) {
	var logzioConfig *LogzioConfig
	if filePath != "" {
		logzioConfig = &LogzioConfig{}
		// Set default values
		logzioConfig.LogCountLimit = defaultLogCountLimit
		logzioConfig.Compress = true
		logzioConfig.InMemoryCapacity = defaultInMemoryCapacity
		logzioConfig.InMemoryQueue = false
		logzioConfig.DrainInterval = defaultDrainInterval
		yamlFile, err := ioutil.ReadFile(filePath)
		if err != nil {
			return nil, err
		}
		err = yaml.Unmarshal(yamlFile, &logzioConfig)
		if err != nil {
			return nil, err
		}
	} else {
		v := viper.New()
		err := convertEnvironmentVariables()
		if err != nil {
			return nil, err
		}
		v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
		v.SetDefault(regionParam, "")
		v.SetDefault(customAPIParam, "")
		v.SetDefault(customListenerParam, "")
		v.SetDefault(customQueueDirParam, "")
		v.SetDefault(inMemoryQueueParam, false)
		v.SetDefault(CompressParam, true)
		v.SetDefault(InMemoryCapacityParam, defaultInMemoryCapacity)
		v.SetDefault(LogCountLimitParam, defaultLogCountLimit)
		v.SetDefault(DrainIntervalParam, defaultDrainInterval)
		v.AutomaticEnv()
		logzioConfig = &LogzioConfig{
			Region:            v.GetString(regionParam),
			AccountToken:      v.GetString(accountTokenParam),
			APIToken:          v.GetString(apiTokenParam),
			CustomAPIURL:      v.GetString(customAPIParam),
			CustomListenerURL: v.GetString(customListenerParam),
			CustomQueueDir:    v.GetString(customQueueDirParam),
			InMemoryQueue:     v.GetBool(inMemoryQueueParam),
			Compress:          v.GetBool(CompressParam),
			InMemoryCapacity:  v.GetUint64(InMemoryCapacityParam),
			LogCountLimit:     v.GetInt(LogCountLimitParam),
			DrainInterval:     v.GetInt(DrainIntervalParam),
		}
	}

	if err := logzioConfig.validate(logger); err != nil {
		return nil, err
	}

	return logzioConfig, nil
}

func convertEnvironmentVariables() error {
	if os.Getenv(InMemoryCapacityParam) != "" {
		if param, err := strconv.Atoi(os.Getenv(InMemoryCapacityParam)); err == nil {
			viper.Set(InMemoryCapacityParam, uint64(param))
		} else {
			return err
		}
	}
	if os.Getenv(LogCountLimitParam) != "" {
		if param, err := strconv.Atoi(os.Getenv(LogCountLimitParam)); err == nil {
			viper.Set(LogCountLimitParam, param)
		} else {
			return err
		}

	}
	if os.Getenv(DrainIntervalParam) != "" {
		if param, err := strconv.Atoi(os.Getenv(DrainIntervalParam)); err == nil {
			viper.Set(DrainIntervalParam, param)
		} else {
			return err
		}
	}
	if os.Getenv(inMemoryQueueParam) != "" {
		if param, err := strconv.ParseBool(os.Getenv(inMemoryQueueParam)); err == nil {
			viper.Set(inMemoryQueueParam, param)
		} else {
			return err
		}
	}
	if os.Getenv(CompressParam) != "" {
		if param, err := strconv.ParseBool(os.Getenv(CompressParam)); err == nil {
			viper.Set(CompressParam, param)
		} else {
			return err
		}
	}
	return nil
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
	if config.Region != "" && strings.ToLower(config.Region) != usRegionCode {
		regionCode = fmt.Sprintf("-%s", strings.ToLower(config.Region))
	}
	return regionCode
}

func (config *LogzioConfig) drainIntervalToDuration() time.Duration {
	if config.DrainInterval != 0 {
		return time.Second * time.Duration(config.DrainInterval)
	} else {
		return time.Second * defaultDrainInterval
	}

}

func (config *LogzioConfig) defaultLogCountLimit() int {
	if config.LogCountLimit != 0 {
		return config.LogCountLimit
	} else {
		return defaultLogCountLimit
	}
}

func (config *LogzioConfig) defaultInMemoryCapacity() uint64 {
	if config.InMemoryCapacity != 0 {
		return config.InMemoryCapacity
	} else {
		return defaultInMemoryCapacity
	}
}

func (config *LogzioConfig) customQueueDir() string {
	s := string(os.PathSeparator)
	if config.CustomQueueDir == "" {
		return fmt.Sprintf("%s%s%s%s%d", os.TempDir(), s, "logzio-buffer", s, time.Now().UnixNano())
	} else if strings.HasSuffix(config.CustomQueueDir, s) {
		path := config.CustomQueueDir[:len(config.CustomQueueDir)-len(s)]
		return fmt.Sprintf("%s%s%s%s%d", path, s, "logzio-buffer", s, time.Now().UnixNano())
	} else {
		return fmt.Sprintf("%s%s%s%s%d", config.CustomQueueDir, s, "logzio-buffer", s, time.Now().UnixNano())
	}
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
