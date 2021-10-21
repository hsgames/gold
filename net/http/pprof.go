package http

import (
	"github.com/hsgames/gold/log"
	"net/http/pprof"
)

func NewPProfServer(name, network, addr string, logger log.Logger, opt ...ServerOption) *Server {
	s := NewServer(name, network, addr,
		Handlers{
			"/debug/pprof/":        pprof.Index,
			"/debug/pprof/cmdline": pprof.Cmdline,
			"/debug/pprof/profile": pprof.Profile,
			"/debug/pprof/symbol":  pprof.Symbol,
			"/debug/pprof/trace":   pprof.Trace,
		},
		logger, opt...)
	return s
}
