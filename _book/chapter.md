# Client to server communication with grpc

In the previous chapters we setup our project and protobufs, and wrote our commit log library. In
this chapter we'll build on our library and turn it into a grpc web service and create both a server
and client.

- We'll see how to define a grpc service in protobuf.
- We'll look at how to implement and test a grpc server.
- We'll take advantage of how grpc provides us with a client for free.

## Why grpc?

grpc allows us to define our service once and then compile that into clients and servers in various
languages that grpc supports. and even if your whole stack is go, grpc handles the efficient and
type-checked serialization of type-checked request and responses, grpc gives us a client for free,
and grpc enable us to easily build streaming apis.

## Defining the service

Open the protobuf file where we defined our RecordBatch and Record types in and add the following
service definition.

```
service Log {
  rpc Produce (ProduceRequest) returns (ProduceResponse) {}
  rpc Consume (ConsumeRequest) returns (ConsumeResponse) {}
}

message ProduceRequest  {
  RecordBatch record_batch = 1;
}

message ProduceResponse  {
  uint64 first_offset = 1;
}

message ConsumeRequest {
  uint64 offset = 1;
}

message ConsumeResponse {
  RecordBatch record_batch = 2;
}
```

This service API simply wraps our log library's API and is similar to the produce/consume API that
Apache Kafka also uses. When compiled this will generate a LogServer and LogClient with
corresponding Produce and Consume methods. We need to compile our protobuf with the grpc plugin.

## Compiling with gogo's grpc plugin

In the root of our project update our build target like so to enable the grpc plugin and compile our
grpc service. We're using the gogo protobuf grpc plugin rather than the one provided by
golang/protobuf. gogo is a popular fork - used by etcd, kubernetes, dropbox, nats, cloudflare, and
others. It provides extra code generation features for go, like being able to generate marshal and
unmarshal and size methods for example which we'll be using, and also being able to embed fields and
use custom types.

```
build:
	protoc -I api/v1/ api/v1/log.proto --gogofast_out=plugins=grpc:api/v1
```

We need to install the protobuf compiler and protoc plugin for Go. If you're on a Mac you do:

```
$ wget https://github.com/google/protobuf/releases/download/v3.5.0/protoc-3.5.0-osx-x86_64.zip && unzip protoc-3.5.0-osx-x86_64.zip -d /usr/local/bin/protoc
$ export PATH="$PATH:/usr/local/bin/protoc/bin"
$ go get -u github.com/gogo/protobuf/protoc-gen-gogofast
```

With those installed and our Makefile in place, we're ready to compile. In the root of your project
run `$ make compile` and look inside the api/v1 directory, there's a new file: log.pb.go. Open it up and
check out the generated code. In there you'll see a gRPC working client, the generated server is
only an interface - it's up to us to implement it. Let's do it.

Create internal/grpc directory tree in the root of your project[^1], you can do that by running `mkdir
-p internal/grpc`. In this directory we'll implement our server in a file called server.go:

```
package grpc

import (
	"context"

	api "github.com/travisjeffery/go-book/api/v1"
	"google.golang.org/grpc"
)

var _ api.LogServer = (*grpcServer)(nil)

func NewAPI(log log) *grpc.Server {
	g := grpc.NewServer()
	s := newgrpcServer(log)
	api.RegisterLogServer(g, s)
	return g
}

func newgrpcServer(log log) *grpcServer {
	return &grpcServer{
		log: log,
	}
}

type grpcServer struct {
	log log
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

type log interface {
	AppendBatch(*api.RecordBatch) (uint64, error)
	ReadBatch(uint64) (*api.RecordBatch, error)
}
```

This is the beauty of library driven development, the code wrapping the libraries tends to be very
simple.

`var _ api.LogServer = (*grpcServer)(nil)` checks that our type implements the interface we want it
to, which in this case is our grpc server implementation.


```
type log interface {
	AppendBatch(*api.RecordBatch) (uint64, error)
	ReadBatch(uint64) (*api.RecordBatch, error)
}
```

Defines our log interface. This way our server doesn't depend on a concrete implementation. We
inject whatever implementation we want instead - maybe we're trying out a different implementation
or maybe we're passing in a mocked log for our tests.

Let's write a test with the grpc client hitting our server.

``` go
package grpc

import (
	"context"
	"net"
	"reflect"
	"testing"
	"time"

	api "github.com/travisjeffery/go-book/api/v1"
	"google.golang.org/grpc"
)

func TestServer(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")[^2]
	check(t, err)

	cc, err := grpc.Dial(l.Addr().String(), grpc.WithInsecure())
	check(t, err)
	defer cc.Close()

    ctx := context.Background()
	srv := NewAPI(make(mocklog)) // 1.

	go func() {
		srv.Serve(l) // 1.
	}()

	defer func() {
		srv.Stop() // 1.
		l.Close()
	}()

	lc := api.NewLogClient(cc) // 1.

	want := &api.RecordBatch{
		Records: []*api.Record{{
			Value: []byte("hello world"),
		}},
	}
	produce, err := lc.Produce(context., &api.ProduceRequest{
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
		t.Fatal(err)b
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
```

- Create our API instance and pass in our mocked log implementation for the API to use so we can focos
on writing and testing our server.
- Tell the server to start handling requests.
- Stop the server and close the listener to free the port.
- Create our grpc client.
- Produce to our API, consume from, and check that the record batch in the request and response
  match.

[^1]: "internal" directories/packages are magical packages in Go that can only be imported by nearby code.
For example: code in /a/b/c/internal/d/e/f can be imported by code rooted by /a/b/c, but not code
rooted by /a/b/g.

[^2]: The 0 port is useful for tests because on unix systems a free port will automatically be
    assigned.
