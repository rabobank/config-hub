package domain

import (
	"fmt"

	err "github.com/gomatbase/go-error"
)

type CredhubConfig struct {
	SourceType string  `json:"type"`
	Client     *string `json:"client,omitempty"`
	Secret     *string `json:"secret,omitempty"`
	Prefix     string  `json:"prefix"`
}

func (cc *CredhubConfig) Type() string {
	return cc.SourceType
}

func (cc *CredhubConfig) FromMap(properties map[string]interface{}) error {
	if properties == nil {
		return nil
	}

	errors := err.Errors()
	if value, found := properties["type"]; !found {
		errors.Add("reading credhub source configuration from source without type")
	} else if v, isType := value.(string); !isType || v != "credhub" {
		errors.Add(fmt.Sprintf("reading credhub source configuration from incompatible source type : %v", value))
	} else {
		cc.SourceType = v
	}

	if value, found := properties["prefix"]; !found {
		errors.Add("reading credhub source configuration from source without a namespace prefix")
	} else if v, isType := value.(string); !isType {
		errors.Add(fmt.Sprintf("reading credhub source configuration with incompatible prefix type : %v", value))
	} else {
		cc.Prefix = v
	}

	if value, found := properties["client"]; found {
		if _, found = properties["secret"]; !found {
			errors.Add("providing client name for credhub source configuration without a secret")
		} else if v, isType := value.(string); !isType {
			errors.Add(fmt.Sprintf("reading credhub source configuration with incompatible client name type : %v", value))
		} else {
			cc.Client = &v
		}
	}

	if value, found := properties["secret"]; found {
		if _, found = properties["client"]; !found {
			errors.Add("providing client secret for credhub source configuration without a client name")
		} else if v, isType := value.(string); !isType {
			errors.Add("reading credhub source configuration with incompatible client secret type")
		} else {
			cc.Secret = &v
		}
	}

	if errors.Count() > 0 {
		return errors
	}
	return nil
}
