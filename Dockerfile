FROM golang:1.24-alpine3.21 AS builder

ENV CGO_ENABLED=0

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY dev.docker.env ./
COPY cmd/ ./cmd/
COPY config/ ./config/
COPY internal/ ./internal/
COPY pkg/ ./pkg/

RUN go build -o family_service ./cmd/server

FROM alpine:3.19

LABEL maintainer="mjgardner@abitofhelp.com"
LABEL version="1.0"
LABEL description="Family Service application"

RUN apk --no-cache add ca-certificates

WORKDIR /app
COPY --from=builder /app/family_service .
COPY --from=builder /app/dev.docker.env .
COPY --from=builder /app/config ./config
COPY entrypoint.sh .
COPY secrets ./secrets

RUN chmod +x /app/entrypoint.sh
RUN mkdir -p /app/secrets && chmod -R 755 /app/secrets

RUN adduser -D appuser
USER appuser

EXPOSE 8080
ENTRYPOINT ["/app/entrypoint.sh"]
CMD ["./family_service"]

HEALTHCHECK --interval=30s --timeout=3s \
  CMD wget --quiet --tries=1 --spider http://localhost:8080/health || exit 1
