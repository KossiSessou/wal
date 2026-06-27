package wal

import (
	"encoding/binary"
	"hash/crc32"
	"os"
	"sync"
	"time"
)

type WAL struct {
	fd        *os.File
	mu        sync.Mutex
	mode      SyncMode
	interval  time.Duration
	offset    uint64
	done      chan struct{}
	stopped   chan struct{}
	closeOnce sync.Once
}

type SyncMode int

const (
	SyncNever SyncMode = iota
	SyncAlways
	SyncInterval
)

type Config struct {
	Mode     SyncMode
	Interval time.Duration
}

func Open(path string, cfg Config) (*WAL, error) {

	fi, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	w := &WAL{
		fd:       fi,
		mode:     cfg.Mode,
		interval: cfg.Interval,
		done:     make(chan struct{}),
	}

	if cfg.Mode == SyncInterval {
		ticker := time.NewTicker(w.interval)
		w.stopped = make(chan struct{})

		go w.syncLoop(ticker)

	}

	return w, nil
}

func (w *WAL) Close() error {
	var err error
	w.closeOnce.Do(func() {
		_ = w.Sync()
		close(w.done)
		if w.mode == SyncInterval {
			<-w.stopped
		}
		err = w.fd.Close()
	})

	return err
}

func (w *WAL) Sync() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.fd.Sync()

}
func (w *WAL) syncLoop(ticker *time.Ticker) {
	defer close(w.stopped)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			_ = w.Sync()
		case <-w.done:
			return
		}
	}

}

func (w *WAL) Append(record []byte) (offset uint64, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	length := len(record)

	lengthBytes := make([]byte, 4)
	checksumBytes := make([]byte, 4)

	binary.LittleEndian.PutUint32(lengthBytes, uint32(length))

	checksumInput := make([]byte, 0, 4+length)
	checksumInput = append(checksumInput, lengthBytes...)
	checksumInput = append(checksumInput, record...)
	checksum := crc32.ChecksumIEEE(checksumInput)

	binary.LittleEndian.PutUint32(checksumBytes, checksum)

	buf := make([]byte, 0, 8+length)
	buf = append(buf, lengthBytes...)
	buf = append(buf, checksumBytes...)
	buf = append(buf, record...)

	pos := w.offset

	written := 0
	var n int
	for written < len(buf) {
		n, err = w.fd.Write(buf[written:])
		if err != nil {
			return 0, err
		}
		written += n
	}

	w.offset = pos + uint64(written)

	if w.mode == SyncAlways {
		if err := w.fd.Sync(); err != nil {
			return pos, err
		}
	}

	return pos, nil

}
