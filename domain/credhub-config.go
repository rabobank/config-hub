package domain

import (
	"fmt"

	"github.com/gomatbase/csn"
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

	errors := csn.Errors()
	errors.Add(extract(Mandatory, properties, "type", &cc.SourceType))
	if cc.SourceType != "credhub" {
		errors.AddErrorMessage(fmt.Sprintf("reading credhub source configuration from incompatible source type : %s", cc.SourceType))
	}

	errors.Add(extract(Mandatory, properties, "prefix", &cc.Prefix))
	errors.Add(extractPtr(Optional, properties, "client", &cc.Client))
	errors.Add(extractPtr(Optional, properties, "secret", &cc.Secret))

	if (cc.Client == nil) != (cc.Secret == nil) {
		errors.AddErrorMessage("if either client or secret is provided both must be provided")
	}

	return errors.NilIfEmpty()
}
