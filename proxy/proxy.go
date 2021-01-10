package proxy

import (
	"context"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/afoninsky/verdite/logger"
	"github.com/afoninsky/verdite/proto"
)

type Proxy struct {
	plugin proto.InterceptorClient
	log    *logger.Logger
}

func New(plugin proto.InterceptorClient, log *logger.Logger) *Proxy {
	return &Proxy{
		plugin: plugin,
		log:    log,
	}
}

func (s *Proxy) Handler(w http.ResponseWriter, r *http.Request) {
	// request grpc plugin what to do with request
	data, err := s.plugin.OnRequest(context.Background(), &proto.OnRequestInput{
		Req: &proto.HTTPRequest{
			Method: r.Method,
			URL:    r.RequestURI,
			// Header: r.Header,
			// Body:   []byte{},
		},
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	// perform actions based on plugin data
	switch data.Action {
	case proto.OnRequestOutput_NONE:
		//
	case proto.OnRequestOutput_REJECT:
		// TODO: return response
	case proto.OnRequestOutput_UPDATE:
		// TODO: copy request
	default:
		http.Error(w, "wrong answer from the plugin", http.StatusServiceUnavailable)
		return
	}

	// https request
	if r.Method == http.MethodConnect {
		tunnelForwarder(w, r)
		return
	}
	httpForwarder(w, r)
}

func copyHeaders(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func httpForwarder(w http.ResponseWriter, r *http.Request) {
	res, err := http.DefaultTransport.RoundTrip(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
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
	io.Copy(destination, source)
}
