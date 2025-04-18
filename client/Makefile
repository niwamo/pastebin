build:
	protoc /pastebin/proto/*.proto \
    	--go_out=/pastebin/proto \
		--go_opt=paths=source_relative \
		--go-grpc_out=/pastebin/proto \
		--go-grpc_opt=paths=source_relative \
		--proto_path=/pastebin/proto \
		/pastebin/proto/pastebin.proto
	go mod tidy
	go build -o ./client/pastebin-cli ./client