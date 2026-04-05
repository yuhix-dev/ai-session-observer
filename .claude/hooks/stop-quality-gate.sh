#!/bin/bash
set -euo pipefail

cd "$CLAUDE_PROJECT_DIR"

if ! OUTPUT=$(go test ./... 2>&1); then
  echo "[Quality Gate] go test failed." >&2
  echo "$OUTPUT" >&2
  exit 2
fi

exit 0
