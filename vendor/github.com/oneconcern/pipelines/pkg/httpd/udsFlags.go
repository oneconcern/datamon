package httpd

import (
	"net"
	"net/http"
	"path/filepath"
	"sync"

	"github.com/kardianos/osext"

	flag "github.com/spf13/pflag"
)

type UnixSocketFlags struct {
	Path       string
	FlagName   string
	Prefix     string
	listenOnce sync.Once
	listener   net.Listener
}

func (u *UnixSocketFlags) RegisterFlags(fs *flag.FlagSet) {
	prefixed := prefixer(u.Prefix)
	if u.Path == "" {
		if nm, err := osext.Executable(); err == nil {
			u.Path = filepath.Join("/var/run", nm+".sock")
		}
	}
	if u.FlagName == "" {
		u.FlagName = prefixed("socket-path")
	}

	fs.StringVar(&u.Path, u.FlagName, u.Path, "the unix socket to listen on")
}

func (u *UnixSocketFlags) Listener() (net.Listener, error) {
	var err error
	u.listenOnce.Do(func() {
		l, e := net.Listen("unix", u.Path)
		if e != nil {
			u.listener = nil
			err = e
			return
		}
		u.listener = l
		err = nil
	})
	if err != nil { // retry on error
		u.listenOnce = sync.Once{}
	}
	return u.listener, err
}

func (u *UnixSocketFlags) Serve(s ServerConfig, wg *sync.WaitGroup) (*http.Server, error) {
	listener, err := u.Listener()
	if err != nil {
		return nil, err
	}

	domainSocket := new(http.Server)
	domainSocket.MaxHeaderBytes = int(s.MaxHeaderSize)
	domainSocket.Handler = s.Handler
	if int64(s.CleanupTimeout) > 0 {
		domainSocket.IdleTimeout = s.CleanupTimeout
	}

	if s.Callbacks != nil {
		s.Callbacks.ConfigureListener(domainSocket, u.Scheme(), u.Path)
	}

	wg.Add(1)
	s.Logger.Printf("Serving at unix://%s", u.Path)

	go func(l net.Listener) {
		defer wg.Done()
		if derr := domainSocket.Serve(l); derr != nil && derr != http.ErrServerClosed {
			s.Logger.Fatalf("%v", derr)
		}
		s.Logger.Printf("Stopped serving at unix://%s", u.Path)
	}(listener)

	return domainSocket, nil
}

func (u *UnixSocketFlags) Scheme() string {
	return schemeUnix
}
