---
resume: Who can use the immosquare DNS provider, how to build Caddy with it, configure it for ACME DNS-01 challenges, use it as a libdns provider in Go, and the HTTP contract it relies on
locale: en
tags:
  - app:caddy-dns-immosquare
  - audience:technique
---

# immosquare DNS provider for Caddy and libdns

`caddy-dns-immosquare` is a Go module with two roles. It is the Caddy DNS provider module `dns.providers.immosquare`, which lets [Caddy](https://github.com/caddyserver/caddy) solve ACME DNS-01 challenges, the only challenge that yields wildcard certificates. It is also a [`libdns`](https://github.com/libdns/libdns) provider that any Go program can use to read and write DNS records. Both roles talk to a single backend, the immosquare DNS API. This page covers who the provider is for, building Caddy with it, configuring it, using it from Go, how it behaves, the HTTP contract it relies on, and how to test it.

## Who can use the immosquare DNS provider

The immosquare DNS provider only manages zones served by the immosquare DNS. It has no backend of its own and speaks no other API. Using it takes three things: the base URL of the immosquare DNS API, a bearer token for that API with the `dns` scope, and a zone that the API serves. A zone the API does not serve answers `404`, and the provider returns that answer as an error.

If your DNS is hosted anywhere else, this module cannot manage your records. Use the provider for your DNS host from the [caddy-dns organization](https://github.com/caddy-dns) instead: there is one for most DNS hosts, and they all plug into Caddy the same way.

The repository is public because Caddy fetches its modules as Go modules when it is compiled. A public module builds with plain `xcaddy` on any machine, without Go proxy credentials or `GOPRIVATE` settings. The HTTP contract in « HTTP contract of the immosquare DNS API used by the provider » is there for the maintainers of the module and of the API. It is not offered as a standard for other servers to implement.

## Building Caddy with the immosquare DNS provider

The immosquare DNS provider is not a standalone binary. It is compiled into Caddy with [xcaddy](https://github.com/caddyserver/xcaddy). The module requires Caddy v2.10 or later, the first release built on libdns v1. xcaddy needs Go 1.25 or later, the version declared in `go.mod`.

```bash
xcaddy build --with github.com/immosquare/caddy-dns-immosquare
```

To build Caddy from a local checkout of the module, for instance before tagging a release, point xcaddy at the working copy:

```bash
xcaddy build --with github.com/immosquare/caddy-dns-immosquare=.
```

The `build.sh` helper at the repository root installs xcaddy when it is missing, then builds Caddy against the latest published version of the module. Once Caddy is built, `caddy list-modules` lists `dns.providers.immosquare`.

## Configuring the immosquare DNS provider in a Caddyfile or in JSON

The immosquare DNS provider takes two options, and both are required. Caddy refuses to load a configuration where either one is missing, so a typo shows up when the configuration is reloaded, not at the first certificate renewal.

| Option      | Description                                                                     |
| ----------- | ------------------------------------------------------------------------------- |
| `api_token` | Bearer token of the immosquare DNS API, issued with the `dns` scope             |
| `endpoint`  | Base URL of the immosquare DNS API, without trailing slash (e.g. `.../api/dns`) |

Both values accept Caddy placeholders, so the token can stay out of the Caddyfile. `{env.IMMOSQUARE_DNS_TOKEN}` is resolved when the module is provisioned. In a site block, the provider goes inside `tls`:

```
*.example.com {
  tls {
    dns immosquare {
      api_token {env.IMMOSQUARE_DNS_TOKEN}
      endpoint  https://dns-api.example.com/api/dns
    }
  }
}
```

The token can also be passed inline, as in `dns immosquare <api_token> { endpoint <endpoint> }`. The global option `acme_dns immosquare { ... }` takes the same block and applies the provider to every site. In JSON, the provider goes in the DNS challenge of the [ACME issuer](https://caddyserver.com/docs/json/apps/tls/automation/policies/issuer/acme/):

```json
{
  "module": "acme",
  "challenges": {
    "dns": {
      "provider": {
        "name": "immosquare",
        "api_token": "YOUR_API_TOKEN",
        "endpoint": "https://dns-api.example.com/api/dns"
      }
    }
  }
}
```

Caddy's standard DNS challenge options (`propagation_delay`, `propagation_timeout`, `resolvers`) work as usual. Setting `resolvers` to the zone's authoritative nameservers lets Caddy check the challenge record where it is published. Otherwise Caddy waits on a recursive resolver that may still have a negative answer cached.

## Using the immosquare DNS provider as a libdns provider in Go

The `Provider` type in the package `immosquare` implements the four libdns interfaces: `RecordGetter`, `RecordAppender`, `RecordSetter` and `RecordDeleter`. Any library built on libdns can use it, certmagic included:

```go
import immosquare "github.com/immosquare/caddy-dns-immosquare"

provider := &immosquare.Provider{
  APIToken: "YOUR_API_TOKEN",
  Endpoint: "https://dns-api.example.com/api/dns",
}

records, err := provider.GetRecords(ctx, "example.com.")
```

Record names are relative to the zone, as libdns expects: `@` for the apex, `_acme-challenge` for a challenge record. Records come back as typed libdns structures (`libdns.Address`, `libdns.TXT`, `libdns.CNAME`, `libdns.MX`, `libdns.NS`), parsed with `libdns.RR.Parse`. A type the parser does not know stays a plain `libdns.RR`. The zone name is passed to the API as given, trailing dot included.

## How the immosquare DNS provider handles TTLs, RRsets and errors

Each libdns operation maps to one call to the immosquare DNS API, and the provider follows the libdns contract for each one:

- **`AppendRecords`** adds the records without touching the existing ones. A TTL below 120 seconds is raised to 120 seconds. certmagic creates challenge records with a TTL of 0, which would otherwise fall back to the zone default and slow down propagation checks.
- **`SetRecords`** makes the given records the only members of their RRset, that is, of their (name, type) pair. Every other record in the zone stays as it is. The API applies the change atomically: if one record is invalid, nothing changes and the call returns an error. The same 120-second minimum TTL applies.
- **`DeleteRecords`** removes the records that match name, type and value exactly. No minimum TTL applies. A record that does not exist makes the call fail.
- **`GetRecords`** lists every record in the zone.

Any answer outside the 2xx range is returned as an error that carries the HTTP status and the API's message. This matters for ACME. certmagic only learns from the returned error that a challenge record was not written or not removed. A swallowed failure would leave stale `_acme-challenge` records behind until the zone reached its TXT limit and renewals started failing.

## HTTP contract of the immosquare DNS API used by the provider

The immosquare DNS provider calls four endpoints under the configured base URL. Every call is authenticated with `Authorization: Bearer <api_token>`, and `{zone}` is the zone name as Caddy passes it, trailing dot included.

| Method   | Path                    | libdns operation | Success | Errors                                                   |
| -------- | ----------------------- | ---------------- | ------- | -------------------------------------------------------- |
| `GET`    | `/zones/{zone}/records` | `GetRecords`     | 200     | 401 bad token, 404 unknown zone, 422 unknown type filter |
| `POST`   | `/zones/{zone}/records` | `AppendRecords`  | 200     | 401, 404, 422 invalid record (nothing is written)        |
| `PUT`    | `/zones/{zone}/records` | `SetRecords`     | 200     | 401, 404, 422 invalid record (nothing is changed)        |
| `DELETE` | `/zones/{zone}/records` | `DeleteRecords`  | 200     | 401, 404, 400 record not found                           |

Writes send the JSON body `{"records": [{"name", "type", "data", "ttl"}]}`, with names relative to the zone. An MX carries its preference in `data`, as in `"10 mail.example.com"`. The `GET` answer is `{"records": [...]}`. Each record in it has a fully qualified `name` without trailing dot, plus a `type`, a `ttl` and a `data` value. An MX has `preference` and `target` fields in place of `data`. The provider converts these names back to names relative to the zone.

## Testing the immosquare DNS provider against a live API

`go vet ./...` and `go build ./...` check the module itself. The program in `test/` runs every libdns operation against a live immosquare DNS API:

```bash
API_TOKEN=your-api-token ENDPOINT=https://dns-api.example.com/api/dns go run ./test
```

The program writes real records into the zone hard-coded in `test/test_provider.go`: a challenge TXT, plus A, CNAME, MX and NS records. Point it at a disposable zone, never at a production one.

## Contributing to caddy-dns-immosquare and its license

Bug reports and pull requests are welcome on GitHub. The changes in each release are listed in [CHANGELOG.md](CHANGELOG.md). This module is open source under the [MIT License](LICENSE).
