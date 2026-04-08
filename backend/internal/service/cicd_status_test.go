package service

import "testing"

func TestDefaultPipelineStatus(t *testing.T) {
	tests := []struct {
		name   string
		input  int8
		expect int8
	}{
		{name: "default zero becomes enabled", input: 0, expect: 1},
		{name: "disabled marker stays disabled", input: -1, expect: -1},
		{name: "enabled stays enabled", input: 1, expect: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := defaultPipelineStatus(tt.input)
			if got != tt.expect {
				t.Fatalf("expected %d, got %d", tt.expect, got)
			}
		})
	}
}
