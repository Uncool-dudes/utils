package httpserver

import (
	"context"
	"net"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/uncool-dudes/utils/errors"
)

var Domain = errors.NewDomain("httpserver")

type HttpServer struct {
	mu     sync.Mutex
	config Config
	router *chi.Mux
	srv    *http.Server
	ln     net.Listener
	log    *zap.Logger
}

func New(cfg Config, log *zap.Logger) *HttpServer {
	r := chi.NewRouter()
	return &HttpServer{
		config: cfg,
		router: r,
		log:    log,
		srv: &http.Server{
			Addr:         cfg.Addr,
			Handler:      r,
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
			IdleTimeout:  cfg.IdleTimeout,
		},
	}
}

// Router exposes the chi mux so callers can mount routes.
func (s *HttpServer) Router() *chi.Mux {
	return s.router
}

// Start binds synchronously so address-in-use errors surface immediately,
// then serves in a background goroutine.
func (s *HttpServer) Start() error {
	ln, err := net.Listen("tcp", s.config.Addr)
	if err != nil {
		return Domain.Mark(err, ErrStartFailed)
	}
	s.ln = ln
	go func() {
		if err := s.srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			s.log.Error("httpserver exited unexpectedly", zap.Error(err))
		}
	}()
	return nil
}

// Reload applies safe runtime changes. Addr changes require a restart.
func (s *HttpServer) Reload(cfg Config) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config = cfg
	s.srv.ReadTimeout = cfg.ReadTimeout
	s.srv.WriteTimeout = cfg.WriteTimeout
	s.srv.IdleTimeout = cfg.IdleTimeout
}

func (s *HttpServer) Shutdown(ctx context.Context) error {
	if ctx.Err() != nil {
		_ = s.ln.Close()
		return ctx.Err()
	}
	if err := s.srv.Shutdown(ctx); err != nil {
		_ = s.ln.Close()
		return Domain.Mark(err, ErrShutdown)
	}
	return nil
}
