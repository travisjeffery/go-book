package grpc

import (
	"context"
	"io/ioutil"
	"net"
	"reflect"
	"testing"

	api "github.com/travisjeffery/go-book/api/v1"
	"github.com/travisjeffery/go-book/internal/log"
	"google.golang.org/grpc"
)

func TestServer(t *testing.T) {
	for scenario, fn := range map[string]func(t *testing.T, srv *grpc.Server, client api.LogClient){
		"consume empty log fails":                             testConsumeEmpty,
		"consume past log boundary fails":                     testConsumePastBoundary,
		"produce/consume a message to/from the log succeeeds": testProduceConsume,
		"consume stream succeeds":                             testConsumeStream,
	} {
		t.Run(scenario, func(t *testing.T) {
			l, err := net.Listen("tcp", "127.0.0.1:0")
			check(t, err)

			cc, err := grpc.Dial(l.Addr().String(), grpc.WithInsecure())
			check(t, err)
			defer cc.Close()

			dir, err := ioutil.TempDir("", "server-test")
			check(t, err)

			srv := NewAPI(&log.Log{Dir: dir})

			go func() {
				srv.Serve(l)
			}()
			defer func() {
				srv.Stop()
				l.Close()
			}()

			client := api.NewLogClient(cc)

			fn(t, srv, client)
		})
	}
}

func testConsumeEmpty(t *testing.T, srv *grpc.Server, client api.LogClient) {
	consume, err := client.Consume(context.Background(), &api.ConsumeRequest{
		Offset: 0,
	})
	if consume != nil {
		t.Fatalf("got consume: %v, want: nil", consume)
	}
	if grpc.Code(err) != grpc.Code(api.ErrOffsetOutOfRange) {
		t.Fatalf("got err: %v, want: %v", err, api.ErrOffsetOutOfRange)
	}
}

func testProduceConsume(t *testing.T, srv *grpc.Server, client api.LogClient) {
	ctx := context.Background()

	want := &api.RecordBatch{
		Records: []*api.Record{{
			Value: []byte("hello world"),
		}},
	}

	produce, err := client.Produce(context.Background(), &api.ProduceRequest{
		RecordBatch: want,
	})
	check(t, err)

	consume, err := client.Consume(ctx, &api.ConsumeRequest{
		Offset: produce.FirstOffset,
	})
	check(t, err)

	if !reflect.DeepEqual(want, consume.RecordBatch) {
		t.Fatalf("API.Produce/Consume, got: %v, want %v", consume.RecordBatch, want)
	}
}

func testConsumePastBoundary(t *testing.T, srv *grpc.Server, client api.LogClient) {
	ctx := context.Background()

	produce, err := client.Produce(ctx, &api.ProduceRequest{
		RecordBatch: &api.RecordBatch{
			Records: []*api.Record{{
				Value: []byte("hello world"),
			}},
		},
	})
	check(t, err)

	consume, err := client.Consume(ctx, &api.ConsumeRequest{
		Offset: produce.FirstOffset + 1,
	})
	if consume != nil {
		t.Fatal("consume not nil")
	}
	if grpc.Code(err) != grpc.Code(api.ErrOffsetOutOfRange) {
		t.Fatalf("got err: %v, want: %v", err, api.ErrOffsetOutOfRange)
	}
}

func testConsumeStream(t *testing.T, srv *grpc.Server, client api.LogClient) {
	ctx := context.Background()

	batches := []*api.RecordBatch{{
		Records: []*api.Record{{
			Value: []byte("first message"),
		}},
	}, {
		Records: []*api.Record{{
			Value: []byte("second message"),
		}},
	}}

	produce, err := client.Produce(ctx, &api.ProduceRequest{
		RecordBatch: batches[0],
	})
	check(t, err)

	_, err = client.Produce(ctx, &api.ProduceRequest{
		RecordBatch: batches[1],
	})
	check(t, err)

	stream, err := client.ConsumeStream(ctx, &api.ConsumeRequest{
		Offset: produce.FirstOffset,
	})
	check(t, err)

	consume, err := stream.Recv()
	if err != nil {
		t.Fatal(err)
	}
	got, want := consume.RecordBatch, batches[0]
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("got batch: %v, want: %v", got, want)
	}

	consume, err = stream.Recv()
	if err != nil {
		t.Fatal(err)
	}
	got, want = consume.RecordBatch, batches[1]
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("got batch: %v, want: %v", got, want)
	}

	consume, err = stream.Recv()
	if consume != nil {
		t.Fatalf("consume not nil")
	}
	if grpc.Code(err) != grpc.Code(api.ErrOffsetOutOfRange) {
		t.Fatalf("got err: %v, want: %v", err, api.ErrOffsetOutOfRange)
	}
}

func check(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
