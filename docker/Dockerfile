FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /nebula ./cmd/nebula

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /nebula /nebula
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY configs/config.yaml /etc/nebula/config.yaml

USER 65532:65532

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD ["/nebula", "-health"]

ENTRYPOINT ["/nebula"]
