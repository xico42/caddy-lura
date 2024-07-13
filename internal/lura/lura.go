package lura

import (
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/luraproject/lura/v2/config"
	"github.com/luraproject/lura/v2/proxy"
	"github.com/luraproject/lura/v2/transport/http/client"
	"github.com/luraproject/lura/v2/transport/http/server"
	"github.com/xico42/caddy-lura/internal/httprouter"
	"go.uber.org/zap"
	"net/http"
	"strings"
)

const (
	defaultDebugPattern = "/__debug/*any"
	defaultEchoPattern  = "/__echo/*any"
)

var (
	logPrefix        = "[Service: Caddy Lura] "
	backendHttpProxy = proxy.CustomHTTPProxyFactory(client.NewHTTPClient)
	allMethods       = []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodHead,
		http.MethodOptions,
		http.MethodConnect,
		http.MethodTrace,
	}
)

type Opts struct {
	ServiceConfig config.ServiceConfig
	ZapLogger     *zap.Logger
	DebugPattern  string
	EchoPattern   string
}

func NewHandler(opts Opts) (caddyhttp.Handler, error) {
	logger := newLogger(opts.ZapLogger)

	luraRouter := httprouter.New()

	proxyFactory := newProxyFactory(logger)
	_ = proxyFactory

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

	server.InitHTTPDefaultTransport(opts.ServiceConfig)

	registerEndpoints(luraRouter, proxyFactory, logger, opts)

	return luraRouter, nil
}

func clientIP(r *http.Request) string {
	return caddyhttp.GetVar(r.Context(), caddyhttp.ClientIPVarKey).(string)
}
