package config

import (
	"os"
	"time"

	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v2"
)

// Config implements proxy configuration
type Config struct {
	Listen       string                 `yaml:"listen" validator:"hostname_port"`
	Interceptors map[string]Interceptor `yaml:"interceptors"`
	Rules        []Rule                 `yaml:"rules"`
}

// Interceptor describes request interceptor
type Interceptor struct {
	Type     string              `yaml:"type" validator:"oneof=grpc response forward"`
	GRPC     InterceptorGRPC     `yaml:"grpc"`
	Response InterceptorResponse `yaml:"response"`
	Request  InterceptorRequest  `yaml:"request"`
}

// InterceptorGRPC sends request to external GRPC service before processing further
type InterceptorGRPC struct {
	Address string        `yaml:"address" validator:"required"`
	Timeout time.Duration `yaml:"timeout"`
}

// InterceptorResponse ...
type InterceptorResponse struct {
	Status  int               `yaml:"status" validator:"gte=100,lte=600"`
	Body    string            `yaml:"body"`
	Headers map[string]string `yaml:"headers"`
}

// Rule ...
type Rule struct {
	Match     Matcher  `yaml:"match"`
	OnRequest []string `yaml:"onRequest"`
	ParseBody bool     `yaml:"parseBoy"`
}

// Matcher describes http matching rules
type Matcher struct {
	Path   string `yaml:"path" validator:"required uri startswith=/"`
	Method string `yaml:"method" validator:"required oneof=GET POST PUT DELETE PATCH DELETE"`
	// TODO: support domains: https://github.com/julienschmidt/httprouter#multi-domain--sub-domains
}

// InterceptorRequest ...
type InterceptorRequest struct {
	Method  string            `yaml:"method" validator:"oneof=GET POST PUT DELETE PATCH DELETE"`
	URL     string            `yaml:"url" validator:"url"`
	Headers map[string]string `yaml:"headers"`
	Body    string            `yaml:"body"`
}

// New returns configruation instance
func New(cfgPath string) (*Config, error) {
	f, err := os.Open(cfgPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var cfg Config
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&cfg)
	if err != nil {
		return nil, err
	}
	if err := validator.New().Struct(cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
