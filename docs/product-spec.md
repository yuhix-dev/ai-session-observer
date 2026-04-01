# Product Spec

## 概要
`ai-session-observer` は、Claude と Codex のセッションやログを tmux 上で横断監視するためのツールである。

## 対象ユースケース
- Claude のサブエージェントやセッションを別ペインで監視したい
- Codex の rollout ログを別ペインで追いたい
- 監視環境を `ai-session` 一発で再現したい

## 初期スコープ
1. `claude-watch`
- `claude-esp` を利用して Claude を read-only 監視する

2. `codex-watch`
- `~/.codex/sessions/YYYY/MM/DD/rollout-*.jsonl` の最新ファイルを自動検出する
- `tail -f` と `jq` で見やすく表示する

3. `ai-session`
- tmux の 3 ペイン構成を起動する
- 同名 session があれば attach する

4. `healthcheck`
- 必須コマンドと前提環境を確認する

## 非スコープ
- Claude / Codex の実行制御
- 既存ログの改変
- `~/.tmux.conf` の変更
- 認証情報の自動セットアップ

## 完了条件
- `ai-session` 一発で監視用 tmux が起動する
- 右上で Claude 観測が動く
- 右下で Codex rollout 観測が動く
- 既存 session があれば attach する
