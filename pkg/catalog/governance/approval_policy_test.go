package governance

import (
	"testing"
)

func testPolicies() []ApprovalPolicy {
	return []ApprovalPolicy{
		{
			ID:          "high-risk",
			DisplayName: "High Risk Approval",
			Enabled:     true,
			Selector: PolicySelector{
				RiskLevels: []string{"high", "critical"},
				Transitions: []TransitionSpec{
					{From: "draft", To: "approved"},
					{From: "approved", To: "archived"},
				},
			},
			Gate: ApprovalGate{
				RequiredCount: 2,
				AllowedRoles:  []string{"admin"},
				DenyOnFirst:   false,
			},
			ExpiryHours: 168,
		},
		{
			ID:          "prod-gate",
			DisplayName: "Production Gate",
			Enabled:     true,
			Selector: PolicySelector{
				Transitions: []TransitionSpec{
					{From: "draft", To: "approved"},
				},
			},
			Gate: ApprovalGate{
				RequiredCount: 1,
				DenyOnFirst:   true,
			},
			ExpiryHours: 72,
		},
		{
			ID:          "disabled-policy",
			DisplayName: "Disabled Policy",
			Enabled:     false,
			Selector: PolicySelector{
				Transitions: []TransitionSpec{
					{From: "*", To: "archived"},
				},
			},
			Gate: ApprovalGate{
				RequiredCount: 1,
			},
		},
		{
			ID:          "plugin-specific",
			DisplayName: "Plugin Specific",
			Enabled:     true,
			Selector: PolicySelector{
				Plugins: []string{"mcp"},
				Kinds:   []string{"mcpserver"},
			},
			Gate: ApprovalGate{
				RequiredCount: 1,
			},
		},
	}
}

func TestApprovalEvaluator_Evaluate(t *testing.T) {
	eval := NewApprovalEvaluator(testPolicies())

	tests := []struct {
		name             string
		plugin           string
		kind             string
		riskLevel        string
		from             LifecycleState
		to               LifecycleState
		wantApproval     bool
		wantPolicyID     string
		wantRequiredCount int
	}{
		{
			name:             "high risk draft to approved matches high-risk policy",
			plugin:           "model",
			kind:             "model",
			riskLevel:        "high",
			from:             StateDraft,
			to:               StateApproved,
			wantApproval:     true,
			wantPolicyID:     "high-risk",
			wantRequiredCount: 2,
		},
		{
			name:             "high risk approved to archived matches high-risk policy",
			plugin:           "model",
			kind:             "model",
			riskLevel:        "high",
			from:             StateApproved,
			to:               StateArchived,
			wantApproval:     true,
			wantPolicyID:     "high-risk",
			wantRequiredCount: 2,
		},
		{
			name:             "medium risk draft to approved matches prod-gate",
			plugin:           "model",
			kind:             "model",
			riskLevel:        "medium",
			from:             StateDraft,
			to:               StateApproved,
			wantApproval:     true,
			wantPolicyID:     "prod-gate",
			wantRequiredCount: 1,
		},
		{
			name:         "approved to deprecated no gate",
			plugin:       "model",
			kind:         "model",
			riskLevel:    "medium",
			from:         StateApproved,
			to:           StateDeprecated,
			wantApproval: false,
		},
		{
			name:         "disabled policy does not match",
			plugin:       "model",
			kind:         "model",
			riskLevel:    "low",
			from:         StateApproved,
			to:           StateArchived,
			wantApproval: false,
		},
		{
			name:             "plugin-specific policy matches mcp mcpserver",
			plugin:           "mcp",
			kind:             "mcpserver",
			riskLevel:        "low",
			from:             StateDeprecated,
			to:               StateArchived,
			wantApproval:     true,
			wantPolicyID:     "plugin-specific",
			wantRequiredCount: 1,
		},
		{
			name:         "plugin-specific policy does not match other plugins",
			plugin:       "knowledge",
			kind:         "source",
			riskLevel:    "low",
			from:         StateDeprecated,
			to:           StateArchived,
			wantApproval: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := eval.Evaluate(tt.plugin, tt.kind, tt.riskLevel, tt.from, tt.to)
			if result.RequiresApproval != tt.wantApproval {
				t.Errorf("Evaluate() RequiresApproval = %v, want %v", result.RequiresApproval, tt.wantApproval)
			}
			if tt.wantApproval {
				if result.PolicyID != tt.wantPolicyID {
					t.Errorf("Evaluate() PolicyID = %q, want %q", result.PolicyID, tt.wantPolicyID)
				}
				if result.RequiredCount != tt.wantRequiredCount {
					t.Errorf("Evaluate() RequiredCount = %d, want %d", result.RequiredCount, tt.wantRequiredCount)
				}
			}
		})
	}
}

