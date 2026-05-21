immosquare DNS module for Caddy
===========================

This package contains a DNS provider module for [Caddy](https://github.com/caddyserver/caddy). It enables Caddy to solve ACME DNS-01 challenges via the immosquare DNS API (used for automatic HTTPS certificate provisioning).

## Installation

This module cannot be used standalone. It must be compiled into Caddy with [xcaddy](https://github.com/caddyserver/xcaddy):

```bash
xcaddy build --with github.com/immosquare/caddy-dns-immosquare
```

A helper script is provided at the repository root:

```bash
./build.sh
```

It installs `xcaddy` if missing and runs the build against the latest tag.

## Configuration

| Option      | Type   | Required | Description                                                       |
| ----------- | ------ | -------- | ----------------------------------------------------------------- |
| `api_token` | String | yes      | API token for the immosquare DNS API                              |
| `endpoint`  | String | no       | Custom API endpoint (defaults to the standard immosquare endpoint) |

## Config examples

JSON — [configure the ACME issuer](https://caddyserver.com/docs/json/apps/tls/automation/policies/issuer/acme/):

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

Caddyfile — globally:

```
{
  acme_dns immosquare <api_token>
}
```

Caddyfile — per site:

```
example.com {
  tls {
    dns immosquare <api_token>
  }
}
```

Caddyfile — with a custom endpoint via block syntax:

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

## Contributing

Bug reports and pull requests are welcome on GitHub.

## License

This module is available as open source under the terms of the [MIT License](LICENSE).
