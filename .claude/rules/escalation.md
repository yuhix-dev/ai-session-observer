---
description: 失敗の再発を観測したらルールを昇格するための運用ルール。
---

# Escalation Rule

## Rule Of Three
- 同じ不具合や手戻りが 1-2 回: `CLAUDE.md` / `AGENTS.md` の注意事項で運用
- 同じ不具合や手戻りが 3 回: `.claude/rules/` に昇格して常時適用
- rules 回避が発生: hook か lint の実行ガードへ昇格

## Recording
- 再発記録は `.ai-harness/claude-progress.txt` の `Known Issues` に追記
- 昇格した理由は `.ai-harness/docs/roadmap-status.md` に残す

## Scope
- project 固有の失敗だけをこの rules に載せる
- global 共通化する前に project 側で有効性を確認する
