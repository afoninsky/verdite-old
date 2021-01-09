package main

import (
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

func main() {
	log := logrus.StandardLogger()
	server := &http.Server{
		Addr: "localhost:8080",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// TODO: auth here
			// TODO: intercept http request
			if r.Method == http.MethodConnect {
				// log.WithField("url", r.URL.String()).Warnln("Unable to intercept HTTP request")
				tunnelForwarder(w, r)
				return
			}
			httpForwarder(w, r)
		}),
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)), // disable HTTP/2
	}
	log.Fatal(server.ListenAndServe())
}

func copyHeaders(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func logRequest(req *http.Request, res *http.Response) {
	logrus.StandardLogger().
		// WithField("request_body", req.Body).
		WithField("url", req.URL.String()).
		WithField("method", req.Method).
		// WithField("response", res.Body).
		Infoln()
}

// forwards http requests
func httpForwarder(w http.ResponseWriter, r *http.Request) {
	res, err := http.DefaultTransport.RoundTrip(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer logRequest(r, res)
	defer res.Body.Close()
	copyHeaders(w.Header(), res.Header)
	w.WriteHeader(res.StatusCode)
	if _, err := io.Copy(w, res.Body); err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
}

func tunnelForwarder(w http.ResponseWriter, r *http.Request) {
	destConn, err := net.DialTimeout("tcp", r.Host, 10*time.Second)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}
	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
	}
	go transfer(destConn, clientConn)
	go transfer(clientConn, destConn)
}

func transfer(destination io.WriteCloser, source io.ReadCloser) {
	defer destination.Close()
	defer source.Close()
	_, _ = io.Copy(destination, source)
}
