package proxy

import (
	"faultline/state"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

// Proxy holds the shared state of rules.
type Proxy struct {
	ruleState *state.RuleState
}

// NewProxy creates and initializes the proxy.
func NewProxy(state *state.RuleState) *Proxy {
	return &Proxy{
		ruleState: state,
	}
}

// HandleRequest is the core logic for the proxy.
func (p *Proxy) HandleRequest(w http.ResponseWriter, r *http.Request) {
	targetURLStr := strings.TrimPrefix(r.URL.Path, "/")
	if r.URL.RawQuery != "" {
		targetURLStr += "?" + r.URL.RawQuery
	}
	reqStart := time.Now()
	client := r.RemoteAddr
	method := r.Method
	log.Printf("[API] -> %s %s from %s", method, targetURLStr, client)

	// Get a read-only copy of the current rules
	rules := p.ruleState.GetRules()

	for _, rule := range rules {
		// Only apply active rules
		if !rule.Enabled {
			continue
		}

		if strings.HasPrefix(targetURLStr, rule.Target) {
			log.Printf("[API MATCH] target=%s failure=%s", rule.Target, rule.Failure.Type)
			p.injectFailure(w, r, &rule, reqStart, client)
			return
		}
	}

	p.serveReverseProxy(targetURLStr, w, r, reqStart, client)
}

// injectFailure applies the failure logic defined in a rule.
func (p *Proxy) injectFailure(w http.ResponseWriter, r *http.Request, rule *state.Rule, reqStart time.Time, client string) {
	targetURL := strings.TrimPrefix(r.URL.Path, "/")

	switch rule.Failure.Type {
	case "latency":
		d := time.Duration(rule.Failure.LatencyMs) * time.Millisecond
		time.Sleep(d)
		log.Printf("[API INJECT] latency=%s target=%s from=%s", d, targetURL, client)
		p.serveReverseProxy(targetURL, w, r, reqStart, client)
	case "error":
		status := rule.Failure.ErrorCode
		body := []byte("FaultLine: Injected Error Response")
		w.WriteHeader(status)
		_, _ = w.Write(body)
		dur := time.Since(reqStart)
		log.Printf("[API INJECT] error status=%d bytes=%d dur=%s target=%s from=%s", status, len(body), dur, targetURL, client)
	case "flaky":
		if rand.Float64() < rule.Failure.Probability {
			status := http.StatusServiceUnavailable
			body := []byte("FaultLine: Injected Flaky Error")
			log.Printf("[API FLAKY] triggered p=%.2f target=%s from=%s", rule.Failure.Probability, targetURL, client)
			w.WriteHeader(status)
			_, _ = w.Write(body)
			dur := time.Since(reqStart)
			log.Printf("[API INJECT] flaky-error status=%d bytes=%d dur=%s target=%s from=%s", status, len(body), dur, targetURL, client)
		} else {
			log.Printf("[API FLAKY] pass-through p=%.2f target=%s from=%s", rule.Failure.Probability, targetURL, client)
			p.serveReverseProxy(targetURL, w, r, reqStart, client)
		}
	default:
		p.serveReverseProxy(targetURL, w, r, reqStart, client)
	}
}

// serveReverseProxy forwards the request to the original destination.
func (p *Proxy) serveReverseProxy(target string, w http.ResponseWriter, r *http.Request, reqStart time.Time, client string) {
	remote, err := url.Parse(target)
	if err != nil {
		http.Error(w, "Invalid target URL", http.StatusBadRequest)
		return
	}
	proxy := httputil.NewSingleHostReverseProxy(remote)

	// Wrap ResponseWriter to capture status and bytes
	lw := &loggingResponseWriter{ResponseWriter: w, status: 0, bytes: 0}

	r.URL.Host = remote.Host
	r.URL.Scheme = remote.Scheme
	r.Header.Set("X-Forwarded-Host", r.Header.Get("Host"))
	r.Host = remote.Host

	// Provide a clearer error handler to log upstream failures
	proxy.ErrorHandler = func(rw http.ResponseWriter, req *http.Request, e error) {
		dur := time.Since(reqStart)
		http.Error(rw, fmt.Sprintf("Upstream error: %v", e), http.StatusBadGateway)
		log.Printf("[API FWD] upstream-error dur=%s target=%s from=%s err=%v", dur, target, client, e)
	}

	log.Printf("[API FWD] -> %s %s from %s", r.Method, target, client)
	start := time.Now()
	proxy.ServeHTTP(lw, r)
	dur := time.Since(start)
	status := lw.status
	if status == 0 {
		status = http.StatusOK
	}
	log.Printf("[API FWD] <- status=%d bytes=%d dur=%s target=%s from=%s", status, lw.bytes, dur, target, client)
}

// loggingResponseWriter captures status code and bytes written.
type loggingResponseWriter struct {
	http.ResponseWriter
	status int
	bytes  int64
}

func (lw *loggingResponseWriter) WriteHeader(statusCode int) {
	lw.status = statusCode
	lw.ResponseWriter.WriteHeader(statusCode)
}

func (lw *loggingResponseWriter) Write(b []byte) (int, error) {
	n, err := lw.ResponseWriter.Write(b)
	lw.bytes += int64(n)
	return n, err
}
