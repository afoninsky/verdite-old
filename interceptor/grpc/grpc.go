package grpc

import (
	"context"

	"github.com/afoninsky/utilities/pkg/logger"
	"github.com/afoninsky/verdite/config"
	"github.com/afoninsky/verdite/proto"
)

// Plugin ...
type Plugin struct{}

// New ...
func New(name string, cfg config.Interceptor) (Plugin, error) {
	return Plugin{}, nil
}

// OnRequest ...
func (s Plugin) OnRequest(ctx context.Context, in *proto.OnRequestInput) (*proto.OnRequestOutput, error) {
	log := logger.New().
		WithField("method", in.Req.Method).
		WithField("url", in.Req.URL)

	for k, v := range in.Req.Headers {
		log = log.WithField(k, v)
	}

	log.Infoln("Request")

	res := proto.OnRequestOutput{
		Action: proto.OnRequestOutput_IGNORE,
	}
	return &res, nil
}
