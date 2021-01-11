package httpproxy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/afoninsky/utilities/pkg/logger"
	"github.com/afoninsky/verdite/config"
	"github.com/afoninsky/verdite/interceptor"
	"github.com/afoninsky/verdite/proto"
	"github.com/julienschmidt/httprouter"
)

// Proxy ...
type Proxy struct {
	log      *logger.Logger
	handlers map[string]interceptor.Interceptor
	router   *httprouter.Router
}

// New ...
func New(cfg *config.Config) (*Proxy, error) {
	s := Proxy{}
	s.log = logger.New()
	s.router = &httprouter.Router{}
	s.router.NotFound = http.HandlerFunc(s.defaultRoute)

	// init http request interceptors
	s.handlers = map[string]interceptor.Interceptor{}
	for name, iCfg := range cfg.Interceptors {
		rh, err := interceptor.New(name, iCfg)
		if err != nil {
			return nil, err
		}
		s.handlers[name] = rh
	}

	// create http request matchers
	for _, rule := range cfg.Rules {
		handler := s.createRequestHandler(rule)
		s.router.HandlerFunc(rule.Match.Method, rule.Match.Path, handler)
		s.log.WithField("interceptors", strings.Join(rule.OnRequest, ",")).
			Infof("Rule added: %s //*%s", rule.Match.Method, rule.Match.Path)
	}

	// s.router.Use(s.loggingMiddleware)

	return &s, nil
}

// Handler returns http default middleware
func (s *Proxy) Handler() *httprouter.Router {
	return s.router
}

// implements default logic if no routes found
func (s *Proxy) defaultRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		tunnelForwarder(w, r)
		return
	}
	httpForwarder(w, r)
}

func (s *Proxy) createRequestHandler(cfg config.Rule) func(w http.ResponseWriter, r *http.Request) {

	return func(w http.ResponseWriter, r *http.Request) {

		chain := []string{}

		defer func() {
			s.log.WithField("rules", strings.Join(chain, ",")).
				Infof("%s %s", r.Method, r.URL)
		}()

		// parse body if according flag is specified
		// by default body is not parsed and passed "as is" to avoid request processing time increase
		var body []byte
		if cfg.ParseBody {
			var err error
			body, err = ioutil.ReadAll(r.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusServiceUnavailable)
				return
			}
		}

		// apply chain of request interceptora
		for _, name := range cfg.OnRequest {
			chain = append(chain, name)
			if !s.callInterceptor(name, body, w, r) {
				break
			}
		}
	}
}

func (s *Proxy) callInterceptor(name string, body []byte, w http.ResponseWriter, r *http.Request) bool {
	handler, ok := s.handlers[name]
	if !ok {
		http.Error(w, fmt.Sprintf(`unable to find "%s" interceptor`, name), http.StatusServiceUnavailable)
		return false
	}

	data, err := handler.OnRequest(context.Background(), &proto.OnRequestInput{
		Req: &proto.HTTPRequest{
			Method:  r.Method,
			URL:     r.RequestURI,
			Headers: mapHeaders(r.Header),
			Body:    body,
		},
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return false
	}

	switch data.Action {
	// do not modify request and pass it further
	case proto.OnRequestOutput_IGNORE:
		return true
	// stop processing request and return response
	case proto.OnRequestOutput_RESPONSE:
		for k, v := range data.Res.Headers {
			w.Header().Set(k, v)
		}
		w.WriteHeader(int(data.Res.Status))
		w.Write(data.Res.Body)
		return false
	// process request but update it
	case proto.OnRequestOutput_FORWARD:
		if data.Req.URL != "" {
			url, err := url.Parse(data.Req.URL)
			if err != nil {
				http.Error(w, err.Error(), http.StatusServiceUnavailable)
				return false
			}
			r.URL = url
		}
		if data.Req.Method != "" {
			r.Method = data.Req.Method
		}

		for k, v := range data.Req.Headers {
			r.Header.Set(k, v)
		}

		if len(data.Req.Body) > 0 {
			r.Body = ioutil.NopCloser(bytes.NewBuffer(data.Req.Body))
		}
		return true
	default:
		http.Error(w, "wrong answer from the plugin", http.StatusServiceUnavailable)
		return false
	}
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
