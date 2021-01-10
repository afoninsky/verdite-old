package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/afoninsky/verdite/logger"
	"github.com/afoninsky/verdite/proto"
	"github.com/afoninsky/verdite/proxy"
	"google.golang.org/grpc"
)

const httpPort = "8080"
const reqIntPort = "9090"

func main() {
	log := logger.New()

	grpcTarget := fmt.Sprintf("localhost:%s", reqIntPort)
	log.WithField("address", grpcTarget).Infoln("Connecting to GRPC interceptor ...")
	grpcClient, err := grpc.Dial(grpcTarget, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(time.Second))
	log.FatalIfErr(err)
	defer grpcClient.Close()

	reqInterceptor := proto.NewInterceptorClient(grpcClient)
	httpProxy := proxy.New(reqInterceptor, log)

	httpAddr := fmt.Sprintf("localhost:%s", httpPort)
	server := &http.Server{
		Addr:         httpAddr,
		Handler:      http.HandlerFunc(httpProxy.Handler),
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)), // disable HTTP/2
	}
	log.WithField("address", httpAddr).Infoln("HTTP/1.0 proxy server started")
	log.Fatal(server.ListenAndServe())
}
