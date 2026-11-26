# Multi-stage build for the taskboard web server.
# Build context: taskboard/

# Stage 1: Builder
# go.mod requires Go >= 1.26.1, so the builder must be 1.26 or newer.
FROM golang:1.26-alpine AS builder
WORKDIR /src

# Copy dependency manifests first for layer caching.
COPY go.mod go.sum ./
RUN go mod download

# Copy the full source tree (templates and static assets are embedded via go:embed).
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/taskboard ./cmd/taskboard

# Stage 2: Runtime
FROM alpine:3.20
RUN apk add --no-cache ca-certificates \
    && addgroup -S -g 10001 taskboard \
    && adduser -S -u 10001 -G taskboard taskboard

WORKDIR /app
COPY --from=builder /app/taskboard /app/taskboard

ENV TASKBOARD_HOST=0.0.0.0
EXPOSE 8080

USER 10001:10001

CMD ["/app/taskboard"]