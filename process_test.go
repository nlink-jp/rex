package main

import (
	"bytes"
	"encoding/json"
	"regexp"
	"strings"
	"testing"
)

func compileRegexes(t *testing.T, patterns ...string) []*regexp.Regexp {
	t.Helper()
	var res []*regexp.Regexp
	for _, p := range patterns {
		re, err := regexp.Compile(p)
		if err != nil {
			t.Fatalf("failed to compile regex %q: %v", p, err)
		}
		res = append(res, re)
	}
	return res
}

// --- applyRegexes tests ---

func TestApplyRegexes_SingleMatch(t *testing.T) {
	regexes := compileRegexes(t, `(?P<ip>\d+\.\d+\.\d+\.\d+)`)
	result := applyRegexes("192.168.1.1 GET /index.html", regexes, false)

	if result["ip"] != "192.168.1.1" {
		t.Errorf("expected ip=192.168.1.1, got %v", result["ip"])
	}
}

func TestApplyRegexes_NoMatch(t *testing.T) {
	regexes := compileRegexes(t, `(?P<ip>\d+\.\d+\.\d+\.\d+)`)
	result := applyRegexes("no ip here", regexes, false)

	if len(result) != 0 {
		t.Errorf("expected empty result, got %v", result)
	}
}

func TestApplyRegexes_MultiplePatterns(t *testing.T) {
	regexes := compileRegexes(t,
		`(?P<user>\w+)@`,
		`action=(?P<action>\w+)`,
	)
	result := applyRegexes("admin@host action=login", regexes, false)

	if result["user"] != "admin" {
		t.Errorf("expected user=admin, got %v", result["user"])
	}
	if result["action"] != "login" {
		t.Errorf("expected action=login, got %v", result["action"])
	}
}

func TestApplyRegexes_UniqueValues(t *testing.T) {
	regexes := compileRegexes(t,
		`src=(?P<addr>\S+)`,
		`dst=(?P<addr>\S+)`,
	)
	// Same value for both — unique should suppress duplicate
	result := applyRegexes("src=10.0.0.1 dst=10.0.0.1", regexes, true)

	if result["addr"] != "10.0.0.1" {
		t.Errorf("expected single addr=10.0.0.1, got %v", result["addr"])
	}
}

func TestApplyRegexes_DuplicateBecomesArray(t *testing.T) {
	regexes := compileRegexes(t,
		`src=(?P<addr>\S+)`,
		`dst=(?P<addr>\S+)`,
	)
	result := applyRegexes("src=10.0.0.1 dst=10.0.0.2", regexes, false)

	slice, ok := result["addr"].([]string)
	if !ok {
		t.Fatalf("expected []string, got %T: %v", result["addr"], result["addr"])
	}
	if len(slice) != 2 || slice[0] != "10.0.0.1" || slice[1] != "10.0.0.2" {
		t.Errorf("expected [10.0.0.1 10.0.0.2], got %v", slice)
	}
}

func TestApplyRegexes_TripleMatch(t *testing.T) {
	regexes := compileRegexes(t,
		`a=(?P<val>\S+)`,
		`b=(?P<val>\S+)`,
		`c=(?P<val>\S+)`,
	)
	result := applyRegexes("a=1 b=2 c=3", regexes, false)

	slice, ok := result["val"].([]string)
	if !ok {
		t.Fatalf("expected []string, got %T: %v", result["val"], result["val"])
	}
	if len(slice) != 3 || slice[0] != "1" || slice[1] != "2" || slice[2] != "3" {
		t.Errorf("expected [1 2 3], got %v", slice)
	}
}

func TestApplyRegexes_EmptyCapture(t *testing.T) {
	regexes := compileRegexes(t, `user=(?P<user>\w*)`)
	result := applyRegexes("user=", regexes, false)

	if result["user"] != "" {
		t.Errorf("expected empty string, got %q", result["user"])
	}
}

