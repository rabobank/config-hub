package domain

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/gomatbase/csn"
	"github.com/rabobank/config-hub/util"
)

const (
	DefaultFetchCacheTtl = 60
	MinimumFetchCacheTtl = 60
)

type GitConfig struct {
	SourceType        string   `json:"type"`
	DeepClone         bool     `json:"deepClone,omitempty"`
	Uri               string   `json:"uri"`
	DefaultLabel      *string  `json:"defaultLabel,omitempty"`
	SearchPaths       []string `json:"searchPaths,omitempty"`
	SkipSslValidation bool     `json:"skipSslValidation"`
	FailOnFetch       bool     `json:"failOnFetch,omitempty"`
	FetchCacheTtl     int      `json:"fetchCacheTtl,omitempty"`

	// Optional parameters for user/password credentials. Also used for az Mi Wif credentials
	Username *string `json:"username,omitempty"`
	Password *string `json:"password,omitempty"`

	// Optional parameter for ssh private key
	PrivateKey *string `json:"privateKey,omitempty"`

	// Optional parameters for azure based authentication with az app registration and credhub stored credentials
	AzTenantId *string `json:"azTenantId,omitempty"`

	// Spn based credentials az app registration and credhub stored credentials
	AzSpn                    bool    `json:"-"`
	AzClient                 *string `json:"azClient,omitempty"`
	AzSecret                 *string `json:"azSecret,omitempty"`
	AzSecretCredhubReference *string `json:"azSecret-credhub-ref,omitempty"`
	AzSecretCredhubClient    *string `json:"azSecret-credhub-client,omitempty"`
	AzSecretCredhubSecret    *string `json:"azSecret-credhub-secret,omitempty"`

	// MI based credentials
	AzMi          bool    `json:"-"`
	AzMiId        *string `json:"azMiId,omitempty"`
	AzMiWifIssuer *string `json:"azMiWifIssuer,omitempty"`
	AzMiWifClient *string `json:"azMiWifClient,omitempty"`
	AzMiWifSecret *string `json:"azMiWifSecret,omitempty"`
}

func stringOrNull(value *string) string {
	if value == nil {
		return "null"
	}
	return *value
}

func (gc *GitConfig) String() string {
	return fmt.Sprintf("GitConfig{Uri:%s, DeepClone: %v, DefaultLabel:%s, SearchPaths:%s, Username:%s, Password:%v, PrivateKey:%v, SkipSslValidation:%v, FailOnFetch: %v, AzMiId: %s}",
		gc.Uri, gc.DeepClone, stringOrNull(gc.DefaultLabel), gc.SearchPaths, stringOrNull(gc.Username), gc.Password != nil && len(*gc.Password) != 0, gc.PrivateKey != nil && len(*gc.PrivateKey) != 0, gc.SkipSslValidation, gc.FailOnFetch, util.EmptyIfNil(gc.AzMiId))
}

func (gc *GitConfig) Type() string {
	return gc.SourceType
}

