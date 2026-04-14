// main.go
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
)

// stringSlice is a custom type for handling multiple -r flags.
type stringSlice []string

func (s *stringSlice) String() string {
	return fmt.Sprintf("%v", *s)
}

func (s *stringSlice) Set(value string) error {
	*s = append(*s, value)
	return nil
}

// DefinitionFile is the struct for the JSON definition file.
type DefinitionFile struct {
	Patterns []string `json:"patterns"`
}

// version is set by the build process using ldflags
var version = "dev"

// loadPatterns merges flag-provided patterns with patterns from a config file.
// If configReader is nil, only flag patterns are returned.
func loadPatterns(flagPatterns []string, configReader io.Reader) ([]string, error) {
	patterns := make([]string, len(flagPatterns))
	copy(patterns, flagPatterns)

	if configReader != nil {
		var defs DefinitionFile
		if err := json.NewDecoder(configReader).Decode(&defs); err != nil {
			return nil, fmt.Errorf("could not parse config file: %w", err)
		}
		patterns = append(patterns, defs.Patterns...)
	}

	return patterns, nil
}

// compilePatterns compiles regex patterns and validates that each has at least
// one named capture group.
func compilePatterns(patterns []string) ([]*regexp.Regexp, error) {
	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid regular expression '%s': %w", pattern, err)
		}
		if len(re.SubexpNames()) <= 1 {
			return nil, fmt.Errorf("regex '%s' must contain at least one named capture group", pattern)
		}
		compiled = append(compiled, re)
	}
	return compiled, nil
}

// run executes the main processing logic, dispatching to text or JSON mode.
func run(writer io.Writer, reader io.Reader, regexes []*regexp.Regexp, unique bool, fieldPath string) error {
	if fieldPath != "" {
		return processJSON(writer, reader, regexes, unique, fieldPath)
	}
	return processLines(writer, reader, regexes, unique)
}

// options holds parsed CLI options for execute().
type options struct {
	patterns    []string
	configFile  string
	inputFile   string
	outputFile  string
	fieldPath   string
	unique      bool
}

// execute runs the full CLI pipeline with the given options and I/O defaults.
// It is separated from main() for testability.
func execute(opts options, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	// Load patterns from config file
	var configReader io.Reader
	if opts.configFile != "" {
		file, err := os.Open(opts.configFile)
		if err != nil {
			return fmt.Errorf("could not open config file %s: %w", opts.configFile, err)
		}
		defer file.Close()
		configReader = file
	}

	patterns, err := loadPatterns(opts.patterns, configReader)
	if err != nil {
		return err
	}

	if len(patterns) == 0 {
		return fmt.Errorf("at least one regex pattern must be provided via -r or -f flag")
	}

	compiledRegexes, err := compilePatterns(patterns)
	if err != nil {
		return err
	}

	// Set up I/O
	var reader io.Reader
	if opts.inputFile != "" {
		file, err := os.Open(opts.inputFile)
		if err != nil {
			return fmt.Errorf("could not open input file %s: %w", opts.inputFile, err)
		}
		defer file.Close()
		reader = file
	} else {
		reader = stdin
	}

	var writer io.Writer
	if opts.outputFile != "" {
		file, err := os.Create(opts.outputFile)
		if err != nil {
			return fmt.Errorf("could not create output file %s: %w", opts.outputFile, err)
		}
		defer file.Close()
		writer = file
	} else {
		writer = stdout
	}

	return run(writer, reader, compiledRegexes, opts.unique, opts.fieldPath)
}

func main() {
	var regexPatterns stringSlice
	flag.Var(&regexPatterns, "r", "Regular expression with named capture groups. Can be specified multiple times.")
	configFile := flag.String("f", "", "Path to a JSON file containing an array of regex patterns.")
	inputFile := flag.String("i", "", "Input file path (default: stdin).")
	outputFile := flag.String("o", "", "Output file path (default: stdout).")
	fieldPath := flag.String("field", "", "JSON field to apply regex to (dot-notation for nested fields). Enables JSON input mode.")
	uniqueValues := flag.Bool("u", false, "Ensure that values for a multi-valued field are unique.")
	showVersion := flag.Bool("version", false, "Show version information and exit.")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "A command-line tool to extract and merge fields from text using all specified regex patterns.\n\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if *showVersion {
		fmt.Printf("rex version %s\n", version)
		os.Exit(0)
	}

	opts := options{
		patterns:   regexPatterns,
		configFile: *configFile,
		inputFile:  *inputFile,
		outputFile: *outputFile,
		fieldPath:  *fieldPath,
		unique:     *uniqueValues,
	}

	if err := execute(opts, os.Stdin, os.Stdout, os.Stderr); err != nil {
		log.Fatalf("Error: %v", err)
	}
}