func TestApplyRegexes_UniqueWithArray(t *testing.T) {
	regexes := compileRegexes(t,
		`a=(?P<v>\S+)`,
		`b=(?P<v>\S+)`,
		`c=(?P<v>\S+)`,
	)
	// a and c have same value "x", b is "y" — unique should deduplicate
	result := applyRegexes("a=x b=y c=x", regexes, true)

	slice, ok := result["v"].([]string)
	if !ok {
		t.Fatalf("expected []string, got %T: %v", result["v"], result["v"])
	}
	if len(slice) != 2 || slice[0] != "x" || slice[1] != "y" {
		t.Errorf("expected [x y], got %v", slice)
	}
}

// --- processLines tests (text mode) ---

func TestProcessLines_BasicExtraction(t *testing.T) {
	regexes := compileRegexes(t, `(?P<ip>\d+\.\d+\.\d+\.\d+)`)
	input := "192.168.1.1 GET /index.html\n10.0.0.1 POST /api\n"
	var buf bytes.Buffer

	if err := processLines(&buf, strings.NewReader(input), regexes, false); err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}

	var obj map[string]interface{}
	json.Unmarshal([]byte(lines[0]), &obj)
	if obj["ip"] != "192.168.1.1" {
		t.Errorf("line 1: expected ip=192.168.1.1, got %v", obj["ip"])
	}
}

func TestProcessLines_EmptyInput(t *testing.T) {
	regexes := compileRegexes(t, `(?P<ip>\d+\.\d+\.\d+\.\d+)`)
	var buf bytes.Buffer

	if err := processLines(&buf, strings.NewReader(""), regexes, false); err != nil {
		t.Fatal(err)
	}

	if buf.String() != "" {
		t.Errorf("expected empty output for empty input, got %q", buf.String())
	}
}

func TestProcessLines_MixedMatchNoMatch(t *testing.T) {
	regexes := compileRegexes(t, `(?P<ip>\d+\.\d+\.\d+\.\d+)`)
	input := "no match\n192.168.1.1 hit\nstill no match\n10.0.0.1 hit\n"
	var buf bytes.Buffer

	if err := processLines(&buf, strings.NewReader(input), regexes, false); err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 output lines (only matches), got %d: %v", len(lines), lines)
	}
}

func TestProcessLines_NoMatchSkipped(t *testing.T) {
	regexes := compileRegexes(t, `(?P<ip>\d+\.\d+\.\d+\.\d+)`)
	input := "no match here\n"
	var buf bytes.Buffer

	if err := processLines(&buf, strings.NewReader(input), regexes, false); err != nil {
		t.Fatal(err)
	}

	if buf.String() != "" {
		t.Errorf("expected empty output, got %q", buf.String())
	}
}

// --- processJSON tests (JSON mode) ---

func TestProcessJSON_BasicFieldExtraction(t *testing.T) {
	regexes := compileRegexes(t, `(?P<ip>\d+\.\d+\.\d+\.\d+)`)
	input := `{"message":"192.168.1.1 GET /index.html","host":"web01"}` + "\n"
	var buf bytes.Buffer

	if err := processJSON(&buf, strings.NewReader(input), regexes, false, "message"); err != nil {
		t.Fatal(err)
	}

	var obj map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &obj); err != nil {
		t.Fatal(err)
	}
	if obj["ip"] != "192.168.1.1" {
		t.Errorf("expected ip=192.168.1.1, got %v", obj["ip"])
	}
	if obj["host"] != "web01" {
		t.Errorf("expected host=web01, got %v", obj["host"])
	}
	if obj["message"] != "192.168.1.1 GET /index.html" {
		t.Errorf("expected message preserved, got %v", obj["message"])
	}
}

func TestProcessJSON_NestedField(t *testing.T) {
	regexes := compileRegexes(t, `user=(?P<user>\w+)`, `action=(?P<action>\w+)`)
	input := `{"event":{"raw":"user=admin action=login"},"id":1}` + "\n"
	var buf bytes.Buffer

	if err := processJSON(&buf, strings.NewReader(input), regexes, false, "event.raw"); err != nil {
		t.Fatal(err)
	}

	var obj map[string]interface{}
	json.Unmarshal(buf.Bytes(), &obj)
	if obj["user"] != "admin" {
		t.Errorf("expected user=admin, got %v", obj["user"])
	}
	if obj["action"] != "login" {
		t.Errorf("expected action=login, got %v", obj["action"])
	}
	// Original nested structure preserved
	event, ok := obj["event"].(map[string]interface{})
	if !ok {
		t.Fatal("expected event to be a map")
	}
	if event["raw"] != "user=admin action=login" {
		t.Errorf("expected event.raw preserved, got %v", event["raw"])
	}
}

