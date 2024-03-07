package git_source

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

func TestReadYamlFile(t *testing.T) {

	if flattenedProperties, e := readYamlFile(openFile("../../tests/configuration-1.yml")); e != nil {
		t.Error(e)
	} else if expectedPropertiesFile, e := os.Open("../../tests/configuration-1.json"); e != nil {
		t.Error(e)
	} else {
		expectedProperties := make(map[string]interface{})
		json.NewDecoder(expectedPropertiesFile).Decode(&expectedProperties)
		if !areEqual(flattenedProperties, expectedProperties) {
			t.Error("Flattened properties are not as expected")
		}
	}
}

func areEqual(flattened map[string]interface{}, expected map[string]interface{}) bool {
	if len(flattened) != len(expected) {
		return false
	}
	for k, v := range expected {
		if v2, ok := flattened[k]; !ok {
			return false
		} else if reflect.TypeOf(v2).Kind() == reflect.Int {
			if int(v.(float64)) != v2.(int) {
				return false
			}
		} else if v2 != v {
			return false
		}
	}
	return true
}
