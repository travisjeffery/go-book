package log

import (
	"sync"

	api "github.com/travisjeffery/go-book/api/v1"
)

type Log struct {
	activeSegment *segment
	mu            sync.RWMutex
	path          string
	segments      []*segment
}

func (l *Log) AppendBatch(batch *api.RecordBatch) (uint64, error) {
	b, err := batch.Marshal()
	if err != nil {
		return 0, err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	offset, position := l.activeSegment.nextOffset, l.activeSegment.position
	_, err = l.activeSegment.Write(b)
	if err != nil {
		return 0, err
	}
	l.activeSegment.index.writeEntry(entry{
		offset:   offset,
		position: position,
		length:   batch.Size(),
	})
	return offset, nil
}

func (l *Log) ReadBatch(offset uint64) (*api.RecordBatch, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	entry, err := l.activeSegment.index.readEntry(offset)
	if err != nil {
		return nil, err
	}
	p := make([]byte, entry.length)
	_, err = l.activeSegment.ReadAt(p, int64(entry.position))
	if err != nil {
		return nil, err
	}
	batch := &api.RecordBatch{}
	err = batch.Unmarshal(p)
	return batch, err

}
