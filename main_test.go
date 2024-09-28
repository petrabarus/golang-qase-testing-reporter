package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseQaseId(t *testing.T) {
	testcases := []struct {
		name     string
		input    string
		expected int
	}{
		{
			name:     "QASE-1 Valid Qase ID",
			input:    "QASE-123",
			expected: 123,
		},
		{
			name:     "QASE-3 Invalid Qase ID",
			input:    "QASE-abc",
			expected: 0,
		},
		{
			name:     "QASE-4 Empty input",
			input:    "",
			expected: 0,
		},
		{
			name:     "QASE-5 No Qase ID",
			input:    "QASE-",
			expected: 0,
		},
		{
			name:     "QASE-6 Will choose the last Qase ID on multiple Qase ID - 1",
			input:    "QASE-123/Halohalo_QASE-456",
			expected: 123,
		},
		{
			name:     "QASE-7 Will choose the last Qase ID on multiple Qase ID - 2",
			input:    "QASE-123/Halohalo_QASE-456",
			expected: 456,
		},
		{
			name:     "QASE-8 Will choose the last Qase ID on multiple Qase ID - 3",
			input:    "QASE-123/Halohalo_QASE-789",
			expected: 789,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := ParseQaseId(tc.input)
			require.Nil(t, err)
			require.Equal(t, tc.expected, actual)
		})
	}
}
