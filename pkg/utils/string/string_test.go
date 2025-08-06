package string

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContainsStringV2(t *testing.T) {
	strArray := []string{"apple", "banana", "cherry"}

	assert.True(t, ContainsStringV2("banana", strArray))
	assert.False(t, ContainsStringV2("grape", strArray))
}

func TestStringsDiffFunc(t *testing.T) {
	ps1 := []string{"apple", "banana", "cherry", "date"}
	ps2 := []string{"banana", "cherry", "date", "elderberry"}

	result := StringsDiffFunc(ps1, ps2)

	assert.Equal(t, []string{"apple"}, result)
}

func TestStringsDiffFunc_NoDifference(t *testing.T) {
	ps1 := []string{"apple", "banana", "cherry", "date"}
	ps2 := []string{"apple", "banana", "cherry", "date"}

	result := StringsDiffFunc(ps1, ps2)

	assert.Empty(t, result)
}

func TestStringsDiffFunc_AllDifferent(t *testing.T) {
	ps1 := []string{"apple", "banana", "cherry", "date"}
	ps2 := []string{"elderberry", "fig", "grape", "honeydew"}

	result := StringsDiffFunc(ps1, ps2)

	assert.Equal(t, ps1, result)
}

func TestRandSeq(t *testing.T) {
	length := 10
	result := RandSeq(length)

	assert.Equal(t, length, len(result))

	for _, char := range result {
		assert.Contains(t, "0123456789abcdefghijklmnopqrstuvwxyz", string(char))
	}
}

func TestCompareStringSlice(t *testing.T) {
	testCases := []struct {
		name     string
		old      []string
		new      []string
		expected bool
	}{
		{
			name:     "Case 1: Identical slices",
			old:      []string{"apple", "banana", "cherry"},
			new:      []string{"apple", "banana", "cherry"},
			expected: true,
		},
		{
			name:     "Case 2: Different order",
			old:      []string{"apple", "banana", "cherry"},
			new:      []string{"banana", "cherry", "apple"},
			expected: true,
		},
		{
			name:     "Case 3: Different slices",
			old:      []string{"apple", "banana", "cherry"},
			new:      []string{"apple", "banana", "grape"},
			expected: false,
		},
		{
			name:     "Case 4: Different length",
			old:      []string{"apple", "banana", "cherry"},
			new:      []string{"apple", "banana"},
			expected: false,
		},
		{
			name:     "Case 5: Empty slices",
			old:      []string{},
			new:      []string{},
			expected: true,
		},
		{
			name:     "Case 6: One empty slice",
			old:      []string{"apple", "banana", "cherry"},
			new:      []string{},
			expected: false,
		},
		{
			name:     "Case 7: Case sensitivity",
			old:      []string{"Apple", "Banana", "Cherry"},
			new:      []string{"apple", "banana", "cherry"},
			expected: false,
		},
		{
			name:     "Case 8: Duplicates in one slice",
			old:      []string{"apple", "banana", "banana"},
			new:      []string{"apple", "banana", "cherry"},
			expected: false,
		},
		{
			name:     "Case 9: Duplicates in both slices",
			old:      []string{"apple", "banana", "banana"},
			new:      []string{"apple", "banana", "banana"},
			expected: true,
		},
		{
			name:     "Case 10: Single element slices",
			old:      []string{"apple"},
			new:      []string{"apple"},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := CompareStringSlice(tc.old, tc.new)
			assert.Equal(t, tc.expected, result)
		})
	}
}
