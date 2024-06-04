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
			name:     "valid qase id",
			input:    "QASE-123",
			expected: 123,
		},
		{
			name:     "invalid qase id",
			input:    "QASE-abc",
			expected: 0,
		},
		{
			name:     "empty input",
			input:    "",
			expected: 0,
		},
		{
			name:     "no qase id",
			input:    "QASE-",
			expected: 0,
		},
		{
			name:     "multiple qase id",
			input:    "QASE-123/Halohalo_QASE-456",
			expected: 456,
		},
		{
			name:     "QASE-123 multiple qase id",
			input:    "QASE-123/Halohalo_QASE-456",
			expected: 456,
		},
		{
			name:     "QASE-123/QASE-456 multiple qase id",
			input:    "QASE-123/Halohalo_QASE-456",
			expected: 456,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := parseQaseId(tc.input)
			require.Nil(t, err)
			require.Equal(t, tc.expected, actual)
		})
	}
}