func TestProcessJSON_MissingFieldPassthrough(t *testing.T) {
	regexes := compileRegexes(t, `(?P<ip>\d+\.\d+\.\d+\.\d+)`)
	input := `{"host":"web01"}` + "\n"
	var buf bytes.Buffer

	if err := processJSON(&buf, strings.NewReader(input), regexes, false, "message"); err != nil {
		t.Fatal(err)
	}

	// Should pass through unchanged
	got := strings.TrimSpace(buf.String())
	var obj map[string]interface{}
	json.Unmarshal([]byte(got), &obj)
	if obj["host"] != "web01" {
		t.Errorf("expected host=web01, got %v", obj["host"])
	}
	if _, exists := obj["ip"]; exists {
		t.Error("ip should not be present when field is missing")
	}
}

func TestProcessJSON_NonStringFieldPassthrough(t *testing.T) {
	regexes := compileRegexes(t, `(?P<ip>\d+\.\d+\.\d+\.\d+)`)
	input := `{"message":12345,"host":"web01"}` + "\n"
	var buf bytes.Buffer

	if err := processJSON(&buf, strings.NewReader(input), regexes, false, "message"); err != nil {
		t.Fatal(err)
	}

	got := strings.TrimSpace(buf.String())
	var obj map[string]interface{}
	json.Unmarshal([]byte(got), &obj)
	if _, exists := obj["ip"]; exists {
		t.Error("ip should not be present when field is not a string")
	}
}

func TestProcessJSON_NonJSONError(t *testing.T) {
	regexes := compileRegexes(t, `(?P<ip>\d+\.\d+\.\d+\.\d+)`)
	input := "this is not json\n"
	var buf bytes.Buffer

	err := processJSON(&buf, strings.NewReader(input), regexes, false, "message")
	if err == nil {
		t.Fatal("expected error for non-JSON input")
	}
	if !strings.Contains(err.Error(), "invalid JSON") {
		t.Errorf("expected 'invalid JSON' in error, got: %v", err)
	}
}

func TestProcessJSON_NoMatchOutputsOriginal(t *testing.T) {
	regexes := compileRegexes(t, `(?P<ip>\d+\.\d+\.\d+\.\d+)`)
	input := `{"message":"no ip here","host":"web01"}` + "\n"
	var buf bytes.Buffer

	if err := processJSON(&buf, strings.NewReader(input), regexes, false, "message"); err != nil {
		t.Fatal(err)
	}

	var obj map[string]interface{}
	json.Unmarshal(buf.Bytes(), &obj)
	if obj["host"] != "web01" {
		t.Errorf("expected host=web01, got %v", obj["host"])
	}
	if _, exists := obj["ip"]; exists {
		t.Error("ip should not be present when regex does not match")
	}
}

func TestProcessJSON_OverwriteExistingField(t *testing.T) {
	regexes := compileRegexes(t, `(?P<ip>\d+\.\d+\.\d+\.\d+)`)
	input := `{"message":"192.168.1.1 GET /","ip":"old-value"}` + "\n"
	var buf bytes.Buffer

	if err := processJSON(&buf, strings.NewReader(input), regexes, false, "message"); err != nil {
		t.Fatal(err)
	}

	var obj map[string]interface{}
	json.Unmarshal(buf.Bytes(), &obj)
	if obj["ip"] != "192.168.1.1" {
		t.Errorf("expected ip=192.168.1.1 (overwritten), got %v", obj["ip"])
	}
}

func TestProcessJSON_UniqueFlag(t *testing.T) {
	regexes := compileRegexes(t,
		`src=(?P<addr>\S+)`,
		`dst=(?P<addr>\S+)`,
	)
	input := `{"message":"src=10.0.0.1 dst=10.0.0.1"}` + "\n"
	var buf bytes.Buffer

	if err := processJSON(&buf, strings.NewReader(input), regexes, true, "message"); err != nil {
		t.Fatal(err)
	}

	var obj map[string]interface{}
	json.Unmarshal(buf.Bytes(), &obj)
	if obj["addr"] != "10.0.0.1" {
		t.Errorf("expected single addr=10.0.0.1, got %v", obj["addr"])
	}
}

