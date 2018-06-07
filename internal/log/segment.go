package log

import (
	"os"
	"sync"
)

type segment struct {
	file        *os.File
	firstOffset uint64
	index       *index
	mu          sync.RWMutex
	nextOffset  uint64
	path        string
	position    uint64
}

func (s *segment) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n, err := s.file.Write(p)
	if err != nil {
		return 0, err
	}
	s.nextOffset++
	s.position += uint64(n)
	return n, nil
}

func (s *segment) Read(p []byte) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.file.Read(p)
}

func (s *segment) ReadAt(p []byte, off int64) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.file.ReadAt(p, off)
}
