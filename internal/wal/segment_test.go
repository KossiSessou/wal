package wal

import "testing"

func TestParseSegmentName(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		wantID uint64
		wantOk bool
	}{
		{"normal", "0000000001.wal", 1, true},
		{"big number", "9999999999.wal", 9999999999, true},
		{"missing extension", "0000000001", 0, false},
		{"wrong extension", "0000000001.log", 0, false},
		{"empty", "", 0, false},
		{"non-numeric", "abc.wal", 0, false},
		{"negative looking", "-1.wal", 0, false},
	}

	for _, tc := range tests {

		got, ok := parseSegmentName(tc.input)

		if got != tc.wantID || ok != tc.wantOk {
			t.Errorf("parseSegment(%q) = (%d, %v); expected (%d, %v)", tc.name, got, ok, tc.wantID, tc.wantOk)
		}
	}
}
