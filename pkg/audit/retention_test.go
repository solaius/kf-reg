package audit

import (
	"testing"
)

func TestNewRetentionWorker(t *testing.T) {
	// Test that the worker is created with correct parameters.
	worker := NewRetentionWorker(nil, 30, nil)

	if worker == nil {
		t.Fatal("expected non-nil worker")
	}

	expectedRetention := 30 * 24 // hours
	actualHours := int(worker.retention.Hours())
	if actualHours != expectedRetention {
		t.Errorf("expected retention %d hours, got %d", expectedRetention, actualHours)
	}

	expectedInterval := 24 // hours
	actualIntervalHours := int(worker.interval.Hours())
	if actualIntervalHours != expectedInterval {
		t.Errorf("expected interval %d hours, got %d", expectedInterval, actualIntervalHours)
	}
}

func TestNewRetentionWorker_ZeroRetention(t *testing.T) {
	// Worker with zero retention should be disabled (Run returns immediately).
	worker := NewRetentionWorker(nil, 0, nil)

	if worker == nil {
		t.Fatal("expected non-nil worker")
	}

	if worker.retention != 0 {
		t.Errorf("expected zero retention, got %v", worker.retention)
	}
}
