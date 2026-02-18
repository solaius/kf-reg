package governance

import "testing"

func TestLifecycleMachine_ValidateTransition(t *testing.T) {
	m := NewLifecycleMachine()

	tests := []struct {
		name    string
		from    LifecycleState
		to      LifecycleState
		wantErr bool
		errCode string
	}{
		// Valid transitions
		{"draft to approved", StateDraft, StateApproved, false, ""},
		{"approved to deprecated", StateApproved, StateDeprecated, false, ""},
		{"deprecated to archived", StateDeprecated, StateArchived, false, ""},
		{"approved to archived", StateApproved, StateArchived, false, ""},
		{"deprecated to approved", StateDeprecated, StateApproved, false, ""},
		{"archived to deprecated", StateArchived, StateDeprecated, false, ""},
		{"archived to draft", StateArchived, StateDraft, false, ""},
		{"same state no-op", StateDraft, StateDraft, false, ""},

		// Denied transitions
		{"draft to deprecated denied", StateDraft, StateDeprecated, true, "LIFECYCLE_TRANSITION_DENIED"},
		{"draft to archived denied", StateDraft, StateArchived, true, "LIFECYCLE_TRANSITION_DENIED"},
		{"archived to approved denied", StateArchived, StateApproved, true, "LIFECYCLE_TRANSITION_DENIED"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := m.ValidateTransition(tt.from, tt.to)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTransition(%s, %s) error = %v, wantErr %v", tt.from, tt.to, err, tt.wantErr)
			}
			if tt.wantErr && tt.errCode != "" {
				te, ok := err.(*TransitionError)
				if !ok {
					t.Errorf("expected TransitionError, got %T", err)
				} else if te.Code != tt.errCode {
					t.Errorf("expected code %s, got %s", tt.errCode, te.Code)
				}
			}
		})
	}
}

func TestLifecycleMachine_RequiresApproval(t *testing.T) {
	m := NewLifecycleMachine()

	tests := []struct {
		name string
		from LifecycleState
		to   LifecycleState
		want bool
	}{
		{"draft to approved requires approval", StateDraft, StateApproved, true},
		{"approved to deprecated no approval", StateApproved, StateDeprecated, false},
		{"deprecated to archived requires approval", StateDeprecated, StateArchived, true},
		{"approved to archived requires approval", StateApproved, StateArchived, true},
		{"deprecated to approved requires approval", StateDeprecated, StateApproved, true},
		{"archived to deprecated requires approval", StateArchived, StateDeprecated, true},
		{"archived to draft requires approval", StateArchived, StateDraft, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := m.RequiresApproval(tt.from, tt.to)
			if got != tt.want {
				t.Errorf("RequiresApproval(%s, %s) = %v, want %v", tt.from, tt.to, got, tt.want)
			}
		})
	}
}

func TestLifecycleMachine_AllowedTransitions(t *testing.T) {
	m := NewLifecycleMachine()

	tests := []struct {
		name     string
		from     LifecycleState
		expected int
	}{
		{"draft has 1 transition", StateDraft, 1},       // only to approved
		{"approved has 2 transitions", StateApproved, 2}, // deprecated or archived
		{"deprecated has 2 transitions", StateDeprecated, 2}, // archived or approved
		{"archived has 2 transitions", StateArchived, 2},     // deprecated or draft
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := m.AllowedTransitions(tt.from)
			if len(got) != tt.expected {
				t.Errorf("AllowedTransitions(%s) = %d states, want %d (got: %v)", tt.from, len(got), tt.expected, got)
			}
		})
	}
}

func TestTransitionError_Error(t *testing.T) {
	err := &TransitionError{
		Code:    "LIFECYCLE_TRANSITION_DENIED",
		From:    StateDraft,
		To:      StateDeprecated,
		Message: "transition from draft to deprecated is not allowed",
	}
	want := "transition from draft to deprecated is not allowed"
	if got := err.Error(); got != want {
		t.Errorf("TransitionError.Error() = %q, want %q", got, want)
	}
}
