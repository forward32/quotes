lint:
	golangci-lint run

test:
	go test -v -race ./...

proto:
	protoc -I /home/yury/Downloads/protoc/include -I schema --go_out=plugins=grpc:api schema/quotes.proto

run:
	go run cmd/*.go

run_race:
	go run -race cmd/*.go
