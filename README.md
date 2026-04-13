# rex

`rex` is a command-line tool that extracts fields from text data using regular expressions and outputs them in JSON format. Like Splunk's `rex` command, its purpose is to easily create structured data from unstructured text such as log files.

With the `--field` option, `rex` can also operate on JSON input — applying regex to a specific field value and merging the extracted fields back into the original JSON object (like Splunk's `rex field=TARGET`).

This tool is written in Go and runs as a single, cross-platform executable.

---

## Features

- **Flexible I/O**: Supports input from standard input (pipes) or files, and writes results to standard output or files.
- **Multiple Regex Patterns**: Specify multiple regular expression patterns from the command line (`-r`) or a configuration file (`-f`).
- **Merged Results**: All specified regular expressions are applied to each line of input, and the matched results are merged into a single JSON object.
- **JSON Field Mode**: With `--field`, apply regex to a specific field within JSON input and merge results back into the original object. Supports dot-notation for nested fields (e.g. `event.raw`).
- **Multi-value Arrays**: If the same field name is captured by multiple patterns, its values are automatically collected into an array.
- **Unique Values**: The `-u` option ensures that values in a multi-valued array are unique.
- **Portable**: Generates a single executable file that runs on machines without a Go runtime environment.

---

## Installation

Pre-compiled binaries for macOS, Windows, and Linux are available on the [Releases](https://github.com/nlink-jp/rex/releases) page.

## Building

To build from source, a Go environment must be set up.

The version of the built binary is automatically determined from git tags. For example, if your latest tag is `v1.0.0`, the binary will be versioned as `v1.0.0`. If there are uncommitted changes, a `-dirty` suffix will be added (e.g., `v1.0.0-dirty`).

Simply run `make` to build the application. The following targets are available:

- **`make build`**: Builds a single executable for your current operating system and architecture in the `dist/` directory.
- **`make build-all`**: Cross-compiles for all target platforms (Linux amd64/arm64, macOS amd64/arm64, Windows amd64) and places the binaries flat in `dist/`.
- **`make package`**: Builds all binaries and creates `.zip` archives for each platform in `dist/`. These archives are ready for distribution.
- **`make clean`**: Removes the `dist/` directory and all build artifacts.

---

## Usage

### Command-Line Options

```
Usage of rex:
A command-line tool to extract and merge fields from text using all specified regex patterns.

  -f string
    	Path to a JSON file containing an array of regex patterns.
  --field string
    	JSON field to apply regex to (dot-notation for nested fields). Enables JSON input mode.
  -i string
    	Input file path (default: stdin).
  -o string
    	Output file path (default: stdout).
  -r value
    	Regular expression with named capture groups. Can be specified multiple times.
  -u	Ensure that values for a multi-valued field are unique.
  --version
      Show version information and exit.
```

### Examples

#### 1. Basic Extraction

Extract information from an Apache access log.

```bash
echo '127.0.0.1 - frank [10/Oct/2000] "GET /api" 200' | \
./rex -r '(?P<client_ip>[^ ]+) - (?P<user>[^ ]+) \[(?P<date>[^\]]+)\] "(?P<method>\w+) (?P<uri>[^ "]+)" (?P<status>\d{3})'
```

**Output:**
```json
{"client_ip":"127.0.0.1","date":"10/Oct/2000","method":"GET","status":"200","uri":"/api","user":"frank"}
```

#### 2. Merging Results from Multiple Patterns

Extract both `level` and `status` from a single log line using different regular expressions.

```bash
echo "request failed with level=error, status=500" | \
./rex -r 'level=(?P<level>\w+)' -r 'status=(?P<status>\d+)'
```

**Output:**
```json
{"level":"error","status":"500"}
```

#### 3. Collecting Multiple Values into an Array

Extract values from both `user=` and `alias=` into a common field named `name`, which will be output as an array.

```bash
echo "user=admin, alias=root" | \
./rex -r 'user=(?P<name>\w+)' -r 'alias=(?P<name>\w+)'
```

**Output:**
```json
{"name":["admin","root"]}
```

#### 4. Making Array Values Unique (with `-u` option)

Even if duplicate values are captured, the `-u` flag ensures only unique values are stored in the array.

```bash
echo "user=admin, alias=root, user=admin" | \
./rex -u -r 'user=(?P<name>\w+)' -r 'alias=(?P<name>\w+)'
```

**Output:**
```json
{"name":["admin","root"]}
```

#### 5. JSON Field Mode (with `--field` option)

Apply regex to a specific field within JSON input and merge extracted fields back into the original object.

```bash
echo '{"message":"192.168.1.1 GET /index.html","host":"web01"}' | \
./rex --field message -r '(?P<ip>\d+\.\d+\.\d+\.\d+)'
```

**Output:**
```json
{"host":"web01","ip":"192.168.1.1","message":"192.168.1.1 GET /index.html"}
```

#### 6. Nested JSON Fields (dot-notation)

Use dot-notation to target fields nested within JSON objects.

```bash
echo '{"event":{"raw":"user=admin action=login"},"id":1}' | \
./rex --field event.raw -r 'user=(?P<user>\w+)' -r 'action=(?P<action>\w+)'
```

**Output:**
```json
{"action":"login","event":{"raw":"user=admin action=login"},"id":1,"user":"admin"}
```

#### 7. Using a Configuration File (with `-f` option)

Create a file named `patterns.json` with the following content:

```json
{
  "patterns": [
    "level=(?P<level>\\w+)",
    "status=(?P<status>\\d+)"
  ]
}
```

Run the tool, specifying the file with the `-f` option.

```bash
echo "level=info, status=200" | ./rex -f patterns.json
```

**Output:**
```json
{"level":"info","status":"200"}
```
---

## License

This project is licensed under the [MIT License](https://opensource.org/licenses/MIT).
