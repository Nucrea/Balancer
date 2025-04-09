package main

import (
	"balancer/internal/balancer"
	"balancer/internal/balancer/backend"
	"balancer/internal/metrics"
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/pprof"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/adaptor"
	"github.com/rs/zerolog"
)

func NewServer(logger zerolog.Logger, addrs ...string) *Server {
	meter := metrics.NewPrometheus("balancer_")

	balancer, err := balancer.NewBalancer(meter, logger, addrs...)
	if err != nil {
		log.Fatal(err)
	}
	return &Server{balancer, meter, logger}
}

type Server struct {
	balancer *balancer.Balancer
	meter    *metrics.Meter
	logger   zerolog.Logger
}

func (s *Server) Run(ctx context.Context, port uint16) {
	app := fiber.New()

	app.Get("/health", func(c fiber.Ctx) error {
		return c.SendStatus(200)
	})
	app.Get("/metrics", adaptor.HTTPHandler(s.meter.HttpHandler()))

	app.Get("/debug/pprof/", adaptor.HTTPHandlerFunc(pprof.Index))
	app.Get("/debug/pprof/mutex", adaptor.HTTPHandlerFunc(pprof.Index))
	app.Get("/debug/pprof/cmdline", adaptor.HTTPHandlerFunc(pprof.Cmdline))
	app.Get("/debug/pprof/profile", adaptor.HTTPHandlerFunc(pprof.Profile))
	app.Get("/debug/pprof/symbol", adaptor.HTTPHandlerFunc(pprof.Symbol))
	app.Get("/debug/pprof/trace", adaptor.HTTPHandlerFunc(pprof.Trace))

	app.All("*", s.handle)

	addr := fmt.Sprintf(":%d", port)
	s.logger.Log().Msgf("listening on %s\n", addr)

	// s.balancer.SetStrategy(strategy.TypeRoundRobin)
	go s.balancer.Routine(ctx)
	app.Listen(addr)
}

func (s *Server) handle(c fiber.Ctx) error {
	headers := map[string]string{}
	headers["Content-Type"] = c.Get("Content-Type")

	request := backend.Request{
		Method:  c.Method(),
		Path:    c.Path(),
		Body:    c.Body(),
		Headers: headers,
	}

	resp, err := s.balancer.Invoke(c.Context(), request)
	if err != nil {
		return c.SendStatus(http.StatusGatewayTimeout)
	}
	for k, v := range resp.Headers {
		c.Set(k, v)
	}

	return c.Status(resp.Status).Send(resp.Body)
}
