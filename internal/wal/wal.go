package wal

import (
	"encoding/binary"
	"errors"
	"hash/crc32"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type SyncMode int

const defaultMaxSegmentSize = 64 << 20

const (
	SyncNever SyncMode = iota
	SyncAlways
	SyncInterval
)

var ErrClosed = errors.New("wal closed")
var ErrRecordTooLarge = errors.New("record too large")

type Offset struct {
	SegmentID uint64
	Position  uint64
}

type WAL struct {
	dir       string
	fd        *os.File
	mu        sync.Mutex
	mode      SyncMode
	interval  time.Duration
	maxSize   uint64
	activeID  uint64
	position  uint64
	closed    bool
	done      chan struct{}
	stopped   chan struct{}
	closeOnce sync.Once
}

type Config struct {
	Mode     SyncMode
	Interval time.Duration
	MaxSize  uint64
}

func Open(dir string, cfg Config) (*WAL, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	ids, err := listSegments(dir)
	if err != nil {
		return nil, err
	}

	maxSize := cfg.MaxSize

	if maxSize == 0 {
		maxSize = defaultMaxSegmentSize
	}
	w := &WAL{
		dir:      dir,
		mode:     cfg.Mode,
		interval: cfg.Interval,
		maxSize:  maxSize,
		done:     make(chan struct{}),
	}
	if len(ids) == 0 {
		w.activeID = 1
	} else {
		w.activeID = ids[len(ids)-1]
	}

	path := filepath.Join(dir, formatSegmentName(w.activeID))

	fi, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}

	info, err := fi.Stat()
	if err != nil {
		_ = fi.Close()
		return nil, err
	}
	w.fd = fi
	w.position = uint64(info.Size())

	if cfg.Mode == SyncInterval {

		ticker := time.NewTicker(cfg.Interval)
		w.stopped = make(chan struct{})
		go w.syncLoop(ticker)
	}

	return w, nil
}

func (w *WAL) Close() error {
	var err error
	w.closeOnce.Do(func() {
		w.mu.Lock()
		w.closed = true
		_ = w.fd.Sync()
		w.mu.Unlock()

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
	if w.closed {
		return ErrClosed
	}

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

func (w *WAL) rotate() (err error) {
	if err = w.fd.Sync(); err != nil {
		return err
	}
	if err = w.fd.Close(); err != nil {
		return err
	}
	w.activeID += 1

	path := formatSegmentName(w.activeID)

	w.fd, err = os.OpenFile(filepath.Join(w.dir, path), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}

	w.position = 0

	return nil

}

func (w *WAL) Append(record []byte) (offset Offset, err error) {

	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return Offset{}, ErrClosed
	}

	recordLen := len(record)

	if uint64(8+recordLen) > w.maxSize {
		return Offset{}, ErrRecordTooLarge

	}

	buf := make([]byte, 8+recordLen)
	binary.LittleEndian.PutUint32(buf[0:4], uint32(recordLen))

	copy(buf[8:], record)

	crc := crc32.Update(0, crc32.IEEETable, buf[0:4])
	crc = crc32.Update(crc, crc32.IEEETable, buf[8:])
	binary.LittleEndian.PutUint32(buf[4:8], crc)

	if uint64(len(buf))+w.position > w.maxSize {

		err = w.rotate()
		if err != nil {
			return Offset{}, err
		}

	}

	var written int

	for written < len(buf) {
		n, err := w.fd.Write(buf[written:])
		if err != nil {
			return Offset{}, err
		}
		written += n

	}
	pos := w.position
	w.position += uint64(written)
	offset = Offset{SegmentID: w.activeID, Position: uint64(pos)}

	if w.mode == SyncAlways {
		if err := w.fd.Sync(); err != nil {
			return offset, err
		}
	}

	return offset, nil
}
