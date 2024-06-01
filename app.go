package caddylura

import (
	"context"
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	coregin "github.com/gin-gonic/gin"
	"github.com/luraproject/lura/v2/config"
	"github.com/luraproject/lura/v2/logging"
	"github.com/luraproject/lura/v2/proxy"
	"github.com/luraproject/lura/v2/router/gin"
	"net/http"
	"os"
	"time"
)

func init() {
	caddy.RegisterModule(new(Lura))
}

type Backend struct {
	Host       []string
	URLPattern string
	AllowList  []string
	Mapping    map[string]string
	Group      string
	Method     string
}

type Endpoint struct {
	URLPattern      string
	Method          string
	ConcurrentCalls int
	Timeout         caddy.Duration
	CacheTTL        caddy.Duration
	Backends        []Backend
}

type Lura struct {
	Endpoints []Endpoint
	Timeout   caddy.Duration
	CacheTTL  caddy.Duration

	handler http.Handler
}

func (l *Lura) Provision(ctx caddy.Context) error {
	endpoints := make([]*config.EndpointConfig, 0, len(l.Endpoints))
	for _, e := range l.Endpoints {
		backends := make([]*config.Backend, 0, len(e.Backends))
		for _, b := range e.Backends {
			backends = append(backends, &config.Backend{
				Host:       b.Host,
				URLPattern: b.URLPattern,
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
	}

	err := cfg.Init()
	if err != nil {
		return err
	}

	logger, _ := logging.NewLogger("DEBUG", os.Stdout, "[LURA]")

	var luraHandler http.Handler

	routerFactory := gin.NewFactory(
		gin.Config{
			Engine:         coregin.Default(),
			Middlewares:    []coregin.HandlerFunc{},
			HandlerFactory: gin.EndpointHandler,
			ProxyFactory:   proxy.DefaultFactory(logger),
			Logger:         logger,
			RunServer: func(ctx context.Context, serviceConfig config.ServiceConfig, handler http.Handler) error {
				luraHandler = handler
				return nil
			},
		},
	)

	routerFactory.New().Run(cfg)

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
