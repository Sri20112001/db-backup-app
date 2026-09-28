package models

import "testing"

func TestCanTransition(t *testing.T) {
	tests := []struct {
		from BackupRunStatus
		to   BackupRunStatus
		want bool
	}{
		// Valid forward transitions
		{RunPending, RunRunning, true},
		{RunPending, RunCancelled, true},
		{RunRunning, RunUploading, true},
		{RunRunning, RunFailed, true},
		{RunRunning, RunCancelled, true},
		{RunUploading, RunVerifying, true},
		{RunUploading, RunFailed, true},
		{RunUploading, RunCancelled, true},
		{RunVerifying, RunCompleted, true},
		{RunVerifying, RunFailed, true},
		// A user cancel wins at every non-terminal stage, including
		// VERIFYING (the agent reports CANCELLED instead of COMPLETED).
		{RunVerifying, RunCancelled, true},

		// Terminal states cannot transition anywhere
		{RunCompleted, RunRunning, false},
		{RunCompleted, RunFailed, false},
		{RunCompleted, RunCancelled, false},
		{RunCompleted, RunPending, false},
		{RunFailed, RunRunning, false},
		{RunFailed, RunCompleted, false},
		{RunFailed, RunPending, false},
		{RunCancelled, RunRunning, false},
		{RunCancelled, RunCompleted, false},
		{RunCancelled, RunPending, false},

		// Cannot skip states
		{RunPending, RunUploading, false},
		{RunPending, RunVerifying, false},
		{RunPending, RunCompleted, false},
		{RunPending, RunFailed, false},
		{RunRunning, RunVerifying, false},
		{RunRunning, RunCompleted, false},
		{RunUploading, RunCompleted, false},
		{RunUploading, RunRunning, false},
	}

	for _, tt := range tests {
		got := CanTransition(tt.from, tt.to)
		if got != tt.want {
			t.Errorf("CanTransition(%s, %s) = %v, want %v", tt.from, tt.to, got, tt.want)
		}
	}
}
