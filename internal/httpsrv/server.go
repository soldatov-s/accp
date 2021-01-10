package httpsrv

import (
	"context"
	"net/http"
	"time"
)

type Server struct {
	*http.Server
}

func NewHTTPServer(addr string, hndl http.Handler) *Server {
	srv := &Server{
		Server: &http.Server{
			Addr:           addr,
			ReadTimeout:    10 * time.Second,
			WriteTimeout:   10 * time.Second,
			MaxHeaderBytes: 1 << 20,
		}}

	srv.Handler = hndl

	return srv
}

func (srv *Server) Shutdown() error {
	return srv.Server.Shutdown(context.Background())
}
