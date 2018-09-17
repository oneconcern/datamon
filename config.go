package trumpet

import (
	"github.com/oneconcern/pipelines/pkg/log"
	opentracing "github.com/opentracing/opentracing-go"
)

type RepoConfig struct {
	Dir       string            `mapstructure:"dir" json:"dir,omitempty" yaml:"dir,omitempty"`
	Overrides map[string]string `mapstructure:"overrides" json:"overrides,omitempty" yaml:"overrides,omitempty"`
	Default   string            `mapstructure:"default" json:"default,omitempty" yaml:"default,omitempty"`
}

type Config struct {
	Repositories RepoConfig `mapstructure:"repositories" json:"repositories,omitempty" yaml:"repositories,omitempty"`
	Metadata     string     `mapstructure:"metadata" json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Blobs        string     `mapstructure:"blobs" json:"blobs,omitempty" yaml:"blobs,omitempty"`

	logger log.Factory
	tracer opentracing.Tracer
}

func (c *Config) Logger() log.Factory        { return c.logger }
func (c *Config) Tracer() opentracing.Tracer { return c.tracer }

func NewConfig(tracer opentracing.Tracer, logs log.Factory) *Config {
	return &Config{
		logger: logs,
		tracer: tracer,
	}
}
