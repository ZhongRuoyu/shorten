FROM golang:1.22-bookworm AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN go build -v -ldflags "-s -w" -o shorten ./cmd/shorten

FROM debian:bookworm-slim

COPY --from=builder /app/shorten /usr/local/bin/shorten

ENTRYPOINT ["/usr/local/bin/shorten"]
