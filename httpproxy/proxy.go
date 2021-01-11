package httpproxy

import (
	"io"
	"net"
	"net/http"
	"time"

	"github.com/afoninsky/utilities/pkg/logger"
	"github.com/afoninsky/verdite/config"
	"github.com/afoninsky/verdite/interceptor"
	"github.com/gorilla/mux"
)

// Proxy ...
type Proxy struct {
	log      *logger.Logger
	router   *mux.Router
	handlers map[string]interceptor.Interceptor
}

// New ...
func New(cfg *config.Config) (*Proxy, error) {
	s := Proxy{
		router: mux.NewRouter(),
		log:    logger.New(),
	}

	// init http request interceptors
	s.handlers = map[string]interceptor.Interceptor{}
	for name, iCfg := range cfg.Interceptors {
		rh, err := interceptor.New(name, iCfg)
		if err != nil {
			return nil, err
		}
		s.handlers[name] = rh
		s.log.WithField("name", name).Infoln("Request interceptor created")
	}

	// create http request matchers
	for name, rCfg := range cfg.Matchers {
		s.addHTTPRoute(name, rCfg)
		s.log.WithField("name", name).Infoln("HTTP route matcher created")
	}

	// s.router.Use(s.loggingMiddleware)

	return &s, nil
}

// Router returns http mux router
func (s *Proxy) Router() *mux.Router {
	return s.router
}

// func (s *Proxy) loggingMiddleware(next http.Handler) http.Handler {
// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		defer func() {
// 			s.log.
// 				WithField("method", r.Method).
// 				WithField("match", r.Context().Value("matcher")).
// 				Info(r.URL)
// 		}()
// 		next.ServeHTTP(w, r)
// 	})
// }

// create mux route based on request matcher
func (s *Proxy) addHTTPRoute(name string, rCfg config.Matcher) {
	route := s.router.Name(name).HandlerFunc(s.createRequestHandler(name, rCfg))
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
}

func (s *Proxy) createRequestHandler(matcherName string, rule config.Matcher) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// ctx := context.WithValue(r.Context(), "matcher", matcherName)
		// r = r.WithContext(ctx)

		// execute set of interceptors
		// for _, hName := range rule.RequestHandlers {
		// 	if !s.callHandler(hName, rule.ParseRequestBody, w, r) {
		// 		return
		// 	}
		// }

		// tunnel https request
		if r.Method == http.MethodConnect {
			tunnelForwarder(w, r)
			return
		}
		httpForwarder(w, r)
	}
}

func (s *Proxy) callHandler(hName string, parseBody bool, w http.ResponseWriter, r *http.Request) bool {
	return true
	// handler, ok := s.handlers[hName]
	// if !ok {
	// 	http.Error(w, "handler does not exist", http.StatusServiceUnavailable)
	// 	return false
	// }

	// var err error
	// var body []byte

	// // read request body into buffer if flag specified
	// if parseBody {
	// 	body, err = ioutil.ReadAll(r.Body)
	// 	if err != nil {
	// 		http.Error(w, err.Error(), http.StatusServiceUnavailable)
	// 		return false
	// 	}
	// }

	// data, err := handler.OnRequest(context.Background(), &proto.OnRequestInput{
	// 	Req: &proto.HTTPRequest{
	// 		Method:  r.Method,
	// 		URL:     r.RequestURI,
	// 		Headers: mapHeaders(r.Header),
	// 		Body:    body,
	// 	},
	// })
	// if err != nil {
	// 	http.Error(w, err.Error(), http.StatusServiceUnavailable)
	// 	return false
	// }

	// s.log.
	// 	WithField("handler", hName).
	// 	WithField("action", data.Action.String()).
	// 	Infof("%s %s", r.Method, r.URL)

	// switch data.Action {
	// case proto.OnRequestOutput_IGNORE:
	// 	return true
	// case proto.OnRequestOutput_RESPONSE:
	// 	for k, v := range data.Res.Headers {
	// 		w.Header().Set(k, v)
	// 	}
	// 	w.WriteHeader(int(data.Res.Status))
	// 	w.Write(data.Res.Body)
	// 	return false

	// case proto.OnRequestOutput_FORWARD:
	// 	if data.Req.URL != "" {
	// 		url, err := url.Parse(data.Req.URL)
	// 		if err != nil {
	// 			http.Error(w, err.Error(), http.StatusServiceUnavailable)
	// 			return false
	// 		}
	// 		r.URL = url
	// 	}
	// 	if data.Req.Method != "" {
	// 		r.Method = data.Req.Method
	// 	}

	// 	for k, v := range data.Req.Headers {
	// 		r.Header.Set(k, v)
	// 	}

	// 	if parseBody && len(data.Req.Body) > 0 {
	// 		r.Body = ioutil.NopCloser(bytes.NewBuffer(data.Req.Body))
	// 	}
	// 	return true
	// default:
	// 	http.Error(w, "wrong answer from the plugin", http.StatusServiceUnavailable)
	// 	return false
	// }
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
