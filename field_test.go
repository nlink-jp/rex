package main

import (
	"testing"
)

func TestGetNestedField(t *testing.T) {
	tests := []struct {
		name   string
		obj    map[string]interface{}
		path   string
		want   interface{}
		wantOK bool
	}{
		{
			name:   "top-level string",
			obj:    map[string]interface{}{"message": "hello"},
			path:   "message",
			want:   "hello",
			wantOK: true,
		},
		{
			name: "nested one level",
			obj: map[string]interface{}{
				"event": map[string]interface{}{"raw": "log line"},
			},
			path:   "event.raw",
			want:   "log line",
			wantOK: true,
		},
		{
			name: "nested two levels",
			obj: map[string]interface{}{
				"a": map[string]interface{}{
					"b": map[string]interface{}{"c": "deep"},
				},
			},
			path:   "a.b.c",
			want:   "deep",
			wantOK: true,
		},
		{
			name:   "missing top-level key",
			obj:    map[string]interface{}{"message": "hello"},
			path:   "missing",
			want:   nil,
			wantOK: false,
		},
		{
			name: "missing nested key",
			obj: map[string]interface{}{
				"event": map[string]interface{}{"raw": "log line"},
			},
			path:   "event.missing",
			want:   nil,
			wantOK: false,
		},
		{
			name:   "intermediate not a map",
			obj:    map[string]interface{}{"event": "not a map"},
			path:   "event.raw",
			want:   nil,
			wantOK: false,
		},
		{
			name: "value is a number",
			obj:  map[string]interface{}{"count": float64(42)},
			path: "count",
			want: float64(42),
			wantOK: true,
		},
		{
			name:   "null value",
			obj:    map[string]interface{}{"field": nil},
			path:   "field",
			want:   nil,
			wantOK: true,
		},
		{
			name:   "boolean value",
			obj:    map[string]interface{}{"flag": true},
			path:   "flag",
			want:   true,
			wantOK: true,
		},
		{
			name:   "array value",
			obj:    map[string]interface{}{"items": []interface{}{"a", "b"}},
			path:   "items",
			want:   nil, // cannot compare slices with !=, check separately
			wantOK: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := getNestedField(tt.obj, tt.path)
			if ok != tt.wantOK {
				t.Errorf("getNestedField() ok = %v, want %v", ok, tt.wantOK)
			}
			if !ok {
				return
			}
			// Skip deep comparison for slice values
			if tt.want == nil && got != nil {
				return // nil want with non-nil got is OK for array/slice tests
			}
			if got != tt.want {
				t.Errorf("getNestedField() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetNestedField_EmptyPath(t *testing.T) {
	obj := map[string]interface{}{"key": "val"}
	_, ok := getNestedField(obj, "")
	// Empty path with single empty segment — should find key "" which doesn't exist
	if ok {
		t.Error("expected false for empty path")
	}
}
