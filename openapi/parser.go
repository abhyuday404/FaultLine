package openapi

import (
	"fmt"
	"log"
	"net/url"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-openapi/loads"
	"github.com/go-openapi/spec"
)

// Endpoint represents a discovered API endpoint
type Endpoint struct {
	Path        string   `json:"path"`
	Method      string   `json:"method"`
	Summary     string   `json:"summary,omitempty"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	BaseURL     string   `json:"baseUrl,omitempty"`
	FullURL     string   `json:"fullUrl,omitempty"`
}

// DiscoveredEndpoints contains all discovered endpoints and metadata
type DiscoveredEndpoints struct {
	Endpoints []Endpoint `json:"endpoints"`
	BaseURLs  []string   `json:"baseUrls"`
	Info      struct {
		Title       string `json:"title,omitempty"`
		Version     string `json:"version,omitempty"`
		Description string `json:"description,omitempty"`
	} `json:"info"`
	Source string `json:"source"` // File path of the OpenAPI spec
}

// ParseOpenAPISpec parses an OpenAPI specification file and extracts all endpoints
func ParseOpenAPISpec(specPath string) (*DiscoveredEndpoints, error) {
	// Load the OpenAPI spec
	doc, err := loads.Spec(specPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load OpenAPI spec from %s: %w", specPath, err)
	}

	// Validate the spec (optional, some specs might have minor issues)
	// Note: Validation is optional as some specs might still be usable despite minor issues
	/*
		if err := doc.Spec().Validate(); err != nil {
			log.Printf("[WARNING] OpenAPI spec validation failed: %v", err)
			// Continue despite validation errors as some specs might still be usable
		}
	*/

	result := &DiscoveredEndpoints{
		Endpoints: []Endpoint{},
		BaseURLs:  []string{},
		Source:    specPath,
	}

	// Extract info
	if doc.Spec().Info != nil {
		result.Info.Title = doc.Spec().Info.Title
		result.Info.Version = doc.Spec().Info.Version
		result.Info.Description = doc.Spec().Info.Description
	}

	// Extract base URLs from servers
	baseURLs := extractBaseURLs(doc.Spec())
	result.BaseURLs = baseURLs

	// Extract endpoints from paths
	if doc.Spec().Paths != nil && doc.Spec().Paths.Paths != nil {
		for path, pathItem := range doc.Spec().Paths.Paths {
			endpoints := extractEndpointsFromPath(path, pathItem, baseURLs)
			result.Endpoints = append(result.Endpoints, endpoints...)
		}
	}

	// Sort endpoints by path and method for consistent output
	sort.Slice(result.Endpoints, func(i, j int) bool {
		if result.Endpoints[i].Path == result.Endpoints[j].Path {
			return result.Endpoints[i].Method < result.Endpoints[j].Method
		}
		return result.Endpoints[i].Path < result.Endpoints[j].Path
	})

	log.Printf("[OPENAPI] Discovered %d endpoints from %s", len(result.Endpoints), filepath.Base(specPath))
	return result, nil
}

// extractBaseURLs extracts base URLs from the OpenAPI spec
func extractBaseURLs(spec *spec.Swagger) []string {
	var baseURLs []string

	// Note: OpenAPI 3.x servers are not directly supported by go-openapi/spec
	// This library primarily supports Swagger 2.0 format
	// For OpenAPI 3.x support, we'd need a different library

	// OpenAPI 2.x (Swagger) host + schemes
	if spec.Host != "" {
		schemes := []string{"https", "http"} // Default schemes
		if len(spec.Schemes) > 0 {
			schemes = spec.Schemes
		}

		basePath := spec.BasePath
		if basePath == "" {
			basePath = "/"
		}

		for _, scheme := range schemes {
			baseURL := fmt.Sprintf("%s://%s%s", scheme, spec.Host, basePath)
			baseURLs = append(baseURLs, strings.TrimSuffix(baseURL, "/"))
		}
	}

	// Fallback to localhost if no host specified
	if len(baseURLs) == 0 {
		baseURLs = append(baseURLs, "http://localhost")
	}

	return baseURLs
}

// extractEndpointsFromPath extracts all HTTP methods for a given path
func extractEndpointsFromPath(path string, pathItem spec.PathItem, baseURLs []string) []Endpoint {
	var endpoints []Endpoint

	operations := map[string]*spec.Operation{
		"GET":     pathItem.Get,
		"POST":    pathItem.Post,
		"PUT":     pathItem.Put,
		"DELETE":  pathItem.Delete,
		"PATCH":   pathItem.Patch,
		"HEAD":    pathItem.Head,
		"OPTIONS": pathItem.Options,
	}

	for method, operation := range operations {
		if operation == nil {
			continue
		}

		endpoint := Endpoint{
			Path:        path,
			Method:      method,
			Summary:     operation.Summary,
			Description: operation.Description,
			Tags:        operation.Tags,
		}

		// Generate full URLs for each base URL
		if len(baseURLs) > 0 {
			endpoint.BaseURL = baseURLs[0] // Use first base URL as primary
			endpoint.FullURL = buildFullURL(baseURLs[0], path)
		}

		endpoints = append(endpoints, endpoint)
	}

	return endpoints
}

// buildFullURL constructs a full URL from base URL and path
func buildFullURL(baseURL, path string) string {
	// Parse base URL
	base, err := url.Parse(baseURL)
	if err != nil {
		return baseURL + path
	}

	// Parse path
	pathURL, err := url.Parse(path)
	if err != nil {
		return baseURL + path
	}

	// Resolve path relative to base
	fullURL := base.ResolveReference(pathURL)
	return fullURL.String()
}

// FindOpenAPISpecs searches for OpenAPI specification files in common locations
func FindOpenAPISpecs(rootDir string) ([]string, error) {
	var specs []string

	// Common OpenAPI spec file patterns
	patterns := []string{
		"*swagger*",
		"*openapi*",
		"*api*",
		"*spec*",
		"swagger.*",
		"openapi.*",
		"api.*",
		"spec.*",
	}

	extensions := []string{".yaml", ".yml", ".json"}

	for _, pattern := range patterns {
		for _, ext := range extensions {
			searchPattern := filepath.Join(rootDir, pattern+ext)
			matches, err := filepath.Glob(searchPattern)
			if err != nil {
				continue
			}
			specs = append(specs, matches...)
		}
	}

	// Remove duplicates
	uniqueSpecs := make(map[string]bool)
	var result []string
	for _, spec := range specs {
		if !uniqueSpecs[spec] {
			uniqueSpecs[spec] = true
			result = append(result, spec)
		}
	}

	return result, nil
}

// ValidateOpenAPIFile checks if a file appears to be an OpenAPI specification
func ValidateOpenAPIFile(filePath string) bool {
	doc, err := loads.Spec(filePath)
	if err != nil {
		return false
	}

	// Check for OpenAPI/Swagger indicators
	spec := doc.Spec()
	if spec == nil {
		return false
	}

	// Check for OpenAPI version or Swagger version
	return spec.Swagger != "" || (spec.Info != nil && spec.Info.Title != "")
}
