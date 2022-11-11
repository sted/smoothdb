package server

import (
	"green/green-ds/api"
	"green/green-ds/database"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func (server *Server) initHTTPServer(dbe *database.DBEngine) {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	//router.Use(gin.Logger())

	root := router.Group("/")

	root.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "Green")
	})
	authMiddleware := server.authMiddleware()

	if server.Config.EnableAdmin {
		api.InitAdminRouter(root, dbe, authMiddleware)
	}
	api.InitSourcesRouter(root, authMiddleware)
	api.InitTestRouter(root, dbe)

	server.HTTP = &http.Server{
		Addr:         server.Config.Address,
		Handler:      router,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
	}
}
