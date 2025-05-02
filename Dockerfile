FROM golang:1.24-bookworm AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download \
  && go mod verify

COPY . .
RUN go build -v -ldflags "-s -w" -o shorten ./cmd/shorten \
  && go build -v -ldflags "-s -w" -o shortenpw ./cmd/shortenpw

FROM debian:bookworm-slim

COPY --from=builder /app/shorten /usr/local/bin/shorten
COPY --from=builder /app/shortenpw /usr/local/bin/shortenpw

ENTRYPOINT ["/usr/local/bin/shorten"]
