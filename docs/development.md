# Development

## 進め方
1. 仕様を `docs/product-spec.md` に固める
2. 責務分離を `docs/architecture.md` で確認する
3. 共通 event model を先に固める
4. `ai-session-observer` を先に作る
5. `claude-watch` と `codex-watch` を互換コマンドとして揃える
6. 最後に `ai-session` で tmux integration を整える

## 実装原則
- read-only observation を崩さない
- 小さいコマンドに分ける
- 依存不足時は明示的に失敗する
- 既存 session がある場合は再利用を優先する
- core の parsing / viewing は Go に寄せる
- shell は entrypoint と tmux integration に留める
- 未決事項は `default-first, escalate-on-ambiguity` で扱う
- 標準表示は summary に寄せ、詳細表示は明示オプトインにする

## テスト方針
- まず smoke test を優先する
- ログ検出は fixture で再現できるようにする
- Claude / Codex の event 正規化を fixture で確認する
- unified viewer と tmux integration を分けて確認する
