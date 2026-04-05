---
name: create-pr
description: |
  ai-session-observer向けのPR本文を作成する手順。
  ユーザーが「PRを作成」「PRドラフト作成」と依頼した時に使う。
---

# Create PR

## Steps
1. `git status` と `git diff main...HEAD` で変更範囲を把握する
2. 変更を3点以内で要約する
3. テスト実行結果を整理する（未実施なら明記）
4. `.github` テンプレートがあれば従って本文を作る

## Output Format
- `## Summary`
- `## Test plan`
- 破壊的変更や未完了事項があれば明記
