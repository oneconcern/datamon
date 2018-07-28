package httpd

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/oneconcern/pipelines/pkg/cli/cflags"
	flag "github.com/spf13/pflag"
)

const (
	schemeHTTP  = "http"
	schemeHTTPS = "https"
	schemeUnix  = "unix"
)

var defaultSchemes []string

func init() {
	defaultSchemes = []string{
		schemeHTTP,
	}
}

// Hook allows for hooking into the lifecycle of the server
type Hook interface {
	ConfigureTLS(*tls.Config)
	ConfigureListener(*http.Server, string, string)
}

var (
	enabledListeners []string
	cleanupTimout    time.Duration
	maxHeaderSize    cflags.ByteSize

	udsFlags  UnixSocketFlags
	httpFlags HTTPFlags
	tlsFlags  TLSFlags
)

func init() {
	maxHeaderSize = cflags.ByteSize(1000000)
	httpFlags.Host = stringEnvOverride(httpFlags.Host, "localhost", "HOST")
	httpFlags.Port = intEnvOverride(httpFlags.Port, 0, "PORT")
	tlsFlags.Host = stringEnvOverride(tlsFlags.Host, httpFlags.Host, "TLS_HOST", "HOST")
	tlsFlags.Port = intEnvOverride(tlsFlags.Port, 0, "TLS_PORT")
	tlsFlags.Certificate = stringEnvOverride(tlsFlags.Certificate, "", "TLS_CERTIFICATE")
	tlsFlags.CertificateKey = stringEnvOverride(tlsFlags.CertificateKey, "", "TLS_PRIVATE_KEY")
	tlsFlags.CACertificate = stringEnvOverride(tlsFlags.CACertificate, "", "TLS_CA_CERTIFICATE")
}

// RegisterFlags to the specified pflag set
func RegisterFlags(fs *flag.FlagSet) {
	fs.StringSliceVar(&enabledListeners, "scheme", defaultSchemes, "the listeners to enable, this can be repeated and defaults to the schemes in the swagger spec")
	fs.DurationVar(&cleanupTimout, "cleanup-timeout", 10*time.Second, "grace period for which to wait before shutting down the server")
	fs.Var(&maxHeaderSize, "max-header-size", "controls the maximum number of bytes the server will read parsing the request header's keys and values, including the request line. It does not limit the size of the request body")

	udsFlags.RegisterFlags(fs)
	httpFlags.RegisterFlags(fs)
	tlsFlags.RegisterFlags(fs)
}

func prefixer(prefix string) func(string) string {
	return func(value string) string {
		if prefix == "" {
			return value
		}
		return strings.Join([]string{prefix, value}, "-")
	}
}

func stringEnvOverride(orig string, def string, keys ...string) string {
	for _, k := range keys {
		if os.Getenv(k) != "" {
			return os.Getenv(k)
		}
	}
	if def != "" && orig == "" {
		return def
	}
	return orig
}

func intEnvOverride(orig int, def int, keys ...string) int {
	for _, k := range keys {
		if os.Getenv(k) != "" {
			v, err := strconv.Atoi(os.Getenv(k))
			if err != nil {
				fmt.Fprintln(os.Stderr, k, "is not a valid number")
				os.Exit(1)
			}
			return v
		}
	}
	if def != 0 && orig == 0 {
		return def
	}
	return orig
}

// Option for the server
type Option func(*defaultServer)

// Hooks allows for registering one or more hooks for the server to call during its lifecycle
func Hooks(hook Hook, extra ...Hook) Option {
	h := &compositeHook{
		hooks: append([]Hook{hook}, extra...),
	}
	return func(s *defaultServer) {
		s.callbacks = h
	}
}

type compositeHook struct {
	hooks []Hook
}

func (c *compositeHook) ConfigureTLS(cfg *tls.Config) {
	for _, h := range c.hooks {
		h.ConfigureTLS(cfg)
	}
}

func (c *compositeHook) ConfigureListener(s *http.Server, scheme, addr string) {
	for _, h := range c.hooks {
		h.ConfigureListener(s, scheme, addr)
	}
}

// HandlesRequestsWith handles the http requests to the server
func HandlesRequestsWith(h http.Handler) Option {
	return func(s *defaultServer) {
		s.handler = h
	}
}

// LogsWith provides a logger to the server
func LogsWith(l Logging) Option {
	return func(s *defaultServer) {
		s.logger = l
	}
}

// EnablesSchemes overrides the enabled schemes
func EnablesSchemes(schemes ...string) Option {
	return func(s *defaultServer) {
		s.EnabledListeners = schemes
	}
}

// OnShutdown runs the provided functions on shutdown
func OnShutdown(handlers ...func()) Option {
	return func(s *defaultServer) {
		if len(handlers) == 0 {
			return
		}
		s.onShutdown = func() {
			for _, run := range handlers {
				run()
			}
		}
	}
}

// WithListeners replaces the default listeners with the provided listeres
func WithListeners(listener ServerListener, extra ...ServerListener) Option {
	all := append([]ServerListener{listener}, extra...)
	return func(s *defaultServer) {
		s.listeners = all
	}
}

// WithExtaListeners appends the provided listeners to the default listeners
func WithExtraListeners(listener ServerListener, extra ...ServerListener) Option {
	all := append([]ServerListener{listener}, extra...)
	return func(s *defaultServer) {
		s.listeners = append(s.listeners, all...)
	}
}

// New creates a new api patmos server but does not configure it
func New(opts ...Option) Server {
	s := new(defaultServer)

	s.EnabledListeners = enabledListeners
	s.CleanupTimeout = cleanupTimout
	s.MaxHeaderSize = maxHeaderSize
	s.shutdown = make(chan struct{})
	s.interrupt = make(chan os.Signal, 1)
	s.logger = &stdLogger{}
	s.onShutdown = func() {}
	s.listeners = []ServerListener{&udsFlags, &httpFlags, &tlsFlags}

	for _, apply := range opts {
		apply(s)
	}
	return s
}

