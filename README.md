---
resume: Build Caddy with the immosquare DNS provider, configure it for ACME DNS-01 challenges, use it as a libdns provider in Go, and the HTTP contract it relies on
locale: en
tags:
  - app:caddy-dns-immosquare
  - audience:technique
---

# immosquare DNS provider for Caddy and libdns

`caddy-dns-immosquare` is a Go module with two roles: it is the Caddy DNS provider module `dns.providers.immosquare`, which lets [Caddy](https://github.com/caddyserver/caddy) solve ACME DNS-01 challenges (the only challenge that yields wildcard certificates), and it is a [`libdns`](https://github.com/libdns/libdns) provider that any Go program can use to read and write DNS records. Both roles talk to the immosquare DNS API, the source of truth for the zones immosquare serves, so the provider only works against that API. This page covers building Caddy with the module, configuring it, using it from Go, how it behaves, the HTTP contract it relies on, and how to test it.

## Building Caddy with the immosquare DNS provider

The immosquare DNS provider is not a standalone binary: it is compiled into Caddy with [xcaddy](https://github.com/caddyserver/xcaddy), which requires Go 1.25 or later.

```bash
xcaddy build --with github.com/immosquare/caddy-dns-immosquare
```

To build Caddy against a local checkout of the module, for instance before tagging a release, point xcaddy at the working copy:

```bash
xcaddy build --with github.com/immosquare/caddy-dns-immosquare=.
```

The `build.sh` helper at the repository root installs xcaddy when it is missing and builds Caddy against the latest published version of the module. Once built, `caddy list-modules` lists `dns.providers.immosquare`.

## Configuring the immosquare DNS provider in a Caddyfile or in JSON

The immosquare DNS provider takes two options, both required. Caddy refuses to load a configuration where either is missing, so a typo surfaces at reload time rather than at the first certificate renewal.

| Option      | Description                                                                         |
| ----------- | ----------------------------------------------------------------------------------- |
| `api_token` | Bearer token of the immosquare DNS API, issued with the `dns` scope                 |
| `endpoint`  | Base URL of the immosquare DNS API, without trailing slash (e.g. `.../api/dns`)     |

Both values accept Caddy placeholders, so the token can stay out of the Caddyfile: `{env.IMMOSQUARE_DNS_TOKEN}` is resolved when the module is provisioned. In a site block, the provider goes inside `tls`:

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

The token can also be given inline, as in `dns immosquare <api_token> { endpoint <endpoint> }`, and the global option `acme_dns immosquare { ... }` takes the same block to apply the provider to every site. In JSON, the provider is declared in the DNS challenge of the [ACME issuer](https://caddyserver.com/docs/json/apps/tls/automation/policies/issuer/acme/):

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

The standard DNS challenge options of Caddy (`propagation_delay`, `propagation_timeout`, `resolvers`) apply unchanged. Pointing `resolvers` at the authoritative nameservers of the zone lets Caddy check the challenge record where it is published, instead of waiting on a recursive resolver that may still cache a negative answer.

## Using the immosquare DNS provider as a libdns provider in Go

The `Provider` type of the package `immosquare` implements the four libdns interfaces (`RecordGetter`, `RecordAppender`, `RecordSetter`, `RecordDeleter`), so it plugs into any library built on libdns, certmagic included:

```go
import immosquare "github.com/immosquare/caddy-dns-immosquare"

provider := &immosquare.Provider{
  APIToken: "YOUR_API_TOKEN",
  Endpoint: "https://dns-api.example.com/api/dns",
}

records, err := provider.GetRecords(ctx, "example.com.")
```

Record names are relative to the zone, as libdns expects: `@` for the apex, `_acme-challenge` for a challenge record. Records come back as their typed libdns structures (`libdns.Address`, `libdns.TXT`, `libdns.CNAME`, `libdns.MX`, `libdns.NS`), parsed with `libdns.RR.Parse`; a type the parser does not know stays a plain `libdns.RR`.

This module replaces `github.com/immosquare/libdns-immosquare`, which is archived. Code that used it only changes its import: the `Provider` type keeps its name and its `APIToken` and `Endpoint` fields, and lives in the package `immosquare` instead of `libdnsimmosquare`.

## How the immosquare DNS provider handles TTLs, RRsets and errors

Each libdns operation maps to one call of the immosquare DNS API, and the provider keeps the libdns contract for each of them:

- **`AppendRecords`** adds the records without touching the existing ones. A TTL below 120 seconds is raised to 120 seconds: certmagic creates challenge records with a TTL of 0, which would otherwise fall back to the zone default and slow down propagation checks.
- **`SetRecords`** makes the given records the only members of their RRset, that is of their (name, type) pair; every other record of the zone stays untouched. The API applies the change atomically: if one record is invalid, nothing changes and the call returns an error. The same 120-second minimum TTL applies.
- **`DeleteRecords`** removes records matching name, type and value exactly, without the TTL minimum. A record that does not exist makes the call fail.
- **`GetRecords`** lists every record of the zone.

Any answer outside the 2xx range is returned as an error carrying the HTTP status and the message of the API. This matters for ACME: certmagic only learns from the returned error that a challenge record was not written or not removed, and a swallowed failure would leave stale `_acme-challenge` records behind until the zone reaches its TXT limit and renewals start failing.

## HTTP contract of the immosquare DNS API used by the provider

The provider calls four endpoints under the configured base URL, all authenticated with `Authorization: Bearer <api_token>`. `{zone}` is the zone name as Caddy passes it, trailing dot included.

| Method   | Path                      | libdns operation | Success | Errors                                                       |
| -------- | ------------------------- | ---------------- | ------- | ------------------------------------------------------------ |
| `GET`    | `/zones/{zone}/records`   | `GetRecords`     | 200     | 401 bad token, 404 unknown zone, 422 unknown type filter     |
| `POST`   | `/zones/{zone}/records`   | `AppendRecords`  | 200     | 401, 404, 422 invalid record (nothing is written)            |
| `PUT`    | `/zones/{zone}/records`   | `SetRecords`     | 200     | 401, 404, 422 invalid record (nothing is changed)            |
| `DELETE` | `/zones/{zone}/records`   | `DeleteRecords`  | 200     | 401, 404, 400 record not found                               |

Writes send a JSON body `{"records": [{"name", "type", "data", "ttl"}]}` with relative names; an MX carries its preference in `data`, as in `"10 mail.example.com"`. The `GET` answer is `{"records": [...]}` where each record has a fully qualified `name` without trailing dot, a `type`, a `ttl` and a `data` value; an MX has `preference` and `target` instead of `data`. The provider converts these names back to names relative to the zone.

## Testing the immosquare DNS provider against a live API

`go vet ./...` and `go build ./...` check the module itself. The program in `test/` exercises every libdns operation against a running immosquare DNS API:

```bash
API_TOKEN=your-api-token ENDPOINT=https://dns-api.example.com/api/dns go run ./test
```

The program writes real records — a challenge TXT, A, CNAME, MX and NS records — into the zone hard-coded in `test/test_provider.go`. Point it at a disposable zone, never at a production one.

## Contributing to caddy-dns-immosquare and its license

Bug reports and pull requests are welcome on GitHub. This module is available as open source under the terms of the [MIT License](LICENSE).
