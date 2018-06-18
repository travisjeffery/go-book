# Client to server communication with gRPC

In the previous chapters we setup our project and protocol buffers, and wrote our commit log library. Starting with a library first helps you focus on writing a flexible and robust API, you don't have to completely ignore the requirements of the end program, but it can be beneficial to ignore it while working on the library. The resulting program we're building should ideally be nothing more than a small layer that ties together configuration and underlying libraries implemented to fulfill its needs.

Currently, our library can only be used on a single computer by a single person at a time, and that person has to run our code, learn our library's API, and store the log on their disk. These are three of the primary reasons why we write services: to take advantage of running over multiple computers, to enable multiple people to interact with the same the data, and to provide a more accessible means of use.

In this chapter, we're gonna build on our library and make a service that other people can call over the network. In doing so we'll face some the following problems/goals all programmers face when building networked services:

- **Simple**: Networked communication is technical and complex and we want to focus on the problem our service is built to solve, rather than the technical minutiae of request-response serialization, and so on.

- **Maintainable**: Writing the first version of a service is a relatively brief period in the time you'll be working on it. Once our service is live and people depend on it we often must maintain backwards compatibility, which is usually done by versioning/running multiple instances of your API.

- **Secure**: With our service exposed on a network we also expose it to whoever is on that network, potentially the entire internet. It's important we can control who has access to our service and permit what they can do.

- **Usable**: We want to enable users of our service to use our service as intended. When using a library the documentation is often immediately accessible and clearly errors when something goes wrong, often neither of which is the case with services.

