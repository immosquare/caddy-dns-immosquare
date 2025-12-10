# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Caddy DNS provider module for immosquare. It enables Caddy to use the immosquare DNS API for ACME DNS challenges (e.g., for automatic HTTPS certificate provisioning).

## Architecture

- **module.go**: The single source file containing the Caddy module implementation
- Wraps `github.com/immosquare/libdns-immosquare` which implements the actual DNS operations
- Registers as `dns.providers.immosquare` in Caddy's module system
- Implements `caddy.Provisioner` and `caddyfile.Unmarshaler` interfaces

## Build Commands

This module is not built standalone. It must be compiled into Caddy using [xcaddy](https://github.com/caddyserver/xcaddy):

```bash
xcaddy build --with github.com/immosquare/caddy-dns-immosquare
```

For local development with an unreleased version:

```bash
xcaddy build --with github.com/immosquare/caddy-dns-immosquare=.
```

## Configuration

The provider accepts two configuration options:
- `api_token` (required): API token for immosquare DNS API
- `endpoint` (optional): Custom API endpoint

Caddyfile syntax:
```
immosquare [<api_token>] {
    api_token <api_token>
    endpoint <endpoint>
}
```
