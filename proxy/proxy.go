package proxy

import (
	"faultline/state"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

// Proxy holds a reference to the shared rule state.
type Proxy struct {
	ruleState *state.RuleState
}

// NewProxy creates and initializes the proxy.
func NewProxy(rs *state.RuleState) *Proxy {
	return &Proxy{
		ruleState: rs,
	}
}

// HandleRequest is the core logic for the proxy.
func (p *Proxy) HandleRequest(w http.ResponseWriter, r *http.Request) {
	// Handle CORS preflight requests (OPTIONS) directly.
	w.Header().Set("Access-Control-Allow-Origin", "*") // Allow any origin
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	targetURLString := strings.TrimPrefix(r.URL.Path, "/")
	if r.URL.RawQuery != "" {
		targetURLString += "?" + r.URL.RawQuery
	}

	// Check if any rule matches the requested URL
	if rule, ok := p.ruleState.FindRuleForTarget(targetURLString); ok {
		log.Printf("[RULE MATCH] Target: %s -> Injecting Failure: %s", rule.Target, rule.Failure.Type)
		p.injectFailure(w, r, rule)
		return
	}

	// If no rule matches, just proxy the request normally
	p.serveReverseProxy(targetURLString, w, r)
}

// injectFailure applies the failure logic defined in a rule.
func (p *Proxy) injectFailure(w http.ResponseWriter, r *http.Request, rule *state.Rule) {
	targetURLString := strings.TrimPrefix(r.URL.Path, "/")

	switch rule.Failure.Type {
	case "latency":
		time.Sleep(time.Duration(rule.Failure.LatencyMs) * time.Millisecond)
		p.serveReverseProxy(targetURLString, w, r)

	case "error":
		w.WriteHeader(rule.Failure.ErrorCode)
		w.Write([]byte("FaultLine: Injected Error Response"))

	default:
		log.Printf("Unknown failure type: %s. Proxying normally.", rule.Failure.Type)
		p.serveReverseProxy(targetURLString, w, r)
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

	// *** THE DEFINITIVE FIX IS HERE ***
	// The original request to our proxy is, for example, GET /https://jsonplaceholder.typicode.com/users
	// The 'Director' must correctly rewrite this to be a valid request to the final server.
	originalPath := r.URL.Path

	director := func(req *http.Request) {
		// Set the scheme and host to the target's
		req.URL.Scheme = remote.Scheme
		req.URL.Host = remote.Host

		// The path sent to the final server should be the target's path, not the one
		// that includes the full URL.
		req.URL.Path = remote.Path

		// Copy the query parameters.
		req.URL.RawQuery = remote.RawQuery

		// Set the host of the request to the target host.
		req.Host = remote.Host

		// Clean up the RequestURI to avoid conflicts.
		req.RequestURI = ""

		log.Printf("Rewriting request from [%s] to [%s%s]", originalPath, req.URL.Host, req.URL.Path)
	}
	proxy.Director = director

	log.Printf("[PROXY] Forwarding request for %s", target)
	proxy.ServeHTTP(w, r)
}
