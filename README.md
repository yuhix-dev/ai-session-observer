# ai-session-observer

Claude と Codex の hidden output を、read-only で横断監視するための viewer です。

この repo は `~/.ai-harness` の共通ルールを前提にしつつ、他の人でも使える multi-agent hidden-output viewer として配布できる形を目指します。

## 目的
- Claude の hidden output を常時観測する
- Codex の hidden output を常時観測する
- Claude / Codex のログを同じ viewer 体験で扱う
- tmux 利用時は観測レイアウトを一発で起動する

## スコープ
- 観測を主機能とし、tmux orchestration は補助機能として扱う
- 既存の `~/.tmux.conf` は変更しない
- Claude / Codex の内部実装を置き換えない
- 破壊的な操作は行わない

## 想定コマンド
- `ai-session-observer`
  Claude / Codex の hidden output をまとめて表示する統合 viewer
- `claude-watch`
  `~/.claude` 配下を直接読む Claude 専用 viewer
- `codex-watch`
  `~/.codex/sessions` 配下の rollout JSONL を読む Codex 専用 viewer
- `ai-session`
  tmux セッションの作成または attach
- `healthcheck`
  `go`, `tmux`, shell 実行環境, Claude / Codex の観測元パス確認

## セキュリティデフォルト
- read-only observation
- デフォルト表示は `summary`
- `details` / `raw` は明示オプトイン
- 機密情報に該当しうる値は redact 前提
- observer 自身はデフォルトで追加保存しない

## ディレクトリ構成
```text
ai-session-observer/
├─ AGENTS.md
├─ CLAUDE.md
├─ README.md
├─ .ai-harness/
│  ├─ project.yaml
│  └─ docs/
├─ docs/
│  ├─ architecture.md
│  ├─ product-spec.md
│  └─ development.md
├─ cmd/
├─ internal/
├─ test/
└─ examples/
```

## 参照ドキュメント
- `docs/product-spec.md`
- `docs/architecture.md`
- `docs/development.md`
- `.ai-harness/docs/index.md`