func TestProcessJSON_EmptyInput(t *testing.T) {
	regexes := compileRegexes(t, `(?P<ip>\d+\.\d+\.\d+\.\d+)`)
	var buf bytes.Buffer

	if err := processJSON(&buf, strings.NewReader(""), regexes, false, "message"); err != nil {
		t.Fatal(err)
	}

	if buf.String() != "" {
		t.Errorf("expected empty output for empty input, got %q", buf.String())
	}
}

func TestProcessJSON_EmptyObject(t *testing.T) {
	regexes := compileRegexes(t, `(?P<ip>\d+\.\d+\.\d+\.\d+)`)
	input := "{}\n"
	var buf bytes.Buffer

	if err := processJSON(&buf, strings.NewReader(input), regexes, false, "message"); err != nil {
		t.Fatal(err)
	}

	// Empty object should pass through (field missing)
	got := strings.TrimSpace(buf.String())
	if got != "{}" {
		t.Errorf("expected {}, got %q", got)
	}
}

func TestProcessJSON_NullFieldValue(t *testing.T) {
	regexes := compileRegexes(t, `(?P<ip>\d+\.\d+\.\d+\.\d+)`)
	input := `{"message":null,"host":"web01"}` + "\n"
	var buf bytes.Buffer

	if err := processJSON(&buf, strings.NewReader(input), regexes, false, "message"); err != nil {
		t.Fatal(err)
	}

	var obj map[string]interface{}
	json.Unmarshal(buf.Bytes(), &obj)
	if _, exists := obj["ip"]; exists {
		t.Error("ip should not be present when field value is null")
	}
}

func TestProcessJSON_ArrayFieldValue(t *testing.T) {
	regexes := compileRegexes(t, `(?P<ip>\d+\.\d+\.\d+\.\d+)`)
	input := `{"message":["a","b"],"host":"web01"}` + "\n"
	var buf bytes.Buffer

	if err := processJSON(&buf, strings.NewReader(input), regexes, false, "message"); err != nil {
		t.Fatal(err)
	}

	var obj map[string]interface{}
	json.Unmarshal(buf.Bytes(), &obj)
	if _, exists := obj["ip"]; exists {
		t.Error("ip should not be present when field value is an array")
	}
}

func TestProcessJSON_DeepNesting(t *testing.T) {
	regexes := compileRegexes(t, `user=(?P<user>\w+)`)
	input := `{"a":{"b":{"c":"user=admin"}}}` + "\n"
	var buf bytes.Buffer

	if err := processJSON(&buf, strings.NewReader(input), regexes, false, "a.b.c"); err != nil {
		t.Fatal(err)
	}

	var obj map[string]interface{}
	json.Unmarshal(buf.Bytes(), &obj)
	if obj["user"] != "admin" {
		t.Errorf("expected user=admin from 3-level nesting, got %v", obj["user"])
	}
}

func TestProcessJSON_MultipleLines(t *testing.T) {
	regexes := compileRegexes(t, `(?P<ip>\d+\.\d+\.\d+\.\d+)`)
	input := `{"message":"192.168.1.1 req"}` + "\n" + `{"message":"10.0.0.1 req"}` + "\n"
	var buf bytes.Buffer

	if err := processJSON(&buf, strings.NewReader(input), regexes, false, "message"); err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 output lines, got %d", len(lines))
	}

	var obj1, obj2 map[string]interface{}
	json.Unmarshal([]byte(lines[0]), &obj1)
	json.Unmarshal([]byte(lines[1]), &obj2)

	if obj1["ip"] != "192.168.1.1" {
		t.Errorf("line 1: expected ip=192.168.1.1, got %v", obj1["ip"])
	}
	if obj2["ip"] != "10.0.0.1" {
		t.Errorf("line 2: expected ip=10.0.0.1, got %v", obj2["ip"])
	}
}
