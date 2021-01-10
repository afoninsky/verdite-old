package dummy

import (
	"context"

	"github.com/afoninsky/verdite/proto"
)

type Plugin struct{}

func (s *Plugin) OnRequest(ctx context.Context, in *proto.OnRequestInput) (*proto.OnRequestOutput, error) {
	// log.Printf("Receive message body from client: %s", in.Body)
	// return &Message{Body: "Hello From the Server!"}, nil
	res := proto.OnRequestOutput{
		Action: proto.OnRequestOutput_NONE,
	}
	return &res, nil
}
