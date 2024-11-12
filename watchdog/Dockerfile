FROM golang:1.23.3 AS builder
LABEL intermediateStageToBeDeleted=true

RUN mkdir -p /build
WORKDIR /build/

COPY go.mod go.sum ./
RUN go mod download

COPY watchdog ./watchdog


RUN CGO_ENABLED=0 GOOS=linux go build -o bin/watchdog ./watchdog

# Stage 2
FROM busybox:latest

COPY --from=builder /build/bin/watchdog /watchdog

ENTRYPOINT ["/watchdog"]