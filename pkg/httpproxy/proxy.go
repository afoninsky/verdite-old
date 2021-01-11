package httpproxy

import (
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	"github.com/afoninsky/utilities/pkg/logger"
	"github.com/afoninsky/verdite/pkg/config"
	"github.com/afoninsky/verdite/pkg/interceptor"
	"github.com/gorilla/mux"
)

type Proxy struct {
	log      *logger.Logger
	cfg      *config.Config
	router   *mux.Router
	handlers map[string]*interceptor.Interceptor
}

func New(cfg *config.Config) (*Proxy, error) {
	s := Proxy{
		// cfg:    cfg,
		router: mux.NewRouter(),
		log:    logger.New(),
	}

	s.handlers = map[string]*interceptor.Interceptor{}
	for name, cfgRH := range cfg.RequestHandler {
		rh, err := interceptor.New(name, cfgRH)
		if err != nil {
			return nil, err
		}
		s.handlers[name] = rh
		s.log.WithField("name", name).Infoln("Request handler created")
	}

	for name, cfgRule := range cfg.Rule {
		if err := s.createHTTPRoute(name, cfgRule, cfg.Route); err != nil {
			return nil, err
		}
		s.log.WithField("name", name).Infoln("Route created")
	}

	return &s, nil
}

func (s *Proxy) createHTTPRoute(name string, rule config.Rule, routes map[string]config.Route) error {
	rCfg, ok := routes[name]
	if !ok {
		return fmt.Errorf(`route "%s" does not exist in config`, name)
	}
	route := s.router.Name(name).HandlerFunc(s.createRequestHandler(rule))

	if rCfg.Host != "" {
		route.Host(rCfg.Host)
	}
	if rCfg.Path != "" {
		route.Path(rCfg.Path)
	}
	if rCfg.PathPrefix != "" {
		route.PathPrefix(rCfg.PathPrefix)
	}
	if len(rCfg.Methods) > 0 {
		route.Methods(rCfg.Methods...)
	}
	if len(rCfg.Schemes) > 0 {
		route.Schemes(rCfg.Schemes...)
	}
	if len(rCfg.Headers) > 0 {
		route.Headers(rCfg.Headers...)
	}
	if len(rCfg.Queries) > 0 {
		route.Queries(rCfg.Queries...)
	}
	return nil
}

func (s *Proxy) createRequestHandler(rule config.Rule) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var err error
		var body []byte
		// read request body into buffer if flag specified
		if rule.ParseRequestBody {
			body, err = ioutil.ReadAll(r.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusServiceUnavailable)
				return
			}
		}
		// execute set of interceptors
		for _, hName := range rule.RequestHandlers {
			if s.callHandler(hName, w, r) {
				return
			}
		}

		// tunnel https request
		if r.Method == http.MethodConnect {
			tunnelForwarder(w, r)
			return
		}
		httpForwarder(w, r)
	}
}

func (s *Proxy) Router() *mux.Router {
	return s.router
}

func (s *Proxy) callHandler(hName string, w http.ResponseWriter, r *http.Request) {
	h, ok := s.handlers[hName]
	if !ok {
		http.Error(w, "handler does not exist", http.StatusServiceUnavailable)
		return
	}
	// TODO
	// // request grpc plugin what to do with request
	// body, err := ioutil.ReadAll(r.Body)
	// if err != nil {
	// 	http.Error(w, err.Error(), http.StatusServiceUnavailable)
	// 	return
	// }
	// data, err := s.plugin.OnRequest(context.Background(), &proto.OnRequestInput{
	// 	Req: &proto.HTTPRequest{
	// 		Method:  r.Method,
	// 		URL:     r.RequestURI,
	// 		Headers: mapHeaders(r.Header),
	// 		Body:    body,
	// 	},
	// })
	// if err != nil {
	// 	http.Error(w, err.Error(), http.StatusServiceUnavailable)
	// 	return
	// }

	// switch data.Action {
	// case proto.OnRequestOutput_IGNORE:
	// 	//
	// case proto.OnRequestOutput_REJECT:
	// 	for k, v := range data.Res.Headers {
	// 		w.Header().Set(k, v)
	// 	}
	// 	w.WriteHeader(int(data.Res.Status))
	// 	w.Write(data.Res.Body)
	// 	return

	// case proto.OnRequestOutput_UPDATE:
	// 	if data.Req.URL != "" {
	// 		url, err := url.Parse(data.Req.URL)
	// 		if err != nil {
	// 			http.Error(w, err.Error(), http.StatusServiceUnavailable)
	// 			return
	// 		}
	// 		r.URL = url
	// 	}
	// 	if data.Req.Method != "" {
	// 		r.Method = data.Req.Method
	// 	}

	// 	for k, v := range data.Req.Headers {
	// 		r.Header.Set(k, v)
	// 	}
	// 	r.Body = ioutil.NopCloser(bytes.NewBuffer(data.Req.Body))
	// default:
	// 	http.Error(w, "wrong answer from the plugin", http.StatusServiceUnavailable)
	// 	return
	// }

	// // https request
	// if r.Method == http.MethodConnect {
	// 	tunnelForwarder(w, r)
	// 	return
	// }
	// httpForwarder(w, r)
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
