package main

import (
	"crypto/tls"
	"net/http"

	"github.com/afoninsky/utilities/pkg/logger"
	"github.com/afoninsky/verdite/pkg/config"
	"github.com/afoninsky/verdite/pkg/httpproxy"
)

func main() {
	log := logger.New()

	cfg, err := config.New("./config.yaml")
	log.FatalIfErr(err)

	proxy, err := httpproxy.New(cfg)
	log.FatalIfErr(err)

	server := &http.Server{
		Addr:    cfg.Listen,
		Handler: proxy.Router(),
		// disable HTTP/2 support
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}
	log.WithField("address", cfg.Listen).Infoln("HTTP/1.0 proxy server started")
	log.Fatal(server.ListenAndServe())

	// log.WithField("address", grpcTarget).Infoln("Connecting to GRPC interceptor ...")
	// // grpcClient, err := grpc.Dial(grpcTarget, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(time.Second))
	// grpcClient, err := grpc.Dial(grpcTarget,
	// 	grpc.WithInsecure(),
	// 	grpc.WithBlock(),
	// 	// grpc.WithKeepaliveParams(keepalive.ClientParameters{}),
	// 	// grpc.WithTimeout(time.Second)
	// )
	// log.FatalIfErr(err)
	// defer grpcClient.Close()

	// reqInterceptor := proto.NewInterceptorClient(grpcClient)
	// httpProxy := proxy.New(reqInterceptor, log)

	// server := &http.Server{
	// 	Addr:         httpAddr,
	// 	Handler:      http.HandlerFunc(httpProxy.Handler),
	// 	TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)), // disable HTTP/2
	// }
	// log.WithField("address", httpAddr).Infoln("HTTP/1.0 proxy server started")
	// log.Fatal(server.ListenAndServe())
}
