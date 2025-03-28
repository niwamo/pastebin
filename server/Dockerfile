# STAGE 1: BUILD BINARY
FROM golang:latest AS builder

WORKDIR /app

COPY ./go.mod ./
COPY ./*.go ./

RUN go get go.mongodb.org/mongo-driver
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o /server.bin

RUN apt -y update && apt -y install openssl

RUN openssl req -new -newkey rsa:4096 -days 365 -nodes -x509 \
    -subj "/C=US/S=NA/L=NA/O=NA/CN=aws-demo-app.local" \
    -keyout /cert.key \
    -out /cert.crt

# STAGE 2

FROM ubuntu:latest

COPY --from=builder /server.bin /opt/server
COPY --from=builder /cert.key /opt/cert.key
COPY --from=builder /cert.crt /opt/cert.crt

COPY ./*.html /opt/

EXPOSE 443

CMD ["/opt/server"]