package proxy

import (
	"faultline/config"
	"log"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

// Proxy holds the configuration and serves as the main handler.
type Proxy struct {
	config *config.Config
	rules  map[string]*config.Rule
}

// NewProxy creates and initializes the proxy.
func NewProxy(cfg *config.Config) (*Proxy, error) {
	p := &Proxy{
		config: cfg,
		rules:  make(map[string]*config.Rule),
	}
	// Pre-process rules for faster lookups
	for i := range cfg.Rules {
		rule := &cfg.Rules[i]
		p.rules[rule.Target] = rule
	}
	rand.Seed(time.Now().UnixNano())
	return p, nil
}

// HandleRequest is the core logic for the proxy.
func (p *Proxy) HandleRequest(w http.ResponseWriter, r *http.Request) {
	// The actual target URL is expected to be in the path
	// e.g., /https://api.example.com/data -> https://api.example.com/data
	targetURL := strings.TrimPrefix(r.URL.Path, "/")
	if r.URL.RawQuery != "" {
		targetURL += "?" + r.URL.RawQuery
	}

	// Check if any rule matches the requested URL
	for target, rule := range p.rules {
		if strings.HasPrefix(targetURL, target) {
			log.Printf("[RULE MATCH] Target: %s -> Injecting Failure: %s", target, rule.Failure.Type)
			p.injectFailure(w, r, rule)
			return
		}
	}

	// If no rule matches, just proxy the request normally
	p.serveReverseProxy(targetURL, w, r)
}

// injectFailure applies the failure logic defined in a rule.
func (p *Proxy) injectFailure(w http.ResponseWriter, r *http.Request, rule *config.Rule) {
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
			// For simplicity, flaky currently just returns an error.
			// This could be expanded to randomly pick between latency and error.
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("FaultLine: Injected Flaky Error"))
		} else {
			log.Printf("[FLAKY] Passed through for %s", targetURL)
			p.serveReverseProxy(targetURL, w, r)
		}

	default:
		log.Printf("Unknown failure type: %s. Proxying normally.", rule.Failure.Type)
		p.serveReverseProxy(targetURL, w, r)
	}
}

// serveReverseProxy forwards the request to the original destination.
func (p *Proxy) serveReverseProxy(target string, w http.ResponseWriter, r *http.Request) {
	remote, err := url.Parse(target)
	if err != nil {
		log.Printf("Error parsing target URL: %v", err)
		http.Error(w, "Invalid target URL", http.StatusBadRequest)
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(remote)
	r.URL = remote
	r.Host = remote.Host
	r.RequestURI = "" // Must be cleared

	log.Printf("[PROXY] Forwarding request to %s", target)
	proxy.ServeHTTP(w, r)
}
