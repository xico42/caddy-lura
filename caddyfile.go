package caddylura

import (
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"strconv"
	"strings"
)

func init() {
	httpcaddyfile.RegisterHandlerDirective("lura", parseCaddyfile)
}

func parseCaddyfile(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	l := new(Lura)

	err := l.UnmarshalCaddyfile(h.Dispenser)

	return l, err
}

func (l *Lura) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	// consume directive name
	d.Next()

	var err error
	endpoints := make([]Endpoint, 0)

	for d.NextBlock(0) {
		switch d.Val() {
		case "timeout":
			l.Timeout, err = unmarshalDuration(d)
			if err != nil {
				return err
			}

			break
		case "cache_ttl":
			l.CacheTTL, err = unmarshalDuration(d)
			if err != nil {
				return err
			}

			break
		case "endpoint":
			e, err := unmarshalEndpoint(d)
			if err != nil {
				return err
			}
			endpoints = append(endpoints, e)

		default:
			return d.Errf("unrecognized subdirective %s", d.Val())
		}
	}

	l.Endpoints = endpoints

	return nil
}

func unmarshalDuration(d *caddyfile.Dispenser) (caddy.Duration, error) {
	durArg, err := unmarshalSingleArg(d)
	if err != nil {
		return 0, err
	}
	dur, err := caddy.ParseDuration(durArg)
	if err != nil {
		return 0, d.Errf("bad duration value %s: %v", d.Val(), err)
	}
	return caddy.Duration(dur), nil
}

func unmarshalEndpoint(d *caddyfile.Dispenser) (e Endpoint, err error) {

	args := d.RemainingArgs()
	if len(args) > 0 {
		e.URLPattern = args[0]
	}

	backends := make([]Backend, 0)

	curNesting := d.Nesting()
	for d.NextBlock(curNesting) {
		switch d.Val() {
		case "url_pattern":
			e.URLPattern, err = unmarshalSingleArg(d)
			if err != nil {
				return
			}
			break

		case "method":
			e.Method, err = unmarshalSingleArg(d)
			if err != nil {
				return
			}
			break

		case "backend":
			var b Backend
			b, err = unmarshalBackend(d)
			if err != nil {
				return
			}
			backends = append(backends, b)

		case "concurrent_calls":
			var arg string
			arg, err = unmarshalSingleArg(d)
			if err != nil {
				return
			}
			e.ConcurrentCalls, err = strconv.Atoi(arg)
			if err != nil {
				err = d.Errf("failed to parse concurrent call: %w", err)
				return
			}
			break

		case "timeout":
			e.Timeout, err = unmarshalDuration(d)
			if err != nil {
				return
			}
			break

		case "cache_ttl":
			e.CacheTTL, err = unmarshalDuration(d)
			if err != nil {
				return
			}
			break

		default:
			err = d.Errf("unrecognized subdirective '%s' while parsing endpoint ", d.Val())
			return
		}
	}

	e.Backends = backends

	return
}

func unmarshalBackend(d *caddyfile.Dispenser) (b Backend, err error) {
	upstreams := make([]string, 0)
	for _, u := range d.RemainingArgs() {
		upstreams = append(upstreams, u)
	}

	curNesting := d.Nesting()
	for d.NextBlock(curNesting) {
		switch d.Val() {
		case "to":
			for _, u := range d.RemainingArgs() {
				upstreams = append(upstreams, u)
			}
			break

		case "url_pattern":
			b.URLPattern, err = unmarshalSingleArg(d)
			if err != nil {
				return
			}
			break

		case "allow":
			b.AllowList = d.RemainingArgs()
			break

		case "group":
			b.Group, err = unmarshalSingleArg(d)
			if err != nil {
				return
			}
			break

		case "method":
			b.Method, err = unmarshalSingleArg(d)
			if err != nil {
				return
			}
			break

		case "mapping":
			mapping := make(map[string]string)
			nesting := d.Nesting()
			for d.NextBlock(nesting) {
				m := d.Val()
				if strings.Contains(m, ">") {
					parts := strings.Split(m, ">")
					mapping[parts[0]] = parts[1]
				} else {
					err = d.Errf("mapping should be in the format source_field>target_field, but got: '%s'", m)
					return
				}
			}
			b.Mapping = mapping
			break
		default:
			err = d.Errf("unrecognized subdirective '%s' while parsing backend ", d.Val())
			return
		}
	}

	b.Host = upstreams

	return
}

func unmarshalSingleArg(d *caddyfile.Dispenser) (string, error) {
	args := d.RemainingArgs()
	if len(args) != 1 {
		return "", d.ArgErr()
	}

	return args[0], nil
}
