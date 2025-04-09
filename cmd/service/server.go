package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync/atomic"
	"time"
)

func NewServer() *Server {
	src := rand.NewSource(time.Now().UnixMicro())
	rand := rand.New(src)
	return &Server{rand, &atomic.Uint32{}}
}

type Server struct {
	rand    *rand.Rand
	counter *atomic.Uint32
}

func (s *Server) Run(ctx context.Context, port uint16) {
	protocols := &http.Protocols{}
	protocols.SetUnencryptedHTTP2(true)

	addr := fmt.Sprintf(":%d", port)
	log.Printf("listening on %s\n", addr)

	srv := http.Server{
		Addr:      addr,
		Handler:   s.router(),
		Protocols: protocols,
	}
	srv.ListenAndServe()
}

func (s *Server) router() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.getHealth)
	mux.HandleFunc("GET /count", s.getCount)
	return mux
}

func (s *Server) getHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
}

func (s *Server) getCount(w http.ResponseWriter, r *http.Request) {
	delay := time.Duration(1+s.rand.Int()%100) * time.Millisecond
	time.Sleep(delay)

	count := s.counter.Add(1)
	body := fmt.Sprintf(`{"count": "%d"}`, count)

	w.WriteHeader(200)
	w.Header().Add("content-type", "application/json")
	w.Write([]byte(body))
}
