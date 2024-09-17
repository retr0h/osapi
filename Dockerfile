FROM golang:1.22-bookworm AS builder

ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64

WORKDIR /build

# Copy only the go.mod and go.sum files to cache dependencies
COPY go.mod go.sum ./

# Download dependencies; this layer will be cached if go.mod and go.sum haven't changed
RUN go mod download

# Copy the rest of the application source code
COPY . .

RUN go build -o osapi .

FROM ubuntu:22.04

WORKDIR /app

COPY --from=builder /build/osapi .
COPY --from=builder /build/osapi.yaml .

# Non root
RUN useradd -m -d /home/nonroot -s /bin/bash nonroot
RUN chown -R nonroot:nonroot /app
USER nonroot

CMD ["./osapi", "server", "start"]
