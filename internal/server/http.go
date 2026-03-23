package server

import (
	"cpamgt/internal/middleware"
	"cpamgt/internal/router"
	"cpamgt/pkg/server/http"
	"cpamgt/web"
	nethttp "net/http"
	"strings"

	"github.com/gin-contrib/static"

	"github.com/gin-gonic/gin"
)

func NewHTTPServer(deps *router.RouterDeps) *http.Server {
	if deps.Conf.Env == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}
	s := http.NewServer(
		gin.Default(),
		deps.Logger,
		http.WithServerHost(deps.Conf.Http.Host),
		http.WithServerPort(deps.Conf.Http.Port),
	)

	fileSystem, err := static.EmbedFolder(web.Assets(), "dist")
	if err != nil {
		panic(err)
	}
	s.Use(static.Serve("/", fileSystem))
	s.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/v1") {
			c.JSON(nethttp.StatusNotFound, gin.H{"error": "API route not found"})
			return
		}
		c.FileFromFS("dist/index.html", nethttp.FS(web.Assets()))
	})

	s.Use(gin.Recovery())
	s.Use(
		middleware.CORSMiddleware(),
		middleware.LoggingMiddleware(deps.Logger),
		middleware.DurationMiddleWare(),
	)

	v1 := s.Group("/v1")

	deps.SetupRouter(v1)

	return s
}
