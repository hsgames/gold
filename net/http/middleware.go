package http

import (
	"github.com/hsgames/gold/log"
	"github.com/hsgames/gold/safe"
	"net/http"
)

type Middleware func(Handler) Handler

func getChainUnaryHandler(ms []Middleware, curr int, finalHandler Handler) Handler {
	if curr == len(ms)-1 {
		return finalHandler
	}
	return func(w ResponseWriter, r *Request) {
		ms[curr+1](getChainUnaryHandler(ms, curr+1, finalHandler))(w, r)
	}
}

func NewRecoverMiddleware(logger log.Logger) Middleware {
	return func(h Handler) Handler {
		return func(w ResponseWriter, r *Request) {
			var err error
			defer func() {
				if err != nil {
					if logger != nil {
						logger.Error("http: recover middleware err:%+v", err)
					}
					w.WriteHeader(http.StatusInternalServerError)
				}
			}()
			defer safe.RecoverError(&err)
			h(w, r)
		}
	}
}
