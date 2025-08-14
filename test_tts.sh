#!/bin/bash
# Test TTS initialization

echo "Testing TTS initialization..."
./glow --tts piper test.md 2>tts_debug.log &
PID=$!

sleep 2
kill $PID 2>/dev/null

echo "Debug output:"
cat tts_debug.log | grep "TTS DEBUG"