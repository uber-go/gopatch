# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## Unreleased
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
