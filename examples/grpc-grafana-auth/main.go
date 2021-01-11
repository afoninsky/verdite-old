package main

import (
	"context"
	"log"
	"net"
	"os"

	"github.com/afoninsky/verdite/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type plugin struct{}

func (s *plugin) OnRequest(ctx context.Context, in *proto.OnRequestInput) (*proto.OnRequestOutput, error) {
	user, ok := in.Req.Headers["X-Grafana-User"]

	// forbid access for anonymous user
	if !ok {
		log.Printf("User anonymous user access denied")
		return &proto.OnRequestOutput{
			Action: proto.OnRequestOutput_RESPONSE,
			Res: &proto.HTTPResponse{
				Status: 401,
				Body:   []byte("Anonymous user access denied"),
			},
		}, nil
	}

	log.Printf("User authenticated: %s\n", user)

	// allow access for authenticated user
	return &proto.OnRequestOutput{
		Action: proto.OnRequestOutput_FORWARD,
		Req: &proto.HTTPRequest{
			Headers: map[string]string{
				"X-Auth-Passed": "true",
			},
		},
	}, nil
}

func main() {
	// create TCP server
	grpcAddr := "localhost:9090"
	if host, ok := os.LookupEnv("LISTEN"); ok {
		grpcAddr = host
	}
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		panic(err)
	}

	// init grpc handlers
	server := grpc.NewServer()
	proto.RegisterInterceptorServer(server, &plugin{})
	reflection.Register(server)
	log.Printf("Grafana datasource auth server started on host %s", grpcAddr)
	log.Fatal(server.Serve(lis))
}
