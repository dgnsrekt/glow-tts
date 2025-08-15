#!/bin/bash

# Build script for glow-tts

echo "Building glow-tts..."
go build -o glow-tts

if [ $? -eq 0 ]; then
    echo "Build successful! Binary created: ./glow-tts"
    echo ""
    echo "To install system-wide:"
    echo "  sudo mv glow-tts /usr/local/bin/"
    echo ""
    echo "Or add to PATH:"
    echo "  export PATH=\$PATH:$(pwd)"
else
    echo "Build failed!"
    exit 1
fi