func TestApprovalEvaluator_EvaluateDecisions(t *testing.T) {
	eval := NewApprovalEvaluator(testPolicies())

	tests := []struct {
		name         string
		policyID     string
		approves     int
		denies       int
		wantStatus   ApprovalStatus
	}{
		{
			name:       "high-risk: 0 approvals, 0 denies -> pending",
			policyID:   "high-risk",
			approves:   0,
			denies:     0,
			wantStatus: ApprovalStatusPending,
		},
		{
			name:       "high-risk: 1 approval, 0 denies -> pending (need 2)",
			policyID:   "high-risk",
			approves:   1,
			denies:     0,
			wantStatus: ApprovalStatusPending,
		},
		{
			name:       "high-risk: 2 approvals, 0 denies -> approved",
			policyID:   "high-risk",
			approves:   2,
			denies:     0,
			wantStatus: ApprovalStatusApproved,
		},
		{
			name:       "high-risk: 1 approval, 1 deny -> pending (denyOnFirst=false)",
			policyID:   "high-risk",
			approves:   1,
			denies:     1,
			wantStatus: ApprovalStatusPending,
		},
		{
			name:       "prod-gate: 1 approval -> approved",
			policyID:   "prod-gate",
			approves:   1,
			denies:     0,
			wantStatus: ApprovalStatusApproved,
		},
		{
			name:       "prod-gate: 0 approvals, 1 deny -> denied (denyOnFirst=true)",
			policyID:   "prod-gate",
			approves:   0,
			denies:     1,
			wantStatus: ApprovalStatusDenied,
		},
		{
			name:       "prod-gate: 1 approval, 1 deny -> denied (denyOnFirst=true)",
			policyID:   "prod-gate",
			approves:   1,
			denies:     1,
			wantStatus: ApprovalStatusDenied,
		},
		{
			name:       "unknown policy: 1 approval -> approved (fallback)",
			policyID:   "nonexistent",
			approves:   1,
			denies:     0,
			wantStatus: ApprovalStatusApproved,
		},
		{
			name:       "unknown policy: 0 approvals -> pending (fallback)",
			policyID:   "nonexistent",
			approves:   0,
			denies:     0,
			wantStatus: ApprovalStatusPending,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := eval.EvaluateDecisions(tt.policyID, tt.approves, tt.denies)
			if got != tt.wantStatus {
				t.Errorf("EvaluateDecisions(%q, %d, %d) = %q, want %q", tt.policyID, tt.approves, tt.denies, got, tt.wantStatus)
			}
		})
	}
}

func TestApprovalEvaluator_NoPolices(t *testing.T) {
	eval := NewApprovalEvaluator(nil)
	result := eval.Evaluate("model", "model", "high", StateDraft, StateApproved)
	if result.RequiresApproval {
		t.Error("expected no approval required when no policies loaded")
	}
}

func TestApprovalEvaluator_GetPolicy(t *testing.T) {
	eval := NewApprovalEvaluator(testPolicies())

	p := eval.GetPolicy("high-risk")
	if p == nil {
		t.Fatal("expected to find high-risk policy")
	}
	if p.ID != "high-risk" {
		t.Errorf("GetPolicy returned %q, want %q", p.ID, "high-risk")
	}

	p = eval.GetPolicy("nonexistent")
	if p != nil {
		t.Error("expected nil for nonexistent policy")
	}
}

func TestApprovalEvaluator_ListPolicies(t *testing.T) {
	policies := testPolicies()
	eval := NewApprovalEvaluator(policies)
	got := eval.ListPolicies()
	if len(got) != len(policies) {
		t.Errorf("ListPolicies returned %d, want %d", len(got), len(policies))
	}
}

func TestPolicySelector_WildcardTransition(t *testing.T) {
	eval := NewApprovalEvaluator([]ApprovalPolicy{
		{
			ID:      "wildcard",
			Enabled: true,
			Selector: PolicySelector{
				Transitions: []TransitionSpec{
					{From: "*", To: "archived"},
				},
			},
			Gate: ApprovalGate{RequiredCount: 1},
		},
	})

	// Any state -> archived should match.
	for _, from := range []LifecycleState{StateDraft, StateApproved, StateDeprecated} {
		result := eval.Evaluate("any", "any", "low", from, StateArchived)
		if !result.RequiresApproval {
			t.Errorf("wildcard from=%s to=archived should require approval", from)
		}
	}

	// Should not match non-archived targets.
	result := eval.Evaluate("any", "any", "low", StateDraft, StateApproved)
	if result.RequiresApproval {
		t.Error("wildcard to=archived should not match to=approved")
	}
}

func TestPolicySelector_EmptyTransitionMatchesAll(t *testing.T) {
	eval := NewApprovalEvaluator([]ApprovalPolicy{
		{
			ID:      "catch-all",
			Enabled: true,
			Selector: PolicySelector{
				RiskLevels: []string{"critical"},
			},
			Gate: ApprovalGate{RequiredCount: 3},
		},
	})

	// Any transition with critical risk should match (no transition spec = matches all).
	result := eval.Evaluate("any", "any", "critical", StateApproved, StateDeprecated)
	if !result.RequiresApproval {
		t.Error("expected critical risk catch-all to require approval")
	}
	if result.RequiredCount != 3 {
		t.Errorf("expected requiredCount=3, got %d", result.RequiredCount)
	}

	// Non-critical should not match.
	result = eval.Evaluate("any", "any", "medium", StateApproved, StateDeprecated)
	if result.RequiresApproval {
		t.Error("expected non-critical to not require approval")
	}
}
