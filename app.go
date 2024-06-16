package caddylura

import (
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/luraproject/lura/v2/config"
	"github.com/xico42/caddy-lura/internal/lura"
	"net/http"
	"time"
)

func init() {
	caddy.RegisterModule(new(Lura))
}

// Lura implements a high-performance API Gateway using the Lura framework (https://luraproject.org/).
//
// This module provides advanced API gateway functionalities.
// It allows defining multiple endpoints, each specifying backend services for request processing.
// The module supports response aggregation and transformation rules per endpoint, facilitating
// complex API orchestrations.
type Lura struct {
	// Set of endpoint definitions representing the gateway public API.
	Endpoints []Endpoint `json:"endpoints,omitempty"`

	// Default timeout applied to all backends. This timeout is applied if not overridden at the endpoint level.
	Timeout caddy.Duration `json:"timeout,omitempty"`

	// Default cache duration applied to all backends. This affects caching headers for responses.
	CacheTTL caddy.Duration `json:"cache_ttl,omitempty"`

	// DebugEndpoint exposes an URL that can be used as a fake backend when the API gateway itself is used as a backend host.
	// It logs all activity to aid in debugging interactions between the gateway and backends when Caddy is run with debug logging.
	DebugEndpoint HelperEndpoint `json:"debug_endpoint,omitempty"`

	// EchoEndpoint is a developer tool similar to DebugEndpoint but instead of logging, responses are printed directly.
	// Useful for debugging configurations without verbose logging.
	EchoEndpoint HelperEndpoint `json:"echo_endpoint,omitempty"`

	// handler is the internal HTTP handler for serving requests handled by the API Gateway module within Caddy.
	handler http.Handler
}

// Endpoint represents a public-facing gateway URL with specific configurations.
type Endpoint struct {
	// URLPattern defines the endpoint URL, supporting path parameters. It specifies the public API endpoint exposed by the gateway.
	//
	// Example: "/users/{id}/permissions"
	URLPattern string `json:"url_pattern,omitempty"`

	// Method specifies the HTTP method for the endpoint. If not specified, "GET" is assumed.
	Method string `json:"method,omitempty"`

	// ConcurrentCalls specifies the number of concurrent calls this endpoint makes to each backend.
	// Controls the concurrency of backend requests to optimize performance.
	ConcurrentCalls int `json:"concurrent_calls,omitempty"`

	// Timeout specifies the timeout duration for requests to this endpoint. Overrides the default timeout if set.
	Timeout caddy.Duration `json:"timeout,omitempty"`

	// CacheTTL specifies the cache duration for responses from this endpoint. Controls caching headers for client caching.
	CacheTTL caddy.Duration `json:"cache_ttl,omitempty"`

	// Backends specifies the set of backend services that serve requests for this endpoint.
	// Responses from multiple backends are aggregated based on rules defined in the gateway configuration.
	Backends []Backend `json:"backends,omitempty"`
}

// Backend represents a backend service that handles requests for an endpoint.
type Backend struct {
	// Host specifies the list of backend hosts. Requests are load balanced between these hosts.
	Host []string `json:"host,omitempty"`

	// URLPattern specifies the URL pattern to locate the resource on the backend service.
	// Path variables defined in the endpoint configuration can be used here, along with any Caddy placeholders.
	//
	// Example: "/api/{header.X-Tenant-Id}/resources/{id}"
	URLPattern string `json:"url_pattern,omitempty"`

	// AllowList specifies the list of response fields allowed from the backend.
	// If empty, all fields are returned. Helps in filtering unnecessary data from responses.
	AllowList []string `json:"allow_list,omitempty"`

	// Mapping specifies renaming rules for response fields. Useful for standardizing response formats.
	Mapping map[string]string `json:"mapping,omitempty"`

	// Group specifies the property name to which the response should be moved.
	// Useful for combining responses from multiple backends.
	Group string `json:"group,omitempty"`

	// Method specifies the HTTP method used for requests to the backend service.
	Method string `json:"method,omitempty"`
}

// HelperEndpoint represents a helper endpoint for developers within the Caddy web server.
type HelperEndpoint struct {
	// URLPattern specifies the URL where the helper endpoint is served.
	// Default value is "/__debug/".
	URLPattern string `json:"url_pattern,omitempty"`

	// Enabled specifies whether the helper endpoint is enabled or not.
	// Default value is "/__echo/"
	Enabled bool `json:"enabled,omitempty"`
}

func (l *Lura) Provision(ctx caddy.Context) error {
	endpoints := make([]*config.EndpointConfig, 0, len(l.Endpoints))
	for _, e := range l.Endpoints {
		endpointParams := newParamsSetFromPattern(e.URLPattern)

		backends := make([]*config.Backend, 0, len(e.Backends))
		for _, b := range e.Backends {
			backendParams := newParamsSetFromPattern(b.URLPattern)

			backends = append(backends, &config.Backend{
				Host:       b.Host,
				URLPattern: processBackendUrlPattern(b.URLPattern, backendParams, endpointParams),
				AllowList:  b.AllowList,
				Mapping:    b.Mapping,
				Group:      b.Group,
				Method:     b.Method,
			})
		}

		endpoints = append(endpoints, &config.EndpointConfig{
			Endpoint:        e.URLPattern,
			Method:          e.Method,
			ConcurrentCalls: e.ConcurrentCalls,
			CacheTTL:        time.Duration(e.CacheTTL),
			Timeout:         time.Duration(e.Timeout),
			Backend:         backends,
		})
	}

	cfg := config.ServiceConfig{
		Version:   3,
		Name:      "Caddy Lura",
		Timeout:   time.Duration(l.Timeout),
		CacheTTL:  time.Duration(l.CacheTTL),
		Endpoints: endpoints,
		Debug:     l.DebugEndpoint.Enabled,
		Echo:      l.EchoEndpoint.Enabled,
	}

	err := cfg.Init()
	if err != nil {
		return err
	}

	for _, e := range cfg.Endpoints {
		for _, b := range e.Backend {
			b.URLPattern = applyCaddyPlaceholders(b.URLPattern)
		}
	}

	luraHandler, err := lura.NewHandler(lura.Opts{
		ServiceConfig: cfg,
		ZapLogger:     ctx.Logger(),
		DebugPattern:  l.DebugEndpoint.URLPattern,
		EchoPattern:   l.EchoEndpoint.URLPattern,
	})
	if err != nil {
		return err
	}

	l.handler = luraHandler

	return nil
}

func (l *Lura) ServeHTTP(rw http.ResponseWriter, req *http.Request, next caddyhttp.Handler) error {
	l.handler.ServeHTTP(rw, req)
	return nil
}

func (*Lura) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "http.handlers.lura",
		New: func() caddy.Module {
			return new(Lura)
		},
	}
}

// Interface guards
var (
	_ caddy.Provisioner = (*Lura)(nil)
	// _ caddy.Validator             = (*Middleware)(nil)
	_ caddyhttp.MiddlewareHandler = (*Lura)(nil)
	_ caddyfile.Unmarshaler       = (*Lura)(nil)
)
