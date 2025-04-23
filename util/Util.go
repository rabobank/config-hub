package util

import (
	"github.com/rabobank/credhub-client"
)

func HasApplication(apps []string) bool {
	for _, app := range apps {
		if app == "application" {
			return true
		}
	}
	return false
}

func EmptyIfNil(str *string) string {
	if str == nil {
		return ""
	}
	return *str
}

func CredhubClient(client, secret *string) (c credhub.Client, e error) {
	if client != nil && secret != nil {
		return credhub.New(&credhub.Options{
			Client: *client,
			Secret: *secret,
		})
	} else {
		return credhub.New(nil)
	}
}
