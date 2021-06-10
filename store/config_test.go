package store

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

const (
	testAccountToken = "testAccountToken"
	testAPIToken     = "testApiToken"
	testRegion       = "testRegion"
	listenerURL      = "https://listener.logz.io:8071"
	listenerURLEu    = "https://listener-eu.logz.io:8071"
	apiURL           = "https://api.logz.io/v1/elasticsearch/_msearch"
	apiURLEu         = "https://api-eu.logz.io/v1/elasticsearch/_msearch"
)

func TestValidate(tester *testing.T) {
	config := LogzioConfig{
		AccountToken: "",
		APIToken:     "",
		Region:       "",
	}

	assert.Error(tester, config.validate(logger), "validation failed, empty account token and api token should produce error")

	config.APIToken = testAccountToken
	assert.NoError(tester, config.validate(logger), "validation failed, one of api token or account token can be empty")

	config.AccountToken = testAccountToken
	config.APIToken = ""
	assert.NoError(tester, config.validate(logger), "validation failed, one of api token or account token can be empty")

}

func TestRegion(tester *testing.T) {
	config := LogzioConfig{
		AccountToken: testAccountToken,
		APIToken:     testAccountToken,
		Region:       "",
	}

	assert.Equal(tester, config.ListenerURL(), listenerURL, "listener url incorrect")
	assert.Equal(tester, config.APIURL(), apiURL, "api url incorrect")

	config.Region = "us"
	assert.Equal(tester, config.ListenerURL(), listenerURL, "listener url incorrect")
	assert.Equal(tester, config.APIURL(), apiURL, "api url incorrect")

	config.Region = "eu"
	assert.Equal(tester, config.ListenerURL(), listenerURLEu, "listener url incorrect")
	assert.Equal(tester, config.APIURL(), apiURLEu, "api url incorrect")
}

func TestCustomQueueDir(tester *testing.T) {
	config := LogzioConfig{
		AccountToken: testAccountToken,
		APIToken:     testAccountToken,
		Region:       "",
		CustomQueueDir: "",
	}
	s := string(os.PathSeparator)
	valid := fmt.Sprintf("%s%s%s%s%s%s", os.Getenv("HOME"), s,"tmp",s, "logzio-buffer", s)
	assert.Equal(tester, config.customQueueDir(), valid, "custom dir path is incorrect")


	config.CustomQueueDir = "/tmp"
	valid = fmt.Sprintf("%s%s%s%s", "/tmp",s, "logzio-buffer", s)
	assert.Equal(tester, config.customQueueDir(), valid, "custom dir path is incorrect")

	config.CustomQueueDir = "/tmp/"
	valid = fmt.Sprintf("%s%s%s%s", "/tmp",s, "logzio-buffer", s)
	assert.Equal(tester, config.customQueueDir(), valid, "custom dir path is incorrect")
}


func TestParseConfig(tester *testing.T) {
	config, err := ParseConfig("fixtures/testConfig.yaml", logger)
	assert.NoError(tester, err)

	assert.Equal(tester, config.Region, testRegion)
	assert.Equal(tester, config.AccountToken, testAccountToken)
	assert.Equal(tester, config.APIToken, testAPIToken)
}
