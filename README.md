---
locale: en
tags:
  - app:caddy-dns-immosquare
  - audience:technique
---

# immosquare DNS module for Caddy

This package contains a DNS provider module for [Caddy](https://github.com/caddyserver/caddy). It enables Caddy to solve ACME DNS-01 challenges via the immosquare DNS API (used for automatic HTTPS certificate provisioning). This page covers building the module into a Caddy binary, then configuring the provider in JSON or in a Caddyfile.

## Building Caddy with the immosquare DNS module

This module cannot be used standalone. It must be compiled into Caddy with [xcaddy](https://github.com/caddyserver/xcaddy):

```bash
xcaddy build --with github.com/immosquare/caddy-dns-immosquare
```

A helper script is provided at the repository root:

```bash
./build.sh
```

It installs `xcaddy` if missing and runs the build against the latest tag.

## Configuring the immosquare DNS provider

The immosquare DNS provider takes two options, one required and one optional:

| Option      | Type   | Required | Description                                                       |
| ----------- | ------ | -------- | ----------------------------------------------------------------- |
| `api_token` | String | yes      | API token for the immosquare DNS API                              |
| `endpoint`  | String | no       | Custom API endpoint (defaults to the standard immosquare endpoint) |

In JSON, the provider is declared in the DNS challenge of the [ACME issuer](https://caddyserver.com/docs/json/apps/tls/automation/policies/issuer/acme/):

```json
{
  "module": "acme",
  "challenges": {
    "dns": {
      "provider": {
        "name": "immosquare",
        "api_token": "YOUR_API_TOKEN",
        "endpoint": "https://custom.example.com"
      }
    }
  }
}
```

In a Caddyfile, `acme_dns immosquare` sets the provider globally:

```
{
  acme_dns immosquare <api_token>
}
```

Per site, `dns immosquare` goes inside the `tls` block of that site:

```
example.com {
  tls {
    dns immosquare <api_token>
  }
}
```

The block syntax of `dns immosquare` adds a custom endpoint:

```
example.com {
  tls {
    dns immosquare {
      api_token <api_token>
      endpoint  <endpoint>
    }
  }
}
```

## Contributing to caddy-dns-immosquare and its license

Bug reports and pull requests are welcome on GitHub. This module is available as open source under the terms of the [MIT License](LICENSE).
