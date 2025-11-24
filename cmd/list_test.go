package cmd

import (
	"testing"
	"time"
)

func TestFormatTimeAgo(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		time     time.Time
		expected string
	}{
		{
			name:     "just now",
			time:     now.Add(-30 * time.Second),
			expected: "just now",
		},
		{
			name:     "1 minute ago",
			time:     now.Add(-1 * time.Minute),
			expected: "1 min ago",
		},
		{
			name:     "5 minutes ago",
			time:     now.Add(-5 * time.Minute),
			expected: "5 mins ago",
		},
		{
			name:     "1 hour ago",
			time:     now.Add(-1 * time.Hour),
			expected: "1 hour ago",
		},
		{
			name:     "3 hours ago",
			time:     now.Add(-3 * time.Hour),
			expected: "3 hours ago",
		},
		{
			name:     "1 day ago",
			time:     now.Add(-24 * time.Hour),
			expected: "1 day ago",
		},
		{
			name:     "3 days ago",
			time:     now.Add(-3 * 24 * time.Hour),
			expected: "3 days ago",
		},
		{
			name:     "1 week ago",
			time:     now.Add(-7 * 24 * time.Hour),
			expected: "1 week ago",
		},
		{
			name:     "2 weeks ago",
			time:     now.Add(-14 * 24 * time.Hour),
			expected: "2 weeks ago",
		},
		{
			name:     "1 month ago",
			time:     now.Add(-30 * 24 * time.Hour),
			expected: "1 month ago",
		},
		{
			name:     "3 months ago",
			time:     now.Add(-90 * 24 * time.Hour),
			expected: "3 months ago",
		},
		{
			name:     "1 year ago",
			time:     now.Add(-365 * 24 * time.Hour),
			expected: "1 year ago",
		},
		{
			name:     "2 years ago",
			time:     now.Add(-2 * 365 * 24 * time.Hour),
			expected: "2 years ago",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatTimeAgo(tt.time)
			if result != tt.expected {
				t.Errorf("formatTimeAgo() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "string shorter than max",
			input:    "short",
			maxLen:   10,
			expected: "short",
		},
		{
			name:     "string equal to max",
			input:    "exactly10!",
			maxLen:   10,
			expected: "exactly10!",
		},
		{
			name:     "string longer than max",
			input:    "this is a very long string",
			maxLen:   10,
			expected: "this is...",
		},
		{
			name:     "empty string",
			input:    "",
			maxLen:   10,
			expected: "",
		},
		{
			name:     "string with maxLen 3",
			input:    "test",
			maxLen:   3,
			expected: "...",
		},
		{
			name:     "long project name",
			input:    "github.com/organization/very-long-repository-name",
			maxLen:   30,
			expected: "github.com/organization/ver...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncate(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
			}
			// Verify result length doesn't exceed maxLen
			if len(result) > tt.maxLen {
				t.Errorf(
					"truncate(%q, %d) result length %d exceeds maxLen %d",
					tt.input,
					tt.maxLen,
					len(result),
					tt.maxLen,
				)
			}
		})
	}
}
