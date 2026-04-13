package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"regexp"
)

// applyRegexes applies all compiled regexes to the target string and returns
// extracted named capture groups merged into a single map.
// If unique is true, duplicate values for the same field are suppressed.
func applyRegexes(target string, regexes []*regexp.Regexp, unique bool) map[string]interface{} {
	result := make(map[string]interface{})

	for _, re := range regexes {
		matches := re.FindStringSubmatch(target)
		if matches == nil {
			continue
		}

		fieldNames := re.SubexpNames()
		for i, name := range fieldNames {
			if i == 0 || name == "" {
				continue
			}
			newValue := matches[i]

			existingValue, ok := result[name]
			if !ok {
				result[name] = newValue
				continue
			}

			if slice, isSlice := existingValue.([]string); isSlice {
				shouldAppend := true
				if unique {
					for _, v := range slice {
						if v == newValue {
							shouldAppend = false
							break
						}
					}
				}
				if shouldAppend {
					result[name] = append(slice, newValue)
				}
			} else {
				existingString := existingValue.(string)
				if !unique || existingString != newValue {
					result[name] = []string{existingString, newValue}
				}
			}
		}
	}

	return result
}

// processLines reads plain text lines, applies regexes, and outputs JSON for
// each line that has at least one match. (text mode — legacy behavior)
func processLines(writer io.Writer, reader io.Reader, regexes []*regexp.Regexp, unique bool) error {
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := scanner.Text()
		result := applyRegexes(line, regexes, unique)

		if len(result) > 0 {
			jsonData, err := json.Marshal(result)
			if err != nil {
				log.Printf("Warning: Could not marshal data to JSON for line: %s. Error: %v", line, err)
				continue
			}
			if _, err := fmt.Fprintln(writer, string(jsonData)); err != nil {
				return fmt.Errorf("failed to write to output: %w", err)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading from input: %w", err)
	}

	return nil
}

// processJSON reads JSON lines, applies regexes to the value at fieldPath
// (dot-notation), merges extracted fields into the top-level object, and
// outputs the enriched JSON. Non-JSON lines cause an error. Lines where
// the target field is missing or not a string are passed through unchanged.
func processJSON(writer io.Writer, reader io.Reader, regexes []*regexp.Regexp, unique bool, fieldPath string) error {
	scanner := bufio.NewScanner(reader)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			return fmt.Errorf("line %d: invalid JSON: %w", lineNum, err)
		}

		target, ok := getNestedField(obj, fieldPath)
		if !ok {
			// Field not found — pass through unchanged
			if _, err := fmt.Fprintln(writer, line); err != nil {
				return fmt.Errorf("failed to write to output: %w", err)
			}
			continue
		}

		targetStr, isStr := target.(string)
		if !isStr {
			// Field is not a string — pass through unchanged
			if _, err := fmt.Fprintln(writer, line); err != nil {
				return fmt.Errorf("failed to write to output: %w", err)
			}
			continue
		}

		extracted := applyRegexes(targetStr, regexes, unique)

		// Merge extracted fields into top-level object (overwrite)
		for k, v := range extracted {
			obj[k] = v
		}

		jsonData, err := json.Marshal(obj)
		if err != nil {
			return fmt.Errorf("line %d: failed to marshal JSON: %w", lineNum, err)
		}
		if _, err := fmt.Fprintln(writer, string(jsonData)); err != nil {
			return fmt.Errorf("failed to write to output: %w", err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading from input: %w", err)
	}

	return nil
}
