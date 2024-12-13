package main

import (
	"testing"
	"time"

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
			expected: 456,
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

func TestParseQaseIds(t *testing.T) {
	testcases := []struct {
		name     string
		input    string
		expected []int
	}{
		{
			name:     "Valid Qase ID",
			input:    "QASE-123",
			expected: []int{123},
		},
		{
			name:     "Invalid Qase ID",
			input:    "QASE-abc",
			expected: []int{},
		},
		{
			name:     "Empty input",
			input:    "",
			expected: []int{},
		},
		{
			name:     "No Qase ID",
			input:    "QASE-",
			expected: []int{},
		},
		{
			name:     "Will choose all valid Qase ID on multiple Qase ID - 1",
			input:    "QASE-123/Halohalo_QASE-456",
			expected: []int{123, 456},
		},
		{
			name:     "Will choose all valid Qase ID on multiple Qase ID - 2",
			input:    "QASE-123/QASE-456Halohalo_QASE-789",
			expected: []int{123, 456, 789},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := ParseQaseIds(tc.input)
			require.Nil(t, err)
			require.Equal(t, tc.expected, actual)
		})
	}
}

func TestProcessLineWithMultipleId(t *testing.T) {
	testcases := []struct {
		name     string
		input    string
		expected []ReportResult
	}{
		{
			name:  "Valid pass line with multiple Qase IDs",
			input: `{"Time":"2024-05-27T19:58:10.573629+07:00","Action":"pass","Package":"github.com/test","Test":"TestCase/QASE-123/QASE-456","Elapsed":10}`,
			expected: []ReportResult{
				{
					TestCaseId: 123,
					Status:     TEST_CASE_RESULT_STATUS_PASSED,
					Package:    "github.com/test",
					Time:       time.Date(2024, 5, 27, 12, 58, 10, 573629000, time.UTC),
					TimeMs:     10000,
				},
				{
					TestCaseId: 456,
					Status:     TEST_CASE_RESULT_STATUS_PASSED,
					Package:    "github.com/test",
					Time:       time.Date(2024, 5, 27, 12, 58, 10, 573629000, time.UTC),
					TimeMs:     10000,
				},
			},
		},
		{
			name:  "Valid fail line with multiple Qase IDs",
			input: `{"Time":"2024-05-27T19:58:10.573629+07:00","Action":"fail","Package":"github.com/test","Test":"TestCase/QASE-123/QASE-456"}`,
			expected: []ReportResult{
				{
					TestCaseId: 123,
					Status:     TEST_CASE_RESULT_STATUS_FAILED,
					Package:    "github.com/test",
					Time:       time.Date(2024, 5, 27, 12, 58, 10, 573629000, time.UTC),
				},
				{
					TestCaseId: 456,
					Status:     TEST_CASE_RESULT_STATUS_FAILED,
					Package:    "github.com/test",
					Time:       time.Date(2024, 5, 27, 12, 58, 10, 573629000, time.UTC),
				},
			},
		},
		{
			name:     "Invalid JSON line",
			input:    "invalid json",
			expected: nil,
		},
		{
			name:     "No test name",
			input:    `{"Time":"2024-05-27T19:58:10.573629+07:00","Action":"pass","Package":"github.com/test"}`,
			expected: nil,
		},
		{
			name:     "No Qase IDs in test name",
			input:    `{"Time":"2024-05-27T19:58:10.573629+07:00","Action":"pass","Package":"github.com/test","Test":"TestCase"}`,
			expected: []ReportResult{},
		},
		{
			name:     "Invalid action",
			input:    `{"Time":"2024-05-27T19:58:10.573629+07:00","Action":"invalid","Package":"github.com/test","Test":"TestCase/QASE-123"}`,
			expected: nil,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := processLineWithMultipleId(tc.input)
			if tc.expected == nil {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, len(tc.expected), len(actual))

			for i := range tc.expected {
				require.Equal(t, tc.expected[i].TestCaseId, actual[i].TestCaseId)
				require.Equal(t, tc.expected[i].Status, actual[i].Status)
				require.Equal(t, tc.expected[i].Package, actual[i].Package)
				require.Equal(t, tc.expected[i].Time.Unix(), actual[i].Time.Unix())
			}
		})
	}
}
