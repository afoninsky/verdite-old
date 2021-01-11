package main

import (
	"crypto/tls"
	"net/http"

	"github.com/afoninsky/utilities/pkg/logger"
	"github.com/afoninsky/verdite/config"
	"github.com/afoninsky/verdite/httpproxy"
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
}
