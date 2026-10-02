package immosquare

import (
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
)

func init() {
	caddy.RegisterModule(Provider{})
}

// CaddyModule returns the Caddy module information.
func (Provider) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "dns.providers.immosquare",
		New: func() caddy.Module { return new(Provider) },
	}
}

// Provision sets up the module. Implements caddy.Provisioner.
func (p *Provider) Provision(ctx caddy.Context) error {
	p.APIToken = caddy.NewReplacer().ReplaceAll(p.APIToken, "")
	p.Endpoint = caddy.NewReplacer().ReplaceAll(p.Endpoint, "")
	return nil
}

// UnmarshalCaddyfile sets up the DNS provider from Caddyfile tokens. Syntax:
//
//	immosquare [<api_token>] {
//	    api_token <api_token>
//	    endpoint  <endpoint>
//	}
//
// Both the API token and the endpoint are required: a missing value fails
// when the configuration loads instead of at the first ACME challenge.
func (p *Provider) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	for d.Next() {
		if d.NextArg() {
			p.APIToken = d.Val()
		}
		if d.NextArg() {
			return d.ArgErr()
		}
		for nesting := d.Nesting(); d.NextBlock(nesting); {
			switch d.Val() {
			case "api_token":
				if p.APIToken != "" {
					return d.Err("API token already set")
				}
				if d.NextArg() {
					p.APIToken = d.Val()
				}
				if d.NextArg() {
					return d.ArgErr()
				}
			case "endpoint":
				if p.Endpoint != "" {
					return d.Err("endpoint already set")
				}
				if d.NextArg() {
					p.Endpoint = d.Val()
				}
				if d.NextArg() {
					return d.ArgErr()
				}
			default:
				return d.Errf("unrecognized subdirective '%s'", d.Val())
			}
		}
	}
	if p.APIToken == "" {
		return d.Err("missing API token")
	}
	if p.Endpoint == "" {
		return d.Err("missing endpoint")
	}
	return nil
}

// Interface guards
var (
	_ caddy.Module          = (*Provider)(nil)
	_ caddyfile.Unmarshaler = (*Provider)(nil)
	_ caddy.Provisioner     = (*Provider)(nil)
)
