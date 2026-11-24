# Dockerfile for Go web server
# Template variables: http://localhost:8080, 8080
# Multi-stage build: golang:1.22-alpine builder -> alpine:3.20 runtime
# The web app runs on HOST (not in Docker). Playwright container reaches it via host.docker.internal.

# Stage 1: Builder
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 go build -o /server ./cmd/server
RUN mkdir -p web

# Stage 2: Runtime
FROM alpine:3.20
RUN apk add --no-cache ca-certificates curl
WORKDIR /app
COPY --from=builder /server /usr/local/bin/server
COPY --from=builder /app/web ./web
EXPOSE 8080
CMD ["server", "-addr", ":8080"]
