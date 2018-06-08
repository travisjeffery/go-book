package grpc

import (
	"context"
	"net"
	"reflect"
	"testing"

	api "github.com/travisjeffery/go-book/api/v1"
	"google.golang.org/grpc"
)

func TestServer(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	check(t, err)
	cc, err := grpc.Dial(l.Addr().String(), grpc.WithInsecure())
	check(t, err)
	defer cc.Close()

	ctx := context.Background()

	srv := NewAPI(make(mocklog))

	go func() {
		srv.Serve(l)
	}()
	defer func() {
		srv.Stop()
		l.Close()
	}()

	lc := api.NewLogClient(cc)

	want := &api.RecordBatch{
		Records: []*api.Record{{
			Value: []byte("hello world"),
		}},
	}
	produce, err := lc.Produce(ctx, &api.ProduceRequest{
		RecordBatch: want,
	})
	check(t, err)

	consume, err := lc.Consume(ctx, &api.ConsumeRequest{
		Offset: produce.FirstOffset,
	})
	check(t, err)

	if !reflect.DeepEqual(want, consume.RecordBatch) {
		t.Fatalf("API.Produce/Consume, got: %v, want %v", consume.RecordBatch, want)
	}
}

func check(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

type mocklog map[uint64]*api.RecordBatch

func (m mocklog) AppendBatch(b *api.RecordBatch) (uint64, error) {
	off := uint64(len(m))
	m[off] = b
	return off, nil
}

func (m mocklog) ReadBatch(off uint64) (*api.RecordBatch, error) {
	return m[off], nil
}
