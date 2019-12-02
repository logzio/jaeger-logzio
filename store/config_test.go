package store

import (
	"testing"
)

const (
	testAccountToken = "testAccountToken"
	testAPIToken     = "testApiToken"
	testRegion       = "testRegion"
)

func TestValidate(tester *testing.T) {
	config := LogzioConfig{
		AccountToken: "",
		APIToken:     "",
		Region:       "",
	}
	err := config.Validate()
	if err == nil {
		tester.Error("validation failed, empty account token and api token should produce error")
	}

	config.APIToken = testAccountToken
	err = config.Validate()
	if err != nil {
		tester.Error("validation failed, one of api token or account token can be empty")
	}

	config.AccountToken = testAccountToken
	config.APIToken = ""
	err = config.Validate()
	if err != nil {
		tester.Error("validation failed, one of api token or account token can be empty")
	}
}

func TestRegion(tester *testing.T) {
	config := LogzioConfig{
		AccountToken: testAccountToken,
		APIToken:     testAccountToken,
		Region:       "",
	}

	if config.ListenerURL() != "https://listener.logz.io:8071" {
		tester.Errorf("listener url incorrect, got: %s, expected: https://listener.logz.io:8071", config.ListenerURL())
	}

	if config.APIURL() != "https://api.logz.io/v1/elasticsearch/_msearch" {
		tester.Errorf("api url incorrect, got: %s, expected: https://api.logz.io/v1/elasticsearch/_msearch", config.APIURL())
	}

	config.Region = "na"
	if config.ListenerURL() != "https://listener.logz.io:8071" {
		tester.Errorf("listener url incorrect, got: %s, expected: https://listener.logz.io:8071", config.ListenerURL())
	}
	if config.APIURL() != "https://api.logz.io/v1/elasticsearch/_msearch" {
		tester.Errorf("api url incorrect, got: %s, expected: https://api.logz.io/v1/elasticsearch/_msearch", config.APIURL())
	}

	config.Region = "eu"
	if config.ListenerURL() != "https://listener-eu.logz.io:8071" {
		tester.Errorf("listener url incorrect, got: %s, expected: https://listener.logz.io:8071", config.ListenerURL())
	}
	if config.APIURL() != "https://api-eu.logz.io/v1/elasticsearch/_msearch" {
		tester.Errorf("api url incorrect, got: %s, expected: https://api-eu.logz.io/v1/elasticsearch/_msearch", config.APIURL())
	}
}

func TestParseConfig(tester *testing.T) {
	config, err := ParseConfig("fixtures/testConfig.yaml")
	if err != nil {
		tester.Errorf("error parsing config file: %s", err.Error())
		return
	}
	if config.Region != testRegion {
		tester.Errorf("wrong listener, expected: testURL, got: %s", config.Region)
	}
	if config.AccountToken != testAccountToken {
		tester.Errorf("wrong account token, expected: testAccountTo	ken, got: %s", config.AccountToken)
	}
	if config.APIToken != testAPIToken {
		tester.Errorf("wrong api token, expected: testApiToken, got: %s", config.AccountToken)
	}
}
