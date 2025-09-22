package state

import (
	"faultline/config"
	"sync"
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
	mu    sync.RWMutex
	rules map[string]Rule
}

// NewRuleState creates a new, thread-safe rule store.
// initialRules can be nil.
func NewRuleState(initialRules []config.Rule) *RuleState {
	rs := &RuleState{
		rules: make(map[string]Rule),
	}
	// This part is for potential future use where we might load initial rules from a file.
	// For now, it starts empty.
	return rs
}

// GetRules returns a slice of all current rules.
func (rs *RuleState) GetRules() []Rule {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	rules := make([]Rule, 0, len(rs.rules))
	for _, rule := range rs.rules {
		rules = append(rules, rule)
	}
	return rules
}

// AddRule adds a new rule to the store.
func (rs *RuleState) AddRule(rule Rule) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	rs.rules[rule.ID] = rule
}

// UpdateRule updates an existing rule. Returns false if the rule is not found.
func (rs *RuleState) UpdateRule(rule Rule) bool {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	if _, ok := rs.rules[rule.ID]; !ok {
		return false
	}
	rs.rules[rule.ID] = rule
	return true
}

// DeleteRule removes a rule by its ID. Returns false if the rule is not found.
func (rs *RuleState) DeleteRule(id string) bool {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	if _, ok := rs.rules[id]; !ok {
		return false
	}
	delete(rs.rules, id)
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
