# Development

## 進め方
1. 仕様を `docs/product-spec.md` に固める
2. 責務分離を `docs/architecture.md` で確認する
3. `healthcheck` を先に作る
4. `claude-watch` と `codex-watch` を個別に作る
5. 最後に `ai-session` で統合する

## 実装原則
- read-only observation を崩さない
- 小さいコマンドに分ける
- 依存不足時は明示的に失敗する
- 既存 session がある場合は再利用を優先する

## テスト方針
- まず smoke test を優先する
- ログ検出は fixture で再現できるようにする
- tmux の attach / create の分岐を確認する
