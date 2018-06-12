# Client to server communication with gRPC

In the previous chapters we setup our project and protocol buffers, and wrote our commit log library. Here's what we have in store this chapter:

- We'll build on our library and turn it into a gRPC web service and create both a server
and client.
- We'll define a gRPC service in protobuf.
- We'll implement a gRPC server.
- We'll create a gRPC client and test our server with it.

## What is gRPC?

gRPC is a high performance RPC framework open sourced by Google. In gGRPC - being RPC (remote
procedure call) - client applications call methods on a server application on a different machine as
if it were a local object, reducing the gap between working on programs that run on a single
computer and working on programs that run on many computers over a network. The server implements an
interface and runs a server to handle gRPC client calls, and the client provides the same methods as
the server,

So why use gRPC? gRPC allows us to define our service once and then compile that into clients and servers in various languages that gRPC supports. Even if your whole stack is Go, gRPC is worth using because it provides efficient, type-checked serialization of your requests and responses; it generates clients for free; and gRPC makes it easy to build streaming APIs.

## Defining the service

Open the protobuf file we defined our RecordBatch and Record types in and add the following service definition.

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

This service API simply wraps our log library's API and is similar to the produce/consume API that Apache Kafka uses - so we're in good company. When compiled, this protocal buffer will turn into a LogServer and LogClient featuring Produce and Consume methods. To do that, we need to compile our protobuf with a gRPC plugin.

## Compiling with gogo's gRPC plugin

Open up our Makefile and update our build target like so to enable the gRPC plugin and compile our gRPC service. We're using the [gogo protobuf](https://github.com/gogo/protobuf) gRPC plugin rather than the [one provided by the Golang team](http://github.com/golang/protobuf/). gogo is a popular fork used by Etcd, Kubernetes, Dropbox, Nats, Cloudflare, and others. It provides extra code generation features for Go, like being able to generate marshal, unmarshal, and size methods for example - which we'll be using - and is also able to embed fields and use custom types.

```
build:
	protoc -I api/v1/ api/v1/log.proto --gogofast_out=plugins=grpc:api/v1

install.deps:
    go get -u github.com/gogo/protobuf/protoc-gen-gogofast
```

Run `$ make install.deps build`, open up the log.pb.go file in the api/v1 directory and check out the generated code. In there you'll see a working gRPC log client, but the log server is left only as an interface - that's because we need to implement it!

## Implementing the gRPC server

Create a internal/grpc directory tree in the root of your project[^1]. You can do that by running `mkdir -p internal/grpc`. In this directory we'll implement our server in a file called server.go and package named grpc.

[^1]: "internal" directories/packages are magical packages in Go that can only be imported by nearby code.
For example: code in /a/b/c/internal/d/e/f can be imported by code rooted by /a/b/c, but not code
rooted by /a/b/g.

The first order of business is to define our server type and a creator function.

```
var _ api.LogServer = (*grpcServer)(nil)

type grpcServer struct {
	log log
}

func newgrpcServer(log log) *grpcServer {
	return &grpcServer{
		log: log,
	}
}
```

The first line is a trick to check that a type satisfies an interface at compile-time. This will help you - the person implementing this type - know when you've fulfilled the requirements. Afterwards it'll help your teammate - the  know what they can or can't change, it acts like type-checked code documentation. To satisfy the interface you saw in log.pb.go we need to implement the Consume and Produce methods.

```
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
```

### Dependency inversion with interfaces

> High-level modules should not depend on low-level modules. Both should depend on abstractions. Abstractions should not depend on details. Details should depend on abstractions.
> –Robert C. Martin

Our server will depend on a log abstraction for it to do anything, generally that'll be our library, but we don't want to be tied to a specific implementation. Our log library stores the logs on disk, but our server doesn't care about the specifics - it only cares that the log it depends on satisfies the log abstraction it's looking for. We do this by defining our dependency as an interface. Aside from the abstract benefits, in practice this eases our server testing.


```
type log interface {
	AppendBatch(*api.RecordBatch) (uint64, error)
	ReadBatch(uint64) (*api.RecordBatch, error)
}
```

## Registering your server

Our server is implemented and we haven't done anything gRPC specific yet. We'll add an exported function to instantiate our server implementation, create a gRPC server, and register our implementation with it. The gRPC server will listen on the network, handle requests, call our server, and respond back to the client with the result.

```
func NewAPI(log log) *grpc.Server {
	g := grpc.NewServer()
	s := newgrpcServer(log)
	api.RegisterLogServer(g, s)
	return g
}
```

## Testing a gRPC server, using a gRPC client

With our gRPC server done let's write some tests and try hitting it with a gRPC client. In the same directory create a server_test.go file.

We start by announcing a listener for our server on the local network address. The 0 port is useful for cases like this where we don't care what port we're using and using 0 will automatically assign us a free port. We then create an insecure connection to our listener and with it, a log client. We then create our server and start serving requests in a goroutine because the Serve method is a blocking call. Lastly, we defer a function that will stop our server and close its connection once our test finishes.

``` go
func TestServer(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	check(t, err)

    conn, err := grpc.Dial(l.Addr().String(), grpc.WithInsecure())
	check(t, err)
	defer conn.Close()

    c := api.NewLogClient(conn)

    ctx := context.Background()
	srv := NewAPI(make(mocklog))

	go func() {
		srv.Serve(l)
	}()

	defer func() {
		srv.Stop()
		l.Close()
	}()

    //...
```

Let's write the test - one that's nice and simple: produce a record batch to our server with our client and check that when we consume it we get the same record batch back.

check is a helper function used to DRY up our error checking.

``` go
    //...

	want := &api.RecordBatch{
		Records: []*api.Record{{
			Value: []byte("hello world"),
		}},
	}

	produce, err := c.Produce(ctx, &api.ProduceRequest{
		RecordBatch: want,
	})
	check(t, err)

	consume, err := c.Consume(ctx, &api.ConsumeRequest{
		Offset: produce.FirstOffset,
	})
	check(t, err)

    got := consume.RecordBatch

	if !reflect.DeepEqual(want, got) {
		t.Fatalf("API.Produce/Consume, got: %v, want %v", got, want)
	}
}

func check(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
```

Back when we created our server, we made and passed in a mock log. The last piece is to implement our mock which keeps our test setup and cleanup simple and lets us focus on testing just our server. As discussed in *Dependency inversion with interfaces* we can pass in our mock since our server depends on our log interface so it doesn't care which specific implementation it uses.

``` go
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

## What you learned

You hit the ground running with gRPC in this chapter. You learned how to define a gRPC service in protobuf, compile it into server and client code, implement the server, and test it with your client.

You know how to build a gRPC server and client and you can use your log over the network. Now we're going to make your log service distributed, turning multiple individual servers into a cluster, connecting them with service discovery and consensus.
