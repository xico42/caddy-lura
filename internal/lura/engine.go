package lura

import (
	"github.com/julienschmidt/httprouter"
	"github.com/luraproject/lura/v2/router/mux"
	"net/http"
	"net/textproto"
)

type HttpRouterEngine struct {
	router *httprouter.Router
}

func newHttpRouterEngine() mux.Engine {
	r := httprouter.New()
	return &HttpRouterEngine{
		router: r,
	}
}

func (e *HttpRouterEngine) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	e.router.ServeHTTP(rw, req)
}

func (e *HttpRouterEngine) Handle(pattern, method string, handler http.Handler) {
	e.router.Handler(method, pattern, handler)
}

var newRequest = mux.NewRequestBuilder(paramExtractor)

var endpointHandler = mux.CustomEndpointHandler(newRequest)

func paramExtractor(r *http.Request) map[string]string {
	p, ok := r.Context().Value(httprouter.ParamsKey).(httprouter.Params)
	if !ok {
		return nil
	}

	out := make(map[string]string, len(p))
	for _, v := range p {
		out[textproto.CanonicalMIMEHeaderKey(v.Key[:1])+v.Key[1:]] = v.Value
	}

	return out
}
