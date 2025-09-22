package codeanalysis

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// EndpointUsage represents an API endpoint found in source code
type EndpointUsage struct {
	URL         string `json:"url"`
	Method      string `json:"method"`
	File        string `json:"file"`
	Line        int    `json:"line"`
	Context     string `json:"context"`
	Type        string `json:"type"` // "fetch", "axios", "request", etc.
	Description string `json:"description,omitempty"`
}

// CodeAnalysisResult contains discovered endpoints and metadata
type CodeAnalysisResult struct {
	Endpoints    []EndpointUsage `json:"endpoints"`
	Files        []string        `json:"files"`
	TotalLines   int             `json:"totalLines"`
	UniqueURLs   []string        `json:"uniqueUrls"`
	MethodCounts map[string]int  `json:"methodCounts"`
	Source       string          `json:"source"` // Directory analyzed
}

// AnalyzeDirectory scans a directory for JavaScript/React files and extracts API endpoints
func AnalyzeDirectory(rootDir string) (*CodeAnalysisResult, error) {
	result := &CodeAnalysisResult{
		Endpoints:    []EndpointUsage{},
		Files:        []string{},
		MethodCounts: make(map[string]int),
		Source:       rootDir,
	}

	// File extensions to analyze
	extensions := map[string]bool{
		".js":   true,
		".jsx":  true,
		".ts":   true,
		".tsx":  true,
		".vue":  true,
		".html": true,
	}

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-relevant files
		if info.IsDir() {
			// Skip common directories we don't want to analyze
			if shouldSkipDirectory(info.Name()) {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if file extension is relevant
		ext := strings.ToLower(filepath.Ext(path))
		if !extensions[ext] {
			return nil
		}

		// Analyze the file
		endpoints, err := analyzeFile(path)
		if err != nil {
			// Log error but continue with other files
			fmt.Printf("[WARNING] Failed to analyze %s: %v\n", path, err)
			return nil
		}

		if len(endpoints) > 0 {
			result.Files = append(result.Files, path)
			result.Endpoints = append(result.Endpoints, endpoints...)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory %s: %w", rootDir, err)
	}

	// Post-process results
	result.UniqueURLs = extractUniqueURLs(result.Endpoints)
	result.MethodCounts = countMethods(result.Endpoints)

	// Sort endpoints by URL for consistency
	sort.Slice(result.Endpoints, func(i, j int) bool {
		if result.Endpoints[i].URL == result.Endpoints[j].URL {
			return result.Endpoints[i].Method < result.Endpoints[j].Method
		}
		return result.Endpoints[i].URL < result.Endpoints[j].URL
	})

	return result, nil
}

// analyzeFile scans a single file for API endpoints
func analyzeFile(filePath string) ([]EndpointUsage, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var endpoints []EndpointUsage
	scanner := bufio.NewScanner(file)
	lineNumber := 0

	// Regular expressions for different API call patterns
	patterns := []struct {
		name    string
		pattern *regexp.Regexp
		method  string
	}{
		// fetch() calls
		{
			name:    "fetch",
			pattern: regexp.MustCompile(`fetch\s*\(\s*['"]([^'"]+)['"]`),
			method:  "GET", // Default, can be overridden
		},
		{
			name:    "fetch-method",
			pattern: regexp.MustCompile(`fetch\s*\(\s*['"]([^'"]+)['"].*method\s*:\s*['"]([^'"]+)['"]`),
			method:  "", // Will be extracted from match
		},
		// axios calls
		{
			name:    "axios-get",
			pattern: regexp.MustCompile(`axios\.get\s*\(\s*['"]([^'"]+)['"]`),
			method:  "GET",
		},
		{
			name:    "axios-post",
			pattern: regexp.MustCompile(`axios\.post\s*\(\s*['"]([^'"]+)['"]`),
			method:  "POST",
		},
		{
			name:    "axios-put",
			pattern: regexp.MustCompile(`axios\.put\s*\(\s*['"]([^'"]+)['"]`),
			method:  "PUT",
		},
		{
			name:    "axios-delete",
			pattern: regexp.MustCompile(`axios\.delete\s*\(\s*['"]([^'"]+)['"]`),
			method:  "DELETE",
		},
		// Template literals with URLs
		{
			name:    "template-url",
			pattern: regexp.MustCompile(`['"]https?://[^'"]+['"]`),
			method:  "GET",
		},
		// Card component endpoint props (specific to this app)
		{
			name:    "card-endpoint",
			pattern: regexp.MustCompile(`endpoint\s*=\s*['"]([^'"]+)['"]`),
			method:  "GET",
		},
	}

	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)

		// Skip comments and empty lines
		if strings.HasPrefix(trimmedLine, "//") || strings.HasPrefix(trimmedLine, "/*") || trimmedLine == "" {
			continue
		}

		// Check each pattern
		for _, p := range patterns {
			matches := p.pattern.FindAllStringSubmatch(line, -1)
			for _, match := range matches {
				if len(match) < 2 {
					continue
				}

				endpointURL := match[1]
				method := p.method

				// For fetch-method pattern, extract the method from the second capture group
				if p.name == "fetch-method" && len(match) >= 3 {
					method = strings.ToUpper(match[2])
				}

				// Validate and clean the URL
				if isValidEndpointURL(endpointURL) {
					endpoint := EndpointUsage{
						URL:     endpointURL,
						Method:  method,
						File:    filePath,
						Line:    lineNumber,
						Context: strings.TrimSpace(line),
						Type:    p.name,
					}

					// Add description based on context
					endpoint.Description = extractDescription(line, lineNumber, scanner)

					endpoints = append(endpoints, endpoint)
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Remove duplicates from the same file
	endpoints = deduplicateEndpoints(endpoints)

	return endpoints, nil
}

// shouldSkipDirectory determines if a directory should be skipped during analysis
func shouldSkipDirectory(dirName string) bool {
	skipDirs := map[string]bool{
		"node_modules": true,
		".git":         true,
		".next":        true,
		"dist":         true,
		"build":        true,
		"coverage":     true,
		".nyc_output":  true,
		"vendor":       true,
		".idea":        true,
		".vscode":      true,
	}
	return skipDirs[dirName]
}

// isValidEndpointURL checks if a string looks like a valid API endpoint
func isValidEndpointURL(s string) bool {
	// Must be a URL or start with http/https
	if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
		// Validate as proper URL
		_, err := url.Parse(s)
		return err == nil
	}

	// Allow relative paths that look like API endpoints
	if strings.HasPrefix(s, "/api/") || strings.HasPrefix(s, "/") {
		return len(s) > 1 && !strings.Contains(s, " ")
	}

	return false
}

// extractDescription tries to extract a meaningful description from the context
func extractDescription(line string, lineNumber int, scanner *bufio.Scanner) string {
	// Look for comments on the same line
	if commentIdx := strings.Index(line, "//"); commentIdx != -1 {
		comment := strings.TrimSpace(line[commentIdx+2:])
		if comment != "" {
			return comment
		}
	}

	// Look for JSDoc or block comments above the line
	// This is a simplified implementation
	if strings.Contains(line, "title") && strings.Contains(line, "description") {
		if titleMatch := regexp.MustCompile(`title\s*=\s*['"]([^'"]+)['"]`).FindStringSubmatch(line); len(titleMatch) > 1 {
			return titleMatch[1]
		}
		if descMatch := regexp.MustCompile(`description\s*=\s*['"]([^'"]+)['"]`).FindStringSubmatch(line); len(descMatch) > 1 {
			return descMatch[1]
		}
	}

	return ""
}

// deduplicateEndpoints removes duplicate endpoints from the same file
func deduplicateEndpoints(endpoints []EndpointUsage) []EndpointUsage {
	seen := make(map[string]bool)
	var result []EndpointUsage

	for _, ep := range endpoints {
		key := fmt.Sprintf("%s|%s|%s", ep.URL, ep.Method, ep.File)
		if !seen[key] {
			seen[key] = true
			result = append(result, ep)
		}
	}

	return result
}

// extractUniqueURLs extracts unique URLs from endpoints
func extractUniqueURLs(endpoints []EndpointUsage) []string {
	urlSet := make(map[string]bool)
	for _, ep := range endpoints {
		urlSet[ep.URL] = true
	}

	var urls []string
	for url := range urlSet {
		urls = append(urls, url)
	}

	sort.Strings(urls)
	return urls
}

// countMethods counts HTTP methods used
func countMethods(endpoints []EndpointUsage) map[string]int {
	counts := make(map[string]int)
	for _, ep := range endpoints {
		counts[ep.Method]++
	}
	return counts
}

// AnalyzeSpecificFiles analyzes only the specified files
func AnalyzeSpecificFiles(filePaths []string) (*CodeAnalysisResult, error) {
	result := &CodeAnalysisResult{
		Endpoints:    []EndpointUsage{},
		Files:        []string{},
		MethodCounts: make(map[string]int),
		Source:       "specific files",
	}

	for _, filePath := range filePaths {
		endpoints, err := analyzeFile(filePath)
		if err != nil {
			fmt.Printf("[WARNING] Failed to analyze %s: %v\n", filePath, err)
			continue
		}

		if len(endpoints) > 0 {
			result.Files = append(result.Files, filePath)
			result.Endpoints = append(result.Endpoints, endpoints...)
		}
	}

	// Post-process results
	result.UniqueURLs = extractUniqueURLs(result.Endpoints)
	result.MethodCounts = countMethods(result.Endpoints)

	return result, nil
}
