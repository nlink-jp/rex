# Changelog

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