package state

import (
	"encoding/json"
	"faultline/config"
	"os"
	"sort"
	"sync"
	"time"
)

// Rule defines the structure for a failure rule, including JSON tags for API communication.
type Rule struct {
	ID       string  `json:"id"`
	Target   string  `json:"target"`
	Failure  Failure `json:"failure"`
	Enabled  bool    `json:"enabled"`
	Category string  `json:"category,omitempty"` // e.g., "api" | "database"
}

// Failure defines the specifics of a failure, using camelCase JSON tags.
type Failure struct {
	Type      string `json:"type"`
	LatencyMs int    `json:"latencyMs,omitempty"`
	ErrorCode int    `json:"errorCode,omitempty"`
}

// RuleState holds the current set of rules in a thread-safe manner.
type RuleState struct {
	mu          sync.RWMutex
	rules       map[string]Rule
	dataFile    string    // Path to persistent storage file
	fileModTime time.Time // Last modification time of the data file
}

// NewRuleState creates a new, thread-safe rule store.
// initialRules can be nil. dataFile specifies where to persist rules.
func NewRuleState(initialRules []config.Rule, dataFile string) *RuleState {
	rs := &RuleState{
		rules:    make(map[string]Rule),
		dataFile: dataFile,
	}

	// Load rules from file if it exists
	if dataFile != "" {
		rs.loadFromFile()
	}

	return rs
}

// loadFromFile loads rules from the persistent storage file
func (rs *RuleState) loadFromFile() error {
	fileInfo, err := os.Stat(rs.dataFile)
	if os.IsNotExist(err) {
		// File doesn't exist, start with empty rules
		return nil
	}
	if err != nil {
		return err
	}

	data, err := os.ReadFile(rs.dataFile)
	if err != nil {
		return err
	}

	var rules []Rule
	if err := json.Unmarshal(data, &rules); err != nil {
		return err
	}

	rs.mu.Lock()
	defer rs.mu.Unlock()

	// Update modification time
	rs.fileModTime = fileInfo.ModTime()
	// Clear existing rules and load from file
	rs.rules = make(map[string]Rule)
	for _, rule := range rules {
		rs.rules[rule.ID] = rule
	}

	return nil
}

// saveToFile saves the current rules to the persistent storage file
func (rs *RuleState) saveToFile() error {
	if rs.dataFile == "" {
		return nil // No file specified, skip saving
	}

	rules := rs.getRulesInternal()
	data, err := json.MarshalIndent(rules, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(rs.dataFile, data, 0644)
}

// getRulesInternal returns rules without locking (internal use)
func (rs *RuleState) getRulesInternal() []Rule {
	rules := make([]Rule, 0, len(rs.rules))
	for _, rule := range rs.rules {
		rules = append(rules, rule)
	}

	// Sort rules by ID to ensure consistent ordering
	sort.Slice(rules, func(i, j int) bool {
		return rules[i].ID < rules[j].ID
	})

	return rules
}

// GetRules returns a slice of all current rules in consistent order (sorted by ID).
func (rs *RuleState) GetRules() []Rule {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	return rs.getRulesInternal()
}

// AddRule adds a new rule to the store and persists to file.
func (rs *RuleState) AddRule(rule Rule) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	rs.rules[rule.ID] = rule
	rs.saveToFile() // Auto-save after adding
}

// UpdateRule updates an existing rule and persists to file. Returns false if the rule is not found.
func (rs *RuleState) UpdateRule(rule Rule) bool {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	if _, ok := rs.rules[rule.ID]; !ok {
		return false
	}
	rs.rules[rule.ID] = rule
	rs.saveToFile() // Auto-save after updating
	return true
}

// DeleteRule removes a rule by its ID and persists to file. Returns false if the rule is not found.
func (rs *RuleState) DeleteRule(id string) bool {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	if _, ok := rs.rules[id]; !ok {
		return false
	}
	delete(rs.rules, id)
	rs.saveToFile() // Auto-save after deleting
	return true
}

// FindRuleForTarget checks if any enabled rule matches the given target URL.
func (rs *RuleState) FindRuleForTarget(targetURL string) (*Rule, bool) {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	for _, rule := range rs.rules {
		// A rule matches if it's enabled and its target is a prefix of the request URL.
		if rule.Enabled && len(rule.Target) > 0 && len(targetURL) >= len(rule.Target) && targetURL[:len(rule.Target)] == rule.Target {
			// Return a copy of the rule to prevent data races.
			r := rule
			return &r, true
		}
	}
	return nil, false
}

// CheckAndReloadIfModified checks if the data file has been modified since last load
// and reloads the rules if necessary. This is used by the proxy to detect CLI changes.
func (rs *RuleState) CheckAndReloadIfModified() error {
	if rs.dataFile == "" {
		return nil // No file to check
	}

	fileInfo, err := os.Stat(rs.dataFile)
	if os.IsNotExist(err) {
		return nil // File doesn't exist
	}
	if err != nil {
		return err
	}

	// Check if file has been modified
	if fileInfo.ModTime().After(rs.fileModTime) {
		return rs.loadFromFile()
	}

	return nil
}
