---
name: release-checklist
description: |
  リリース前チェックを標準化する手順。
  ユーザーが「リリース前確認」「最終チェック」を依頼した時に使う。
---

# Release Checklist

## Checklist
1. `go test ./...`
2. 主要コマンドの動作確認（`ai-session-observer`, `healthcheck`）
3. docs更新有無の確認（`docs/`）
4. 機密情報混入チェック

## Output
- PASS/FAIL の一覧
- FAIL がある場合はブロック理由と修正案
