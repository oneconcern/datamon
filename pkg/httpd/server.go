package httpd

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/go-openapi/runtime/flagext"
	"github.com/go-openapi/swag"
	flag "github.com/spf13/pflag"
	"golang.org/x/net/netutil"
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
	maxHeaderSize    flagext.ByteSize

	socketPath string

	host         string
	port         int
	listenLimit  int
	keepAlive    time.Duration
	readTimeout  time.Duration
	writeTimeout time.Duration

	tlsHost           string
	tlsPort           int
	tlsListenLimit    int
	tlsKeepAlive      time.Duration
	tlsReadTimeout    time.Duration
	tlsWriteTimeout   time.Duration
	tlsCertificate    string
	tlsCertificateKey string
	tlsCACertificate  string
)

func init() {
	maxHeaderSize = flagext.ByteSize(1000000)
	host = stringEnvOverride(host, "localhost", "HOST")
	port = intEnvOverride(port, 0, "PORT")
	tlsHost = stringEnvOverride(tlsHost, host, "TLS_HOST", "HOST")
	tlsPort = intEnvOverride(tlsPort, 0, "TLS_PORT")
	tlsCertificate = stringEnvOverride(tlsCertificate, "", "TLS_CERTIFICATE")
	tlsCertificateKey = stringEnvOverride(tlsCertificateKey, "", "TLS_PRIVATE_KEY")
	tlsCACertificate = stringEnvOverride(tlsCACertificate, "", "TLS_CA_CERTIFICATE")
}

