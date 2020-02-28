package influxdb

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"time"
)

// Option configures an exporter
type Option func(*Exporter)

// WithStore sets the influxdb store for this exporter
func WithStore(s Store) Option {
	return func(e *Exporter) {
		if s != nil {
			e.store = s
		}
	}
}

// WithErrorHandler sets an error handler for this exporter
func WithErrorHandler(h func(error)) Option {
	return func(e *Exporter) {
		if h != nil {
			e.errorHandler = h
		}
	}
}

// WithTags sets or	adds some tags to every record posted to the store
func WithTags(tags map[string]string) Option {
	return func(e *Exporter) {
		if len(tags) > 0 {
			if len(e.customTags) == 0 {
				e.customTags = tags
				return
			}
			for k, v := range tags {
				e.customTags[k] = v
			}
		}
	}
}

// StoreOption configures an influxdb client
type StoreOption func(*influxDB)

// WithDatabase sets the database to use
func WithDatabase(db string) StoreOption {
	return func(s *influxDB) {
		if db != "" {
			s.database = db
		}
	}
}

// WithAddr sets the influxdb server URL
func WithAddr(addr string) StoreOption {
	return func(s *influxDB) {
		if addr != "" {
			s.config.Addr = addr
		}
	}
}

// WithUser sets the database user to connect to an influxdb database
func WithUser(user string) StoreOption {
	return func(s *influxDB) {
		s.config.Username = user
	}
}

// WithPassword sets the database password to connect to an influxdb database
func WithPassword(pwd string) StoreOption {
	return func(s *influxDB) {
		s.config.Password = pwd
	}
}

// WithInsecureSkipVerify toggles TLS server certificate check by the client
func WithInsecureSkipVerify(skip bool) StoreOption {
	return func(s *influxDB) {
		s.config.InsecureSkipVerify = skip
	}
}

// WithTimeout sets write timeouts for the client
func WithTimeout(d time.Duration) StoreOption {
	return func(s *influxDB) {
		s.config.Timeout = d
	}
}

// WithTLSConfig sets TLS configuration for an https client
func WithTLSConfig(config *tls.Config) StoreOption {
	return func(s *influxDB) {
		s.config.TLSConfig = config
	}
}

// WithProxy configures a proxy for the http client
func WithProxy(proxy func(*http.Request) (*url.URL, error)) StoreOption {
	return func(s *influxDB) {
		s.config.Proxy = proxy
	}
}

// WithMapper specifies a name mapping function, which translates a measurement name and a set of tags into another
// one. This allows for converting measurement names into tags and reduce the number of time series handled by influxdb.
func WithMapper(mapper func(string, map[string]string) (string, map[string]string)) StoreOption {
	return func(s *influxDB) {
		s.mapper = mapper
	}
}

// WithNameAsTag is a helper which specifies a simple mapper converting a measurement name into a
// "metric" tag and the time series name is predefined.
func WithNameAsTag(timeseries string) StoreOption {
	return func(s *influxDB) {
		s.mapper = func(name string, tags map[string]string) (string, map[string]string) {
			tags["metric"] = name
			return timeseries, tags
		}
	}
}

// WithURL combines user, password and host address in one single URI notation (e.g. http://user:password@host:port)
func WithURL(r string) StoreOption {
	return func(s *influxDB) {
		if r != "" {
			u, _ := url.Parse(r)
			if u.User != nil {
				s.config.Username = u.User.Username()
				if pwd, ok := u.User.Password(); ok {
					s.config.Password = pwd
				}
			}
			s.config.Addr = u.Scheme + "://" + u.Host
		}
	}
}
