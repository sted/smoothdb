package main

import (
	"green/green-ds/api"
	"green/green-ds/database"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func InitHTTPServer(addr string, dbe *database.DBEngine) *http.Server {
	router := gin.Default()

	root := router.Group("/")

	admin := api.InitAdminRouter(root, dbe, AdminOnly())
	api.InitSourcesRouter(root, Authenticated())

	api.InitTestRouter(root, dbe)
	admin.Use(AdminOnly())

	server := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	return server
}
