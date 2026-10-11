# Implementation Plan: クエスト管理の基本機能

**Branch**: `001-quest-management` | **Date**: 2026-10-11 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/001-quest-management/spec.md`

**Note**: This template is filled in by the `/speckit-plan` command; its definition describes the execution workflow.

## Summary

ギルド職員がクエストを登録・閲覧し、冒険者が受注・完了報告できる最小の縦切りを作る。
バックエンドは Go + Echo の REST API を DDD + クリーンアーキテクチャで構成し、状態遷移
（募集中 → 受注済 → 完了）と受注ルールを `Quest` 集約に閉じ込める。同時受注は `version` 列による
楽観的ロックで防ぐ。フロントエンドは Next.js で、役割と冒険者を画面上で選ぶ（認証なし）。

## Technical Context

**Language/Version**: Go（実装開始時点の最新安定版、mise で固定）/ TypeScript + Node.js LTS

**Primary Dependencies**: Echo v4、pgx/v5、sqlc、Atlas、google/uuid / Next.js (App Router)、
Tailwind CSS、shadcn/ui

**Storage**: PostgreSQL（Docker Compose）。スキーマは `backend/db/schema.sql` を正とし、Atlas の
versioned migrations で適用（research R4）

**Testing**: `go test`（domain/usecase/handler は単体テスト、リポジトリは `integration` タグ付きの
統合テスト）。フロントエンドの自動テストは本機能では対象外（research R10）

**Target Platform**: ローカル開発環境（macOS + Docker）。API は Linux コンテナでも動く単一バイナリ

**Project Type**: Web アプリケーション（REST API + SPA 的なフロントエンド）

**Performance Goals**: クエスト 1,000 件で一覧・詳細が 1 秒以内（SC-003）

**Constraints**: 同時受注で二重受注 0 件（SC-004）、認証なし、日時は API で UTC・表示は日本時間

**Scale/Scope**: クエスト 〜1,000 件、冒険者は seed の数名、画面 3 つ（一覧・詳細・登録）

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| 原則 | 判定 | 根拠 |
|------|------|------|
| I. 仕様駆動開発 | PASS | spec → clarify（3 問）→ 本 plan の順に進行。実装は tasks 生成後 |
| II. DDD + クリーンアーキテクチャ | PASS | domain / usecase / handler / infrastructure の 4 層。domain は標準ライブラリと uuid のみに依存し、Echo・sqlc・pgx を参照しない（R2） |
| III. テストファースト | PASS | domain/usecase はフェイク実装で単体テストを先に書く。排他制御は統合テストで検証（R10） |
| IV. スキーマ・クエリの明示管理 | PASS | スキーマ変更は Atlas の migrations のみ、`atlas.sum` をコミット。クエリは sqlc 生成、生成物は手編集しない（R4） |
| V. シンプルさ | PASS | DI ライブラリ・OpenAPI コード生成・testcontainers は導入しない。楽観的ロック列は FR-009 の要求に対する最小手段 |
| 技術スタック制約 | PASS | Go + Echo v4 / PostgreSQL / sqlc + pgx / Atlas / Next.js + Tailwind + shadcn/ui / mise |

**Post-design re-check（Phase 1 後）**: PASS。data-model・contracts に憲法違反なし。
Complexity Tracking への記載事項なし。

## Project Structure

### Documentation (this feature)

```text
specs/001-quest-management/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/
│   └── openapi.yaml     # Phase 1 output
├── checklists/
│   └── requirements.md
└── tasks.md             # Phase 2 output (/speckit-tasks)
```

### Source Code (repository root)

```text
mise.toml                     # Go / Node / Atlas / sqlc とタスク定義
compose.yaml                  # Postgres（dev 用 DB とテスト用 DB）

backend/
├── go.mod
├── atlas.hcl
├── sqlc.yaml
├── cmd/server/main.go        # 依存の組み立てと起動
├── db/
│   ├── schema.sql            # あるべきスキーマ（Atlas / sqlc の入力）
│   ├── migrations/           # atlas migrate diff の出力 + atlas.sum
│   ├── queries/              # sqlc のクエリ定義
│   └── seed.sql              # 冒険者の初期データ
└── internal/
    ├── domain/
    │   ├── quest/            # Quest 集約、Status、ドメインエラー、QuestRepository
    │   └── adventurer/       # Adventurer、AdventurerRepository
    ├── usecase/              # CreateQuest / ListQuests / GetQuest / AcceptQuest / CompleteQuest / ListAdventurers、Clock
    ├── handler/              # Echo ハンドラ、DTO、エラー → HTTP 変換、ルーティング
    └── infrastructure/
        └── postgres/
            ├── sqlcgen/      # sqlc 生成コード（手編集禁止）
            ├── quest_repository.go
            └── adventurer_repository.go

frontend/
├── package.json
├── next.config.ts            # /api/* → backend への rewrites
└── src/
    ├── app/
    │   ├── page.tsx              # クエスト一覧（絞り込み・ページ送り）
    │   ├── quests/new/page.tsx   # クエスト登録（職員）
    │   └── quests/[id]/page.tsx  # クエスト詳細（受注・完了報告）
    ├── components/           # shadcn/ui ベースの部品、役割・冒険者セレクタ
    └── lib/                  # API クライアント、型、日時フォーマット
```

**Structure Decision**: バックエンドとフロントエンドを `backend/` と `frontend/` に分けるモノレポ構成
（research R1）。テストは Go の慣習に従い、各パッケージ内に `_test.go` として置く。

## Complexity Tracking

憲法違反なし。記載事項なし。
