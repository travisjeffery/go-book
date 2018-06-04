package log

import (
	"bytes"
	"encoding/binary"
	"os"
	"sync"

	"github.com/tysontate/gommap"
)

var (
	encoding = binary.BigEndian
)

const (
	offsetWidth   = 4
	positionWidth = 4
	lengthWidth   = 4
	entryWidth    = offsetWidth + positionWidth + lengthWidth
)

type index struct {
	mu       sync.Mutex
	mmap     gommap.MMap
	position uint64
	file     *os.File
}

type entry struct {
	offset   uint64
	position uint64
	length   uint64
}

func (i *index) readEntry(offset uint64) (e entry, err error) {
	p := make([]byte, entryWidth)
	pos := offset * entryWidth
	copy(p, i.mmap[pos:pos+entryWidth])
	b := bytes.NewReader(p)
	err = binary.Read(b, encoding, &e)
	return e, err
}

func (i *index) writeEntry(e entry) error {
	b := new(bytes.Buffer)
	if err := binary.Write(b, encoding, e); err != nil {
		return err
	}
	n, err := i.WriteAt(b.Bytes(), int64(i.position))
	if err != nil {
		return err
	}
	i.position += uint64(n)
	return nil

}

func (i *index) WriteAt(p []byte, offset int64) (int, error) {
	i.mu.Lock()
	defer i.mu.Unlock()
	n := copy(i.mmap[offset:offset+entryWidth], p)
	return n, nil
}
