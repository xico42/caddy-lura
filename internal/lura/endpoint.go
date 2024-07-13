// Copyright © 2021 Lura Project a Series of LF Projects, LLC
// Use of this source code is governed by Apache License, Version 2.0 that can be found
// at https://github.com/luraproject/lura/blob/master/LICENSE

package lura

import (
	"context"
	"errors"
	"fmt"
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/luraproject/lura/v2/config"
	"github.com/luraproject/lura/v2/core"
	"github.com/luraproject/lura/v2/logging"
	"github.com/luraproject/lura/v2/proxy"
	"github.com/luraproject/lura/v2/router"
	"github.com/luraproject/lura/v2/router/mux"
	"github.com/luraproject/lura/v2/transport/http/server"
	"github.com/xico42/caddy-lura/internal/httprouter"
	"net/http"
	"net/textproto"
	"strings"
)

func registerEndpoints(luraRouter *httprouter.Router, proxyFactory proxy.Factory, logger logging.Logger, opts Opts) {
	if opts.ServiceConfig.Debug {
		debugHandler := mux.DebugHandler(logger)
		for _, method := range allMethods {
			luraRouter.HandlerFunc(method, opts.DebugPattern, func(rw http.ResponseWriter, req *http.Request) error {
				debugHandler.ServeHTTP(rw, req)
				return nil
			})
		}
	}

	if opts.ServiceConfig.Echo {
		echoHandler := mux.EchoHandler()
		for _, method := range allMethods {
			luraRouter.HandlerFunc(method, opts.EchoPattern, func(rw http.ResponseWriter, req *http.Request) error {
				echoHandler.ServeHTTP(rw, req)
				return nil
			})
		}
	}

	for _, c := range opts.ServiceConfig.Endpoints {
		proxyStack, err := proxyFactory.New(c)
		if err != nil {
			logger.Error(logPrefix, "could not instantiate the proxy stack", err.Error())
			continue
		}

		handler := buildEndpointHandle(c, proxyStack)

		method := strings.ToTitle(c.Method)
		path := c.Endpoint
		if method != http.MethodGet && len(c.Backend) > 1 {
			if !router.IsValidSequentialEndpoint(c) {
				logger.Error(logPrefix, method, " endpoints with sequential proxy enabled only allow a non-GET in the last backend! Ignoring", path)
				return
			}
		}

		switch method {
		case http.MethodGet:
		case http.MethodPost:
		case http.MethodPut:
		case http.MethodPatch:
		case http.MethodDelete:
		default:
			logger.Error(logPrefix, "Unsupported method", method)
			return
		}
		logger.Debug(logPrefix, "Registering the endpoint", method, path)

		luraRouter.Handle(method, path, handler)
	}
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

func buildEndpointHandle(configuration *config.EndpointConfig, prxy proxy.Proxy) httprouter.Handle {
	cacheControlHeaderValue := fmt.Sprintf("public, max-age=%d", int(configuration.CacheTTL.Seconds()))
	isCacheEnabled := configuration.CacheTTL.Seconds() != 0
	render := getRender(configuration)

	headersToSend := configuration.HeadersToPass
	if len(headersToSend) == 0 {
		headersToSend = server.HeadersToSend
	}
	method := strings.ToTitle(configuration.Method)

	return func(w http.ResponseWriter, r *http.Request, params httprouter.Params) (err error) {
		w.Header().Set(core.KrakendHeaderName, core.KrakendHeaderValue)
		if r.Method != method {
			w.Header().Set(server.CompleteResponseHeaderName, server.HeaderIncompleteResponseValue)
			http.Error(w, "", http.StatusMethodNotAllowed)
			return caddyhttp.Error(http.StatusMethodNotAllowed, fmt.Errorf("unexepected method: %s", r.Method))
		}

		requestCtx, cancel := context.WithTimeout(r.Context(), configuration.Timeout)

		proxyRequest := buildProxyRequest(r, configuration.QueryString, headersToSend, params)
		response, err := prxy(requestCtx, proxyRequest)

		select {
		case <-requestCtx.Done():
			if err == nil {
				err = server.ErrInternalError
			}
		default:
		}

		if response != nil && len(response.Data) > 0 {
			if response.IsComplete {
				w.Header().Set(server.CompleteResponseHeaderName, server.HeaderCompleteResponseValue)
				if isCacheEnabled {
					w.Header().Set("Cache-Control", cacheControlHeaderValue)
				}
			} else {
				w.Header().Set(server.CompleteResponseHeaderName, server.HeaderIncompleteResponseValue)
			}

			for k, vs := range response.Metadata.Headers {
				for _, v := range vs {
					w.Header().Add(k, v)
				}
			}
		} else {
			w.Header().Set(server.CompleteResponseHeaderName, server.HeaderIncompleteResponseValue)
			if err != nil {
				var responseError errorWithStatusCode
				if errors.As(err, &responseError) {
					err = caddyhttp.Error(responseError.StatusCode(), err)
				} else {
					err = caddyhttp.Error(http.StatusInternalServerError, err)
				}
				cancel()
				return
			}
		}

		err = render(w, response)
		cancel()
		return err
	}
}

func buildProxyRequest(r *http.Request, queryString, headersToSend []string, reqParams httprouter.Params) *proxy.Request {
	replacer := r.Context().Value(caddy.ReplacerCtxKey).(*caddy.Replacer)

	for _, p := range reqParams {
		replacer.Set(p.Key, p.Value)
	}
	headers := make(map[string][]string, 3+len(headersToSend))

	for _, k := range headersToSend {
		if k == "*" {
			headers = r.Header

			break
		}

		if h, ok := r.Header[textproto.CanonicalMIMEHeaderKey(k)]; ok {
			headers[k] = h
		}
	}

	headers["X-Forwarded-For"] = []string{clientIP(r)}
	headers["X-Forwarded-Host"] = []string{r.Host}
	// if User-Agent is not forwarded using headersToSend, we set
	// the KrakenD router User Agent value
	if _, ok := headers["User-Agent"]; !ok {
		headers["User-Agent"] = server.UserAgentHeaderValue
	} else {
		headers["X-Forwarded-Via"] = server.UserAgentHeaderValue
	}

	query := make(map[string][]string, len(queryString))
	queryValues := r.URL.Query()
	for i := range queryString {
		if queryString[i] == "*" {
			query = queryValues

			break
		}

		if v, ok := queryValues[queryString[i]]; ok && len(v) > 0 {
			query[queryString[i]] = v
		}
	}

	return &proxy.Request{
		Path:    r.URL.Path,
		Method:  r.Method,
		Query:   query,
		Body:    r.Body,
		Params:  nil,
		Headers: headers,
	}
}

type errorWithStatusCode interface {
	error
	StatusCode() int
}
