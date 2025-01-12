package proxy

import (
	"context"
	"github.com/go-chi/chi"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"github.com/whereiskurt/tiogo/pkg/cache"
	"github.com/whereiskurt/tiogo/pkg/config"
	"github.com/whereiskurt/tiogo/pkg/metrics"
	"github.com/whereiskurt/tiogo/pkg/proxy/middleware"
	"github.com/whereiskurt/tiogo/pkg/tenable"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

// Server is built on go-chi
type Server struct {
	Context           context.Context
	Router            *chi.Mux
	HTTP              *http.Server
	Finished          context.CancelFunc
	ServiceBaseURL    string
	DiskCache         *cache.Disk
	Log               *log.Logger
	CacheFolder       string
	ListenPort        string
	Metrics           *metrics.Metrics
	MetricsListenPort string
}

// NewServer configs the HTTP, router, context, log and a DB to mock the ACME HTTP API
func NewServer(config *config.Config, metrics *metrics.Metrics, serverLog *log.Logger) (server Server) {
	if config == nil {
		log.Fatalf("error: config cannot be nil value.")
	}

	server.ServiceBaseURL = config.Server.ServiceBaseURL
	server.Log = serverLog
	server.ListenPort = config.Server.ListenPort
	server.CacheFolder = config.Server.CacheFolder
	server.MetricsListenPort = config.Server.MetricsListenPort

	if config.Server.CacheResponse {
		server.EnableCache(config.Server.CacheFolder, config.Server.CacheKey)
	}

	server.Context = config.Context
	server.Router = chi.NewRouter()
	server.HTTP = &http.Server{
		Addr:         ":" + server.ListenPort, // TODO: Take this as parameter
		Handler:      server.Router,
		IdleTimeout:  time.Duration(2 * time.Second),
		ReadTimeout:  time.Duration(2 * time.Second),
		WriteTimeout: time.Duration(2 * time.Second),
		// Ian Kent recommends these timeouts be set:
		//   https://www.youtube.com/watch?v=YF1qSfkDGAQ&t=333s
	}
	server.Metrics = metrics
	return
}
func (s *Server) ListenAndServeMetrics() {

	s.Log.Infof("Starting metrics server...")
	// Start the /metrics server
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		_ = http.ListenAndServe(":"+s.MetricsListenPort, nil)
	}()

}

// ListenAndServe will attempt to bind and provide HTTP service. It's hooked for signals and smooth Shutdown.
func (s *Server) ListenAndServe() {
	s.Log.Infof("Starting Tenable.io proxy server...")
	s.hookShutdownSignal()

	// Start the HTTP server
	go func() {
		err := s.HTTP.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			s.Log.Errorf("error serving: %+v", err)
			return
		}
		s.Log.Infof("Proxy server was signaled to shutdown..")

		s.Finished()
	}()

	select {
	case <-s.Context.Done():
		s.Log.Infof("Proxy server Context.Done() to shutdown")
	}

	return
}

func (s *Server) hookShutdownSignal() {
	stop := make(chan os.Signal)

	signal.Notify(stop, syscall.SIGTERM)
	signal.Notify(stop, syscall.SIGINT)

	s.Context, s.Finished = context.WithCancel(s.Context)
	go func() {
		sig := <-stop
		s.Log.Infof("termination signal '%s' received for server", sig)
		s.Finished()
	}()

	return
}

// EnableCache will create a new Disk Cache for all request.
func (s *Server) EnableCache(cacheFolder string, cryptoKey string) {
	var useCrypto = false
	if cryptoKey != "" {
		useCrypto = true
	}
	s.DiskCache = cache.NewDisk(cacheFolder, cryptoKey, useCrypto)
	return
}

func (s *Server) cacheClear(r *http.Request, endPoint tenable.EndPointType, service metrics.EndPointType) {
	if s.DiskCache == nil {
		return
	}
	if s.Metrics != nil {
		s.Metrics.CacheInc(service, metrics.Methods.Cache.Invalidate)
	}

	filename, _ := tenable.ToCacheFilename(endPoint, middleware.ContextMap(r))
	filename = filepath.Join(".", s.DiskCache.CacheFolder, filename)

	s.DiskCache.Clear(filename)
}
func (s *Server) cacheStore(w http.ResponseWriter, r *http.Request, bb []byte, endPoint tenable.EndPointType, service metrics.EndPointType) {
	if s.DiskCache == nil {
		return
	}
	// Metrics!
	if s.Metrics != nil {
		s.Metrics.CacheInc(service, metrics.Methods.Cache.Store)
	}

	filename, _ := tenable.ToCacheFilename(endPoint, middleware.ContextMap(r))
	prettyCache := middleware.NewPrettyPrint(w).Prettify(bb)

	_ = s.DiskCache.Store(filename, prettyCache)

}
func (s *Server) cacheFetch(r *http.Request, endPoint tenable.EndPointType, service metrics.EndPointType) (bb []byte, err error) {
	if s.DiskCache == nil {
		return
	}

	filename, _ := tenable.ToCacheFilename(endPoint, middleware.ContextMap(r))

	bb, err = s.DiskCache.Fetch(filename)

	if err == nil && len(bb) > 0 && s.Metrics != nil {
		s.Metrics.CacheInc(service, metrics.Methods.Cache.Hit)
	} else {
		s.Metrics.CacheInc(service, metrics.Methods.Cache.Miss)
	}

	return
}
