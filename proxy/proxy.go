package proxy

import (
	"faultline/state"
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

	// Get a read-only copy of the current rules
	rules := p.ruleState.GetRules()

	for _, rule := range rules {
		// Only apply active rules
		if !rule.Enabled {
			continue
		}

		if strings.HasPrefix(targetURLStr, rule.Target) {
			log.Printf("[PROXY MATCH] Target: %s -> Injecting Failure: %s", rule.Target, rule.Failure.Type)
			p.injectFailure(w, r, &rule)
			return
		}
	}

	p.serveReverseProxy(targetURLStr, w, r)
}

// injectFailure applies the failure logic defined in a rule.
func (p *Proxy) injectFailure(w http.ResponseWriter, r *http.Request, rule *state.Rule) {
	targetURL := strings.TrimPrefix(r.URL.Path, "/")

	switch rule.Failure.Type {
	case "latency":
		time.Sleep(time.Duration(rule.Failure.LatencyMs) * time.Millisecond)
		p.serveReverseProxy(targetURL, w, r)
	case "error":
		w.WriteHeader(rule.Failure.ErrorCode)
		w.Write([]byte("FaultLine: Injected Error Response"))
	case "flaky":
		if rand.Float64() < rule.Failure.Probability {
			log.Printf("[FLAKY] Failure triggered for %s", targetURL)
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("FaultLine: Injected Flaky Error"))
		} else {
			log.Printf("[FLAKY] Passed through for %s", targetURL)
			p.serveReverseProxy(targetURL, w, r)
		}
	default:
		p.serveReverseProxy(targetURL, w, r)
	}
}

// serveReverseProxy forwards the request to the original destination.
func (p *Proxy) serveReverseProxy(target string, w http.ResponseWriter, r *http.Request) {
	remote, err := url.Parse(target)
	if err != nil {
		http.Error(w, "Invalid target URL", http.StatusBadRequest)
		return
	}
	proxy := httputil.NewSingleHostReverseProxy(remote)
	r.URL.Host = remote.Host
	r.URL.Scheme = remote.Scheme
	r.Header.Set("X-Forwarded-Host", r.Header.Get("Host"))
	r.Host = remote.Host

	log.Printf("[PROXY] Forwarding request to %s", target)
	proxy.ServeHTTP(w, r)
}
