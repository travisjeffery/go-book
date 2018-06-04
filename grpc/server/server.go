package server

import (
	"context"

	"github.com/apex/log"
	api "github.com/travisjeffery/go-book/api/v1"
	"github.com/travisjeffery/go-book/log"
)

var _ api.LogServer = (*server)(nil)

type server struct {
	log *log.Log
}

func New(log *log.Log) *server {
	return &server{log: log}
}

func (s *server) Produce(ctx context.Context, req *api.ProduceRequest) (*api.ProduceResponse, error) {
	offset, err := s.log.AppendBatch(req.RecordBatch)
	if err != nil {
		return nil, err
	}
	return &api.ProduceResponse{FirstOffset: offset}, nil
}

func (s *server) Consume(ctx context.Context, req *api.ConsumeRequest) (*api.ConsumeResponse, error) {
	batch, err := s.log.ReadBatch(req.Offset)
	if err != nil {
		return nil, err
	}
	return &api.ConsumeResponse{RecordBatch: batch}, nil
}
