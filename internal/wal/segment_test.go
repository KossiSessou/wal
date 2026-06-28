package wal

import (
	"os"
	"path/filepath"
	"testing"
)

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

func TestFormatSegmentName(t *testing.T) {

	tests := []struct {
		name      string
		id        uint64
		wantValue string
	}{
		{"normal", 1, "0000000001.wal"},
	}

	for _, tc := range tests {
		got := formatSegmentName(tc.id)

		if got != tc.wantValue {
			t.Errorf("formatSegmentName(%q) = %q; expected %q", tc.id, got, tc.wantValue)
		}
	}
}

func TestListSegements(t *testing.T) {

	dir := t.TempDir()
	for _, name := range []string{"0000000001.wal", "0000000003.wal", "0000000002.wal", "0000000004.wal", "garbage.txt", "notes.md"} {
		f, err := os.Create(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		err = f.Close()
		if err != nil {
			t.Fatal(err)
		}

	}

	ids, err := listSegments(dir)
	if err != nil {
		t.Fatal(err)
	}

	want := []uint64{1, 2, 3, 4}

	for i := range ids {
		if ids[i] != want[i] {
			t.Errorf("ids[%d] = %d; expected %d\n ids = %v; expected %v", i, ids[i], want[i], ids, want)
		}
	}
}
