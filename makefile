compile:
	protoc api/v1/*.proto --go_out=. \
    	--go_opt=paths=source_relative \
    	--proto_path=.


test-race:
	go test -race ./...
