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
	"net/http"
	"os"
)

var (
	backendHttpProxy = proxy.CustomHTTPProxyFactory(client.NewHTTPClient)
)

func NewHandler(cfg config.ServiceConfig) (http.Handler, error) {
	logger, _ := logging.NewLogger("DEBUG", os.Stdout, "[LURA]")

	var luraHandler http.Handler

	proxyFactory := newProxyFactory(logger)

	routerFactory := mux.NewFactory(
		mux.Config{
			Engine:         newHttpRouterEngine(),
			Middlewares:    []mux.HandlerMiddleware{},
			HandlerFactory: endpointHandler,
			ProxyFactory:   proxyFactory,
			Logger:         logger,
			DebugPattern:   mux.DefaultDebugPattern,
			EchoPattern:    mux.DefaultEchoPattern,
			RunServer: func(ctx context.Context, serviceConfig config.ServiceConfig, handler http.Handler) error {
				luraHandler = handler
				return nil
			},
		},
	)

	routerFactory.New().Run(cfg)

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
