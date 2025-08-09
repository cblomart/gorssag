# Build stage
FROM golang:1.23-alpine AS builder

# Install build dependencies for CGO
RUN apk add --no-cache gcc musl-dev

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Debug: List static files and templates
RUN find internal/web/static
RUN find internal/web/templates

# Build the application with CGO enabled
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o gorssag .

# Final stage
FROM alpine:latest

# Add build arguments (optional - works with standard docker build)
ARG VERSION=dev
ARG BUILD_TIME=""
ARG GIT_COMMIT=""

# Add labels for metadata
LABEL org.opencontainers.image.title="RSS Aggregator"
LABEL org.opencontainers.image.description="RSS feed aggregator with web interface"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.created="${BUILD_TIME}"
LABEL org.opencontainers.image.revision="${GIT_COMMIT}"

# Install runtime dependencies for SQLite
RUN apk --no-cache add ca-certificates sqlite

WORKDIR /app

# Set environment variables from build args (so runtime can access them)
ENV APP_VERSION=${VERSION}
ENV BUILD_TIME=${BUILD_TIME}
ENV GIT_COMMIT=${GIT_COMMIT}

# Create data directory
RUN mkdir -p /app/data

# Copy the binary from builder stage
COPY --from=builder /app/gorssag .

# Copy template and static files to final stage
COPY --from=builder /app/internal/web/templates ./internal/web/templates
COPY --from=builder /app/internal/web/static ./internal/web/static

# Expose port
EXPOSE 8080

# Run the application
CMD ["./gorssag"] 