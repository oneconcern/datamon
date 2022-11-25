package core

import (
	"github.com/oneconcern/datamon/pkg/dlogger"
	"go.uber.org/zap"
)

type (
	// PurgeOption modifies the behavior of the purge operations.
	PurgeOption func(*purgeOptions)

	purgeOptions struct {
		force          bool
		localStorePath string
		l              *zap.Logger
	}
)

func WithPurgeForce(enabled bool) PurgeOption {
	return func(o *purgeOptions) {
		o.force = true
	}
}

func WithPurgeLocalStore(pth string) PurgeOption {
	return func(o *purgeOptions) {
		o.localStorePath = pth
	}
}

func WithPurgeLogger(zlg *zap.Logger) PurgeOption {
	return func(o *purgeOptions) {
		o.l = zlg
	}
}

func defaultPurgeOptions(opts []PurgeOption) *purgeOptions {
	o := &purgeOptions{
		localStorePath: ".datamon-index",
		l:              dlogger.MustGetLogger("info"),
	}

	for _, apply := range opts {
		apply(o)
	}

	return o
}
