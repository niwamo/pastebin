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

# STAGE 2: PACKAGE APP
FROM scratch AS app

COPY --from=builder /server.bin /server
COPY --from=builder /cert.key /cert.key
COPY --from=builder /cert.crt /cert.crt

COPY ./root /var/www/html/

EXPOSE 443

CMD ["/server"]