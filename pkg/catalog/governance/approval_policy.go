package governance

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// ApprovalPolicyFile is the top-level structure of the approval policies YAML file.
type ApprovalPolicyFile struct {
	Policies []ApprovalPolicy `yaml:"policies" json:"policies"`
}

// ApprovalPolicy defines a single approval rule.
type ApprovalPolicy struct {
	ID            string           `yaml:"id" json:"id"`
	DisplayName   string           `yaml:"displayName" json:"displayName"`
	Description   string           `yaml:"description" json:"description,omitempty"`
	Enabled       bool             `yaml:"enabled" json:"enabled"`
	Selector      PolicySelector   `yaml:"selector" json:"selector"`
	Gate          ApprovalGate     `yaml:"gate" json:"gate"`
	ExpiryHours   int              `yaml:"expiryHours" json:"expiryHours,omitempty"`
}

// PolicySelector determines which assets and transitions a policy applies to.
type PolicySelector struct {
	Plugins       []string         `yaml:"plugins,omitempty" json:"plugins,omitempty"`
	Kinds         []string         `yaml:"kinds,omitempty" json:"kinds,omitempty"`
	RiskLevels    []string         `yaml:"riskLevels,omitempty" json:"riskLevels,omitempty"`
	Transitions   []TransitionSpec `yaml:"transitions,omitempty" json:"transitions,omitempty"`
}

// TransitionSpec matches a specific lifecycle transition.
type TransitionSpec struct {
	From string `yaml:"from" json:"from"`
	To   string `yaml:"to" json:"to"`
}

// ApprovalGate defines how many approvals are needed and who can approve.
type ApprovalGate struct {
	RequiredCount int      `yaml:"requiredCount" json:"requiredCount"`
	AllowedRoles  []string `yaml:"allowedRoles,omitempty" json:"allowedRoles,omitempty"`
	DenyOnFirst   bool     `yaml:"denyOnFirst" json:"denyOnFirst,omitempty"`
}

// ApprovalEvaluator loads approval policies and evaluates them against transitions.
type ApprovalEvaluator struct {
	policies []ApprovalPolicy
}

// NewApprovalEvaluator creates an evaluator with the given policies.
func NewApprovalEvaluator(policies []ApprovalPolicy) *ApprovalEvaluator {
	return &ApprovalEvaluator{policies: policies}
}

// LoadApprovalPolicies loads policies from a YAML file.
// Returns an empty evaluator if the file does not exist.
func LoadApprovalPolicies(path string) (*ApprovalEvaluator, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return NewApprovalEvaluator(nil), nil
		}
		return nil, fmt.Errorf("read approval policies: %w", err)
	}

	var pf ApprovalPolicyFile
	if err := yaml.Unmarshal(data, &pf); err != nil {
		return nil, fmt.Errorf("parse approval policies: %w", err)
	}

	return NewApprovalEvaluator(pf.Policies), nil
}

// EvaluationResult describes whether an approval gate applies and its requirements.
type EvaluationResult struct {
	RequiresApproval bool
	PolicyID         string
	PolicyName       string
	RequiredCount    int
	AllowedRoles     []string
	ExpiryHours      int
}

// Evaluate checks all enabled policies against a given transition context.
// Returns the first matching policy's gate requirements, or a result with
// RequiresApproval=false if no policy matches.
func (e *ApprovalEvaluator) Evaluate(plugin, kind string, riskLevel string, from, to LifecycleState) EvaluationResult {
	for _, p := range e.policies {
		if !p.Enabled {
			continue
		}
		if !e.matchSelector(p.Selector, plugin, kind, riskLevel, from, to) {
			continue
		}
		return EvaluationResult{
			RequiresApproval: true,
			PolicyID:         p.ID,
			PolicyName:       p.DisplayName,
			RequiredCount:    p.Gate.RequiredCount,
			AllowedRoles:     p.Gate.AllowedRoles,
			ExpiryHours:      p.ExpiryHours,
		}
	}

	return EvaluationResult{RequiresApproval: false}
}

// matchSelector returns true if the selector matches the given context.
func (e *ApprovalEvaluator) matchSelector(sel PolicySelector, plugin, kind, riskLevel string, from, to LifecycleState) bool {
	if len(sel.Plugins) > 0 && !containsString(sel.Plugins, plugin) {
		return false
	}
	if len(sel.Kinds) > 0 && !containsString(sel.Kinds, kind) {
		return false
	}
	if len(sel.RiskLevels) > 0 && !containsString(sel.RiskLevels, riskLevel) {
		return false
	}
	if len(sel.Transitions) > 0 {
		matched := false
		for _, ts := range sel.Transitions {
			fromMatch := ts.From == "" || ts.From == "*" || ts.From == string(from)
			toMatch := ts.To == "" || ts.To == "*" || ts.To == string(to)
			if fromMatch && toMatch {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	return true
}

// EvaluateDecisions checks whether the current set of decisions meets the
// gate requirements, and returns the resolved status.
func (e *ApprovalEvaluator) EvaluateDecisions(policyID string, approveCount, denyCount int) ApprovalStatus {
	for _, p := range e.policies {
		if p.ID != policyID {
			continue
		}
		if p.Gate.DenyOnFirst && denyCount > 0 {
			return ApprovalStatusDenied
		}
		if approveCount >= p.Gate.RequiredCount {
			return ApprovalStatusApproved
		}
		return ApprovalStatusPending
	}
	// Policy not found; if we have at least 1 approval, approve.
	if approveCount > 0 {
		return ApprovalStatusApproved
	}
	return ApprovalStatusPending
}

// GetPolicy returns a policy by ID, or nil if not found.
func (e *ApprovalEvaluator) GetPolicy(id string) *ApprovalPolicy {
	for i := range e.policies {
		if e.policies[i].ID == id {
			return &e.policies[i]
		}
	}
	return nil
}

// ListPolicies returns all loaded policies.
func (e *ApprovalEvaluator) ListPolicies() []ApprovalPolicy {
	return e.policies
}

func containsString(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}
