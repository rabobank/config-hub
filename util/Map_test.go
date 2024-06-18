package util

import (
	"testing"
)

func TestGet(t *testing.T) {
	m := map[string]any{
		"a": 1,
		"b": map[string]any{
			"c": 2,
		},
		"b.c": 1,
		"c": map[string]any{
			"a":   2,
			"b.c": 3,
		},
		"c.a": map[string]any{
			"b": 3,
			"c.a": map[string]any{
				"a": 2,
				"b": 3,
			},
			"c": map[string]any{
				"a": map[string]any{
					"b": 1,
				},
				"b.c": 3,
			},
		},
	}
	v, found := Get("a", m)
	if !found {
		t.Errorf("Expected to find key 'a'")
	}
	if v != 1 {
		t.Errorf("Expected value 1, got %v", v)
	}
	v, found = Get("b.c", m)
	if !found {
		t.Errorf("Expected to find key 'b.c'")
	}
	if v != 1 {
		t.Errorf("Expected value 1, got %v", v)
	}
	v, found = Get("b.d", m)
	if found {
		t.Errorf("Expected not to find key 'b.d'")
	}
	v, found = Get("c.a", m)
	if !found {
		t.Errorf("Expected to find key 'c.a'")
	}
	// if !Identical(m["c.a"], v) {
	//     t.Errorf("Expected value %v, got %v", m["c.a"], v)
	// }
	v, found = Get("c.b.c", m)
	if !found {
		t.Errorf("Expected to find key 'b.c'")
	}
	if v != 3 {
		t.Errorf("Expected value 3, got %v", v)
	}
	v, found = Get("c.a.c.a.b", m)
	if !found {
		t.Errorf("Expected to find key 'c.a.c.a.b'")
	}
	if v != 1 {
		t.Errorf("Expected value 1, got %v", v)
	}
	v, found = Get("c.a.c.a.a", m)
	if !found {
		t.Errorf("Expected to find key 'c.a.c.a.a'")
	}
	if v != 2 {
		t.Errorf("Expected value 2, got %v", v)
	}
}
