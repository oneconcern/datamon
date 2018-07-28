package httpd

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/go-openapi/swag"
	flag "github.com/spf13/pflag"
	"golang.org/x/net/netutil"
)

type TLSFlags struct {
	HTTPFlags
	Certificate    string
	CertificateKey string
	CACertificate  string
}

func (t *TLSFlags) RegisterFlags(fs *flag.FlagSet) {
	prefixed := prefixer(t.Prefix)
	fs.StringVar(&t.Host, prefixed("tls-host"), t.Host, "the IP to listen on")
	fs.IntVar(&t.Port, prefixed("tls-port"), t.Port, "the port to listen on for secure connections, defaults to a random value")
	fs.StringVar(&t.Certificate, prefixed("tls-certificate"), t.Certificate, "the certificate to use for secure connections")
	fs.StringVar(&t.CertificateKey, prefixed("tls-key"), t.CertificateKey, "the private key to use for secure conections")
	fs.StringVar(&t.CACertificate, prefixed("tls-ca"), t.CACertificate, "the certificate authority file to be used with mutual tls auth")
	fs.IntVar(&t.ListenLimit, prefixed("tls-listen-limit"), 0, "limit the number of outstanding requests")
	fs.DurationVar(&t.KeepAlive, prefixed("tls-keep-alive"), 3*time.Minute, "sets the TCP keep-alive timeouts on accepted connections. It prunes dead TCP connections ( e.g. closing laptop mid-download)")
	fs.DurationVar(&t.ReadTimeout, prefixed("tls-read-timeout"), 30*time.Second, "maximum duration before timing out read of the request")
	fs.DurationVar(&t.WriteTimeout, prefixed("tls-write-timeout"), 30*time.Second, "maximum duration before timing out write of the response")
}

func (t *TLSFlags) ApplyDefaults(values *HTTPFlags) {
	// Use http host if https host wasn't defined
	if t.Host == "" {
		t.Host = values.Host
	}
	// Use http listen limit if https listen limit wasn't defined
	if t.ListenLimit == 0 {
		t.ListenLimit = values.ListenLimit
	}
	// Use http tcp keep alive if https tcp keep alive wasn't defined
	if int64(t.KeepAlive) == 0 {
		t.KeepAlive = values.KeepAlive
	}
	// Use http read timeout if https read timeout wasn't defined
	if int64(t.ReadTimeout) == 0 {
		t.ReadTimeout = values.ReadTimeout
	}
	// Use http write timeout if https write timeout wasn't defined
	if int64(t.WriteTimeout) == 0 {
		t.WriteTimeout = values.WriteTimeout
	}
}

func (t *TLSFlags) Listener() (net.Listener, error) {
	var err error
	t.listenOnce.Do(func() {
		l, e := net.Listen("tcp", net.JoinHostPort(t.Host, strconv.Itoa(t.Port)))
		if e != nil {
			t.listener = nil
			err = e
			return
		}
		hh, p, e := swag.SplitHostPort(l.Addr().String())
		if e != nil {
			t.listener = nil
			err = e
			return
		}
		t.Host = hh
		t.Port = p
		if t.ListenLimit > 0 {
			l = netutil.LimitListener(l, t.ListenLimit)
		}
		t.listener = l
		err = nil
	})
	if err != nil { // retry on error
		t.listenOnce = sync.Once{}
	}
	return t.listener, err
}

func (t *TLSFlags) Serve(s ServerConfig, wg *sync.WaitGroup) (*http.Server, error) {
	listener, err := t.Listener()
	if err != nil {
		return nil, err
	}

	prefixed := prefixer(t.Prefix)

	httpsServer := new(http.Server)
	httpsServer.MaxHeaderBytes = int(s.MaxHeaderSize)
	httpsServer.ReadTimeout = t.ReadTimeout
	httpsServer.WriteTimeout = t.WriteTimeout
	httpsServer.SetKeepAlivesEnabled(int64(t.KeepAlive) > 0)
	if int64(s.CleanupTimeout) > 0 {
		httpsServer.IdleTimeout = s.CleanupTimeout
	}
	httpsServer.Handler = s.Handler

	// Inspired by https://blog.bracebin.com/achieving-perfect-ssl-labs-score-with-go
	httpsServer.TLSConfig = &tls.Config{
		// Causes servers to use Go's default ciphersuite preferences,
		// which are tuned to avoid attacks. Does nothing on clients.
		PreferServerCipherSuites: true,
		// Only use curves which have assembly implementations
		// https://github.com/golang/go/tree/master/src/crypto/elliptic
		CurvePreferences: []tls.CurveID{
			tls.CurveP256,
			tls.X25519,
		},
		NextProtos: []string{"http/1.1", "h2"},
		// https://www.owasp.org/index.php/Transport_Layer_Protection_Cheat_Sheet#Rule_-_Only_Support_Strong_Protocols
		MinVersion: tls.VersionTLS12,
		// Use modern tls mode https://wiki.mozilla.org/Security/Server_Side_TLS#Modern_compatibility
		// See security linter code: https://github.com/securego/gosec/blob/master/rules/tls_config.go#L11
		// These ciphersuites support Forward Secrecy: https://en.wikipedia.org/wiki/Forward_secrecy
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},
	}

	if t.Certificate != "" && t.CertificateKey != "" {
		httpsServer.TLSConfig.Certificates = make([]tls.Certificate, 1)
		httpsServer.TLSConfig.Certificates[0], err = tls.LoadX509KeyPair(t.Certificate, t.CertificateKey)
	}

	if t.CACertificate != "" {
		caCert, caCertErr := ioutil.ReadFile(t.CACertificate)
		if caCertErr != nil {
			return nil, caCertErr
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)
		httpsServer.TLSConfig.ClientCAs = caCertPool
		httpsServer.TLSConfig.ClientAuth = tls.RequireAndVerifyClientCert
	}

	if s.Callbacks != nil {
		s.Callbacks.ConfigureTLS(httpsServer.TLSConfig)
	}
	httpsServer.TLSConfig.BuildNameToCertificate()

	if err != nil {
		return nil, err
	}

	if len(httpsServer.TLSConfig.Certificates) == 0 {
		if t.Certificate == "" {
			if t.CertificateKey == "" {
				return nil, fmt.Errorf("the required flags %q and %q were not specified", prefixed("tls-certificate"), prefixed("tls-key"))
			}
			return nil, fmt.Errorf("the required flag %q was not specified", prefixed("tls-certificate"))
		}
		if t.CertificateKey == "" {
			return nil, fmt.Errorf("the required flag %q was not specified", prefixed("tls-key"))
		}
	}

	if s.Callbacks != nil {
		s.Callbacks.ConfigureListener(httpsServer, "https", listener.Addr().String())
	}

	wg.Add(1)
	s.Logger.Printf("Serving at https://%s", listener.Addr())
	go func(l net.Listener) {
		defer wg.Done()
		if terr := httpsServer.Serve(l); terr != nil && terr != http.ErrServerClosed {
			s.Logger.Fatalf("%v", terr)
		}
		s.Logger.Printf("Stopped serving at https://%s", l.Addr())
	}(tls.NewListener(listener, httpsServer.TLSConfig))
	return httpsServer, nil
}

func (t *TLSFlags) Scheme() string {
	return schemeHTTPS
}
