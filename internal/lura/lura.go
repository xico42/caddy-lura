package lura

import (
	"context"
	"errors"
	"github.com/caddyserver/caddy/v2"
	"github.com/luraproject/lura/v2/config"
	"github.com/luraproject/lura/v2/logging"
	"github.com/luraproject/lura/v2/proxy"
	"github.com/luraproject/lura/v2/router/mux"
	"github.com/luraproject/lura/v2/transport/http/client"
	"go.uber.org/zap"
	"net/http"
	"strings"
)

const (
	defaultDebugPattern = "/__debug/*any"
	defaultEchoPattern  = "/__echo/*any"
)

var (
	backendHttpProxy = proxy.CustomHTTPProxyFactory(client.NewHTTPClient)
)

type Opts struct {
	ServiceConfig config.ServiceConfig
	ZapLogger     *zap.Logger
	DebugPattern  string
	EchoPattern   string
}

func NewHandler(opts Opts) (http.Handler, error) {
	logger := newLogger(opts.ZapLogger)

	var luraHandler http.Handler

	proxyFactory := newProxyFactory(logger)

	if opts.DebugPattern == "" {
		opts.DebugPattern = defaultDebugPattern
	} else {
		opts.DebugPattern = strings.TrimRight(opts.DebugPattern, "/") + "/*any"
	}

	if opts.EchoPattern == "" {
		opts.EchoPattern = defaultEchoPattern
	} else {
		opts.EchoPattern = strings.TrimRight(opts.EchoPattern, "/") + "/*any"
	}

	routerFactory := mux.NewFactory(
		mux.Config{
			Engine:         newHttpRouterEngine(),
			Middlewares:    []mux.HandlerMiddleware{},
			HandlerFactory: endpointHandler,
			ProxyFactory:   proxyFactory,
			Logger:         logger,
			DebugPattern:   opts.DebugPattern,
			EchoPattern:    opts.EchoPattern,
			RunServer: func(ctx context.Context, serviceConfig config.ServiceConfig, handler http.Handler) error {
				luraHandler = handler
				return nil
			},
		},
	)

	routerFactory.New().Run(opts.ServiceConfig)

	return luraHandler, nil
}

func newProxyFactory(logger logging.Logger) proxy.Factory {
	return proxy.NewDefaultFactory(newBackendFactory(), logger)
}

func newBackendFactory() proxy.BackendFactory {
	return func(remote *config.Backend) proxy.Proxy {
		next := backendHttpProxy(remote)
		return func(ctx context.Context, request *proxy.Request) (*proxy.Response, error) {
			request.GeneratePath(remote.URLPattern)
			request.Params = nil
			replacer, ok := ctx.Value(caddy.ReplacerCtxKey).(*caddy.Replacer)
			if !ok {
				return nil, errors.New("could not find caddy replacer")
			}
			path, err := replacer.ReplaceOrErr(request.Path, true, true)
			if err != nil {
				return nil, err
			}
			request.Path = path
			request.URL.Path = path
			return next(ctx, request)
		}
	}
}
