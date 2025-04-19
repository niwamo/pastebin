# STAGE 1: BUILD BINARY
FROM golang:latest AS builder

# BUILD CONTEXT IS PARENT DIR, NOT IMMEDIATE DIR

WORKDIR /pastebin

COPY ./go.mod /pastebin/go.mod
COPY ./proto /pastebin/proto
COPY ./server/*.go /pastebin/server/

RUN apt update && apt install -y protobuf-compiler
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

RUN go get go.mongodb.org/mongo-driver
RUN go mod tidy

RUN export PATH="$PATH:$(go env GOPATH)/bin"
RUN protoc /pastebin/proto/*.proto \
    --go_out=/pastebin/proto \
    --go_opt=paths=source_relative \
    --go-grpc_out=/pastebin/proto \
    --go-grpc_opt=paths=source_relative \
    --proto_path=/pastebin/proto \
    /pastebin/proto/pastebin.proto

RUN CGO_ENABLED=0 GOOS=linux go build -o /server.bin ./server

RUN apt -y update && apt -y install openssl

RUN openssl req -new -newkey rsa:4096 -days 365 -nodes -x509 \
    -subj "/C=US/S=NA/L=NA/O=NA/CN=aws-demo-app.local" \
    -keyout /cert.key \
    -out /cert.crt

# STAGE 2: PACKAGE APP
FROM scratch AS app

COPY --from=builder /server.bin /server
COPY --from=builder /cert.key /cert.key
COPY --from=builder /cert.crt /cert.crt

COPY ./server/root /var/www/html/

EXPOSE 443

CMD ["/server"]