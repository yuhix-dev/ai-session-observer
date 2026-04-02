# Product Spec

## 概要
`ai-session-observer` は、Claude と Codex の hidden output を read-only で横断表示する viewer である。

## 対象ユースケース
- Claude の hidden output を常時見たい
- Codex の hidden output を常時見たい
- Claude / Codex の表示を同じ viewer で見たい
- tmux 利用時は監視環境を `ai-session` 一発で再現したい

## 初期スコープ
1. `ai-session-observer`
- Claude / Codex の最新ログを自動検出する
- 共通 event model に正規化して 1 画面に表示する
- 表示は `summary` を標準とし、`details` / `raw` は明示オプトインにする
- 機密情報に該当しうる値は redact を前提に扱う

2. `claude-watch`
- `~/.claude` 配下の session / project 情報を直接読み取る
- read-only で Claude の状態を観測する
- 統合 viewer と同じ表示モードを持つ

3. `codex-watch`
- `~/.codex/sessions/YYYY/MM/DD/rollout-*.jsonl` の最新ファイルを自動検出する
- read-only で Codex の状態を観測する
- 統合 viewer と同じ表示モードを持つ

4. `ai-session`
- tmux の観測用レイアウトを起動する
- 同名 session があれば attach する

5. `healthcheck`
- 必須コマンドと前提環境を確認する
- `go`, `tmux`, shell 実行環境、Claude / Codex の観測元パスを確認する

## 非スコープ
- Claude / Codex の実行制御
- 既存ログの改変
- `~/.tmux.conf` の変更
- 認証情報の自動セットアップ
- observer 自身によるデフォルト保存

## セキュリティ方針
- read-only observation を前提にする
- observer 自身は外部送信しない
- デフォルト表示は `summary` とし、露出を最小限にする
- `details` / `raw` は明示オプトインにする
- 機密情報に該当しうる文字列は redact 可能な設計にする

## 完了条件
- `ai-session-observer` で Claude / Codex の hidden output を同時に見られる
- `claude-watch` と `codex-watch` は引き続き個別に使える
- `ai-session` 一発で観測用 tmux が起動する
- 既存 session があれば attach する
