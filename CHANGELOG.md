# Changelog

## v1.2.0 - 2026-04-14

### Changed

- Refactored main.go for testability: extracted `loadPatterns()`,
  `compilePatterns()`, `run()`, and `execute()` as pure functions
- Architecture documentation added (`docs/en/`, `docs/ja/`)
- Test coverage: 44% → 79% (56 tests, up from 20)

## v1.1.0 - 2026-04-14

### Added

- `--field` option: JSON input mode — apply regex to a specific field value within
  JSON input and merge extracted fields back into the original object
  (like Splunk's `rex field=TARGET`)
- Dot-notation support for nested JSON fields (e.g. `--field event.raw`)
- Lines where the target field is missing or not a string are passed through unchanged
- Non-JSON input in `--field` mode causes an error with line number

### Changed

- Refactored into separate files (`process.go`, `field.go`) for testability
- Added comprehensive test suite (`field_test.go`, `process_test.go`)
- Version output changed from `rex-go version` to `rex version`

## v1.0.2 - 2026-03-28

### Changed
- Unified Makefile: replaced macOS universal binary with separate `darwin/amd64` and `darwin/arm64` targets; standardized targets (`build`, `build-all`, `test`, `lint`, `check`, `package`, `clean`, `help`) and output layout (`dist/` flat directory, `.zip` archives).

## v1.0.1 - 2026-03-28

### Internal

- Added `go.mod` to establish Go module (`github.com/nlink-jp/rex`) following repository transfer to nlink-jp organization.
- Renamed binary from `rex-go` to `rex`.

## v1.0.0 - 2025-08-21

### Added

- **Dynamic Versioning**: The `Makefile` has been updated to automatically derive the binary version from git tags using `git describe --tags --always --dirty`. This ensures that released binaries reflect the exact version from the repository.
- **Packaging Target**: A new `make package` target has been added to the `Makefile`. This target automates the creation of distributable archives (.zip for Windows, .tar.gz for Linux/macOS) for all cross-compiled binaries, streamlining the release process.
- **License File**: An MIT `LICENSE` file has been added to the project root, clearly defining the terms of use and distribution.
- **Documentation Updates**: `README.md` and `README.ja.md` have been updated to reflect these new build features, providing clear instructions on how to use the dynamic versioning and the new packaging target.