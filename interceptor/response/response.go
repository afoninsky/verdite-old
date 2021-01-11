// Package response implements http request interceptor with the following defaults:
// 	- request marked as allowed to proceed
// 	- request entities are added if specified
package response

import (
	"context"

	"github.com/afoninsky/verdite/config"
	"github.com/afoninsky/verdite/proto"
)

// Plugin ...
type Plugin struct {
	cfg config.InterceptorResponse
}

// New ...
func New(name string, cfg config.Interceptor) (Plugin, error) {
	return Plugin{
		cfg: cfg.Response,
	}, nil
}

// OnRequest ...
func (s Plugin) OnRequest(ctx context.Context, in *proto.OnRequestInput) (*proto.OnRequestOutput, error) {

	httpRes := proto.HTTPResponse{}
	if s.cfg.Status != 0 {
		httpRes.Status = uint32(s.cfg.Status)
	}
	if s.cfg.Body != "" {
		httpRes.Body = []byte(s.cfg.Body)
	}
	httpRes.Headers = map[string]string{}
	for k, v := range s.cfg.Headers {
		httpRes.Headers[k] = v
	}

	res := proto.OnRequestOutput{
		Action: proto.OnRequestOutput_RESPONSE,
		Res:    &httpRes,
	}
	return &res, nil
}
