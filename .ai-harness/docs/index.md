# Project Docs Index

この repo は Claude / Codex の hidden output 観測を中心に扱い、tmux オーケストレーションは補助機能として扱う。

## まず読むもの
1. `docs/product-spec.md`
2. `docs/architecture.md`
3. `docs/development.md`
4. `.claude/rules/escalation.md`
5. `.ai-harness/claude-progress.txt`

## project 固有の前提
- 監視対象は Claude と Codex
- 主コマンドは `ai-session-observer`
- 既存設定や既存ログを破壊しない
- 共通ルールは `~/.ai-harness` を参照する

## Harness Ops
- Rules: `.claude/rules/`
- Skills: `.claude/skills/`
- Hooks: `.claude/hooks/`
- Roadmap status: `.ai-harness/docs/roadmap-status.md`
