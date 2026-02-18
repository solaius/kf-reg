package jobs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRefreshJobTableName(t *testing.T) {
	j := RefreshJob{}
	assert.Equal(t, "refresh_jobs", j.TableName())
}

func TestRefreshJobIsTerminal(t *testing.T) {
	tests := []struct {
		state    JobState
		terminal bool
	}{
		{JobStateQueued, false},
		{JobStateRunning, false},
		{JobStateSucceeded, true},
		{JobStateFailed, true},
		{JobStateCanceled, true},
	}

	for _, tc := range tests {
		t.Run(string(tc.state), func(t *testing.T) {
			j := &RefreshJob{State: tc.state}
			assert.Equal(t, tc.terminal, j.IsTerminal())
		})
	}
}
