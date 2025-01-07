package domain

import (
	"fmt"
	"net/url"
	"strings"

	err "github.com/gomatbase/go-error"
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
	Username          *string  `json:"username,omitempty"`
	Password          *string  `json:"password,omitempty"`
	PrivateKey        *string  `json:"privateKey,omitempty"`
	SkipSslValidation bool     `json:"skipSslValidation"`
	FailOnFetch       bool     `json:"failOnFetch,omitempty"`
	FetchCacheTtl     int      `json:"fetchCacheTtl,omitempty"`
}

func stringOrNull(value *string) string {
	if value == nil {
		return "null"
	}
	return *value
}

func (gc *GitConfig) String() string {
	return fmt.Sprintf("GitConfig{Uri:%s, DeepClone: %v, DefaultLabel:%s, SearchPaths:%s, Username:%s, Password:%v, PrivateKey:%v, SkipSslValidation:%v, FailOnFetch: %v}",
		gc.Uri, gc.DeepClone, stringOrNull(gc.DefaultLabel), gc.SearchPaths, stringOrNull(gc.Username), gc.Password != nil && len(*gc.Password) != 0, gc.PrivateKey != nil && len(*gc.PrivateKey) != 0, gc.SkipSslValidation, gc.FailOnFetch)
}

func (gc *GitConfig) Type() string {
	return gc.SourceType
}

func (gc *GitConfig) FromMap(properties map[string]interface{}) error {
	if properties == nil {
		return nil
	}

	errors := err.Errors()
	if value, found := properties["type"]; !found {
		errors.Add("reading git source configuration from source without type")
	} else if v, isType := value.(string); !isType || v != "git" {
		errors.Add(fmt.Sprintf("reading git source configuration from incompatible source type : %v", value))
	} else {
		gc.SourceType = v
	}

	if value, found := properties["uri"]; !found {
		errors.Add("reading git source configuration from source without a uri")
	} else if v, isType := value.(string); !isType {
		errors.Add(fmt.Sprintf("reading git source configuration with incompatible uri type : %v", value))
	} else if uri, e := url.Parse(v); e != nil {
		if !strings.HasPrefix(v, "git@") {
			errors.Add(fmt.Sprintf("reading git source configuration with invalid uri : %v", value))
		} else {
			gc.Uri = v
		}
	} else if uri.Scheme != "http" && uri.Scheme != "https" {
		errors.Add(fmt.Sprintf("reading git source configuration with incompatible uri scheme : %s", uri.Scheme))
	} else {
		gc.Uri = v
	}

	if value, found := properties["defaultLabel"]; found {
		if v, isType := value.(string); !isType {
			errors.Add(fmt.Sprintf("reading git source configuration with incompatible defaultLabele type : %v", value))
		} else {
			gc.DefaultLabel = &v
		}
	}

	if value, found := properties["deepClone"]; found {
		if v, isType := value.(bool); !isType {
			errors.Add(fmt.Sprintf("reading git source configuration with incompatible deepClone type : %v", value))
		} else {
			gc.DeepClone = v
		}
	}

	if value, found := properties["searchPaths"]; found {
		if v, isType := value.([]interface{}); !isType {
			errors.Add(fmt.Sprintf("reading git source configuration with incompatible searchTypes type : %v", value))
		} else {
			gc.SearchPaths = make([]string, len(v))
			for i, av := range v {
				if s, isType := av.(string); !isType {
					errors.Add(fmt.Sprintf("reading git source configuration with incompatible searchTypes array value type : %v", av))
				} else {
					gc.SearchPaths[i] = strings.TrimSpace(s)
				}
			}
		}
	}
	gc.SearchPaths = append(gc.SearchPaths, "")

	if value, found := properties["defaultLabel"]; found {
		if v, isType := value.(string); !isType {
			errors.Add(fmt.Sprintf("reading git source configuration with incompatible defaultLabele type : %s", v))
		} else {
			gc.DefaultLabel = &v
		}
	}

	if value, found := properties["username"]; found {
		if v, isType := value.(string); !isType {
			errors.Add(fmt.Sprintf("reading git source configuration with incompatible username type : %s", v))
		} else {
			gc.Username = &v
		}
	}

	if value, found := properties["password"]; found {
		if v, isType := value.(string); !isType {
			errors.Add("reading git source configuration with incompatible password type")
		} else {
			gc.Password = &v
		}
	}

	if value, found := properties["privateKey"]; found {
		if v, isType := value.(string); !isType {
			errors.Add(fmt.Sprintf("reading git source configuration with incompatible privateKey type : %s", v))
		} else {
			gc.PrivateKey = &v
		}
	}

	if value, found := properties["skipSslValidation"]; found {
		if v, isType := value.(bool); !isType {
			errors.Add(fmt.Sprintf("reading git source configuration with incompatible skipSslValidation type : %v", v))
		} else {
			gc.SkipSslValidation = v
		}
	}

	if value, found := properties["failOnFetch"]; found {
		if v, isType := value.(bool); !isType {
			errors.Add(fmt.Sprintf("reading git source configuration with incompatible failOnFetch type : %v", v))
		} else {
			gc.FailOnFetch = v
		}
	}

	gc.FetchCacheTtl = DefaultFetchCacheTtl
	if value, found := properties["fetchCacheTtl"]; found {
		if v, isType := value.(int); !isType {
			errors.Add(fmt.Sprintf("reading git source configuration with incompatible fetchCacheTtl type : %v", v))
		} else if v > MinimumFetchCacheTtl {
			// JV: ignoring smaller values, but perhaps an error can also be raised
			gc.FetchCacheTtl = v
		}
	}

	if errors.Count() > 0 {
		return errors
	}

	return nil
}
