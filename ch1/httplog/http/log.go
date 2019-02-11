package http

import (
	"fmt"
	"sync"
)

type CommitLog struct {
	sync.Mutex
	currOffset uint64
	data       map[uint64]RecordBatch
}

func NewCommitLog() *CommitLog {
	return &CommitLog{
		data: make(map[uint64]RecordBatch),
	}
}

func (c *CommitLog) AppendBatch(batch RecordBatch) (uint64, error) {
	c.Lock()
	defer c.Unlock()
	c.currOffset++
	c.data[c.currOffset] = batch
	return c.currOffset, nil
}

func (c *CommitLog) ReadBatch(offset uint64) (RecordBatch, error) {
	c.Lock()
	defer c.Unlock()
	batch, ok := c.data[offset]
	if !ok {
		return RecordBatch{}, ErrOffsetNotFound
	}
	return batch, nil
}

type Record struct {
	Value       []byte `json:"value"`
	OffsetDelta uint32 `json:"offset_delta"`
}

type RecordBatch struct {
	FirstOffset uint64   `json:"first_offset"`
	Records     []Record `json:"records"`
}

var ErrOffsetNotFound = fmt.Errorf("offset not found")
