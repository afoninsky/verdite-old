// Package forward implements http request interceptor with the following defaults:
// 	- request marked as allowed to proceed
// 	- request entities are added if specified
package forward

import (
	"context"

	"github.com/afoninsky/verdite/pkg/config"
	"github.com/afoninsky/verdite/pkg/proto"
)

// Plugin ...
type Plugin struct {
	cfg config.InterceptorRequest
}

// New ...
func New(name string, cfg config.RequestHandler) (Plugin, error) {
	return Plugin{
		cfg: cfg.Request,
	}, nil
}

// OnRequest ...
func (s Plugin) OnRequest(ctx context.Context, in *proto.OnRequestInput) (*proto.OnRequestOutput, error) {

	httpReq := proto.HTTPRequest{}
	if s.cfg.Method != "" {
		httpReq.Method = s.cfg.Method
	}
	if s.cfg.URL != "" {
		httpReq.URL = s.cfg.URL
	}
	if s.cfg.Body != "" {
		httpReq.Body = []byte(s.cfg.Body)
	}
	httpReq.Headers = map[string]string{}
	for k, v := range s.cfg.Headers {
		httpReq.Headers[k] = v
	}

	res := proto.OnRequestOutput{
		Action: proto.OnRequestOutput_FORWARD,
		Req:    &httpReq,
	}
	return &res, nil
}
