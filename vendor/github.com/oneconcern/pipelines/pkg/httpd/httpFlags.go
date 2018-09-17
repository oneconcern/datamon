package httpd

import (
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/go-openapi/swag"
	flag "github.com/spf13/pflag"
	"golang.org/x/net/netutil"
)

type HTTPFlags struct {
	Prefix       string
	Host         string
	Port         int
	ListenLimit  int
	KeepAlive    time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration

	listenOnce sync.Once
	listener   net.Listener
}

func (h *HTTPFlags) RegisterFlags(fs *flag.FlagSet) {
	prefixed := prefixer(h.Prefix)
	fs.StringVar(&h.Host, prefixed("host"), h.Host, "the IP to listen on")
	fs.IntVar(&h.Port, prefixed("port"), h.Port, "the port to listen on for http connections, defaults to a random value")
	fs.IntVar(&h.ListenLimit, prefixed("listen-limit"), 0, "limit the number of outstanding requests")
	fs.DurationVar(&h.KeepAlive, prefixed("keep-alive"), 3*time.Minute, "sets the TCP keep-alive timeouts on accepted connections. It prunes dead TCP connections ( e.g. closing laptop mid-download)")
	fs.DurationVar(&h.ReadTimeout, prefixed("read-timeout"), 30*time.Second, "maximum duration before timing out read of the request")
	fs.DurationVar(&h.WriteTimeout, prefixed("write-timeout"), 30*time.Second, "maximum duration before timing out write of the response")
}

func (h *HTTPFlags) Listener() (net.Listener, error) {
	var err error
	h.listenOnce.Do(func() {
		l, e := net.Listen("tcp", net.JoinHostPort(h.Host, strconv.Itoa(h.Port)))
		if e != nil {
			h.listener = nil
			err = e
			return
		}
		hh, p, e := swag.SplitHostPort(l.Addr().String())
		if e != nil {
			h.listener = nil
			err = e
			return
		}
		h.Host = hh
		h.Port = p
		if h.ListenLimit > 0 {
			l = netutil.LimitListener(l, h.ListenLimit)
		}
		h.listener = l
		err = nil
	})
	if err != nil { // retry on error
		h.listenOnce = sync.Once{}
	}
	return h.listener, err
}

func (h *HTTPFlags) Serve(s ServerConfig, wg *sync.WaitGroup) (*http.Server, error) {
	listener, err := h.Listener()
	if err != nil {
		return nil, err
	}

	httpServer := new(http.Server)
	httpServer.MaxHeaderBytes = int(s.MaxHeaderSize)
	httpServer.ReadTimeout = h.ReadTimeout
	httpServer.WriteTimeout = h.WriteTimeout
	httpServer.SetKeepAlivesEnabled(int64(h.KeepAlive) > 0)
	if int64(s.CleanupTimeout) > 0 {
		httpServer.IdleTimeout = s.CleanupTimeout
	}

	httpServer.Handler = s.Handler
	if s.Callbacks != nil {
		s.Callbacks.ConfigureListener(httpServer, h.Scheme(), listener.Addr().String())
	}

	wg.Add(1)
	s.Logger.Printf("Serving at http://%s", listener.Addr())
	go func(l net.Listener) {
		defer wg.Done()
		if herr := httpServer.Serve(l); herr != nil && herr != http.ErrServerClosed {
			s.Logger.Fatalf("%v", herr)
		}
		s.Logger.Printf("Stopped serving at http://%s", l.Addr())
	}(listener)
	return httpServer, nil
}

func (h *HTTPFlags) Scheme() string {
	return schemeHTTP
}
