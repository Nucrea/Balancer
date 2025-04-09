package main

import (
	"balancer/internal/balancer"
	"balancer/internal/balancer/backend"
	"balancer/internal/metrics"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/pprof"

	"github.com/rs/zerolog"
)

func NewServer(logger zerolog.Logger, addrs ...string) *Server {
	meter := metrics.NewPrometheus("balancer_")

	balancer, err := balancer.NewBalancer(meter, logger, addrs...)
	profiler := NewProfiler()
	if err != nil {
		log.Fatal(err)
	}
	return &Server{balancer, profiler, meter, logger}
}

type Server struct {
	balancer *balancer.Balancer
	profiler *Profiler
	meter    *metrics.Meter
	logger   zerolog.Logger
}

func (s *Server) Run(ctx context.Context, port uint16) {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/mutex", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	mux.Handle("/metrics", s.meter.HttpHandler())
	mux.HandleFunc("/", s.handle)

	addr := fmt.Sprintf(":%d", port)
	s.logger.Log().Msgf("listening on %s\n", addr)

	// s.balancer.SetStrategy(strategy.TypeRoundRobin)
	go s.balancer.Routine(ctx)
	http.ListenAndServe(addr, mux)
}

func (s *Server) handle(w http.ResponseWriter, r *http.Request) {
	headers := map[string]string{}
	headers["Content-Type"] = r.Header.Get("Content-Type")

	body, _ := io.ReadAll(r.Body)
	request := backend.Request{
		Method:  r.Method,
		Path:    r.URL.Path,
		Body:    body,
		Headers: headers,
	}

	resp, err := s.balancer.Invoke(r.Context(), request)
	if err != nil {
		w.WriteHeader(http.StatusGatewayTimeout)
		return
	}
	for k, v := range resp.Headers {
		w.Header().Add(k, v)
	}

	w.WriteHeader(resp.Status)
	w.Write(resp.Body)
}

func (s *Server) ProfileOn(w http.ResponseWriter, r *http.Request) {
	status, body := 200, []byte(nil)

	fileName, err := s.profiler.Start(context.Background())
	if err != nil {
		status, body = 400, []byte(fmt.Sprintf(`{"status": "error, "message": "%s"}`, err.Error()))
	} else {
		status, body = 200, []byte(fmt.Sprintf(`{"status": "ok", "file": "%s"}`, fileName))
	}

	w.WriteHeader(status)
	w.Write(body)
}

func (s *Server) ProfileOff(w http.ResponseWriter, r *http.Request) {
	status, body := 200, []byte(nil)

	fileName, err := s.profiler.Stop()
	if err != nil {
		status, body = 400, []byte(fmt.Sprintf(`{"status": "error, "message": "%s"}`, err.Error()))
	} else {
		status, body = 200, []byte(fmt.Sprintf(`{"status": "ok", "file": "%s"}`, fileName))
	}

	w.WriteHeader(status)
	w.Write(body)
}
