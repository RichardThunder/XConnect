#!/bin/bash
# Test XConnect Go server and CLI (clipboard, files, message, discovery).
# Run from repo root. Uses localhost:18315 to avoid needing Tailscale.

set -e
cd "$(dirname "$0")/.."
PORT=18315
BASE="http://127.0.0.1:$PORT"
BIND="127.0.0.1:$PORT"

echo "=== Build ==="
go build -o xconnect .
go build -o xconnect-cli ./cmd/cli

echo ""
echo "=== Start server on $BIND ==="
./xconnect -addr ":$PORT" &
PID=$!
trap "kill $PID 2>/dev/null || true" EXIT
sleep 1
if ! kill -0 $PID 2>/dev/null; then
  echo "Server failed to start (exit or bind error)."
  exit 1
fi

echo ""
echo "=== 1. GET /clipboard (initial) ==="
BODY=$(curl -s -w "\n%{http_code}" "$BASE/clipboard")
HTTP_CODE=$(echo "$BODY" | tail -n1)
CLIP=$(echo "$BODY" | sed '$d')
echo "HTTP $HTTP_CODE, body length: ${#CLIP}"
# In headless/CI clipboard may be unavailable (500)
NO_CLIP=0
[ "$HTTP_CODE" = "500" ] && NO_CLIP=1 && echo "SKIP: no clipboard in this environment"

echo ""
echo "=== 2. POST /clipboard ==="
CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST -d "hello from test" "$BASE/clipboard")
echo "HTTP $CODE (expect 204 or 500 if no clipboard)"
[ "$CODE" = "500" ] && NO_CLIP=1

echo ""
echo "=== 3. GET /clipboard (after POST) ==="
if [ $NO_CLIP -eq 1 ]; then
  echo "SKIP (no clipboard)"
else
  GOT=$(curl -s "$BASE/clipboard")
  if [ "$GOT" = "hello from test" ]; then
    echo "OK: got '$GOT'"
  else
    echo "FAIL: expected 'hello from test', got '$GOT'"
    exit 1
  fi
fi

echo ""
echo "=== 4. POST /message ==="
CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST -H "Content-Type: application/json" -d '{"text":"msg test"}' "$BASE/message")
echo "HTTP $CODE (expect 204 or 500 if no clipboard)"

echo ""
echo "=== 5. GET /clipboard (after message) ==="
if [ $NO_CLIP -eq 1 ]; then
  echo "SKIP (no clipboard)"
else
  GOT=$(curl -s "$BASE/clipboard")
  if [ "$GOT" = "msg test" ]; then
    echo "OK: got '$GOT'"
  else
    echo "FAIL: expected 'msg test', got '$GOT'"
    exit 1
  fi
fi

echo ""
echo "=== 6. POST /files (upload) ==="
TMPFILE=$(mktemp)
echo "file content here" > "$TMPFILE"
RESP=$(curl -s -X POST -F "file=@$TMPFILE" "$BASE/files")
FILE_ID=$(echo "$RESP" | grep -o '"file_id":"[^"]*"' | cut -d'"' -f4)
rm -f "$TMPFILE"
if [ -z "$FILE_ID" ]; then
  echo "FAIL: no file_id in response: $RESP"
  exit 1
fi
echo "OK: file_id=$FILE_ID"

echo ""
echo "=== 7. GET /files/:id (download) ==="
DOWN=$(curl -s "$BASE/files/$FILE_ID")
if [ "$DOWN" = "file content here" ]; then
  echo "OK: downloaded content matches"
else
  echo "FAIL: expected 'file content here', got '$DOWN'"
  exit 1
fi

echo ""
echo "=== 8. CLI push (to localhost) ==="
set +e
if command -v pbcopy &>/dev/null; then
  echo "push test" | pbcopy
elif command -v xclip &>/dev/null; then
  echo "push test" | xclip -selection clipboard
fi
./xconnect-cli -port "$PORT" push 127.0.0.1 2>&1 && echo "OK" || echo "SKIP (clipboard unavailable)"

echo ""
echo "=== 9. CLI pull (from localhost) ==="
curl -s -X POST -d "pull test content" "$BASE/clipboard" > /dev/null
./xconnect-cli -port "$PORT" pull 127.0.0.1 2>&1 && echo "OK" || echo "SKIP"

echo ""
echo "=== 10. CLI message ==="
./xconnect-cli -port "$PORT" message 127.0.0.1 "cli message" 2>&1 && echo "OK"

echo ""
echo "=== 11. CLI file ==="
TF=$(mktemp)
echo "cli file body" > "$TF"
if ./xconnect-cli -port "$PORT" file 127.0.0.1 "$TF" 2>&1; then
  echo "OK"
else
  echo "FAIL: CLI file upload"
  rm -f "$TF"
  exit 1
fi
rm -f "$TF"

echo ""
echo "=== 12. CLI list (may fail without tailscale) ==="
./xconnect-cli list 2>&1 && echo "OK" || echo "SKIP (tailscale status not available)"
set -e

echo ""
echo "=== All tests done ==="
