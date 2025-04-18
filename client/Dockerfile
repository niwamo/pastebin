# STAGE 1: BUILD BINARY
FROM golang:latest AS builder

RUN apt update && apt install -y make protobuf-compiler

RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

WORKDIR /out

ENTRYPOINT [ "make", "build" ]