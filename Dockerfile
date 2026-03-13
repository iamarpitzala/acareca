# ── Build stage ─────────────────────────────
FROM golang:1.25-alpine3.21 AS builder

WORKDIR /app

# install git (needed for some go modules)
RUN apk add --no-cache git

# cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# copy source
COPY . .

# build optimized binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o server ./cmd/api


# ── Runtime stage ───────────────────────────
FROM alpine:3.21

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# copy binary
COPY --from=builder /app/server .

# copy migrations only if needed
COPY migrations ./migrations

# create non-root user
RUN adduser -D appuser
USER appuser

EXPOSE 8080

ENTRYPOINT ["./server"]