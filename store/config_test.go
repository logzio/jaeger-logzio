package store

import (
	"testing"
)

func TestValidate(tester *testing.T) {
	config := LogzioConfig{
		AccountToken:	"",
		ListenerURL:	"",
	}
	err := config.Validate()
	if err == nil {
		tester.Error("validation failed, empty account token should produce error")
	}

	if config.ListenerURL != "https://listener.logz.io:8071" {
		tester.Errorf("listener url incorrect, got: %s, expected: https://listener.logz.io:8071", config.ListenerURL)
	}

	config.ListenerURL = "fakeURL"
	config.Validate()
	if config.ListenerURL != "fakeURL" {
		tester.Error("listener url changed, should have stayed the same")
	}
}

func TestParseConfig(tester *testing.T) {
	config, err := ParseConfig("../testConfig.yaml")
	if err != nil {
		tester.Errorf("error parsing config file: %s", err.Error())
		return
	}
	if config.ListenerURL != "testURL" {
		tester.Errorf("wrong listener, expected: testURL, got: %s", config.ListenerURL)
	}
	if config.AccountToken != "fakeToken" {
		tester.Errorf("wrong account token, expected: fakeToken, got: %s", config.AccountToken)
	}
}