FROM golang:1.21-alpine AS builder

# Set working directory
WORKDIR /app

# Copy source files
COPY *.go go.mod ./

# Build the binary
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath -o deps *.go

# Final stage - minimal image
FROM alpine:3.18

# Install git (needed for some GitHub operations)
RUN apk --no-cache add git

# Copy the binary from builder
COPY --from=builder /app/deps /usr/local/bin/deps

# Create entrypoint script
RUN echo '#!/bin/sh' > /entrypoint.sh && \
    echo 'cd "${WORKING_DIR:-.}"' >> /entrypoint.sh && \
    echo 'exec deps "$@"' >> /entrypoint.sh && \
    chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
