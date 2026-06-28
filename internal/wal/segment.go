package wal

import (
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
)

// parseSegmentName extracts the segment ID from a filename of the form
// "%010d.wal". Returns (id, true) on success, (0, false) on any
// malformed input.

func parseSegmentName(name string) (uint64, bool) {

	if !strings.HasSuffix(name, ".wal") {
		return 0, false
	}
	base := strings.TrimSuffix(name, ".wal")
	id, err := strconv.ParseUint(base, 10, 64)
	if err != nil {
		return 0, false
	}
	return id, true
}

// formatSegmentName takes the uint64 id and format it into a
// 10-digits zero-padded filename with a wal extension and return
// it as a string.
func formatSegmentName(id uint64) string {

	return fmt.Sprintf("%010d.wal", id)
}

// listSegments lists all the .wal file in a directory, sort them
// in ascending order and return the list on success and (nil, error)
// on error
func listSegments(dir string) ([]uint64, error) {

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	result := make([]uint64, 0, len(entries))

	for _, file := range entries {

		if file.IsDir() {
			continue
		}

		id, ok := parseSegmentName(file.Name())

		if ok {
			result = append(result, id)
		}
	}

	slices.Sort(result)
	return result, nil
}
