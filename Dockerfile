# Stage 1: Build binary with cached dependencies
FROM golang:1.22.10-alpine AS builder

WORKDIR /app

# Download dependencies before copying source code for optimal docker caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code and build statically linked executable
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o main .

# Stage 2: Minimal runtime image
FROM alpine:3.20

# Add security certificate store and create non-root user
RUN apk --no-cache add ca-certificates \
    && addgroup -S appgroup \
    && adduser -S appuser -G appgroup

WORKDIR /app

# Copy binary and static files from build stage
COPY --from=builder /app/main /app/main
COPY --from=builder /app/static /app/static

# Change ownership to non-root user
RUN chown -R appuser:appgroup /app

# Security constraint: Do not run as root
USER appuser

EXPOSE 8080

ENTRYPOINT ["/app/main"]
