package domain

import (
	"fmt"

	"github.com/gomatbase/csn"
)

const (
	Mandatory = true
	Optional  = false
)

func extract[T any](mandatory bool, properties map[string]any, property string, placeholder *T) error {
	if value, found := properties[property]; !found {
		if mandatory {
			return csn.Error(fmt.Sprintf("reading git source configuration from source without %s", property))
		}
	} else if v, isType := value.(T); !isType {
		return csn.Error(fmt.Sprintf("reading git source configuration with invalid %s : %v", property, value))
	} else {
		*placeholder = v
	}
	return nil
}

func extractPtr[T any](mandatory bool, properties map[string]any, property string, placeholder **T) error {
	if value, found := properties[property]; !found {
		if mandatory {
			return csn.Error(fmt.Sprintf("reading git source configuration from source without %s", property))
		}
	} else if v, isType := value.(T); !isType {
		return csn.Error(fmt.Sprintf("reading git source configuration with invalid %s : %v", property, value))
	} else {
		*placeholder = &v
	}
	return nil
}
