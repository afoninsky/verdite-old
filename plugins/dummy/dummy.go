package plugin

import (
	"context"
	"math/rand"
	"net/http"

	"github.com/afoninsky/utilities/pkg/logger"
	"github.com/afoninsky/verdite/proto"
)

type Plugin struct{}

func (s *Plugin) OnRequest(ctx context.Context, in *proto.OnRequestInput) (*proto.OnRequestOutput, error) {
	log := logger.New()
	var action proto.OnRequestOutput_Action
	var request proto.HTTPRequest
	var response proto.HTTPResponse

	switch rand.Intn(3) {
	// pass request without changes
	case 0:
		action = proto.OnRequestOutput_IGNORE
	// reject request with information about authentication
	case 1:
		action = proto.OnRequestOutput_REJECT
		response = proto.HTTPResponse{
			Status: http.StatusUnauthorized,
			Headers: map[string]string{
				"X-Verdite-Tested": "true",
			},
			Body: []byte("Authentication Required"),
		}
	// allow request adding check header
	case 2:
		action = proto.OnRequestOutput_UPDATE
		request = proto.HTTPRequest{
			Headers: map[string]string{
				"X-Verdite-Tested": "true",
			},
		}
	}
	log.WithField("action", action).Infoln(in.Req.URL)

	res := proto.OnRequestOutput{
		Action: action,
		Req:    &request,
		Res:    &response,
	}
	return &res, nil
}