func (gc *GitConfig) FromMap(properties map[string]any) error {
	if properties == nil {
		return nil
	}

	errors := csn.Errors()
	errors.Add(extract(Mandatory, properties, "type", &gc.SourceType))
	if gc.SourceType != "git" {
		errors.AddErrorMessage(fmt.Sprintf("reading git source configuration from incompatible source type : %s", gc.SourceType))
	}

	errors.Add(extract(Mandatory, properties, "uri", &gc.Uri))
	if uri, e := url.Parse(gc.Uri); e != nil {
		if !strings.HasPrefix(gc.Uri, "git@") {
			errors.AddErrorMessage(fmt.Sprintf("reading git source configuration with invalid uri : %v", gc.Uri))
		}
	} else if uri.Scheme != "http" && uri.Scheme != "https" {
		errors.AddErrorMessage(fmt.Sprintf("reading git source configuration with incompatible uri scheme : %s", uri.Scheme))
	}

	errors.Add(extractPtr(Optional, properties, "defaultLabel", &gc.DefaultLabel))
	errors.Add(extract(Optional, properties, "deepClone", &gc.DeepClone))

	var searchPaths []any
	errors.Add(extract(Optional, properties, "searchPaths", &searchPaths))
	gc.SearchPaths = make([]string, len(searchPaths))
	for i, v := range searchPaths {
		if s, isType := v.(string); !isType {
			errors.AddErrorMessage(fmt.Sprintf("reading git source configuration with incompatible searchTypes array value type : %v", v))
		} else {
			gc.SearchPaths[i] = strings.TrimSpace(s)
		}
	}
	gc.SearchPaths = append(gc.SearchPaths, "")

	errors.Add(extractPtr(Optional, properties, "username", &gc.Username))
	errors.Add(extractPtr(Optional, properties, "password", &gc.Password))
	errors.Add(extractPtr(Optional, properties, "privateKey", &gc.PrivateKey))
	errors.Add(extract(Optional, properties, "skipSslValidation", &gc.SkipSslValidation))
	errors.Add(extract(Optional, properties, "failOnFetch", &gc.FailOnFetch))

	errors.Add(extract(Optional, properties, "fetchCacheTtl", &gc.FetchCacheTtl))
	if gc.FetchCacheTtl < MinimumFetchCacheTtl {
		// JV: ignoring smaller values, but perhaps an error can also be raised
		gc.FetchCacheTtl = DefaultFetchCacheTtl
	}

	// extract az based credentials, if present
	errors.Add(extractPtr(Optional, properties, "azTenantId", &gc.AzTenantId))
	// Az SPN based credentials potentially with a credhub service instance holding the SPN secret
	errors.Add(extractPtr(Optional, properties, "azClient", &gc.AzClient))
	errors.Add(extractPtr(Optional, properties, "azSecret", &gc.AzSecret))
	errors.Add(extractPtr(Optional, properties, "azSecret-credhub-ref", &gc.AzSecretCredhubReference))
	errors.Add(extractPtr(Optional, properties, "azSecret-credhub-client", &gc.AzSecretCredhubClient))
	errors.Add(extractPtr(Optional, properties, "azSecret-credhub-secret", &gc.AzSecretCredhubSecret))
	// Az MI WIF based credentials (username and password would in this case have the WIF credentials
	errors.Add(extractPtr(Optional, properties, "azMiId", &gc.AzMiId))
	errors.Add(extractPtr(Optional, properties, "azMiWifIssuer", &gc.AzMiWifIssuer))
	errors.Add(extractPtr(Optional, properties, "azMiWifClient", &gc.AzMiWifClient))
	errors.Add(extractPtr(Optional, properties, "azMiWifSecret", &gc.AzMiWifSecret))

	// if Tenant id is given check that either SPN or MI wif creadentials are fully given
	if gc.AzTenantId != nil {

		if gc.AzClient != nil || gc.AzSecret != nil || gc.AzSecretCredhubReference != nil {
			if gc.AzClient == nil || gc.AzSecret == nil && gc.AzSecretCredhubReference == nil {
				errors.AddErrorMessage("Invalid AZ SPN configuration. It requires Tenant ID, Client ID and Secret to be defined.")
			} else {
				gc.AzSpn = true
			}
		}

		if gc.AzMiId != nil || gc.AzMiWifIssuer != nil {
			if gc.AzMiId == nil || gc.AzMiWifIssuer == nil || gc.Username == nil || gc.Password == nil {
				errors.AddErrorMessage("Invalid AZ MI configuration. It requires Tenant ID, Az MI name and WIF issuer credentials (username/password).")
			} else {
				gc.AzMi = true
			}
		}

		if gc.AzSpn && gc.AzMi {
			errors.AddErrorMessage("Configuring both AZ SPN and AZ MI is not supported")
		} else if !gc.AzMi && !gc.AzSpn {
			errors.AddErrorMessage("Az Tenant ID provided and neither a valid MI nor SPN configuration was provided")
		}
	}

	return errors.NilIfEmpty()
}
