package grpc

import (
	"context"

	"github.com/afoninsky/verdite/config"
	"github.com/afoninsky/verdite/proto"
	"google.golang.org/grpc"
)

// Plugin ...
type Plugin struct {
	client proto.InterceptorClient
}

// TODO: handle things like reconnect/health checks etc

// New ...
func New(name string, cfg config.Interceptor) (Plugin, error) {
	s := Plugin{}
	conn, err := grpc.Dial(cfg.GRPC.Address, grpc.WithInsecure())
	if err != nil {
		return s, err
	}
	s.client = proto.NewInterceptorClient(conn)

	return s, nil
}

// OnRequest ...
func (s Plugin) OnRequest(ctx context.Context, in *proto.OnRequestInput) (*proto.OnRequestOutput, error) {
	return s.client.OnRequest(ctx, in)
}
