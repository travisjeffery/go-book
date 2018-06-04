build:
	protoc -I api/v1/ api/v1/log.proto --go_out=plugins=grpc:api/v1

clean:
	rm -f api/v1/log.pb.go

test:
	go test -v ./...