// RegisterFlags to the specified pflag set
func RegisterFlags(fs *flag.FlagSet) {
	fs.StringSliceVar(&enabledListeners, "scheme", defaultSchemes, "the listeners to enable, this can be repeated and defaults to the schemes in the swagger spec")
	fs.DurationVar(&cleanupTimout, "cleanup-timeout", 10*time.Second, "grace period for which to wait before shutting down the server")
	fs.Var(&maxHeaderSize, "max-header-size", "controls the maximum number of bytes the server will read parsing the request header's keys and values, including the request line. It does not limit the size of the request body")

	fs.StringVar(&socketPath, "socket-path", "/var/run/trumpetd.sock", "the unix socket to listen on")

	fs.StringVar(&host, "host", host, "the IP to listen on")
	fs.IntVar(&port, "port", port, "the port to listen on for insecure connections, defaults to a random value")
	fs.IntVar(&listenLimit, "listen-limit", 0, "limit the number of outstanding requests")
	fs.DurationVar(&keepAlive, "keep-alive", 3*time.Minute, "sets the TCP keep-alive timeouts on accepted connections. It prunes dead TCP connections ( e.g. closing laptop mid-download)")
	fs.DurationVar(&readTimeout, "read-timeout", 30*time.Second, "maximum duration before timing out read of the request")
	fs.DurationVar(&writeTimeout, "write-timeout", 30*time.Second, "maximum duration before timing out write of the response")

	fs.StringVar(&tlsHost, "tls-host", tlsHost, "the IP to listen on")
	fs.IntVar(&tlsPort, "tls-port", tlsPort, "the port to listen on for secure connections, defaults to a random value")
	fs.StringVar(&tlsCertificate, "tls-certificate", tlsCertificate, "the certificate to use for secure connections")
	fs.StringVar(&tlsCertificateKey, "tls-key", tlsCertificateKey, "the private key to use for secure conections")
	fs.StringVar(&tlsCACertificate, "tls-ca", tlsCACertificate, "the certificate authority file to be used with mutual tls auth")
	fs.IntVar(&tlsListenLimit, "tls-listen-limit", 0, "limit the number of outstanding requests")
	fs.DurationVar(&tlsKeepAlive, "tls-keep-alive", 3*time.Minute, "sets the TCP keep-alive timeouts on accepted connections. It prunes dead TCP connections ( e.g. closing laptop mid-download)")
	fs.DurationVar(&tlsReadTimeout, "tls-read-timeout", 30*time.Second, "maximum duration before timing out read of the request")
	fs.DurationVar(&tlsWriteTimeout, "tls-write-timeout", 30*time.Second, "maximum duration before timing out write of the response")
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

// New creates a new api patmos server but does not configure it
func New(opts ...Option) Server {
	s := new(defaultServer)

	s.EnabledListeners = enabledListeners
	s.CleanupTimeout = cleanupTimout
	s.MaxHeaderSize = maxHeaderSize
	s.SocketPath = socketPath
	s.Host = host
	s.Port = port
	s.ListenLimit = listenLimit
	s.KeepAlive = keepAlive
	s.ReadTimeout = readTimeout
	s.WriteTimeout = writeTimeout
	s.TLSHost = tlsHost
	s.TLSPort = tlsPort
	s.TLSCertificate = tlsCertificate
	s.TLSCertificateKey = tlsCertificateKey
	s.TLSCACertificate = tlsCACertificate
	s.TLSListenLimit = tlsListenLimit
	s.TLSKeepAlive = tlsKeepAlive
	s.TLSReadTimeout = tlsReadTimeout
	s.TLSWriteTimeout = tlsWriteTimeout
	s.shutdown = make(chan struct{})
	s.interrupt = make(chan os.Signal, 1)
	s.logger = &stdLogger{}
	s.onShutdown = func() {}

	for _, apply := range opts {
		apply(s)
	}
	return s
}

// defaultServer for the patmos API
type defaultServer struct {
	EnabledListeners []string
	CleanupTimeout   time.Duration
	MaxHeaderSize    flagext.ByteSize

	SocketPath    string
	domainSocketL net.Listener

	Host         string
	Port         int
	ListenLimit  int
	KeepAlive    time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	httpServerL  net.Listener

	TLSHost           string
	TLSPort           int
	TLSCertificate    string
	TLSCertificateKey string
	TLSCACertificate  string
	TLSListenLimit    int
	TLSKeepAlive      time.Duration
	TLSReadTimeout    time.Duration
	TLSWriteTimeout   time.Duration
	httpsServerL      net.Listener

	handler      http.Handler
	hasListeners bool
	shutdown     chan struct{}
	shuttingDown int32
	interrupted  bool
	interrupt    chan os.Signal
	chanLock     sync.RWMutex
	callbacks    Hook
	logger       Logging
	onShutdown   func()
}

func (s *defaultServer) configureListener(hsrv *http.Server, scheme, addr string) {
	if s.callbacks != nil {
		s.callbacks.ConfigureListener(hsrv, scheme, addr)
	}
}

func (s *defaultServer) configureTLS(cfg *tls.Config) {
	if s.callbacks != nil {
		s.callbacks.ConfigureTLS(cfg)
	}
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
	if !s.hasListeners {
		if err = s.Listen(); err != nil {
			return err
		}
	}

	var wg sync.WaitGroup
	once := new(sync.Once)
	signalNotify(s.interrupt)
	go handleInterrupt(once, s)

	servers := []*http.Server{}
	wg.Add(1)
	go s.handleShutdown(&wg, &servers)

	if s.hasScheme(schemeUnix) {
		domainSocket := new(http.Server)
		domainSocket.MaxHeaderBytes = int(s.MaxHeaderSize)
		domainSocket.Handler = s.handler
		if int64(s.CleanupTimeout) > 0 {
			domainSocket.IdleTimeout = s.CleanupTimeout
		}

		s.configureListener(domainSocket, "unix", string(s.SocketPath))

		wg.Add(1)
		s.logger.Printf("Serving at unix://%s", s.SocketPath)
		go func(l net.Listener) {
			defer wg.Done()
			if derr := domainSocket.Serve(l); derr != nil && derr != http.ErrServerClosed {
				s.logger.Fatalf("%v", derr)
			}
			s.logger.Printf("Stopped serving at unix://%s", s.SocketPath)
		}(s.domainSocketL)
		servers = append(servers, domainSocket)
	}

	if s.hasScheme(schemeHTTP) {
		httpServer := new(http.Server)
		httpServer.MaxHeaderBytes = int(s.MaxHeaderSize)
		httpServer.ReadTimeout = s.ReadTimeout
		httpServer.WriteTimeout = s.WriteTimeout
		httpServer.SetKeepAlivesEnabled(int64(s.KeepAlive) > 0)
		if s.ListenLimit > 0 {
			s.httpServerL = netutil.LimitListener(s.httpServerL, s.ListenLimit)
		}

		if int64(s.CleanupTimeout) > 0 {
			httpServer.IdleTimeout = s.CleanupTimeout
		}

		httpServer.Handler = s.handler

		s.configureListener(httpServer, "http", s.httpServerL.Addr().String())

		wg.Add(1)
		s.logger.Printf("Serving at http://%s", s.httpServerL.Addr())
		go func(l net.Listener) {
			defer wg.Done()
			if herr := httpServer.Serve(l); herr != nil && herr != http.ErrServerClosed {
				s.logger.Fatalf("%v", herr)
			}
			s.logger.Printf("Stopped serving at http://%s", l.Addr())
		}(s.httpServerL)
		servers = append(servers, httpServer)
	}

	if s.hasScheme(schemeHTTPS) {
		httpsServer := new(http.Server)
		httpsServer.MaxHeaderBytes = int(s.MaxHeaderSize)
		httpsServer.ReadTimeout = s.TLSReadTimeout
		httpsServer.WriteTimeout = s.TLSWriteTimeout
		httpsServer.SetKeepAlivesEnabled(int64(s.TLSKeepAlive) > 0)
		if s.TLSListenLimit > 0 {
			s.httpsServerL = netutil.LimitListener(s.httpsServerL, s.TLSListenLimit)
		}
		if int64(s.CleanupTimeout) > 0 {
			httpsServer.IdleTimeout = s.CleanupTimeout
		}
		httpsServer.Handler = s.handler

		// Inspired by https://blog.bracebin.com/achieving-perfect-ssl-labs-score-with-go
		httpsServer.TLSConfig = &tls.Config{
			// Causes servers to use Go's default ciphersuite preferences,
			// which are tuned to avoid attacks. Does nothing on clients.
			PreferServerCipherSuites: true,
			// Only use curves which have assembly implementations
			// https://github.com/golang/go/tree/master/src/crypto/elliptic
			CurvePreferences: []tls.CurveID{tls.CurveP256},
			// Use modern tls mode https://wiki.mozilla.org/Security/Server_Side_TLS#Modern_compatibility
			NextProtos: []string{"http/1.1", "h2"},
			// https://www.owasp.org/index.php/Transport_Layer_Protection_Cheat_Sheet#Rule_-_Only_Support_Strong_Protocols
			MinVersion: tls.VersionTLS12,
			// These ciphersuites support Forward Secrecy: https://en.wikipedia.org/wiki/Forward_secrecy
			CipherSuites: []uint16{
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			},
		}

		if s.TLSCertificate != "" && s.TLSCertificateKey != "" {
			httpsServer.TLSConfig.Certificates = make([]tls.Certificate, 1)
			httpsServer.TLSConfig.Certificates[0], err = tls.LoadX509KeyPair(s.TLSCertificate, s.TLSCertificateKey)
		}

		if s.TLSCACertificate != "" {
			caCert, caCertErr := ioutil.ReadFile(s.TLSCACertificate)
			if caCertErr != nil {
				log.Fatal(caCertErr)
			}
			caCertPool := x509.NewCertPool()
			caCertPool.AppendCertsFromPEM(caCert)
			httpsServer.TLSConfig.ClientCAs = caCertPool
			httpsServer.TLSConfig.ClientAuth = tls.RequireAndVerifyClientCert
		}

		s.configureTLS(httpsServer.TLSConfig)
		httpsServer.TLSConfig.BuildNameToCertificate()

		if err != nil {
			return err
		}

		if len(httpsServer.TLSConfig.Certificates) == 0 {
			if s.TLSCertificate == "" {
				if s.TLSCertificateKey == "" {
					s.logger.Fatalf("the required flags `--tls-certificate` and `--tls-key` were not specified")
				}
				s.logger.Fatalf("the required flag `--tls-certificate` was not specified")
			}
			if s.TLSCertificateKey == "" {
				s.logger.Fatalf("the required flag `--tls-key` was not specified")
			}
		}

		s.configureListener(httpsServer, "https", s.httpsServerL.Addr().String())

		wg.Add(1)
		s.logger.Printf("Serving at https://%s", s.httpsServerL.Addr())
		go func(l net.Listener) {
			defer wg.Done()
			if terr := httpsServer.Serve(l); terr != nil && terr != http.ErrServerClosed {
				s.logger.Fatalf("%v", terr)
			}
			s.logger.Printf("Stopped serving at https://%s", l.Addr())
		}(tls.NewListener(s.httpsServerL, httpsServer.TLSConfig))

		servers = append(servers, httpsServer)
	}

	wg.Wait()
	return nil
}

// Listen creates the listeners for the server
func (s *defaultServer) Listen() error {
	if s.hasListeners { // already done this
		return nil
	}

	if s.hasScheme(schemeHTTPS) {
		// Use http host if https host wasn't defined
		if s.TLSHost == "" {
			s.TLSHost = s.Host
		}
		// Use http listen limit if https listen limit wasn't defined
		if s.TLSListenLimit == 0 {
			s.TLSListenLimit = s.ListenLimit
		}
		// Use http tcp keep alive if https tcp keep alive wasn't defined
		if int64(s.TLSKeepAlive) == 0 {
			s.TLSKeepAlive = s.KeepAlive
		}
		// Use http read timeout if https read timeout wasn't defined
		if int64(s.TLSReadTimeout) == 0 {
			s.TLSReadTimeout = s.ReadTimeout
		}
		// Use http write timeout if https write timeout wasn't defined
		if int64(s.TLSWriteTimeout) == 0 {
			s.TLSWriteTimeout = s.WriteTimeout
		}
	}

	if s.hasScheme(schemeUnix) {
		domSockListener, err := net.Listen("unix", string(s.SocketPath))
		if err != nil {
			return err
		}
		s.domainSocketL = domSockListener
	}

	if s.hasScheme(schemeHTTP) {
		listener, err := net.Listen("tcp", net.JoinHostPort(s.Host, strconv.Itoa(s.Port)))
		if err != nil {
			return err
		}

		h, p, err := swag.SplitHostPort(listener.Addr().String())
		if err != nil {
			return err
		}
		s.Host = h
		s.Port = p
		s.httpServerL = listener
	}

	if s.hasScheme(schemeHTTPS) {
		tlsListener, err := net.Listen("tcp", net.JoinHostPort(s.TLSHost, strconv.Itoa(s.TLSPort)))
		if err != nil {
			return err
		}

		sh, sp, err := swag.SplitHostPort(tlsListener.Addr().String())
		if err != nil {
			return err
		}
		s.TLSHost = sh
		s.TLSPort = sp
		s.httpsServerL = tlsListener
	}

	s.hasListeners = true
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
	if !s.hasListeners {
		if err := s.Listen(); err != nil {
			return nil, err
		}
	}
	return s.domainSocketL, nil
}

// HTTPListener returns the http listener
func (s *defaultServer) HTTPListener() (net.Listener, error) {
	if !s.hasListeners {
		if err := s.Listen(); err != nil {
			return nil, err
		}
	}
	return s.httpServerL, nil
}

// TLSListener returns the https listener
func (s *defaultServer) TLSListener() (net.Listener, error) {
	if !s.hasListeners {
		if err := s.Listen(); err != nil {
			return nil, err
		}
	}
	return s.httpsServerL, nil
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
