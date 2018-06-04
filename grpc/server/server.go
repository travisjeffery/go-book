package server

import (
	"context"

	log "github.com/travisjeffery/log/api/v1"
)

var _ log.LogServer = (*server)(nil)

type server struct {
}

func (s *server) Produce(ctx context.Context, req *log.ProduceRequest) (*log.ProduceResponse, error) {
	return nil, nil
}

func (s *server) Consume(ctx context.Context, req *log.ConsumeRequest) (*log.ConsumeResponse, error) {
	return nil, nil
}