- **Performant**: We want our service to be fast. [Amazon found every 100ms of latency cost them 1% in sales](https://www.fastcompany.com/1825005/how-one-second-could-cost-amazon-16-billion-sales) - that's 1.6 billion! [Google found an extra .5 seconds in search page generation time dropped traffic by 20%](http://glinden.blogspot.com/2006/11/marissa-mayer-at-web-20.html).

- **Scales**: We want to easily scale up our service the more it's used by balancing the load across multiple computers.

To write our service we'll be using gRPC which has the following advantages over REST:

- gRPC transparently handles the serialization of our requests and responses.
- gRPC eases versioning our services and running multiple services at the same time.
- gRPC checks client calls are type-safe and tells the user they're making a bad call.
- gRPC speeds up request handling, your service will be about [25 times](https://husobee.github.io/golang/rest/grpc/2016/05/28/golang-rest-v-grpc.html) faster than the equivalent REST service.
- gRPC supports load balancing built-in.

What is gRPC? gRPC is a high performance RPC framework open sourced by Google. In gGRPC - being RPC (remote procedure call) - client applications call methods on a server application on a different machine as if it were a local object. It enables client and server applications to communicate transparently, and makes it easier to build connected systems.

Why use gRPC? gRPC allows us to define our service once and then compile that into clients and servers in various languages that gRPC supports. Even if your whole stack is Go, gRPC is worth using because it provides efficient, type-checked serialization of your requests and responses; it generates clients for free; and gRPC also makes it easy to build streaming APIs. gRPC is also extendable with an active community working on doing so.

What can you build with gRPC? gRPC can be used anywhere with request-response communication, you can even use it for bidirectional streaming RPC where client and server send messages back and forth using a read-write stream. Anywhere you're using REST/HTTP and you control the clients you could use gRPC instead - and often benefit. REST allows you to loosely couple your server and clients by implementing [hypermedia](https://en.wikipedia.org/wiki/HATEOAS). When using hypermedia REST services, the REST client enters the application through a fixed URL and all future actions the client can take are discovered within resource representations returned from the server - it's like how you browse the web with the hyperlinks on each page telling you where you can go next. However, hypermedia is not straightforward to build, nor use, and often not necessary since the server and client are co-developed, and they can take advantage of that. Furthermore, REST is all about resources. Resources are easy to reason about when the resources are tangible, but harder to reason about when intangible - the user's session as a resource or password reset as a resource in a web app, for instance. On other hand, RPC is all about APIs and actions. This makes RPC more accessible and more intuitive because it's matches how we think in the real world - you don't want to create a password reset - you want to reset your password.

Let's build your first, of many, gRPC services.

## Define the service

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

## Compile with gogo's gRPC plugin

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

## Register your server

Our server is implemented and we haven't done anything gRPC specific yet. We'll add an exported function to instantiate our server implementation, create a gRPC server, and register our implementation with it. The gRPC server will listen on the network, handle requests, call our server, and respond back to the client with the result.

```
func NewAPI(log log) *grpc.Server {
	g := grpc.NewServer()
	s := newgrpcServer(log)
	api.RegisterLogServer(g, s)
	return g
}
```

## Test your gRPC server and use a gRPC client

With our gRPC server done let's write some tests and try hitting it with a gRPC client. In the same directory create a server_test.go file.

Our tests will [table driven](https://github.com/golang/go/wiki/TableDrivenTests). With table driven tests, each entry is a complete and concisely written test case. The entry may be structure or function defining the input and expected results. If you find yourself using copy and paste when writing a test, consider refactoring the code to be table driven or into a helper function.

We start by defining our top-level test function TestServer which will contain and run the test table, including the setup common for each test case. For us that setup includes creating a listener for our server on the local network address. The 0 port is useful for cases like this where we don't care what port we're using and using 0 will automatically assign us a free port. We create an insecure connection to our listener and with it, a log client. We create our server and start serving requests in a goroutine because the Serve method is a blocking call and our test case would never run otherwise. We defer a function that will stop our server and close its connection once our test case finishes. Finally we call the function representing our test case and past in our setup server and client for it to use.

``` go
func TestServer(t *testing.T) {
	for scenario, fn := range map[string]func(t *testing.T, srv *grpc.Server, client api.LogClient){} {
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
```

Our table is empty though so let's start adding test cases. The initial test case will test what happens when a user consumes the initial, empty state of the log. The second case tests a successful produce and consume. And the final case tests what happens when the consumer reaches teh end of the log. Add the test cases to the table.

```
for scenario, fn := range map[string]func(t *testing.T, srv *grpc.Server, client api.LogClient){
  	"consume empty log fails":                             testConsumeEmpty,
	"produce/consume a message to/from the log succeeeds": testProduceConsume,
	"consume past log boundary fails":                     testConsumePastBoundary,
} {
//...
```

Now implement our test cases. In the initial case we check that the returned batch is nil and that we get the ErrOffsetOutOfRange error returned by our log library. The code we defined the error with is included in the response, and we can look it up with grpc's Code() function. Similarly, you can look up its description via the ErrorDesc() function. The error's code and description provide information to the caller on what went wrong, similar to HTTP status codes.

```
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
    if grpc.ErrorDesc(err) != grpc.ErrorDesc(api.ErrOffsetOutOfRange) {
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
	equal(t, consume.RecordBatch, want)
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
```

check is a helper function used to DRY up our error checking. Similarly, equal is a helper function used to DRY up our equality checking.

``` go
func check(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}


func equal(t *testing.T, got, want interface{}) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got: %v, want %v", got, want)
	}
}
```

## Stream messages

Communication between applications built on logs is rarely one and done, instead these applications
communicate by streaming messages with one another. Data pipeline creators jumped on board with streams sooner than others because their pipelines fit the model and to reap the performance gains. Now developers building user facing applications are taking advantage of streams to build features like streaming, real-time messages such as in Twitch or Twitter. We'll now add streaming capability to our service.

Back to our protobuf file, we're going to add two methods:

1. ConsumeStream - a server streaming RPC to allow the client to make a single consume request and the server will stream back the corresponding message, along with all subsequent messages. If you were writing a service that reads from the log and stores the messages in a database, you'd use this type of API.
2. ProduceStream - a bidirectional streaming RPC to allow the client to stream produce requests and responses, appending messages to the log while optionally using the response to check errors, progress, or some other notification. If you were writing a service that ingested messages into your system then it would use this type of API.

Your service will look like the following. You prepend "stream" before your request or response or both to indicate it will stream. Run `$ make clean build` to compile it code and have a look at log.go.pb to see what the compiler generated and the API you'll be using.

```
service Log {
  rpc Produce (ProduceRequest) returns (ProduceResponse) {}
  rpc Consume (ConsumeRequest) returns (ConsumeResponse) {}
  rpc ConsumeStream(ConsumeRequest) returns (stream ConsumeResponse) {}
  rpc ProduceStream(stream ProduceRequest) returns (stream ProduceResponse) {}
}
```

Implementing these APIs in our servers is easy as gRPC's are easy to use and we can build on our existing methods. In server.go we add the following methods. Streaming APIs are passed a similar stream interface with two core methods: Send() - to send messages to the stream and Recv() - to receive messages from the stream. Additionally you can use Context() method to check whether the RPC was timed out, terminated, or canceled. For example if your service was Twitch-like and streamed messages to users, when a user turned off chat you'd want to cancel the stream and prevent wasting bandwidth.

Our streaming methods are basically just our existing unary methods wrapped with for loops. ConsumeStream tracks the offset of the next message to send to the client which will be sent the next time the client calls Recv(). ProduceStream is simpler - receiving a batch from the stream, appending it to the log, and sending back a response.

``` go
func (s *grpcServer) ConsumeStream(req *api.ConsumeRequest, stream api.Log_ConsumeStreamServer) error {
	for {
		res, err := s.Consume(stream.Context(), req)
		if err != nil {
			return err
		}
		if err = stream.Send(res); err != nil {
			return err
		}
		req.Offset++
	}
}

func (s *grpcServer) ProduceStream(stream api.Log_ProduceStreamServer) error {
	for {
		req, err := stream.Recv()
		if err != nil {
			return err
		}
		res, err := s.Produce(stream.Context(), req)
		if err != nil {
			return err
		}
		if err = stream.Send(res); err != nil {
			return err
		}
	}
}
```

Let's add a test for our stream methods and see how streaming works from the client-side.

Add a case to our table.

```
for scenario, fn := range map[string]func(t *testing.T, srv *grpc.Server, client api.LogClient){
		"consume empty log fails":                             testConsumeEmpty,
		"produce/consume a message to/from the log succeeeds": testProduceConsume,
		"consume past log boundary fails":                     testConsumePastBoundary,
		"produce/consume stream succeeds":                     testProduceConsumeStream,
	} {...}
```

And implement the test.

```
func testProduceConsumeStream(t *testing.T, srv *grpc.Server, client api.LogClient) {
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

	{
		stream, err := client.ProduceStream(ctx)
		check(t, err)

		for _, batch := range batches {
			err = stream.Send(&api.ProduceRequest{
				RecordBatch: batch,
			})
			check(t, err)
		}

	}

	{
		stream, err := client.ConsumeStream(ctx, &api.ConsumeRequest{Offset: 0})
		check(t, err)

		for _, batch := range batches {
			res, err := stream.Recv()
			check(t, err)
			equal(t, res.RecordBatch, batch)
		}
	}
}
```

## What you learned

You hit the ground running with gRPC in this chapter. You learned how to define a gRPC service in protobuf, compile it into server and client code, implement the server, and test it with your client.

You know how to build a gRPC server and client and you can use your log over the network. Now we're going to make your log service distributed, turning multiple individual servers into a cluster, connecting them with service discovery and consensus.
