package api

import (
	"encoding/json"
	"faultline/cli"
	"faultline/codeanalysis"
	"faultline/openapi"
	"faultline/state"
	"log"
	"net/http"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// ApiHandler holds a reference to the shared rule state and persistence manager.
type ApiHandler struct {
	ruleState    *state.RuleState
	ruleManager  *cli.RuleManager
	openAPISpecs []string // Cache for discovered OpenAPI specs
}

// NewApiHandler creates a new handler for the API.
func NewApiHandler(rm *cli.RuleManager) *ApiHandler {
	return &ApiHandler{
		ruleState:   rm.GetRuleState(),
		ruleManager: rm,
	}
}

// *** THIS IS THE MISSING FUNCTION ***
// RegisterHandlers sets up the routing for the API endpoints.
func RegisterHandlers(router *mux.Router, rm *cli.RuleManager) {
	h := NewApiHandler(rm)

	// Define the API routes and link them to the handler methods
	router.HandleFunc("/api/rules", h.GetRules).Methods("GET")
	router.HandleFunc("/api/rules", h.AddRule).Methods("POST")
	router.HandleFunc("/api/rules/{id}", h.UpdateRule).Methods("PUT")
	router.HandleFunc("/api/rules/{id}", h.DeleteRule).Methods("DELETE")

	// OpenAPI endpoints discovery routes
	router.HandleFunc("/api/endpoints", h.GetEndpoints).Methods("GET")
	router.HandleFunc("/api/endpoints/discover", h.DiscoverEndpoints).Methods("POST")
	router.HandleFunc("/api/endpoints/specs", h.GetOpenAPISpecs).Methods("GET")

	// Code analysis endpoints
	router.HandleFunc("/api/endpoints/analyze-code", h.AnalyzeCodeEndpoints).Methods("GET")
	router.HandleFunc("/api/endpoints/analyze-directory", h.AnalyzeDirectory).Methods("POST")
}

// GetRules returns the list of current failure rules as JSON.
func (h *ApiHandler) GetRules(w http.ResponseWriter, r *http.Request) {
	// Check if rules file has been modified and reload if necessary (for CLI changes)
	if err := h.ruleState.CheckAndReloadIfModified(); err != nil {
		// Log error but continue with current state
		// log.Printf("Warning: Failed to reload rules: %v", err)
	}

	rules := h.ruleState.GetRules()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rules)
}

// AddRule adds a new failure rule from a JSON payload.
func (h *ApiHandler) AddRule(w http.ResponseWriter, r *http.Request) {
	var newRule state.Rule
	if err := json.NewDecoder(r.Body).Decode(&newRule); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Assign a new UUID and enable by default
	newRule.ID = uuid.New().String()
	newRule.Enabled = true

	// Default category to "api" if not provided, to preserve current behavior
	if newRule.Category == "" {
		newRule.Category = "api"
	}
	h.ruleState.AddRule(newRule)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newRule)
}

