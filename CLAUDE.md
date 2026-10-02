# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Go module that is both the Caddy DNS provider module `dns.providers.immosquare` and a `libdns` provider. It lets Caddy solve ACME DNS-01 challenges (wildcard certificates included) through the immosquare DNS API exposed by monitoring (`api/dns`), the source of truth for the zones immosquare serves. It replaces `libdns-immosquare`, archived since v1.1.0.

## Architecture

- **provider.go**: the `Provider` type and the four libdns interfaces (`GetRecords`, `AppendRecords`, `SetRecords`, `DeleteRecords`), each mapped to one call of the API (`GET`, `POST`, `PUT`, `DELETE` on `/zones/{zone}/records`)
- **module.go**: registers the same `Provider` type as the Caddy module, with `caddy.Provisioner` (placeholder replacement) and `caddyfile.Unmarshaler`
- **test/test_provider.go**: manual program that exercises every operation against a live API — it writes real records into the zone hard-coded in the file
- Depends on `github.com/libdns/libdns` v1.1.0, the version Caddy ships with; typed records come from `libdns.RR.Parse`

## Contract to Preserve

- Every non-2xx API answer must be returned as an error: certmagic only learns from the error that a challenge record was not written or not deleted, and a swallowed failure leaves stale `_acme-challenge` records behind
- `SetRecords` replaces only the RRsets (name, type) it receives; the API applies it atomically
- Names sent to the API are relative to the zone; names read from `GET` are fully qualified and converted back with `libdns.RelativeName`
- MX travel as `"<preference> <target>"` in `data` on writes; `GET` returns `preference` and `target` fields
- `AppendRecords` and `SetRecords` raise TTLs below 120 seconds to 120 seconds; `DeleteRecords` does not

Any change to this contract must be mirrored in monitoring (`app/controllers/api/dns/records_controller.rb`).

## Build Commands

This module is not built standalone. It must be compiled into Caddy using [xcaddy](https://github.com/caddyserver/xcaddy):

```bash
xcaddy build --with github.com/immosquare/caddy-dns-immosquare
```

For local development with an unreleased version:

```bash
xcaddy build --with github.com/immosquare/caddy-dns-immosquare=.
```

`go vet ./...` and `go build ./...` check the module itself. A `build.sh` helper at the repo root installs `xcaddy` if missing and builds against `@latest` — useful for quick smoke tests against the published version.

## Conventions

- Go code is indented with 2 spaces, as configured for the repository editor; files are not run through `gofmt`
- Releases are tags `vX.Y.Z`; the frontals compile Caddy with this module unpinned (`immosquare-ansible`, `playbooks/caddy.yml`), so a new tag only reaches them at the next Caddy recompilation

## Configuration

The provider requires both options, checked when the configuration loads:

- `api_token`: bearer token of the immosquare DNS API, with the `dns` scope
- `endpoint`: base URL of the API

Caddyfile syntax:

```
immosquare [<api_token>] {
    api_token <api_token>
    endpoint  <endpoint>
}
```
