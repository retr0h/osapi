FROM golang:1.22-bookworm AS builder

ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64

WORKDIR /build

COPY . .

RUN go build -o osapi .

FROM ubuntu:22.04

WORKDIR /app

COPY --from=builder /build/osapi .
COPY --from=builder /build/osapi.yaml .

CMD ["./osapi", "server", "start"]
