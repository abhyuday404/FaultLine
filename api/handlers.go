package api

import (
	"encoding/json"
	"faultline/state"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// ApiHandler holds a reference to the shared rule state.
type ApiHandler struct {
	ruleState *state.RuleState
}

// NewApiHandler creates a new handler for the API.
func NewApiHandler(rs *state.RuleState) *ApiHandler {
	return &ApiHandler{ruleState: rs}
}

// GetRules returns the list of current failure rules as JSON.
func (h *ApiHandler) GetRules(w http.ResponseWriter, r *http.Request) {
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
