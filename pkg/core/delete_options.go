package core

type (
	DeleteOption func(*deleteOptions)

	deleteOptions struct {
		skipCheckRepo   bool
		skipDeleteLabel bool
	}
)

func deleteOptionsWithDefaults(opts []DeleteOption) *deleteOptions {
	o := &deleteOptions{}

	for _, apply := range opts {
		apply(o)
	}

	return o
}

func WithDeleteSkipCheckRepo(skip bool) DeleteOption {
	return func(o *deleteOptions) {
		o.skipCheckRepo = skip
	}
}

func WithDeleteSkipDeleteLabel(skip bool) DeleteOption {
	return func(o *deleteOptions) {
		o.skipDeleteLabel = skip
	}
}
