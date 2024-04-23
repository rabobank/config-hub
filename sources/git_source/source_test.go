package git_source

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"
)

// func TestReadYamlFile(t *testing.T) {
//
// 	if flattenedProperties, e := readYamlFile(openFile("../../tests/configuration-1.yml")); e != nil {
// 		t.Error(e)
// 	} else if expectedPropertiesFile, e := os.Open("../../tests/configuration-1.json"); e != nil {
// 		t.Error(e)
// 	} else {
// 		expectedProperties := make(map[string]interface{})
// 		json.NewDecoder(expectedPropertiesFile).Decode(&expectedProperties)
// 		if !areEqual(flattenedProperties, expectedProperties) {
// 			t.Error("Flattened properties are not as expected")
// 		}
// 		// enc := json.NewEncoder(os.Stdout)
// 		// enc.SetIndent("", "  ")
// 		//
// 		// enc.Encode(flattenedProperties)
// 	}
// }

func areEqual(flattened map[string]interface{}, expected map[string]interface{}) bool {
	if len(flattened) != len(expected) {
		return false
	}
	for k, v := range expected {
		if v2, ok := flattened[k]; !ok {
			return false
		} else if v2 != nil && reflect.TypeOf(v2).Kind() == reflect.Int {
			if int(v.(float64)) != v2.(int) {
				fmt.Println(k)
				return false
			}
		} else if v2 != nil && reflect.TypeOf(v2).Kind() == reflect.Array {
			if _, isType := v.([]interface{}); !isType || len(v2.([]interface{})) != 0 || len(v.([]interface{})) != 0 {
				// the only way an array will go through is if it's empty
				return false
			}
		} else if v2 != nil && reflect.TypeOf(v2).Kind() == reflect.Slice {
			if _, isType := v.([]interface{}); !isType || len(v2.([]interface{})) != 0 || len(v.([]interface{})) != 0 {
				// the only way an array will go through is if it's empty
				return false
			}
		} else if v2 != v {
			fmt.Println(k)
			return false
		}
	}
	return true
}

func ReadYamlFile2(t *testing.T) {

	if flattenedProperties, e := readYamlFile(openFile("../../tests/configuration-2.yml")); e != nil {
		t.Error(e)
	} else if expectedPropertiesFile, e := os.Open("../../tests/configuration-2.json"); e != nil {
		t.Error(e)
	} else {
		expectedProperties := make(map[string]interface{})
		json.NewDecoder(expectedPropertiesFile).Decode(&expectedProperties)
		if !areEqual(flattenedProperties, expectedProperties) {
			t.Error("Flattened properties are not as expected")
		}
		// enc := json.NewEncoder(os.Stdout)
		// enc.SetIndent("", "  ")
		//
		// enc.Encode(flattenedProperties)
	}
}
