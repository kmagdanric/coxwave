# Build stage
FROM golang:1.24 as builder

WORKDIR /app

# Install buf and plugins
RUN curl -sSL \
    "https://github.com/bufbuild/buf/releases/download/v1.30.0/buf-$(uname -s)-$(uname -m)" \
    -o /usr/local/bin/buf && \
    chmod +x /usr/local/bin/buf

# Install Go plugins
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest && \
    go install github.com/bufbuild/connect-go/cmd/protoc-gen-connect-go@latest

# Copy the entire project
COPY . .

# Generate protobuf files using buf
RUN buf generate

# Initialize and tidy the module
RUN go mod tidy

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/server

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from the builder stage
COPY --from=builder /server .

# Expose the service port
EXPOSE 8080

CMD ["./server"]
