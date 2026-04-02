# Architecture

## 方針
この repo は「制御」ではなく「観測」を中心に設計する。
観測は tool wrapper ではなく、Claude / Codex のローカル状態を直接読み取る viewer として設計する。

## 主要責務
1. unified viewing
- Claude / Codex のログを共通 event model に正規化する
- 1 画面で summary / details / raw を切り替えて表示する

2. per-agent observation
- `~/.claude` と `~/.codex/sessions/` の観測元を直接読む
- read-only 観測
- 個別 watcher を維持する

3. session orchestration
- tmux session の作成
- レイアウト適用
- attach / reuse

4. environment validation
- 依存コマンドと前提条件の確認
- 観測元パスの読み取り可否確認

## 責務分離
- `ai-session-observer`
  unified viewer を担当する
- `ai-session`
  tmux integration のみ担当する
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
- core は Go で実装し、shell は起動ラッパーに留める