type ServerConfig struct {
	MaxHeaderSize  int
	Logger         Logging
	Handler        http.Handler
	Callbacks      Hook
	CleanupTimeout time.Duration
}

type ServerListener interface {
	Listener() (net.Listener, error)
	Serve(ServerConfig, *sync.WaitGroup) (*http.Server, error)
	Scheme() string
}

// defaultServer for the patmos API
type defaultServer struct {
	EnabledListeners []string
	CleanupTimeout   time.Duration
	MaxHeaderSize    cflags.ByteSize

	handler http.Handler

	shutdown     chan struct{}
	shuttingDown int32
	interrupted  bool
	interrupt    chan os.Signal
	callbacks    Hook
	logger       Logging
	onShutdown   func()
	listeners    []ServerListener
}

func (s *defaultServer) hasScheme(scheme string) bool {
	schemes := s.EnabledListeners
	if len(schemes) == 0 {
		schemes = defaultSchemes
	}

	for _, v := range schemes {
		if v == scheme {
			return true
		}
	}
	return false
}

// Serve the api
func (s *defaultServer) Serve() (err error) {
	if err := s.Listen(); err != nil {
		return err
	}

	var wg sync.WaitGroup
	once := new(sync.Once)
	signalNotify(s.interrupt)
	go handleInterrupt(once, s)

	servers := []*http.Server{}
	wg.Add(1)
	go s.handleShutdown(&wg, &servers)

	for _, server := range s.listeners {
		if !s.hasScheme(server.Scheme()) {
			continue
		}
		sc := ServerConfig{
			Callbacks:      s.callbacks,
			CleanupTimeout: s.CleanupTimeout,
			MaxHeaderSize:  int(s.MaxHeaderSize),
			Handler:        s.handler,
			Logger:         s.logger,
		}
		if hs, err := server.Serve(sc, &wg); err == nil {
			servers = append(servers, hs)
		} else {
			return err
		}
	}

	wg.Wait()
	return nil
}

// Listen creates the listeners for the server
func (s *defaultServer) Listen() error {
	for _, server := range s.listeners {
		if !s.hasScheme(server.Scheme()) {
			continue
		}
		_, err := server.Listener()
		if err != nil {
			return err
		}
	}
	return nil
}

// Shutdown server and clean up resources
func (s *defaultServer) Shutdown() error {
	if atomic.CompareAndSwapInt32(&s.shuttingDown, 0, 1) {
		close(s.shutdown)
	}
	return nil
}

func (s *defaultServer) handleShutdown(wg *sync.WaitGroup, serversPtr *[]*http.Server) {
	// wg.Done must occur last, after s.api.ServerShutdown()
	// (to preserve old behaviour)
	defer wg.Done()

	<-s.shutdown

	servers := *serversPtr

	ctx, cancel := context.WithTimeout(context.TODO(), 15*time.Second)
	defer cancel()

	shutdownChan := make(chan bool)
	for i := range servers {
		server := servers[i]
		go func() {
			var success bool
			defer func() {
				shutdownChan <- success
			}()
			if err := server.Shutdown(ctx); err != nil {
				// Error from closing listeners, or context timeout:
				s.logger.Printf("HTTP server Shutdown: %v", err)
			} else {
				success = true
			}
		}()
	}

	// Wait until all listeners have successfully shut down before calling ServerShutdown
	success := true
	for range servers {
		success = success && <-shutdownChan
	}
	if success {
		if s.onShutdown != nil {
			s.onShutdown()
		}
	}
}

// GetHandler returns a handler useful for testing
func (s *defaultServer) GetHandler() http.Handler {
	return s.handler
}

// UnixListener returns the domain socket listener
func (s *defaultServer) UnixListener() (net.Listener, error) {
	if !s.hasScheme(udsFlags.Scheme()) {
		return nil, nil
	}
	return udsFlags.Listener()
}

// HTTPListener returns the http listener
func (s *defaultServer) HTTPListener() (net.Listener, error) {
	if !s.hasScheme(httpFlags.Scheme()) {
		return nil, nil
	}
	return httpFlags.Listener()
}

// TLSListener returns the https listener
func (s *defaultServer) TLSListener() (net.Listener, error) {
	if !s.hasScheme(tlsFlags.Scheme()) {
		return nil, nil
	}
	return tlsFlags.Listener()
}

func handleInterrupt(once *sync.Once, s *defaultServer) {
	once.Do(func() {
		for range s.interrupt {
			if s.interrupted {
				continue
			}
			s.logger.Printf("Shutting down... ")
			s.interrupted = true
			if err := s.Shutdown(); err != nil {
				s.logger.Printf("[WARN] error during server shutdown: %v", err)
			}
		}
	})
}

func signalNotify(interrupt chan<- os.Signal) {
	signal.Notify(interrupt, syscall.SIGINT, syscall.SIGTERM)
}

// Server is the interface a server implements
type Server interface {
	GetHandler() http.Handler
	TLSListener() (net.Listener, error)
	HTTPListener() (net.Listener, error)
	UnixListener() (net.Listener, error)
	Listen() error
	Serve() error
	Shutdown() error
}

// Logging the logger interface for the server
type Logging interface {
	Printf(string, ...interface{})
	Fatalf(string, ...interface{})
}

type stdLogger struct {
}

func (s *stdLogger) Printf(format string, args ...interface{}) {
	log.Printf(format, args...)
}
func (s *stdLogger) Fatalf(format string, args ...interface{}) {
	log.Fatalf(format, args...)
}
