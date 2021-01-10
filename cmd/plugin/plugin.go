package main

import (
	"fmt"
	"net"

	"github.com/afoninsky/verdite/logger"
	"github.com/afoninsky/verdite/plugin/dummy"
	"github.com/afoninsky/verdite/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const grpcPort = "9090"

func main() {
	log := logger.New()

	grpcAddr := fmt.Sprintf("localhost:%s", grpcPort)
	lis, err := net.Listen("tcp", grpcAddr)
	log.FatalIfErr(err)

	server := grpc.NewServer()
	proto.RegisterInterceptorServer(server, &dummy.Plugin{})
	reflection.Register(server)
	log.WithField("address", grpcAddr).Infoln("GRPC interceptor started")
	log.Fatal(server.Serve(lis))
}
