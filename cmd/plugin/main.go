package main

import (
	"net"
	"os"

	"github.com/afoninsky/utilities/pkg/logger"
	// "github.com/afoninsky/verdite/plugin/dummy"
	plugin "github.com/afoninsky/verdite/plugin/grafana-ds-auth"

	"github.com/afoninsky/verdite/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const grpcPort = "9090"

func main() {
	log := logger.New()
	log.Infoln("test")

	grpcAddr := "localhost:9090"
	if host, ok := os.LookupEnv("LISTEN"); ok {
		grpcAddr = host
	}
	lis, err := net.Listen("tcp", grpcAddr)
	log.FatalIfErr(err)

	server := grpc.NewServer()
	proto.RegisterInterceptorServer(server, &plugin.Plugin{})
	reflection.Register(server)
	log.WithField("address", grpcAddr).Infoln("GRPC interceptor started")
	log.Fatal(server.Serve(lis))
}
