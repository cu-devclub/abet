package http

import (
	"net/http"
	"net/http/pprof"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	"abet/main-service/internal/adapter/inbound/http/handler"
	"abet/main-service/internal/adapter/inbound/http/middleware"
	"abet/main-service/internal/core/port"
)

// Registrar is implemented by every module handler: it mounts its routes on the given group.
type Registrar interface {
	Register(rg *gin.RouterGroup)
}

// RouterConfig carries the router-level settings and dependencies shared by all modules.
type RouterConfig struct {
	EnablePprof      bool
	EnableSwagger    bool
	InternalAPIToken string
	ActorLoader      port.ActorLoader
	// DevActAsEnabled turns on the cookie-based actor swap in the auth middleware (dev only, D28).
	DevActAsEnabled bool
}

// Routes lists the module handlers per route group. Adding a module = append its handler to the right slice in main.go.
type Routes struct {
	// API handlers are mounted under /api/v1 behind the X-User-Id authentication middleware.
	API []Registrar
	// Internal handlers are mounted under /internal/v1 behind the X-Internal-Token middleware.
	Internal []Registrar
}

// SetupRouter builds the Gin engine with the global middlewares, probes and all module routes.
func SetupRouter(cfg RouterConfig, health *handler.HealthHandler, routes Routes) *gin.Engine {
	r := gin.New()
	r.Use(middleware.Recovery())
	r.Use(otelgin.Middleware("main-service", otelgin.WithFilter(traceFilter)))
	r.Use(middleware.RequestLog())
	r.Use(middleware.PrometheusMetrics())

	r.GET("/healthz", health.LivenessProbe)
	r.GET("/readyz", health.ReadinessProbe)
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	if cfg.EnablePprof {
		registerPprof(r)
	}
	if cfg.EnableSwagger {
		registerSwagger(r)
	}

	api := r.Group("/api/v1", middleware.Authenticate(cfg.ActorLoader, cfg.DevActAsEnabled))
	for _, reg := range routes.API {
		reg.Register(api)
	}

	internal := r.Group("/internal/v1", middleware.InternalToken(cfg.InternalAPIToken))
	for _, reg := range routes.Internal {
		reg.Register(internal)
	}

	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
	})
	return r
}

// traceFilter keeps probes, metrics and the Swagger UI out of the traces.
func traceFilter(r *http.Request) bool {
	p := r.URL.Path
	return p != "/healthz" && p != "/readyz" && p != "/metrics" && !strings.HasPrefix(p, "/swagger")
}

func registerPprof(r *gin.Engine) {
	debug := r.Group("/debug/pprof")
	debug.GET("/", gin.WrapF(pprof.Index))
	debug.GET("/cmdline", gin.WrapF(pprof.Cmdline))
	debug.GET("/profile", gin.WrapF(pprof.Profile))
	debug.GET("/symbol", gin.WrapF(pprof.Symbol))
	debug.POST("/symbol", gin.WrapF(pprof.Symbol))
	debug.GET("/trace", gin.WrapF(pprof.Trace))
	for _, name := range []string{"allocs", "block", "goroutine", "heap", "mutex", "threadcreate"} {
		debug.GET("/"+name, gin.WrapH(pprof.Handler(name)))
	}
}

func registerSwagger(r *gin.Engine) {
	r.GET("/swagger/*any", func(c *gin.Context) {
		if c.Request.URL.Path == "/swagger/spec.yaml" {
			c.File("./api/openapi/server/spec.yaml")
			return
		}
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, swaggerUIHTML)
	})
}

const swaggerUIHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>ABET main-service - Swagger UI</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.onload = () => {
      window.ui = SwaggerUIBundle({ url: '/swagger/spec.yaml', dom_id: '#swagger-ui' });
    };
  </script>
</body>
</html>`
