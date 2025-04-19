FROM golang:latest

# BUILD CONTEXT IS PARENT DIR, NOT IMMEDIATE DIR

WORKDIR /pastebin

COPY ./go.mod /pastebin/go.mod
COPY ./client/Makefile /pastebin/Makefile
COPY ./proto /pastebin/proto

RUN apt update && apt install -y make protobuf-compiler
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

ENTRYPOINT [ "make", "build" ]