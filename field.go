package main

import "strings"

// getNestedField resolves a dot-notation field path (e.g. "event.raw")
// against a nested map structure. Returns the value and true if found,
// or nil and false if any segment is missing or not a map.
func getNestedField(obj map[string]interface{}, path string) (interface{}, bool) {
	parts := strings.Split(path, ".")
	var current interface{} = obj

	for _, part := range parts {
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil, false
		}
		current, ok = m[part]
		if !ok {
			return nil, false
		}
	}

	return current, true
}
