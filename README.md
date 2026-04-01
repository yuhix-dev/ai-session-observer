# ai-session-observer

Claude / Codex など複数の AI agent の動作を、tmux 上で横断監視するためのツールです。

この repo は `~/.ai-harness` の共通ルールを前提にしつつ、他の人でも使える監視ツールとして配布できる形を目指します。

## 目的
- Claude のセッションを read-only に監視する
- Codex の rollout ログを read-only に監視する
- tmux で観測用レイアウトを一発で起動する
- 既存 session があれば再利用する

## スコープ
- 観測とオーケストレーションに限定する
- 既存の `~/.tmux.conf` は変更しない
- Claude / Codex の内部実装を置き換えない
- 破壊的な操作は行わない

## 想定コマンド
- `ai-session`
  tmux セッションの作成または attach
- `claude-watch`
  `claude-esp` を使った Claude 観測
- `codex-watch`
  最新 Codex rollout JSONL の検出と表示
- `healthcheck`
  `tmux`, `jq`, `go`, `claude-esp` などの依存確認

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
