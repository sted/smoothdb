package main

import (
	"context"
	"net/http"

	"github.com/sted/heligo"
	"github.com/sted/smoothdb/logging"
	"github.com/sted/smoothdb/plugins"
)

type examplePlugin struct {
	logger *logging.Logger
	router *heligo.Router
}

func (p *examplePlugin) Prepare(h plugins.Host) error {
	p.logger = h.GetLogger()
	p.logger.Info().Msg("examplePlugin: Preparing")
	p.router = h.GetRouter()
	p.router.Handle("GET", "/example", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		w.Write([]byte("Here we are"))
		return http.StatusOK, nil
	})
	return nil
}

func (p *examplePlugin) Run() error {
	p.logger.Info().Msg("examplePlugin: Started")
	return nil
}

var Plugin examplePlugin
