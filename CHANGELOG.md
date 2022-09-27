# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## Unreleased
### Added
- ([#64]) `--diff` flag reports a diff of the changes that would be made, but does not
  make them.
- ([#70]) `--print-only` prints the modified contents of all matched files to stdout in
  their entirety without modifying files on disk.
- ([#77]) `-P`/`--patches-file` reads a list of patch files from another file on-disk.

  [#64]: https://github.com/uber-go/gopatch/pull/64
  [#70]: https://github.com/uber-go/gopatch/pull/70
  [#77]: https://github.com/uber-go/gopatch/pull/77

## 0.1.1 - 2022-07-26
### Fixed
- ([#54]) Preserve top-level comments in files when updating imports.
- Parse generics syntax introduced in Go 1.18.

  [#54]: https://github.com/uber-go/gopatch/issues/54

Thanks to @breml for their contribution to this release.

## 0.1.0 - 2021-08-19
Starting this release, we will include pre-built binaries of gopatch for
different systems.

### Added
- ([#7]): Add support for verbose logging with a `-v` flag.
- Add [introductory documentation] and [patch guide].

  [introductory documentation]: https://github.com/uber-go/gopatch/blob/main/README.md
  [patch guide]: https://github.com/uber-go/gopatch/blob/main/docs/PatchesInDepth.md
  [#7]: https://github.com/uber-go/gopatch/issues/7

### Changed
- Only the `--version` flag now prints the version number. The `-v` is used for
  verbose logging instead.

### Fixed
- ([#2]): Patches with named imports now support matching and manipulating any
  import.
- Fix issue where rewrites of unnamed imports would end up with duplicate
  entries.

  [#2]: https://github.com/uber-go/gopatch/issues/2

## 0.0.2 - 2020-11-04
### Fixed
- Fixed unintended deletion of unchanged named imports.

## 0.0.1 - 2020-01-14

- Initial alpha release.
