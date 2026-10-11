# Quickstart: クエスト管理の基本機能

**Feature**: `001-quest-management` | **Date**: 2026-10-11

実装後に、機能が端から端まで動くことを確かめるための手順。API の詳細は
[contracts/openapi.yaml](contracts/openapi.yaml)、状態と制約は [data-model.md](data-model.md) を参照。

## 前提

- Docker（Compose v2）が動いていること
- `mise install` で Go / Node.js / Atlas / sqlc が入ること

## セットアップ

```bash
mise install
mise run db:up        # Postgres（開発用・テスト用）を起動
mise run db:migrate   # atlas migrate apply
mise run db:seed      # 冒険者の初期データを投入
mise run dev:backend  # http://localhost:8080
mise run dev:frontend # http://localhost:3000（別ターミナル）
```

## 自動テスト

```bash
mise run test              # domain / usecase / handler の単体テスト
mise run test:integration  # Postgres を使うリポジトリ統合テスト（同時受注 100 並列を含む）
```

期待結果: すべて成功。同時受注テストでは成功がちょうど 1 件、残り 99 件が `ErrConcurrentModification`
または `ErrQuestNotOpen` になる（SC-004）。

## API での確認シナリオ

以下の `$A` / `$B` は `GET /api/adventurers` で得た 2 人の冒険者 ID。

| # | 操作 | 期待結果 | 対応 |
|---|------|----------|------|
| 1 | `POST /api/quests` に `{"title":"薬草採取","reward":100,"deadline":"<未来日時>"}` | 201、`status=open` | US1-1 |
| 2 | `POST /api/quests` に `{"title":"","reward":100}` | 400 `INVALID_TITLE` | US1-2 |
| 3 | `POST /api/quests` に `{"title":"x","reward":-1}` | 400 `INVALID_REWARD` | US1-3 |
| 4 | `GET /api/quests` | 200、新しい順、`per_page=20`、`total_count`・`total_pages` あり | US2-1 |
| 5 | `GET /api/quests?status=open` | `open` のみ | US2-2 |
| 6 | `GET /api/quests/<存在しない UUID>` | 404 `QUEST_NOT_FOUND` | US2-5 |
| 7 | #1 のクエストに `POST .../accept` `{"adventurer_id":"$A"}` | 200、`status=accepted`、`assignee.id=$A` | US3-1 |
| 8 | 同じクエストに `POST .../accept` `{"adventurer_id":"$B"}` | 409 `QUEST_NOT_OPEN` | US3-2 |
| 9 | 同じクエストに `POST .../complete` `{"adventurer_id":"$B"}` | 403 `NOT_ASSIGNEE` | US4-2 |
| 10 | 同じクエストに `POST .../complete` `{"adventurer_id":"$A"}` | 200、`status=completed`、`completed_at` あり | US4-1 |
| 11 | 未受注のクエストに `POST .../complete` | 409 `QUEST_NOT_ACCEPTED` | US4-3 |

期限切れ（US2-4, US3-3）は、API からは過去の期限で登録できないため、統合テストで時刻を固定して確認する。
手動で確かめる場合は、期限の近いクエストを登録して期限経過後に次を確認する:
`GET /api/quests` で `is_expired=true`、`?exclude_expired=true` で除外、`accept` が 409 `QUEST_EXPIRED`。

## 画面での確認

1. http://localhost:3000 を開き、役割「ギルド職員」を選ぶ → クエストを登録し、一覧に「募集中」で出る。
2. 役割「冒険者」と冒険者 A を選ぶ → 一覧から 3 操作以内で受注できる（SC-002）。
3. 冒険者 B に切り替える → 同じクエストに受注ボタンが出ない、または押すと理由が表示される。
4. 冒険者 A に戻る → 完了報告し、詳細で「完了」と完了日時が表示される。
5. クエストが 0 件の状態で一覧を開くと、空である旨が表示される。

## 性能の目安（SC-003）

`mise run db:seed:bulk` で 1,000 件のクエストを投入し、一覧の各ページと詳細が 1 秒以内に表示されることを
確認する。
