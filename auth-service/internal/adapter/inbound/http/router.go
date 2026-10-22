package http

import (
	"net/http"
	"net/http/pprof"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	"abet/auth-service/internal/adapter/inbound/http/handler"
	"abet/auth-service/internal/adapter/inbound/http/middleware"
)

type RouterConfig struct {
	EnablePprof   bool
	EnableSwagger bool
}

// SetupRouter constructs and configures the Gin engine with all routes and middlewares.
func SetupRouter(
	cfg RouterConfig,
	healthHandler *handler.HealthHandler,
	authHandler *handler.AuthHandler,
) (*gin.Engine, error) {
	r := gin.New()

	// Global Middlewares
	r.Use(gin.Recovery())
	r.Use(otelgin.Middleware("auth-service", otelgin.WithFilter(traceFilter)))
	r.Use(middleware.RequestLog())
	r.Use(middleware.PrometheusMetrics())

	// 1. Health & Observability Endpoints
	r.GET("/healthz", healthHandler.LivenessProbe)
	r.GET("/readyz", healthHandler.ReadinessProbe)

	// 2. Prometheus Metrics
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// 3. Go Profiling (pprof)
	if cfg.EnablePprof {
		registerPprof(r)
	}

	// 4. Swagger UI (OpenAPI specification)
	if cfg.EnableSwagger {
		registerSwagger(r)
	}

	// 5. Application API Routes (v1)
	v1 := r.Group("/api/v1")
	{
		authHandler.Register(v1)
	}

	// 6. Internal routes (docker network only; the gateway does not route /internal)
	internal := r.Group("/internal/v1")
	{
		authHandler.RegisterInternal(internal)
	}

	return r, nil
}

// traceFilter keeps probes, metrics and the Swagger UI out of the traces.
func traceFilter(r *http.Request) bool {
	p := r.URL.Path
	return p != "/healthz" && p != "/readyz" && p != "/metrics" && !strings.HasPrefix(p, "/swagger")
}

func registerPprof(r *gin.Engine) {
	debug := r.Group("/debug/pprof")
	{
		debug.GET("/", gin.WrapF(pprof.Index))
		debug.GET("/cmdline", gin.WrapF(pprof.Cmdline))
		debug.GET("/profile", gin.WrapF(pprof.Profile))
		debug.POST("/symbol", gin.WrapF(pprof.Symbol))
		debug.GET("/symbol", gin.WrapF(pprof.Symbol))
		debug.GET("/trace", gin.WrapF(pprof.Trace))
		debug.GET("/allocs", gin.WrapH(pprof.Handler("allocs")))
		debug.GET("/block", gin.WrapH(pprof.Handler("block")))
		debug.GET("/goroutine", gin.WrapH(pprof.Handler("goroutine")))
		debug.GET("/heap", gin.WrapH(pprof.Handler("heap")))
		debug.GET("/mutex", gin.WrapH(pprof.Handler("mutex")))
		debug.GET("/threadcreate", gin.WrapH(pprof.Handler("threadcreate")))
	}
}

func registerSwagger(r *gin.Engine) {
	r.GET("/swagger/*any", func(c *gin.Context) {
		if c.Request.URL.Path == "/swagger/spec.yaml" {
			c.File("./api/openapi/server/spec.yaml")
			return
		}

		swaggerUIHTML := `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>ABET Auth Service - Swagger UI</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.onload = () => {
      window.ui = SwaggerUIBundle({
        url: '/swagger/spec.yaml',
        dom_id: '#swagger-ui',
      });
    };
  </script>
</body>
</html>`
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, swaggerUIHTML)
	})
}
