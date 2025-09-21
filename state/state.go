package state

import (
	"faultline/config"
	"sync"

	"github.com/google/uuid"
)

// Failure matches the config.Failure struct but is used for in-memory state.
type Failure config.Failure

// Rule is the in-memory representation of a failure rule.
// It includes an ID and an enabled/disabled state.
type Rule struct {
	ID      string  `json:"id"`
	Target  string  `json:"target"`
	Failure Failure `json:"failure"`
	Enabled bool    `json:"enabled"`
}

// RuleState holds the current list of rules and a mutex for safe concurrent access.
type RuleState struct {
	sync.RWMutex
	rules []Rule
}

// NewRuleState creates a new, thread-safe rule store.
func NewRuleState(initialRules []config.Rule) *RuleState {
	rs := &RuleState{
		rules: make([]Rule, 0),
	}
	// Convert initial rules from config format to state format
	for _, r := range initialRules {
		rs.rules = append(rs.rules, Rule{
			ID:      uuid.New().String(),
			Target:  r.Target,
			Failure: Failure(r.Failure),
			Enabled: true, // Rules from YAML are enabled by default
		})
	}
	return rs
}

// GetRules returns a copy of the current rules for safe reading.
func (rs *RuleState) GetRules() []Rule {
	rs.RLock()
	defer rs.RUnlock()
	// Return a copy to prevent modification of the underlying slice
	rulesCopy := make([]Rule, len(rs.rules))
	copy(rulesCopy, rs.rules)
	return rulesCopy
}

// AddRule adds a new rule to the state.
func (rs *RuleState) AddRule(rule Rule) {
	rs.Lock()
	defer rs.Unlock()
	rs.rules = append(rs.rules, rule)
}

// UpdateRule updates an existing rule by its ID.
func (rs *RuleState) UpdateRule(updatedRule Rule) bool {
	rs.Lock()
	defer rs.Unlock()
	for i, r := range rs.rules {
		if r.ID == updatedRule.ID {
			rs.rules[i] = updatedRule
			return true
		}
	}
	return false // Not found
}

// DeleteRule removes a rule by its ID.
func (rs *RuleState) DeleteRule(id string) bool {
	rs.Lock()
	defer rs.Unlock()
	for i, r := range rs.rules {
		if r.ID == id {
			// Remove the element by slicing
			rs.rules = append(rs.rules[:i], rs.rules[i+1:]...)
			return true
		}
	}
	return false // Not found
}
