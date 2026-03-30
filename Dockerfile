FROM golang:1.26-trixie AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download \
  && go mod verify

COPY . .
RUN go build -v -ldflags "-s -w" -o shorten ./cmd/shorten \
  && go build -v -ldflags "-s -w" -o shortenkey ./cmd/shortenkey

FROM debian:trixie-slim

COPY --from=builder /app/shorten /usr/local/bin/shorten
COPY --from=builder /app/shortenkey /usr/local/bin/shortenkey

ENTRYPOINT ["/usr/local/bin/shorten"]
