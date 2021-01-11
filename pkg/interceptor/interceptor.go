package interceptor

import (
	"context"
	"fmt"

	"github.com/afoninsky/verdite/pkg/config"
	"github.com/afoninsky/verdite/pkg/interceptor/forward"
	"github.com/afoninsky/verdite/pkg/interceptor/grpc"
	"github.com/afoninsky/verdite/pkg/interceptor/response"
	"github.com/afoninsky/verdite/pkg/proto"
)

type Interceptor interface {
	OnRequest(context.Context, *proto.OnRequestInput) (*proto.OnRequestOutput, error)
}

func New(name string, cfg config.RequestHandler) (*Interceptor, error) {
	var err error
	var t Interceptor

	switch cfg.Type {
	case "grpc":
		t, err = grpc.New(name, cfg)
	case "response":
		t, err = response.New(name, cfg)
	case "forward":
		t, err = forward.New(name, cfg)
	default:
		err = fmt.Errorf(`unsupported interceptor type: %s`, cfg.Type)
	}

	return &t, err
}
