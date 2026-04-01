# Architecture

## 方針
この repo は「制御」ではなく「観測」を中心に設計する。
観測は tool wrapper ではなく、Claude / Codex のローカル状態を直接読み取る observer として設計する。

## 主要責務
1. session orchestration
- tmux session の作成
- レイアウト適用
- attach / reuse

2. Claude observation
- `~/.claude` 配下の観測元を直接読む
- read-only 観測
- summary / details / raw の表示モードを持つ

3. Codex observation
- `~/.codex/sessions/` 配下の最新 rollout JSONL の検出
- follow 表示
- summary / details / raw の表示モードを持つ

4. environment validation
- 依存コマンドと前提条件の確認
- 観測元パスの読み取り可否確認

## 責務分離
- `ai-session`
  orchestration のみ担当する
- `claude-watch`
  Claude 観測のみ担当する
- `codex-watch`
  Codex 観測のみ担当する
- `healthcheck`
  診断のみ担当する

## 設計制約
- `~/.tmux.conf` を変更しない
- 既存 session を壊さない
- 既存ログを壊さない
- 観測対象への書き込みを避ける
- observer 自身はデフォルトで追加保存しない
- 機密情報に該当しうる値は redact 前提で扱う
- 初期実装は shell-first とし、外部 CLI の接着に留める
