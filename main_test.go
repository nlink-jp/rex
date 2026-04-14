package main

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// --- loadPatterns tests ---

func TestLoadPatterns_FlagOnly(t *testing.T) {
	patterns, err := loadPatterns([]string{`(?P<ip>\S+)`, `(?P<user>\w+)`}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(patterns) != 2 {
		t.Fatalf("expected 2 patterns, got %d", len(patterns))
	}
	if patterns[0] != `(?P<ip>\S+)` {
		t.Errorf("expected first pattern preserved, got %q", patterns[0])
	}
}

func TestLoadPatterns_ConfigOnly(t *testing.T) {
	config := `{"patterns": ["(?P<level>\\w+)", "(?P<status>\\d+)"]}`
	patterns, err := loadPatterns(nil, strings.NewReader(config))
	if err != nil {
		t.Fatal(err)
	}
	if len(patterns) != 2 {
		t.Fatalf("expected 2 patterns, got %d", len(patterns))
	}
}

func TestLoadPatterns_Combined(t *testing.T) {
	config := `{"patterns": ["(?P<level>\\w+)"]}`
	patterns, err := loadPatterns([]string{`(?P<ip>\S+)`}, strings.NewReader(config))
	if err != nil {
		t.Fatal(err)
	}
	if len(patterns) != 2 {
		t.Fatalf("expected 2 patterns, got %d", len(patterns))
	}
	if patterns[0] != `(?P<ip>\S+)` {
		t.Errorf("flag pattern should come first, got %q", patterns[0])
	}
}

func TestLoadPatterns_InvalidJSON(t *testing.T) {
	_, err := loadPatterns(nil, strings.NewReader(`{invalid`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "could not parse") {
		t.Errorf("expected parse error, got: %v", err)
	}
}

func TestLoadPatterns_EmptyConfig(t *testing.T) {
	config := `{"patterns": []}`
	patterns, err := loadPatterns(nil, strings.NewReader(config))
	if err != nil {
		t.Fatal(err)
	}
	if len(patterns) != 0 {
		t.Errorf("expected 0 patterns, got %d", len(patterns))
	}
}

func TestLoadPatterns_NilConfigNoMutation(t *testing.T) {
	original := []string{"a", "b"}
	patterns, err := loadPatterns(original, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Returned slice should be a copy
	patterns[0] = "modified"
	if original[0] != "a" {
		t.Error("loadPatterns should not mutate input slice")
	}
}

// --- compilePatterns tests ---

func TestCompilePatterns_Valid(t *testing.T) {
	regexes, err := compilePatterns([]string{`(?P<ip>\d+\.\d+\.\d+\.\d+)`, `level=(?P<level>\w+)`})
	if err != nil {
		t.Fatal(err)
	}
	if len(regexes) != 2 {
		t.Fatalf("expected 2 compiled regexes, got %d", len(regexes))
	}
}

func TestCompilePatterns_InvalidRegex(t *testing.T) {
	_, err := compilePatterns([]string{`(?P<ip>[`})
	if err == nil {
		t.Fatal("expected error for invalid regex")
	}
	if !strings.Contains(err.Error(), "invalid regular expression") {
		t.Errorf("expected regex error, got: %v", err)
	}
}

func TestCompilePatterns_NoNamedGroup(t *testing.T) {
	_, err := compilePatterns([]string{`\d+\.\d+\.\d+\.\d+`})
	if err == nil {
		t.Fatal("expected error for regex without named group")
	}
	if !strings.Contains(err.Error(), "named capture group") {
		t.Errorf("expected named group error, got: %v", err)
	}
}

func TestCompilePatterns_Empty(t *testing.T) {
	regexes, err := compilePatterns(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(regexes) != 0 {
		t.Errorf("expected 0 regexes, got %d", len(regexes))
	}
}

// --- execute tests ---

func TestExecute_TextMode(t *testing.T) {
	input := strings.NewReader("192.168.1.1 GET /\n10.0.0.1 POST /api\n")
	var stdout bytes.Buffer

	opts := options{
		patterns: []string{`(?P<ip>\d+\.\d+\.\d+\.\d+)`},
	}
	if err := execute(opts, input, &stdout, nil); err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
}

func TestExecute_JSONMode(t *testing.T) {
	input := strings.NewReader(`{"msg":"192.168.1.1 test"}` + "\n")
	var stdout bytes.Buffer

	opts := options{
		patterns:  []string{`(?P<ip>\d+\.\d+\.\d+\.\d+)`},
		fieldPath: "msg",
	}
	if err := execute(opts, input, &stdout, nil); err != nil {
		t.Fatal(err)
	}

	var obj map[string]interface{}
	json.Unmarshal(stdout.Bytes(), &obj)
	if obj["ip"] != "192.168.1.1" {
		t.Errorf("expected ip=192.168.1.1, got %v", obj["ip"])
	}
}

func TestExecute_NoPatterns(t *testing.T) {
	err := execute(options{}, strings.NewReader(""), nil, nil)
	if err == nil {
		t.Fatal("expected error for no patterns")
	}
	if !strings.Contains(err.Error(), "at least one regex pattern") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestExecute_InvalidRegex(t *testing.T) {
	opts := options{patterns: []string{`(?P<ip>[`}}
	err := execute(opts, strings.NewReader(""), nil, nil)
	if err == nil {
		t.Fatal("expected error for invalid regex")
	}
}

func TestExecute_NoNamedGroup(t *testing.T) {
	opts := options{patterns: []string{`\d+`}}
	err := execute(opts, strings.NewReader(""), nil, nil)
	if err == nil {
		t.Fatal("expected error for regex without named group")
	}
}

func TestExecute_ConfigFileNotFound(t *testing.T) {
	opts := options{configFile: "/nonexistent/config.json"}
	err := execute(opts, strings.NewReader(""), nil, nil)
	if err == nil {
		t.Fatal("expected error for missing config file")
	}
}

func TestExecute_InputFileNotFound(t *testing.T) {
	opts := options{
		patterns:  []string{`(?P<ip>\S+)`},
		inputFile: "/nonexistent/input.txt",
	}
	err := execute(opts, strings.NewReader(""), nil, nil)
	if err == nil {
		t.Fatal("expected error for missing input file")
	}
}

func TestExecute_WithConfigFile(t *testing.T) {
	// Create temp config file
	configContent := `{"patterns": ["(?P<level>\\w+)"]}`
	tmpFile := t.TempDir() + "/patterns.json"
	if err := os.WriteFile(tmpFile, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	input := strings.NewReader("info something\n")
	var stdout bytes.Buffer

	opts := options{configFile: tmpFile}
	if err := execute(opts, input, &stdout, nil); err != nil {
		t.Fatal(err)
	}

	var obj map[string]interface{}
	json.Unmarshal(stdout.Bytes(), &obj)
	if obj["level"] != "info" {
		t.Errorf("expected level=info, got %v", obj["level"])
	}
}

func TestExecute_WithInputFile(t *testing.T) {
	tmpFile := t.TempDir() + "/input.txt"
	if err := os.WriteFile(tmpFile, []byte("192.168.1.1 test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	opts := options{
		patterns:  []string{`(?P<ip>\d+\.\d+\.\d+\.\d+)`},
		inputFile: tmpFile,
	}
	if err := execute(opts, nil, &stdout, nil); err != nil {
		t.Fatal(err)
	}

	var obj map[string]interface{}
	json.Unmarshal(stdout.Bytes(), &obj)
	if obj["ip"] != "192.168.1.1" {
		t.Errorf("expected ip=192.168.1.1, got %v", obj["ip"])
	}
}

func TestExecute_WithOutputFile(t *testing.T) {
	tmpFile := t.TempDir() + "/output.jsonl"
	input := strings.NewReader("192.168.1.1 test\n")

	opts := options{
		patterns:   []string{`(?P<ip>\d+\.\d+\.\d+\.\d+)`},
		outputFile: tmpFile,
	}
	if err := execute(opts, input, nil, nil); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatal(err)
	}
	var obj map[string]interface{}
	json.Unmarshal(data, &obj)
	if obj["ip"] != "192.168.1.1" {
		t.Errorf("expected ip=192.168.1.1, got %v", obj["ip"])
	}
}

func TestExecute_UniqueFlag(t *testing.T) {
	input := strings.NewReader("src=10.0.0.1 dst=10.0.0.1\n")
	var stdout bytes.Buffer

	opts := options{
		patterns: []string{`src=(?P<addr>\S+)`, `dst=(?P<addr>\S+)`},
		unique:   true,
	}
	if err := execute(opts, input, &stdout, nil); err != nil {
		t.Fatal(err)
	}

	var obj map[string]interface{}
	json.Unmarshal(stdout.Bytes(), &obj)
	if obj["addr"] != "10.0.0.1" {
		t.Errorf("expected single addr=10.0.0.1, got %v", obj["addr"])
	}
}

// --- run tests ---

func TestRun_TextMode(t *testing.T) {
	regexes := compileRegexes(t, `(?P<ip>\d+\.\d+\.\d+\.\d+)`)
	input := "192.168.1.1 GET /\n"
	var buf bytes.Buffer

	if err := run(&buf, strings.NewReader(input), regexes, false, ""); err != nil {
		t.Fatal(err)
	}

	var obj map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &obj); err != nil {
		t.Fatal(err)
	}
	if obj["ip"] != "192.168.1.1" {
		t.Errorf("expected ip=192.168.1.1, got %v", obj["ip"])
	}
}

func TestRun_JSONMode(t *testing.T) {
	regexes := compileRegexes(t, `(?P<ip>\d+\.\d+\.\d+\.\d+)`)
	input := `{"msg":"192.168.1.1 test"}` + "\n"
	var buf bytes.Buffer

	if err := run(&buf, strings.NewReader(input), regexes, false, "msg"); err != nil {
		t.Fatal(err)
	}

	var obj map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &obj); err != nil {
		t.Fatal(err)
	}
	if obj["ip"] != "192.168.1.1" {
		t.Errorf("expected ip=192.168.1.1, got %v", obj["ip"])
	}
	if obj["msg"] != "192.168.1.1 test" {
		t.Errorf("expected msg preserved, got %v", obj["msg"])
	}
}
