package proxy

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
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
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	data, err := s.plugin.OnRequest(context.Background(), &proto.OnRequestInput{
		Req: &proto.HTTPRequest{
			Method:  r.Method,
			URL:     r.RequestURI,
			Headers: mapHeaders(r.Header),
			Body:    body,
		},
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	switch data.Action {
	case proto.OnRequestOutput_IGNORE:
		//
	case proto.OnRequestOutput_REJECT:
		for k, v := range data.Res.Headers {
			w.Header().Set(k, v)
		}
		// w.Body = []byte{}
		w.WriteHeader(int(data.Res.Status))
		w.Write([]byte{})
		return

	case proto.OnRequestOutput_UPDATE:
		url, err := url.Parse(data.Req.URL)
		if err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		r.Method = data.Req.Method
		r.URL = url
		for k, v := range data.Req.Headers {
			r.Header.Set(k, v)
		}
		r.Body = ioutil.NopCloser(bytes.NewBuffer(data.Req.Body))
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

// convert http.Header slice to a map containing headers
func mapHeaders(src http.Header) map[string]string {
	dst := map[string]string{}
	for k, vv := range src {
		for _, v := range vv {
			dst[k] = v
		}
	}
	return dst
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
