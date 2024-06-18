package util

import (
	"reflect"
	"strings"
)

func Get(key string, m map[string]any) (any, bool) {
	if v, found := m[key]; found {
		return v, true
	}
	rest := key
	key = ""
	for i := strings.Index(rest, "."); i != -1; i = strings.Index(rest, ".") {
		key += rest[:i]
		rest = rest[i+1:]
		if v, found := m[key]; found {
			if m, ok := v.(map[string]any); ok {
				if v, found := Get(rest, m); found {
					return v, true
				}
			}
		}
		key += "."
	}
	return nil, false
}

func Identical(expected, actual any) bool {
	expectedT := reflect.TypeOf(expected)
	actualT := reflect.TypeOf(actual)
	if expectedT.Kind() != actualT.Kind() {
		return false
	}
	if expectedT == nil {
		return actualT == nil
	}
	switch expectedT.Kind() {
	case reflect.Array, reflect.Slice:
		return compareArrays(expectedT, actualT)
	case reflect.Map:
		return compareMaps(expectedT, actualT)
	default:
		return expectedT.Comparable() && expected == actual
	}
}

func compareMaps(expected, actual reflect.Type) bool {
	return false
}

func compareArrays(expected, actual reflect.Type) bool {

	return false
}
