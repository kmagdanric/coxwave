#!/bin/bash

# Install buf if needed
if ! command -v buf &> /dev/null; then
    echo "Installing buf..."
    curl -sSL "https://github.com/bufbuild/buf/releases/download/v1.28.1/buf-$(uname -s)-$(uname -m)" -o /tmp/buf
    chmod +x /tmp/buf
    sudo mv /tmp/buf /usr/local/bin/buf
fi

go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install github.com/bufbuild/connect-go/cmd/protoc-gen-connect-go@latest
