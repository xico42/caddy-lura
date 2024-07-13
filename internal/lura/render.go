// Copyright © 2021 Lura Project a Series of LF Projects, LLC
// Use of this source code is governed by Apache License, Version 2.0 that can be found
// at https://github.com/luraproject/lura/blob/master/LICENSE

package lura

import (
	"encoding/json"
	"errors"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"io"
	"net/http"
	"sync"

	"github.com/luraproject/lura/v2/config"
	"github.com/luraproject/lura/v2/encoding"
	"github.com/luraproject/lura/v2/proxy"
)

// Render defines the signature of the functions to be use for the final response
// encoding and rendering
type Render func(http.ResponseWriter, *proxy.Response) error

// NEGOTIATE defines the value of the OutputEncoding for the negotiated render
const NEGOTIATE = "negotiate"

var (
	mutex          = &sync.RWMutex{}
	renderRegister = map[string]Render{
		encoding.STRING:   stringRender,
		encoding.JSON:     jsonRender,
		encoding.NOOP:     noopRender,
		"json-collection": jsonCollectionRender,
	}
)

// RegisterRender allows clients to register their custom renders
func RegisterRender(name string, r Render) {
	mutex.Lock()
	renderRegister[name] = r
	mutex.Unlock()
}

func getRender(cfg *config.EndpointConfig) Render {
	fallback := jsonRender
	if len(cfg.Backend) == 1 {
		fallback = getWithFallback(cfg.Backend[0].Encoding, fallback)
	}

	if cfg.OutputEncoding == "" {
		return fallback
	}

	return getWithFallback(cfg.OutputEncoding, fallback)
}

func getWithFallback(key string, fallback Render) Render {
	mutex.RLock()
	r, ok := renderRegister[key]
	mutex.RUnlock()
	if !ok {
		return fallback
	}
	return r
}

var (
	emptyResponse   = []byte("{}")
	emptyCollection = []byte("[]")
)

func jsonRender(w http.ResponseWriter, response *proxy.Response) error {
	w.Header().Set("Content-Type", "application/json")
	if response == nil {
		w.Write(emptyResponse)
		return nil
	}

	js, err := json.Marshal(response.Data)
	if err != nil {
		return caddyhttp.Error(http.StatusInternalServerError, err)
	}
	w.Write(js)
	return nil
}

func jsonCollectionRender(w http.ResponseWriter, response *proxy.Response) error {
	w.Header().Set("Content-Type", "application/json")
	if response == nil {
		w.Write(emptyCollection)
		return nil
	}
	col, ok := response.Data["collection"]
	if !ok {
		w.Write(emptyCollection)
		return nil
	}

	js, err := json.Marshal(col)
	if err != nil {
		return caddyhttp.Error(http.StatusInternalServerError, err)
	}
	w.Write(js)
	return nil
}

func stringRender(w http.ResponseWriter, response *proxy.Response) error {
	w.Header().Set("Content-Type", "text/plain")
	if response == nil {
		w.Write([]byte{})
		return nil
	}
	d, ok := response.Data["content"]
	if !ok {
		w.Write([]byte{})
		return nil
	}
	msg, ok := d.(string)
	if !ok {
		w.Write([]byte{})
		return nil
	}
	w.Write([]byte(msg))
	return nil
}

func noopRender(w http.ResponseWriter, response *proxy.Response) error {
	if response == nil {
		http.Error(w, "", http.StatusInternalServerError)
		return caddyhttp.Error(http.StatusInternalServerError, errors.New("empty response"))
	}

	for k, vs := range response.Metadata.Headers {
		for _, v := range vs {
			w.Header().Add(k, v)
		}
	}
	if response.Metadata.StatusCode != 0 {
		w.WriteHeader(response.Metadata.StatusCode)
	}

	if response.Io == nil {
		return nil
	}
	io.Copy(w, response.Io)
	return nil
}
