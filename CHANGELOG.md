# Changelog

## [1.1.0] - 2026-10-02

### Changed

- Merge `libdns-immosquare` into this module: a single `Provider` type implements both the libdns interfaces and the Caddy module, in the package `immosquare`
- Depend on `github.com/libdns/libdns` v1.1.0 directly
- Build typed records with `libdns.RR.Parse` instead of hand-written parsing
- Require the `endpoint` option in the Caddyfile, so a missing value fails at load time

### Fixed

- Return every non-2xx API answer as an error (`DeleteRecords` used to swallow failures and leave stale `_acme-challenge` records behind)
- Read the fields the API serves in `GetRecords` (`data`, `preference`, `target`) and return names relative to the zone

## [1.0.9] - 2026-02-10

### Changed

- Update `libdns-immosquare` from v1.0.3 to v1.0.4
- Update `caddy` from v2.10.0 to v2.10.2 and indirect dependencies

## [1.0.8] - 2025-07-10

### Changed

- Convert indentation from tabs to spaces for consistency

## [1.0.7] - 2025-07-10

### Added

- Initial release of caddy-dns-immosquare module
- DNS provider for ACME DNS challenges via immosquare API
- Support for `api_token` and `endpoint` configuration
