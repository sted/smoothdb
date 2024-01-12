package server

import (
	"context"
	"net/http"

	"github.com/rs/cors"
	"github.com/sted/heligo"
)

func (s *Server) initCORS() {

	router := s.GetRouter()

	cors := cors.New(cors.Options{
		AllowedOrigins: s.Config.CORSAllowedOrigins,
		AllowedMethods: []string{
			http.MethodHead,
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
		},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: s.Config.CORSAllowCredentials,
		MaxAge:           86400,
	})
	router.Use(heligo.AdaptMiddleware(cors.Handler))

	router.Handle("OPTIONS", "/*path", func(ctx context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		return http.StatusOK, nil
	})
}