// UpdateRule updates an existing rule from a JSON payload.
func (h *ApiHandler) UpdateRule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var updatedRule state.Rule
	if err := json.NewDecoder(r.Body).Decode(&updatedRule); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	updatedRule.ID = id // Ensure the ID from the URL is used

	if !h.ruleState.UpdateRule(updatedRule) {
		http.Error(w, "Rule not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedRule)
}

// DeleteRule removes a rule by its ID.
func (h *ApiHandler) DeleteRule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if !h.ruleState.DeleteRule(id) {
		http.Error(w, "Rule not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetEndpoints returns discovered endpoints from OpenAPI specs
func (h *ApiHandler) GetEndpoints(w http.ResponseWriter, r *http.Request) {
	specPath := r.URL.Query().Get("spec")

	if specPath == "" {
		// If no specific spec provided, try to find OpenAPI specs in current directory
		specs, err := openapi.FindOpenAPISpecs(".")
		if err != nil || len(specs) == 0 {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"endpoints": []interface{}{},
				"baseUrls":  []string{},
				"message":   "No OpenAPI specifications found. Please provide a spec file path or discover endpoints first.",
			})
			return
		}
		specPath = specs[0] // Use first found spec
	}

	// Parse the OpenAPI spec and extract endpoints
	discovered, err := openapi.ParseOpenAPISpec(specPath)
	if err != nil {
		log.Printf("[ERROR] Failed to parse OpenAPI spec %s: %v", specPath, err)
		http.Error(w, "Failed to parse OpenAPI specification", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(discovered)
}

// DiscoverEndpoints searches for OpenAPI specs and parses them
func (h *ApiHandler) DiscoverEndpoints(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Directory string `json:"directory,omitempty"`
		SpecPath  string `json:"specPath,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// If no body provided, use current directory
		req.Directory = "."
	}

	var allEndpoints []openapi.DiscoveredEndpoints

	if req.SpecPath != "" {
		// Parse specific spec file
		discovered, err := openapi.ParseOpenAPISpec(req.SpecPath)
		if err != nil {
			log.Printf("[ERROR] Failed to parse OpenAPI spec %s: %v", req.SpecPath, err)
			http.Error(w, "Failed to parse OpenAPI specification", http.StatusBadRequest)
			return
		}
		allEndpoints = append(allEndpoints, *discovered)
	} else {
		// Discover specs in directory
		directory := req.Directory
		if directory == "" {
			directory = "."
		}

		specs, err := openapi.FindOpenAPISpecs(directory)
		if err != nil {
			log.Printf("[ERROR] Failed to find OpenAPI specs: %v", err)
			http.Error(w, "Failed to discover OpenAPI specifications", http.StatusInternalServerError)
			return
		}

		h.openAPISpecs = specs // Cache discovered specs

		for _, specPath := range specs {
			if !openapi.ValidateOpenAPIFile(specPath) {
				log.Printf("[WARNING] Skipping invalid OpenAPI file: %s", specPath)
				continue
			}

			discovered, err := openapi.ParseOpenAPISpec(specPath)
			if err != nil {
				log.Printf("[WARNING] Failed to parse OpenAPI spec %s: %v", specPath, err)
				continue
			}
			allEndpoints = append(allEndpoints, *discovered)
		}
	}

	response := map[string]interface{}{
		"specs":           allEndpoints,
		"totalEndpoints":  getTotalEndpoints(allEndpoints),
		"discoveredSpecs": len(allEndpoints),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetOpenAPISpecs returns list of discovered OpenAPI specification files
func (h *ApiHandler) GetOpenAPISpecs(w http.ResponseWriter, r *http.Request) {
	directory := r.URL.Query().Get("directory")
	if directory == "" {
		directory = "."
	}

	specs, err := openapi.FindOpenAPISpecs(directory)
	if err != nil {
		log.Printf("[ERROR] Failed to find OpenAPI specs: %v", err)
		// Return empty result instead of error to prevent client issues
		response := map[string]interface{}{
			"specs":     []interface{}{},
			"directory": directory,
			"total":     0,
			"error":     err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Validate each spec and include metadata
	var validSpecs []map[string]interface{}
	for _, specPath := range specs {
		specInfo := map[string]interface{}{
			"path":  specPath,
			"name":  filepath.Base(specPath),
			"valid": openapi.ValidateOpenAPIFile(specPath),
		}

		if specInfo["valid"].(bool) {
			// Try to get basic info from the spec
			if discovered, err := openapi.ParseOpenAPISpec(specPath); err == nil {
				specInfo["title"] = discovered.Info.Title
				specInfo["version"] = discovered.Info.Version
				specInfo["endpointCount"] = len(discovered.Endpoints)
				specInfo["baseUrls"] = discovered.BaseURLs
			}
		}

		validSpecs = append(validSpecs, specInfo)
	}

	response := map[string]interface{}{
		"specs":     validSpecs,
		"directory": directory,
		"total":     len(validSpecs),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// AnalyzeCodeEndpoints analyzes source code to find actual API endpoints
func (h *ApiHandler) AnalyzeCodeEndpoints(w http.ResponseWriter, r *http.Request) {
	directory := r.URL.Query().Get("directory")
	if directory == "" {
		directory = "./showcase-app" // Default to showcase app
	}

	result, err := codeanalysis.AnalyzeDirectory(directory)
	if err != nil {
		log.Printf("[ERROR] Failed to analyze code in %s: %v", directory, err)
		// Return empty result instead of error to prevent client issues
		response := map[string]interface{}{
			"endpoints":  []interface{}{},
			"files":      []string{},
			"totalLines": 0,
			"uniqueUrls": []string{},
			"error":      err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// AnalyzeDirectory analyzes a specific directory for API endpoints
func (h *ApiHandler) AnalyzeDirectory(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Directory string   `json:"directory,omitempty"`
		Files     []string `json:"files,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// If no body provided, use default
		req.Directory = "./showcase-app"
	}

	var result *codeanalysis.CodeAnalysisResult
	var err error

	if len(req.Files) > 0 {
		// Analyze specific files
		result, err = codeanalysis.AnalyzeSpecificFiles(req.Files)
	} else {
		// Analyze directory
		directory := req.Directory
		if directory == "" {
			directory = "./showcase-app"
		}
		result, err = codeanalysis.AnalyzeDirectory(directory)
	}

	if err != nil {
		log.Printf("[ERROR] Failed to analyze code: %v", err)
		http.Error(w, "Failed to analyze source code", http.StatusInternalServerError)
		return
	}

	// Enhance the result with additional metadata
	response := map[string]interface{}{
		"analysis":       result,
		"totalEndpoints": len(result.Endpoints),
		"analyzedFiles":  len(result.Files),
		"uniqueUrls":     len(result.UniqueURLs),
		"methodCounts":   result.MethodCounts,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Helper function to calculate total endpoints across all specs
func getTotalEndpoints(specs []openapi.DiscoveredEndpoints) int {
	total := 0
	for _, spec := range specs {
		total += len(spec.Endpoints)
	}
	return total
}
