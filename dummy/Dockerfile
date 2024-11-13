FROM golang:1.23.3 AS builder
LABEL intermediateStageToBeDeleted=true

RUN mkdir -p /build
WORKDIR /build/

COPY go.mod go.sum ./
RUN go mod download

COPY dummy ./dummy


RUN CGO_ENABLED=0 GOOS=linux go build -o bin/dummy ./dummy

# Stage 2
FROM busybox:latest

EXPOSE 80

COPY --from=builder /build/bin/dummy /dummy

ENTRYPOINT ["/dummy"]