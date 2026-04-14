# rex — Architecture

## Purpose

Pipe-friendly CLI tool that extracts named fields from text or JSON streams
using regular expressions and outputs structured JSON. Analogous to Splunk's
`rex` command.

## Modes of Operation

### Text Mode (default)

```
stdin (plain text lines)
  → apply regexes to each line
  → output JSON for lines with ≥1 match (skip non-matching lines)
```

### JSON Mode (`--field`)

```
stdin (JSONL)
  → parse each line as JSON object
  → resolve target field (dot-notation)
  → apply regexes to field value
  → merge extracted fields into original object
  → output enriched JSON (always, even if no match)
```

## Module Structure

```
main.go      CLI flags, validation, I/O setup, mode dispatch
process.go   applyRegexes(), processLines(), processJSON()
field.go     getNestedField() — dot-notation field resolution
```

### Dependency Graph

```
main.go
  ├── process.go
  │     └── field.go    (JSON mode only)
  └── encoding/json     (config file loading)
```

No external dependencies. Standard library only.

## Data Flow

### Text Mode

```
reader ──► bufio.Scanner ──► line ──► applyRegexes() ──► map[string]interface{}
                                                              │
                                                  len > 0 ?   │
                                                  ┌───yes──────┘
                                                  ▼
                                          json.Marshal() ──► writer
```

### JSON Mode

```
reader ──► bufio.Scanner ──► line ──► json.Unmarshal() ──► map[string]interface{}
                                                              │
                                                  getNestedField(obj, fieldPath)
                                                              │
                                              ┌── not found / not string ──► passthrough
                                              ▼
                                     applyRegexes(fieldValue)
                                              │
                                     merge into top-level obj
                                              │
                                     json.Marshal(obj) ──► writer
```

## Key Behaviors

### Multi-value Fields

When the same named group appears in multiple patterns:

1. First capture → string
2. Second capture → `[]string{first, second}`
3. Subsequent → append to slice

With `-u`: duplicates suppressed before append.

### Field Overwrite (JSON mode)

Extracted fields overwrite existing keys in the original JSON object.
No merge or array concatenation — pure overwrite.

### Error Semantics

| Condition | Behavior |
|-----------|----------|
| Invalid regex | Fatal exit |
| No named groups | Fatal exit |
| Non-JSON in `--field` mode | Fatal exit with line number |
| No match (text mode) | Silent skip |
| Missing/non-string field (JSON) | Passthrough unchanged |
| JSON marshal failure | Warning log, skip line |

## Configuration

### Pattern Sources

Patterns can be provided via:
- `-r` flag (repeatable): inline regex
- `-f` flag: JSON file with `{"patterns": [...]}`
- Both combined: patterns are appended

### I/O

| Flag | Default | Description |
|------|---------|-------------|
| `-i` | stdin | Input source |
| `-o` | stdout | Output destination |

## Testing Strategy

### Unit-testable Functions

| Function | File | Inputs | Dependencies |
|----------|------|--------|-------------|
| `applyRegexes` | process.go | string, regexes, unique flag | None |
| `processLines` | process.go | io.Writer, io.Reader, regexes, unique | None |
| `processJSON` | process.go | io.Writer, io.Reader, regexes, unique, fieldPath | `getNestedField` |
| `getNestedField` | field.go | map, path string | None |

All core logic accepts `io.Reader`/`io.Writer` — fully testable without file I/O.

### main.go (integration only)

CLI flag parsing, file open/close, mode dispatch. Tested via built binary
or by extracting a `run()` function.

### Coverage Targets

| Module | Current | Target |
|--------|---------|--------|
| field.go | 100% | 100% |
| process.go | 70-83% | 90%+ |
| main.go | 0% | 60%+ (via `run()` extraction) |
| **Overall** | 44% | **80%+** |
