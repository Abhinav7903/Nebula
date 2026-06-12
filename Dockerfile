FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /nebula ./cmd/nebula

FROM alpine:3.21

RUN apk add --no-cache ca-certificates curl

COPY --from=builder /nebula /nebula
COPY configs/config.yaml /etc/nebula/config.yaml

RUN adduser -D -u 1001 nebula && \
    chown -R nebula:nebula /nebula /etc/nebula

USER nebula

EXPOSE 8080

ENV NEBULA_CONFIG=/etc/nebula/config.yaml

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
  CMD curl -sf http://localhost:8080/health || exit 1

ENTRYPOINT ["/nebula"]
