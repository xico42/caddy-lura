package caddylura

import (
	"context"
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
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
	httpcaddyfile.RegisterHandlerDirective("lura", func(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
		return new(Lura), nil
	})
}

type Lura struct {
	handler http.Handler
}

func (l *Lura) Provision(ctx caddy.Context) error {
	cfg := config.ServiceConfig{
		Version:  3,
		Name:     "Mey Lovely gateway",
		Timeout:  10 * time.Second,
		CacheTTL: 3600 * time.Second,
		Endpoints: []*config.EndpointConfig{
			{
				Endpoint: "/users/{user}",
				Method:   "GET",
				Backend: []*config.Backend{
					{
						Host:       []string{"http://mock:8081"},
						URLPattern: "/registered/{user}",
						AllowList: []string{
							"id",
							"name",
							"email",
						},
						Mapping: map[string]string{
							"email": "personal_email",
						},
					},
					{
						Host:       []string{"http://mock:8081"},
						URLPattern: "/users/{user}/permissions",
						Group:      "permissions",
					},
				},
				ConcurrentCalls: 2,
				Timeout:         1000 * time.Second,
				CacheTTL:        3600 * time.Second,
			},
		},
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
	// _ caddyfile.Unmarshaler       = (*Middleware)(nil)
)
