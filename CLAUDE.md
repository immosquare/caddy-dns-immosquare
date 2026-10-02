# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Go module that is both the Caddy DNS provider module `dns.providers.immosquare` and a `libdns` provider, backed by the immosquare DNS API. `README.md` covers configuration, behavior and the HTTP contract.

- **provider.go**: the `Provider` type and the four libdns operations, each mapped to one API call (`GET`, `POST`, `PUT`, `DELETE` on `/zones/{zone}/records`)
- **module.go**: registers the same `Provider` as the Caddy module (Caddyfile parsing, placeholder replacement)
- **test/test_provider.go**: manual program run against a live API — it writes real records

## Contract to Preserve

- Every non-2xx API answer is returned as an error: certmagic relies on it to know a challenge record was not written or not deleted
- `SetRecords` replaces only the RRsets (name, type) it receives
- `AppendRecords` and `SetRecords` raise TTLs below 120 seconds to 120 seconds; `DeleteRecords` does not

Any change to this contract must be mirrored on the API side.

## Commands

```bash
go vet ./... && go build ./...
xcaddy build --with github.com/immosquare/caddy-dns-immosquare=.
```

## Conventions

- Go code is indented with 2 spaces; files are not run through `gofmt`
- Releases are tags `vX.Y.Z`, with the `Version` constant in `provider.go` and `CHANGELOG.md` updated
