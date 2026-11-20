package main

import "testing"

func TestCanTransition(t *testing.T) {
	tests := []struct {
		name    string
		current JobStatus
		next    JobStatus
		want    bool
	}{
		{
			name:    "Queued to Assigned",
			current: JobStatusQueued,
			next:    JobStatusAssigned,
			want:    true,
		},
		{
			name:    "Assigned to Running",
			current: JobStatusAssigned,
			next:    JobStatusRunning,
			want:    true,
		},
		{
			name:    "Running to Completed",
			current: JobStatusRunning,
			next:    JobStatusCompleted,
			want:    true,
		},
		{
			name:    "Running to Failed",
			current: JobStatusRunning,
			next:    JobStatusFailed,
			want:    true,
		},
		{
			name:    "Queued to Running",
			current: JobStatusQueued,
			next:    JobStatusRunning,
			want:    false,
		},
		{
			name:    "Queued to Completed",
			current: JobStatusQueued,
			next:    JobStatusCompleted,
			want:    false,
		},
		{
			name:    "Completed to Running",
			current: JobStatusCompleted,
			next:    JobStatusRunning,
			want:    false,
		},
		{
			name:    "Failed to Completed",
			current: JobStatusFailed,
			next:    JobStatusCompleted,
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := canTransition(tt.current, tt.next)

			if got != tt.want {
				t.Errorf(
					"canTransition(%s, %s) = %v, want %v",
					tt.current,
					tt.next,
					got,
					tt.want,
				)
			}
		})
	}
}
