package governance

import "fmt"

// TransitionRule defines an allowed lifecycle transition.
type TransitionRule struct {
	From             LifecycleState
	To               LifecycleState
	RequiresApproval bool
}

// DefaultTransitions defines the allowed lifecycle state transitions.
var DefaultTransitions = []TransitionRule{
	{From: StateDraft, To: StateApproved, RequiresApproval: true},
	{From: StateApproved, To: StateDeprecated, RequiresApproval: false},
	{From: StateDeprecated, To: StateArchived, RequiresApproval: true},
	{From: StateApproved, To: StateArchived, RequiresApproval: true},
	{From: StateDeprecated, To: StateApproved, RequiresApproval: true},
	{From: StateArchived, To: StateDeprecated, RequiresApproval: true},
	{From: StateArchived, To: StateDraft, RequiresApproval: true},
}

// DisallowedTransitions are explicitly forbidden (return specific error).
var DisallowedTransitions = map[LifecycleState][]LifecycleState{
	StateDraft:    {StateDeprecated, StateArchived},
	StateArchived: {StateApproved},
}

// LifecycleMachine validates lifecycle state transitions.
type LifecycleMachine struct {
	transitions []TransitionRule
	disallowed  map[LifecycleState][]LifecycleState
}

// NewLifecycleMachine creates a machine with default rules.
func NewLifecycleMachine() *LifecycleMachine {
	return &LifecycleMachine{
		transitions: DefaultTransitions,
		disallowed:  DisallowedTransitions,
	}
}

// ValidateTransition checks if a transition from->to is allowed.
// Returns nil if allowed, an error with a machine-readable code if not.
func (m *LifecycleMachine) ValidateTransition(from, to LifecycleState) error {
	// Same state is a no-op, allow it.
	if from == to {
		return nil
	}

	// Check disallowed first.
	if disallowed, ok := m.disallowed[from]; ok {
		for _, d := range disallowed {
			if d == to {
				return &TransitionError{
					Code:    "LIFECYCLE_TRANSITION_DENIED",
					From:    from,
					To:      to,
					Message: fmt.Sprintf("transition from %s to %s is not allowed", from, to),
				}
			}
		}
	}

	// Check allowed transitions.
	for _, t := range m.transitions {
		if t.From == from && t.To == to {
			return nil
		}
	}

	return &TransitionError{
		Code:    "LIFECYCLE_INVALID_TRANSITION",
		From:    from,
		To:      to,
		Message: fmt.Sprintf("no transition defined from %s to %s", from, to),
	}
}

// RequiresApproval returns true if the transition requires approval.
func (m *LifecycleMachine) RequiresApproval(from, to LifecycleState) bool {
	for _, t := range m.transitions {
		if t.From == from && t.To == to {
			return t.RequiresApproval
		}
	}
	return true // default to requiring approval for unknown transitions
}

// AllowedTransitions returns all valid target states from the given state.
func (m *LifecycleMachine) AllowedTransitions(from LifecycleState) []LifecycleState {
	var allowed []LifecycleState
	for _, t := range m.transitions {
		if t.From == from {
			allowed = append(allowed, t.To)
		}
	}
	return allowed
}

// TransitionError is a structured error for invalid transitions.
type TransitionError struct {
	Code    string         `json:"code"`
	From    LifecycleState `json:"from"`
	To      LifecycleState `json:"to"`
	Message string         `json:"message"`
}

func (e *TransitionError) Error() string {
	return e.Message
}
