# Changelog

## [Unreleased]

### Fixed

- **The Linux archives no longer carry macOS file metadata.** macOS `tar` wrote
  each bundled file's extended attributes (`com.apple.provenance`, and a Dropbox
  attribute where the tree is synced) into the `.tar.gz` twice: as AppleDouble
  `._` members, which GNU tar extracts as stray `._<name>` files beside the real
  ones, and as `LIBARCHIVE.xattr.*` / `SCHILY.xattr.*` pax headers, which it
  reports as unknown keywords. `make package` now archives with
  `COPYFILE_DISABLE=1 tar --no-xattrs`; each setting stops one of the two.
  Archives already published still carry them; the files themselves are
  unaffected.

### Internal

- `make verify-release` also judges each Linux archive: no AppleDouble or other
  macOS metadata members — listed with `--options 'tar:!mac-ext'`, because a
  plain macOS listing folds `._` members away — no extended attributes as pax
  headers, and exactly the canonical binary, `README.md` and `LICENSE`, compared
  in the C locale.
- The Linux-archive check in `make verify-release` reads each archive's pax
  headers with Python's `tarfile` instead of grepping the decompressed stream,
  which also matched file text that names the keywords (a bundled CHANGELOG,
  for one).

## v1.3.0 - 2026-07-12

### Removed

- **darwin/amd64 (Intel) pre-built binary.** macOS releases now ship
  **arm64 only**, per the org-wide policy (darwin is Apple-Silicon only; no
  universal binaries). Intel Mac users can build from source.

### Changed

- **Linux release archives are now `.tar.gz`** (darwin/windows remain `.zip`),
  per `nlink-jp/.github` CONVENTIONS.md §Release Archive Standard. Archives
  still bundle `LICENSE` + `README.md` alongside the canonical binary.
- **darwin code-signature identifier** is now the canonical `rex`
  (was `rex-darwin-arm64`), set via `codesign -i` so it stays stable after
  the archived binary is renamed to its canonical name.

No change to the binary's behaviour — a packaging / build-config release.

## v1.2.1 - 2026-05-23

### Changed

- **Darwin releases are now Developer ID signed and Apple-notarized.**
  `rex-v1.2.1-darwin-{amd64,arm64}.zip` carry full Apple Developer ID
  Application signatures and notarization tickets from Apple. End
  users on macOS no longer need to bypass Gatekeeper with right-click
  → Open or `xattr -d com.apple.quarantine` on first launch; local
  users who place `rex` under Dropbox-synced (or any other
  FileProvider-managed) paths are no longer killed by macOS's
  ad-hoc + provenance distrust policy. Pipeline:
  `scripts/codesign-darwin.sh` + `scripts/notarize-darwin.sh`,
  driven by `make package`. Adopts the org-wide convention in
  `nlink-jp/.github` CONVENTIONS.md §Code Signing.

No behaviour change to the binary itself — feature-wise this is
identical to v1.2.0.

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
- **Documentation Updates**: `README.md` and `README.ja.md` have been updated to reflect these new build features, providing clear instructions on how to use the dynamic versioning and the new packaging target.## [Unreleased]

### Fixed

- **`make verify-release` now fails closed.** Its last block chained unzip, the
  packaged binary's `--version` and `spctl` with `&&` and ended the whole chain
  in `|| true`, so a zip that did not unpack or a binary that did not run exited
  0 and the upload proceeded. Each step is now judged on its own, the packaged
  binary's `--version` must contain the tag being released, and only the
  informational `spctl` line may be ignored. Matches the org template
  (CONVENTIONS.md §Code Signing → Verifying a release).

