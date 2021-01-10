package config

import (
	"os"
	"time"

	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v2"
)

// Config implements proxy configuration
type Config struct {
	Listen      string                    `yaml:"listen" validator:"hostname_port"`
	Route       map[string]Route          `yaml:"route"`
	Interceptor map[string]RequestHandler `yaml:"requestHandler"`
	Rules       map[string]Rule           `yaml:"rule"`
}

// Route describes http matching rules: https://github.com/gorilla/mux#matching-routes
type Route struct {
	Host       string   `yaml:"host" validator:"hostname"`
	Path       string   `yaml:"path" validator:"uri startswith=/"`
	PathPrefix string   `yaml:"pathPrefix" validator:"uri startswith=/"`
	Methods    []string `yaml:"methods" validator:"oneof=GET POST PUT DELETE PATCH DELETE"`
	Schemes    []string `yaml:"schemes" validator:"oneof=http https"`
	Headers    []string `yaml:"headers"`
	Queries    []string `yaml:"queries"`
}

// RequestHandler describes request interceptor
type RequestHandler struct {
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

// InterceptorRequest ...
type InterceptorRequest struct {
	Method  []string          `yaml:"method" validator:"oneof=GET POST PUT DELETE PATCH DELETE"`
	URL     string            `yaml:"url" validator:"url"`
	Headers map[string]string `yaml:"headers"`
	Body    string            `yaml:"body"`
}

// Rule specifies http(s) request routing table
type Rule struct {
	RequestHandlers []string `yaml:"matchers" validator:"alphanum"`
	// if not specified - body will not be parsed into string
	// usefull when you want to keep memory/cpu resources and speedup request, and middleware don't need it
	// disabled by default
	ParseRequestBody bool `yaml:"parseRequestBody"`
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