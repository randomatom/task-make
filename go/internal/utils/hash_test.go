package utils

import (
	"testing"
)

func TestGetChecksum(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantLen int
	}{
		{"empty_string", "", 8},
		{"simple_test", "test", 8},
		{"hello_world", "hello world", 8},
		{"numbers", "123456789", 8},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetChecksum(tt.input)
			if len(got) != tt.wantLen {
				t.Errorf("GetChecksum(%q) length = %d, want %d", tt.input, len(got), tt.wantLen)
			}
			got2 := GetChecksum(tt.input)
			if got != got2 {
				t.Errorf("GetChecksum(%q) not consistent: %s != %s", tt.input, got, got2)
			}
		})
	}
}

func TestGetChecksum_Uniqueness(t *testing.T) {
	input1 := "test_input_1"
	input2 := "test_input_2"

	hash1 := GetChecksum(input1)
	hash2 := GetChecksum(input2)

	if hash1 == hash2 {
		t.Errorf("Different inputs produced same hash: %s vs %s", hash1, hash2)
	}
}
