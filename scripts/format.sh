#!/bin/bash
# =============================================================================
# format.sh - Go プロジェクトのフォーマット処理
# =============================================================================

set -e

echo "Running go fmt..."
go fmt ./...

echo "Running go vet..."
go vet ./...

# golangci-lint がインストールされていれば実行
if command -v golangci-lint &> /dev/null; then
    echo "Running golangci-lint..."
    golangci-lint run --fix ./...
else
    echo "Note: golangci-lint not found, skipping"
fi

echo "Format complete!"
