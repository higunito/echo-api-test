# 冒険者ギルド（echo-api-test）

Go + Echo の学習用アプリ。ギルドがクエストを登録し、冒険者が受注・完了報告する。
開発は [Spec Kit](https://github.com/github/spec-kit) の仕様駆動で進めている（憲法: `.specify/memory/constitution.md`）。

| 領域 | 技術 |
|------|------|
| バックエンド | Go / Echo v4 / DDD + クリーンアーキテクチャ（`backend/`） |
| DB | PostgreSQL（Docker Compose）/ Atlas（マイグレーション）/ sqlc + pgx |
| フロントエンド | Next.js（App Router）/ Tailwind CSS / shadcn/ui（`frontend/`） |
| ツール管理 | mise（`mise.toml`） |

## セットアップと起動

前提: Docker（Compose v2）と mise。

```bash
mise install
(cd frontend && npm install)
mise run db:up        # Postgres（開発用 :5432 / テスト用 :5433）
mise run db:migrate   # マイグレーション適用
mise run db:seed      # 冒険者の初期データ
mise run dev:backend  # API: http://localhost:8080
mise run dev:frontend # 画面: http://localhost:3000（別ターミナル）
```

画面右上で役割（ギルド職員 / 冒険者）と操作する冒険者を選ぶ（認証なし）。

## テストとチェック

```bash
mise run test              # 単体テスト（domain / usecase / handler）
mise run test:integration  # Postgres を使う統合テストを含む
mise run lint              # golangci-lint
mise run check             # atlas.sum の整合性と sqlc 生成物が最新か
```

## スキーマとクエリの変更

- スキーマ: `backend/db/schema.sql` を編集 → `mise run db:diff <名前>` → `mise run db:migrate`
- クエリ: `backend/db/queries/*.sql` を編集 → `mise run gen`
- `backend/db/migrations/` の既存ファイルと `sqlcgen/` の生成コードは手編集しない

## ドキュメント

- 機能仕様・設計: [`specs/001-quest-management/`](specs/001-quest-management/)
- 動作確認手順: [`specs/001-quest-management/quickstart.md`](specs/001-quest-management/quickstart.md)
- API 契約: [`specs/001-quest-management/contracts/openapi.yaml`](specs/001-quest-management/contracts/openapi.yaml)
