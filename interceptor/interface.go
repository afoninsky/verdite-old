package interceptor

import (
	"context"
	"fmt"

	"github.com/afoninsky/verdite/config"
	"github.com/afoninsky/verdite/interceptor/forward"
	"github.com/afoninsky/verdite/interceptor/grpc"
	"github.com/afoninsky/verdite/interceptor/response"
	"github.com/afoninsky/verdite/proto"
)

// Interceptor describes interceptor plugin interface
type Interceptor interface {
	OnRequest(context.Context, *proto.OnRequestInput) (*proto.OnRequestOutput, error)
}

// New ...
func New(name string, cfg config.Interceptor) (Interceptor, error) {
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

	return t, err
}
