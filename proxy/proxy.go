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
	// Handle CORS preflight requests (OPTIONS) directly. Only set CORS
	// headers here for preflight responses. For proxied responses we use
	// the reverse proxy's ModifyResponse to normalize headers, to avoid
	// sending duplicate Access-Control-Allow-* values.
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Origin", "*") // Allow any origin
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.WriteHeader(http.StatusOK)
		return
	}

	targetURLString := strings.TrimPrefix(r.URL.Path, "/")
	if r.URL.RawQuery != "" {
		targetURLString += "?" + r.URL.RawQuery
	}

	// Check if any rule matches the requested URL (category is ignored here; UI uses it for grouping only)
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

	// Capture the incoming request's Origin header so we can echo it back in
	// the response. This avoids using '*' which combined with an upstream
	// origin value could create multiple values and fail the browser check.
	incomingOrigin := r.Header.Get("Origin")

	// Normalize/remove upstream CORS headers then set a single, appropriate
	// Access-Control-Allow-Origin value. Prefer the incoming Origin (if
	// present) otherwise fall back to a wildcard.
	proxy.ModifyResponse = func(resp *http.Response) error {
		// Remove any upstream CORS headers we don't control
		resp.Header.Del("Access-Control-Allow-Origin")
		resp.Header.Del("Access-Control-Allow-Headers")
		resp.Header.Del("Access-Control-Allow-Methods")

		// Use the incoming origin when available to avoid '*' + origin
		if incomingOrigin != "" {
			resp.Header.Set("Access-Control-Allow-Origin", incomingOrigin)
		} else {
			resp.Header.Set("Access-Control-Allow-Origin", "*")
		}
		resp.Header.Set("Access-Control-Allow-Headers", "Content-Type")
		resp.Header.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")

		return nil
	}

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
