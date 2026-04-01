# Product Spec

## 概要
`ai-session-observer` は、Claude と Codex のセッションやログを tmux 上で横断監視するためのツールである。

## 対象ユースケース
- Claude のサブエージェントやセッションを別ペインで監視したい
- Codex の rollout ログを別ペインで追いたい
- 監視環境を `ai-session` 一発で再現したい

## 初期スコープ
1. `claude-watch`
- `~/.claude` 配下の session / history / project 情報を直接読み取る
- read-only で Claude の状態を観測する
- 表示は `summary` を標準とし、`details` / `raw` は明示オプトインにする
- 機密情報に該当しうる値は redact を前提に扱う

2. `codex-watch`
- `~/.codex/sessions/YYYY/MM/DD/rollout-*.jsonl` の最新ファイルを自動検出する
- `tail -f` と `jq` で見やすく表示する
- 表示は `summary` を標準とし、`details` / `raw` は明示オプトインにする
- 機密情報に該当しうる値は redact を前提に扱う

3. `ai-session`
- tmux の 3 ペイン構成を起動する
- 同名 session があれば attach する

4. `healthcheck`
- 必須コマンドと前提環境を確認する
- `tmux`, `jq`, shell 実行環境、Claude / Codex の観測元パスを確認する
- `go` は将来拡張向けの任意依存として扱う

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
- `ai-session` 一発で監視用 tmux が起動する
- 右上で Claude 観測が動く
- 右下で Codex rollout 観測が動く
- 既存 session があれば attach する
