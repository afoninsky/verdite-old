package response

import (
	"context"

	"github.com/afoninsky/utilities/pkg/logger"
	"github.com/afoninsky/verdite/pkg/config"
	"github.com/afoninsky/verdite/pkg/proto"
)

type plugin struct{}

func New(name string, cfg config.RequestHandler) (*plugin, error) {
	return &plugin{}, nil
}

func (s *plugin) OnRequest(ctx context.Context, in *proto.OnRequestInput) (*proto.OnRequestOutput, error) {
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
