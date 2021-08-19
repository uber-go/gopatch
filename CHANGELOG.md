# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
