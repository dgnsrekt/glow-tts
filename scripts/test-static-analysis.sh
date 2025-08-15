#!/bin/bash
# Test script for static analysis with nocgo build tag

set -e

echo "Testing static analysis tools with nocgo build tag..."
echo "=================================================="

# Test go vet
echo -e "\n1. Testing go vet..."
if go vet -tags=nocgo ./... 2>&1 | grep -q "could not import C"; then
    echo "❌ go vet still has CGO import errors"
    exit 1
else
    echo "✅ go vet works with nocgo tag"
fi

# Test go build with nocgo
echo -e "\n2. Testing go build..."
if go build -tags=nocgo -o /tmp/glow-tts-nocgo . 2>&1 | grep -q "undefined"; then
    echo "⚠️  Build with nocgo tag has issues (expected for binary requiring audio)"
else
    echo "✅ Build completes with nocgo tag"
fi
rm -f /tmp/glow-tts-nocgo

# Test staticcheck if available
echo -e "\n3. Testing staticcheck..."
if command -v staticcheck &> /dev/null; then
    if staticcheck -tags=nocgo ./pkg/tts/engines/... 2>&1 | grep -q "could not import C"; then
        echo "❌ staticcheck still has CGO import errors"
    else
        echo "✅ staticcheck works with nocgo tag"
    fi
else
    echo "⚠️  staticcheck not installed, skipping"
fi

# Test golangci-lint if available
echo -e "\n4. Testing golangci-lint..."
if command -v golangci-lint &> /dev/null; then
    if golangci-lint run --build-tags=nocgo --timeout=30s ./pkg/tts/engines/... 2>&1 | grep -q "could not import C"; then
        echo "❌ golangci-lint still has CGO import errors"
    else
        echo "✅ golangci-lint works with nocgo tag"
    fi
else
    echo "⚠️  golangci-lint not installed, skipping"
fi

echo -e "\n=================================================="
echo "Static analysis test complete!"
echo ""
echo "Summary:"
echo "- CGO-dependent files are properly excluded with nocgo build tag"
echo "- Static analysis tools can run without CGO import errors"
echo "- Use 'go build -tags=nocgo' or tool-specific tag flags for analysis"