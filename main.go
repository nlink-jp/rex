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

func main() {
	// --- Define command-line flags ---
	var regexPatterns stringSlice
	flag.Var(&regexPatterns, "r", "Regular expression with named capture groups. Can be specified multiple times.")
	configFile := flag.String("f", "", "Path to a JSON file containing an array of regex patterns.")
	inputFile := flag.String("i", "", "Input file path (default: stdin).")
	outputFile := flag.String("o", "", "Output file path (default: stdout).")
	fieldPath := flag.String("field", "", "JSON field to apply regex to (dot-notation for nested fields). Enables JSON input mode.")
	uniqueValues := flag.Bool("u", false, "Ensure that values for a multi-valued field are unique.")
	showVersion := flag.Bool("version", false, "Show version information and exit.")

	// --- Customize help message ---
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

	// --- Load regex patterns from config file ---
	if *configFile != "" {
		file, err := os.Open(*configFile)
		if err != nil {
			log.Fatalf("Error: Could not open config file %s: %v", *configFile, err)
		}
		defer file.Close()

		var defs DefinitionFile
		decoder := json.NewDecoder(file)
		if err := decoder.Decode(&defs); err != nil {
			log.Fatalf("Error: Could not parse config file %s: %v", *configFile, err)
		}
		regexPatterns = append(regexPatterns, defs.Patterns...)
	}

	// --- Check for required flags ---
	if len(regexPatterns) == 0 {
		log.Println("Error: At least one regex pattern must be provided via -r or -f flag.")
		flag.Usage()
		os.Exit(1)
	}

	// --- Compile all regex patterns ---
	var compiledRegexes []*regexp.Regexp
	for _, pattern := range regexPatterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			log.Fatalf("Error: Invalid regular expression '%s': %v", pattern, err)
		}
		if len(re.SubexpNames()) <= 1 {
			log.Fatalf("Error: Regex '%s' must contain at least one named capture group.", pattern)
		}
		compiledRegexes = append(compiledRegexes, re)
	}

	// --- Set up input source ---
	var reader io.Reader
	if *inputFile != "" {
		file, err := os.Open(*inputFile)
		if err != nil {
			log.Fatalf("Error: Could not open input file %s: %v", *inputFile, err)
		}
		defer file.Close()
		reader = file
	} else {
		reader = os.Stdin
	}

	// --- Set up output destination ---
	var writer io.Writer
	if *outputFile != "" {
		file, err := os.Create(*outputFile)
		if err != nil {
			log.Fatalf("Error: Could not create output file %s: %v", *outputFile, err)
		}
		defer file.Close()
		writer = file
	} else {
		writer = os.Stdout
	}

	// --- Run the main processing logic ---
	var err error
	if *fieldPath != "" {
		err = processJSON(writer, reader, compiledRegexes, *uniqueValues, *fieldPath)
	} else {
		err = processLines(writer, reader, compiledRegexes, *uniqueValues)
	}
	if err != nil {
		log.Fatalf("Error during processing: %v", err)
	}
}
