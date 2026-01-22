package jobq

import (
	"testing"
	"time"
)

func TestWithRetryInterval(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected time.Duration
	}{
		{
			name:     "set 2 seconds retry interval",
			duration: time.Second * 2,
			expected: time.Second * 2,
		},
		{
			name:     "set 500ms retry interval",
			duration: time.Millisecond * 500,
			expected: time.Millisecond * 500,
		},
		{
			name:     "set zero retry interval",
			duration: time.Duration(0),
			expected: time.Duration(0),
		},
		{
			name:     "set negative seconds retry interval",
			duration: time.Second * -1,
			expected: time.Second * -1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := NewOptions(WithRetryInterval(tt.duration))
			if opts.retryInterval != tt.expected {
				t.Errorf("WithRetryInterval() expected %v, got %v", tt.expected, opts.retryInterval)
			}
		})
	}
}
