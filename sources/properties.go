package sources

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	err "github.com/gomatbase/go-error"
)

func flattenProperties(prefix string, object interface{}, properties *map[string]interface{}) error {

	errors := err.Errors()

	if object == nil {
		object = ""
	}

	t := reflect.ValueOf(object).Kind()
	if t == reflect.Pointer {
		object = reflect.ValueOf(object).Elem().Interface()
		t = reflect.ValueOf(object).Kind()
	}

	switch t {
	case reflect.Map:
		// if it's a map we expect it to be a type of map[string]interface{}, although yaml allows for numbers to be keys in which case they are meant to array/map indexes...
		if m, isType := object.(map[string]interface{}); isType {
			for key, value := range m {
				if e := flattenProperties(prefix+"."+key, value, properties); e != nil {
					errors.AddError(e)
				}
			}
		} else {
			for key, value := range object.(map[any]any) {
				if reflect.TypeOf(key).Kind() == reflect.Int {
					if e := flattenProperties(fmt.Sprintf("%s[%v]", prefix, key), value, properties); e != nil {
						errors.AddError(e)
					}
				} else {
					if e := flattenProperties(fmt.Sprintf("%s.%v", prefix, key), value, properties); e != nil {
						errors.AddError(e)
					}
				}
			}
		}
	case reflect.Slice:
		// if it's an array we expect it to be a type of []]interface{}
		if len(object.([]interface{})) == 0 {
			if len(prefix) == 0 || prefix[0] != '.' {
				(*properties)[prefix] = object
			} else {
				(*properties)[prefix[1:]] = object
			}
		} else {
			for i, value := range object.([]interface{}) {
				if e := flattenProperties(prefix+"["+strconv.Itoa(i)+"]", value, properties); e != nil {
					errors.AddError(e)
				}
			}
		}
	case reflect.Array:
		// if it's an array we expect it to be a type of []]interface{}
		if len(object.([]interface{})) == 0 {
			if len(prefix) == 0 || prefix[0] != '.' {
				(*properties)[prefix] = object
			} else {
				(*properties)[prefix[1:]] = object
			}
		} else {
			for i, value := range object.([]interface{}) {
				if e := flattenProperties(prefix+"["+strconv.Itoa(i)+"]", value, properties); e != nil {
					errors.AddError(e)
				}
			}
		}
	default:
		if t == reflect.String {
			// special string-to-boolean cases
			switch strings.ToUpper(object.(string)) {
			case "OFF":
				object = false
			case "ON":
				object = true
			}
		}

		if len(prefix) == 0 || prefix[0] != '.' {
			(*properties)[prefix] = object
		} else {
			(*properties)[prefix[1:]] = object
		}
	}

	if errors.Count() > 0 {
		return errors
	}

	return nil
}
