package grpc

import (
	"context"

	api "github.com/travisjeffery/go-book/api/v1"
	"google.golang.org/grpc"
)

var _ api.LogServer = (*grpcServer)(nil)

func NewAPI(log logger) *grpc.Server {
	g := grpc.NewServer()
	s := newgrpcServer(log)
	api.RegisterLogServer(g, s)
	return g
}

func newgrpcServer(log logger) *grpcServer {
	return &grpcServer{
		log: log,
	}
}

type grpcServer struct {
	log logger
}

func (s *grpcServer) Produce(ctx context.Context, req *api.ProduceRequest) (*api.ProduceResponse, error) {
	offset, err := s.log.AppendBatch(req.RecordBatch)
	if err != nil {
		return nil, err
	}
	return &api.ProduceResponse{FirstOffset: offset}, nil
}

func (s *grpcServer) Consume(ctx context.Context, req *api.ConsumeRequest) (*api.ConsumeResponse, error) {
	batch, err := s.log.ReadBatch(req.Offset)
	if err != nil {
		return nil, err
	}
	return &api.ConsumeResponse{RecordBatch: batch}, nil
}

type logger interface {
	AppendBatch(*api.RecordBatch) (uint64, error)
	ReadBatch(uint64) (*api.RecordBatch, error)
}
