# Architecture

## 方針
この repo は「制御」ではなく「観測」を中心に設計する。

## 主要責務
1. session orchestration
- tmux session の作成
- レイアウト適用
- attach / reuse

2. Claude observation
- `claude-esp` の起動
- read-only 観測

3. Codex observation
- 最新 rollout JSONL の検出
- follow 表示

4. environment validation
- 依存コマンドと前提条件の確認

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
