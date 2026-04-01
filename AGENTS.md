# Project Agent Entry

この repo では `~/.ai-harness/AGENTS.md` を土台にしつつ、この repo の `.ai-harness/` を優先する。

## 最初に見る場所
1. `.ai-harness/project.yaml`
2. `.ai-harness/docs/index.md`
3. `docs/product-spec.md`
4. `docs/architecture.md`
5. `docs/development.md`

## project 固有ルール
- この repo は監視とオーケストレーションに限定する
- read-only 観測を優先し、既存 session やログを破壊しない
- 既存の `~/.tmux.conf` は変更しない
- 実装は責務ごとに小さく分離する
