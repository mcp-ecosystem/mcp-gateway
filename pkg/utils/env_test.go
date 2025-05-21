package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMapToEnvList(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]string
		expected []string
	}{
		{
			name:     "empty map",
			input:    map[string]string{},
			expected: []string{},
		},
		{
			name: "single key-value pair",
			input: map[string]string{
				"KEY": "value",
			},
			expected: []string{"KEY=value"},
		},
		{
			name: "multiple key-value pairs",
			input: map[string]string{
				"KEY1": "value1",
				"KEY2": "value2",
				"KEY3": "value3",
			},
			expected: []string{"KEY1=value1", "KEY2=value2", "KEY3=value3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MapToEnvList(tt.input)
			assert.ElementsMatch(t, tt.expected, result)
		})
	}
}
