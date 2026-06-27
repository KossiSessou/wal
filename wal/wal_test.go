package wal

import (
	"os"
	"testing"
	"time"
)

func TestAppendBasic(t *testing.T) {
	dir := t.TempDir()

	path := dir + "test.wal"

	w, err := Open(path, Config{Mode: SyncNever})

	if err != nil {
		t.Fatal(err)
	}

	off1, err := w.Append([]byte("hello"))

	if err != nil {
		t.Fatal(err)
	}

	off2, err := w.Append([]byte("world"))

	if err != nil {
		t.Fatal(err)
	}

	if off1 != 0 {
		t.Errorf("First offset = %d; want 0", off1)
	}

	if off2 != 13 {
		t.Errorf("Second offset = %d; want 13", off2)
	}

	info, _ := os.Stat(path)

	if info.Size() != 26 {
		t.Errorf("File size = %d; want 26", info.Size())
	}
}

func TestAppendSyncInterval(t *testing.T) {
	dir := t.TempDir()
	path := dir + "test.go"
	w, _ := Open(path, Config{Mode: SyncInterval, Interval: 50 * time.Millisecond})
	go func() {
		for i := 0; i < 1000; i++ {
			w.Append([]byte("hello"))
		}
	}()
	time.Sleep(500 * time.Millisecond)
	w.Close()
}
