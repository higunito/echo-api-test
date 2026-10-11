---

description: "Task list for 001-quest-management"
---

# Tasks: クエスト管理の基本機能

**Input**: Design documents from `/specs/001-quest-management/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/openapi.yaml, quickstart.md

**Tests**: 憲法 III（テストファースト）により、domain / usecase の単体テストは必須。各ストーリーでテストを
先に書いて失敗を確認してから実装する。handler テストと DB 統合テストも research R10 に従って含める。
フロントエンドの自動テストは対象外。

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1〜US4)
- パスはリポジトリルートからの相対パス。Go のパッケージは `backend/internal/...`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: ツール・プロジェクトの初期化

- [X] T001 `mise.toml` に go（最新安定版）・node（最新 LTS）・atlas・sqlc・golangci-lint を追加し、`mise install` が通ることを確認する（既存の `uv = "0.13.0"` は残す）
- [X] T002 `compose.yaml` を作成し、Postgres（最新メジャー版）を 2 サービス定義する: 開発用 `db`（port 5432、DB 名 `guild`）とテスト用 `db-test`（port 5433、DB 名 `guild_test`、tmpfs）。ユーザー/パスワードは `guild`/`guild`
- [X] T003 `backend/` で `go mod init github.com/higunito/echo-api-test/backend` を実行し、依存（`github.com/labstack/echo/v4`、`github.com/jackc/pgx/v5`、`github.com/google/uuid`）を `backend/go.mod` に追加する
- [X] T004 [P] `frontend/` に Next.js（App Router、TypeScript、Tailwind CSS、`src/` ディレクトリ）を作成し、shadcn/ui を初期化して button / input / textarea / select / table / badge / card / label コンポーネントを追加する
- [X] T005 [P] `backend/.golangci.yml` に最小構成（govet, staticcheck, errcheck, gofmt）を置き、`.gitignore` に `frontend/node_modules`・`frontend/.next`・`backend/bin` を追記する
- [X] T006 `mise.toml` に tasks を定義する: `db:up`（docker compose up -d）、`db:migrate`（atlas migrate apply --env local）、`db:migrate:test`（--env test）、`db:diff`（atlas migrate diff --env local）、`db:seed`（`docker compose exec -T db psql -U guild -d guild < db/seed.sql` で投入。ローカルの psql には依存しない）、`gen`（sqlc generate）、`test`（go test ./...）、`test:integration`（go test -tags=integration ./...）、`lint`（golangci-lint run ./...）、`check`（`atlas migrate validate --env local` と `sqlc diff`）、`dev:backend`（go run ./cmd/server）、`dev:frontend`（npm run dev）。backend 系は `dir = "backend"`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: すべてのストーリーが依存するスキーマ・ドメインの骨格・サーバ基盤・画面の土台

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

### スキーマとマイグレーション（憲法 IV）

- [X] T007 `backend/db/schema.sql` を data-model.md に従って書く。`adventurers(id uuid PK, name text NOT NULL CHECK (char_length(name) BETWEEN 1 AND 50), created_at timestamptz NOT NULL DEFAULT now())`。`quests(id uuid PK, title text NOT NULL CHECK (char_length(title) BETWEEN 1 AND 100), description text NOT NULL DEFAULT '' CHECK (char_length(description) <= 2000), reward integer NOT NULL CHECK (reward >= 0), deadline timestamptz NULL, status text NOT NULL CHECK (status IN ('open','accepted','completed')), assignee_id uuid NULL REFERENCES adventurers(id), accepted_at timestamptz NULL, completed_at timestamptz NULL, created_at timestamptz NOT NULL, version integer NOT NULL DEFAULT 1)`。状態ごとの不変条件（`open` ⇔ assignee_id・accepted_at・completed_at がすべて NULL、`accepted` ⇔ assignee_id・accepted_at が非 NULL かつ completed_at が NULL、`completed` ⇔ 3 つとも非 NULL）を CHECK 制約で表現する。インデックス `quests(created_at DESC, id DESC)` と `quests(status, created_at DESC)`
- [X] T008 `backend/atlas.hcl` を作成する。`env "local"`（url = 開発用 DB、src = `file://db/schema.sql`、migration dir = `file://db/migrations`、dev = `docker://postgres/<T002 と同じメジャー版>/dev`）と `env "test"`（url = テスト用 DB、他は同じ）
- [X] T009 `mise run db:diff` で初回マイグレーションを `backend/db/migrations/` に生成し（`atlas.sum` 含む）、`mise run db:up && mise run db:migrate && mise run db:migrate:test` が通ることを確認する
- [X] T010 [P] `backend/db/seed.sql` に冒険者 3 名（例: アリア、ボルグ、セナ）を固定 UUID で投入する（`ON CONFLICT (id) DO NOTHING` で再実行可能に）
- [X] T011 `backend/sqlc.yaml` を作成する（engine postgresql、schema `db/schema.sql`、queries `db/queries/`、gen.go: package `sqlcgen`、out `internal/infrastructure/postgres/sqlcgen`、sql_package `pgx/v5`、emit_json_tags false、uuid は `github.com/google/uuid.UUID` に override）

### ドメインの骨格

- [X] T012 [P] `backend/internal/domain/quest/status.go` に `Status` 型（`StatusOpen="open"`、`StatusAccepted="accepted"`、`StatusCompleted="completed"`）と `ParseStatus(string) (Status, error)` を定義する
- [X] T013 [P] `backend/internal/domain/quest/errors.go` に data-model.md のドメインエラーを `errors.New` で定義する: `ErrInvalidTitle`、`ErrInvalidDescription`、`ErrInvalidReward`、`ErrDeadlineInPast`、`ErrQuestNotFound`、`ErrQuestNotOpen`、`ErrQuestExpired`、`ErrQuestNotAccepted`、`ErrQuestAlreadyCompleted`、`ErrNotAssignee`、`ErrConcurrentModification`
- [X] T014 `backend/internal/domain/quest/quest.go` に `Quest` 構造体（非公開フィールド: id, title, description, reward, deadline *time.Time, status, assigneeID *uuid.UUID, acceptedAt, completedAt *time.Time, createdAt, version int）、ゲッター、永続化からの復元用 `Reconstruct(...)` を定義する。生成（NewQuest）と状態遷移メソッドは各ストーリーで追加する。依存は標準ライブラリと `github.com/google/uuid` のみ（憲法 II）
- [X] T015 `backend/internal/domain/quest/repository.go` に `QuestRepository` インターフェース（`Create`、`FindByID`、`Update`、`List(ctx, Filter, now time.Time, page int) ([]*Quest, int, error)`）と `Filter{Status *Status; ExcludeExpired bool}`、定数 `PageSize = 20` を定義する
- [X] T016 [P] `backend/internal/domain/adventurer/adventurer.go` に `Adventurer`（id, name, createdAt）・`Reconstruct`・`ErrAdventurerNotFound`・`Repository` インターフェース（`FindByID`、`ListAll`）を定義する
- [X] T017 [P] `backend/internal/usecase/clock.go` に `Clock` インターフェース（`Now() time.Time`）と `SystemClock`（UTC を返す）を定義し、テスト用に `backend/internal/usecase/fake_test.go` を用意して `FixedClock` を置く（以後のフェイクリポジトリもこのファイルに追加する）

### インフラとサーバ基盤

- [X] T018 `backend/internal/infrastructure/postgres/db.go` に `NewPool(ctx, databaseURL) (*pgxpool.Pool, error)` を実装する
- [X] T019 [P] `backend/db/queries/adventurers.sql` に `GetAdventurer :one` と `ListAdventurers :many`（name 昇順）を書き、`mise run gen` を実行する
- [X] T020 `backend/internal/infrastructure/postgres/adventurer_repository.go` に `adventurer.Repository` の実装を書く（`pgx.ErrNoRows` → `adventurer.ErrAdventurerNotFound`）
- [X] T021 [P] `backend/internal/infrastructure/postgres/testutil_test.go`（build tag `integration`）に、環境変数 `TEST_DATABASE_URL`（既定 `postgres://guild:guild@localhost:5433/guild_test?sslmode=disable`）でプールを作り、各テスト前に `TRUNCATE quests, adventurers` するヘルパーを書く
- [X] T022 [P] `backend/internal/handler/errors.go` にエラーレスポンス `{"error":{"code","message"}}` と、ドメインエラー → (HTTP ステータス, code, 日本語メッセージ) の対応表を contracts/openapi.yaml どおりに実装する（ErrInvalidTitle→400 INVALID_TITLE、ErrInvalidDescription→400 INVALID_DESCRIPTION、ErrInvalidReward→400 INVALID_REWARD、ErrDeadlineInPast→400 DEADLINE_IN_PAST、ErrQuestNotFound→404 QUEST_NOT_FOUND、ErrAdventurerNotFound→404 ADVENTURER_NOT_FOUND、ErrQuestNotOpen→409 QUEST_NOT_OPEN、ErrQuestExpired→409 QUEST_EXPIRED、ErrQuestNotAccepted→409 QUEST_NOT_ACCEPTED、ErrQuestAlreadyCompleted→409 QUEST_ALREADY_COMPLETED、ErrNotAssignee→403 NOT_ASSIGNEE、バインド失敗→400 VALIDATION_ERROR、その他→500 INTERNAL_ERROR）。Echo の `HTTPErrorHandler` として登録できる形にし、500 は slog にエラーを出す
- [X] T023 [P] `backend/internal/handler/errors_test.go` に対応表の各行と 500 のフォールバックを確認するテーブル駆動テストを書く
- [X] T024 `backend/internal/usecase/list_adventurers.go` に `ListAdventurers` usecase、`backend/internal/handler/adventurer_handler.go` に `GET /api/adventurers`（`{"items":[{id,name}]}`）を実装する
- [X] T025 `backend/internal/handler/router.go` に `NewRouter(deps) *echo.Echo` を実装する（RequestID・Recover ミドルウェア、slog による JSON アクセスログ、T022 のエラーハンドラ、`/api` グループ）。`backend/cmd/server/main.go` で環境変数 `DATABASE_URL`（既定は開発用 DB）と `PORT`（既定 8080）を読み、プール・リポジトリ・usecase・ルータを手動で組み立てて起動する
- [X] T026 `mise run db:seed && mise run dev:backend`（`backend/cmd/server/main.go`）の状態で `curl localhost:8080/api/adventurers` が 3 名を返すことを確認する

### フロントエンドの土台

- [X] T027 [P] `frontend/next.config.ts` に `/api/:path*` → `http://localhost:8080/api/:path*` の rewrites を設定する
- [X] T028 [P] `frontend/src/lib/api.ts` に fetch ラッパーを実装する（JSON 送受信、エラー時は `{code, message}` を持つ `ApiError` を throw）。`frontend/src/lib/types.ts` に contracts/openapi.yaml の `Adventurer`・`Quest`・`QuestStatus`・`QuestPage`・`ErrorResponse` 型を書く。`frontend/src/lib/format.ts` に UTC の ISO 文字列を日本時間（`Asia/Tokyo`）で表示する関数と、状態の日本語ラベル（open=募集中、accepted=受注済、completed=完了）を置く
- [X] T029 `frontend/src/components/actor-context.tsx` に役割（`guild` / `adventurer`）と選択中の冒険者 ID を保持する React Context を実装する（localStorage に保存、読み書きは try/catch）。`frontend/src/components/actor-selector.tsx` に `GET /api/adventurers` を使った役割・冒険者のセレクタを実装し、`frontend/src/app/layout.tsx` のヘッダに配置する

**Checkpoint**: 基盤完成。ここから各ストーリーに着手できる

---

## Phase 3: User Story 1 - ギルドがクエストを登録する (Priority: P1) 🎯 MVP

**Goal**: ギルド職員がタイトル・説明・報酬・期限を入力してクエストを登録でき、「募集中」で保存される

**Independent Test**: `POST /api/quests` で登録し 201・`status=open` が返る。空タイトル・負の報酬・過去の期限が 400 で拒否される（quickstart #1〜#3）

### Tests for User Story 1 ⚠️ 先に書いて失敗を確認する

- [X] T030 [P] [US1] `backend/internal/domain/quest/quest_test.go` に `NewQuest` のテーブル駆動テストを書く: 正常（status=open、version=1、createdAt=now、assignee 等が nil）、タイトルは「1〜100 文字、必須（前後の空白は除去してから判定）」→ 空・空白のみ・101 文字で `ErrInvalidTitle`、100 文字は成功、説明「0〜2000 文字」→ 2001 文字で `ErrInvalidDescription`、報酬「0 以上」→ -1 で `ErrInvalidReward`・0 は成功、期限「登録時点より未来」→ now と同時刻・過去で `ErrDeadlineInPast`・nil は成功
- [X] T031 [P] [US1] `backend/internal/usecase/create_quest_test.go` にフェイクリポジトリと `FixedClock` を使った `CreateQuest` のテストを書く（正常時にリポジトリへ保存されること、ドメインエラーがそのまま返ること）
- [X] T032 [P] [US1] `backend/internal/handler/quest_handler_test.go` に `POST /api/quests` の httptest テストを書く（201 とレスポンス形、不正 JSON で 400 VALIDATION_ERROR、ドメインエラーで 400 と各 code）
- [X] T033 [P] [US1] `backend/internal/infrastructure/postgres/quest_repository_test.go`（build tag `integration`）に `Create` → `FindByID` で全フィールドが往復することを確認するテストを書く

### Implementation for User Story 1

- [X] T034 [US1] `backend/internal/domain/quest/quest.go` に `NewQuest(id uuid.UUID, title, description string, reward int, deadline *time.Time, now time.Time) (*Quest, error)` を実装し T030 を通す（title は `strings.TrimSpace` 後に `utf8.RuneCountInString` で判定）
- [X] T035 [P] [US1] `backend/db/queries/quests.sql` に `CreateQuest :exec` と `GetQuest :one`（`adventurers` を LEFT JOIN して受注者名も取得）を書き、`mise run gen` を実行する
- [X] T036 [US1] `backend/internal/infrastructure/postgres/quest_repository.go` に `Create` と `FindByID`（`pgx.ErrNoRows` → `quest.ErrQuestNotFound`）を実装し T033 を通す
- [X] T037 [US1] `backend/internal/usecase/create_quest.go` に `CreateQuest`（入力 DTO → `uuid.NewV7()` で ID 生成 → `NewQuest` → `Create`）を実装し T031 を通す
- [X] T038 [US1] `backend/internal/handler/quest_handler.go` に `POST /api/quests`（リクエスト `{title, description?, reward, deadline?}`、レスポンスは openapi の `Quest`。`is_expired` は `false`、`assignee` は null）と Quest → レスポンス DTO 変換関数を実装し、`router.go` に登録して T032 を通す
- [X] T039 [US1] `frontend/src/app/quests/new/page.tsx` に登録フォームを実装する（タイトル・説明・報酬・期限（任意、datetime-local を日本時間として UTC に変換して送信））。API のエラー `message` をフォーム上に表示し、成功時は「クエスト『<タイトル>』を登録しました」と表示してフォームを空に戻す（詳細ページは US2 で作るため、この時点では遷移しない）。役割が `guild` でないときはフォームの代わりに案内を表示する

**Checkpoint**: クエスト登録が API と画面で単独に動く

---

## Phase 4: User Story 2 - クエストを一覧・詳細表示する (Priority: P1)

**Goal**: 一覧（新しい順・20 件ページング・状態絞り込み・期限切れ除外）と詳細を閲覧できる

**Independent Test**: 状態の異なるクエストと期限切れクエストを用意し、`GET /api/quests` の並び順・`total_count`/`total_pages`・`status` 絞り込み・`exclude_expired`・範囲外ページ、`GET /api/quests/{id}` の 200/404 を確認する（quickstart #4〜#6）

### Tests for User Story 2 ⚠️ 先に書いて失敗を確認する

- [X] T040 [P] [US2] `backend/internal/domain/quest/quest_test.go` に `IsExpired(now)` のテストを追加する: open かつ deadline <= now で true、deadline > now・deadline nil・accepted/completed（期限が過去でも）で false
- [X] T041 [P] [US2] `backend/internal/usecase/list_quests_test.go` と `backend/internal/usecase/get_quest_test.go` を書く: page 未指定は 1、page < 1 はエラー、`total_pages = ceil(total_count / 20)`（0 件なら 0）、Clock の時刻がリポジトリに渡ること、存在しない ID で `ErrQuestNotFound`
- [X] T042 [P] [US2] `backend/internal/handler/quest_handler_test.go` に `GET /api/quests`（クエリ `status`・`exclude_expired`・`page` のパース、不正な status や page=0 で 400 VALIDATION_ERROR、レスポンス `{items, page, per_page:20, total_count, total_pages}`）と `GET /api/quests/{id}`（不正 UUID で 400、存在しなければ 404 QUEST_NOT_FOUND、`is_expired` と `assignee{id,name}` を含む）のテストを追加する
- [X] T043 [P] [US2] `backend/internal/infrastructure/postgres/quest_repository_test.go`（integration）に `List` のテストを追加する: 25 件で 1 ページ目 20 件・2 ページ目 5 件・3 ページ目 0 件、`created_at DESC, id DESC` の順、status 絞り込み、`ExcludeExpired` で「open かつ deadline <= now」だけが除外され受注済の期限切れは残ること、絞り込み時の総件数

### Implementation for User Story 2

- [X] T044 [US2] `backend/internal/domain/quest/quest.go` に `IsExpired(now time.Time) bool`（`status == open && deadline != nil && !deadline.After(now)`）を実装し T040 を通す
- [X] T045 [P] [US2] `backend/db/queries/quests.sql` に `ListQuests :many` と `CountQuests :one` を追加する。条件は `(sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status'))` と `(NOT @exclude_expired::bool OR NOT (status = 'open' AND deadline IS NOT NULL AND deadline <= @now))`、`ORDER BY created_at DESC, id DESC LIMIT @limit OFFSET @offset`、受注者名を LEFT JOIN で取得。`mise run gen` を実行する
- [X] T046 [US2] `backend/internal/infrastructure/postgres/quest_repository.go` に `List` を実装し T043 を通す
- [X] T047 [US2] `backend/internal/usecase/list_quests.go`（`ListQuests`: 結果に items・page・total_count・total_pages を返す）と `backend/internal/usecase/get_quest.go`（`GetQuest`: Quest と受注者の Adventurer を返す）を実装し T041 を通す
- [X] T048 [US2] `backend/internal/handler/quest_handler.go` に `GET /api/quests` と `GET /api/quests/:id` を実装し（`is_expired` は Clock の現在時刻で計算）、`router.go` に登録して T042 を通す
- [X] T049 [US2] `frontend/src/app/page.tsx` に一覧画面を実装する: 表（タイトル・報酬・期限・状態バッジ、期限切れは「期限切れ」バッジ）、状態セレクタ（すべて/募集中/受注済/完了）と「期限切れを除く」チェック、ページ送り（総件数・現在ページ/総ページ数を表示）。絞り込みとページは URL のクエリ（`?status=&exclude_expired=&page=`）で保持し、ページ移動後も維持する。0 件のとき「クエストがありません」を表示する。役割が `guild` のときは「クエストを登録」ボタンを表示する
- [X] T050 [US2] `frontend/src/app/quests/[id]/page.tsx` に詳細画面を実装する（説明・報酬・期限・状態・期限切れ表示・受注者名・受注日時・完了日時を日本時間で表示）。404 のときは「クエストが見つかりません」を表示する。受注・完了報告ボタンは置き場所だけ用意し、US3/US4 で実装する。あわせて `frontend/src/app/quests/new/page.tsx` の登録成功時の動作を、作成したクエストの詳細ページ（`/quests/[id]`）への遷移に変更する

**Checkpoint**: US1 と US2 で「登録して一覧・詳細で確認できる」MVP が完成

---

## Phase 5: User Story 3 - 冒険者がクエストを受注する (Priority: P2)

**Goal**: 冒険者が募集中かつ期限内のクエストを受注でき、同時受注でも二重受注にならない

**Independent Test**: 冒険者 A の受注で `status=accepted`・`assignee.id=A`、B の受注が 409 QUEST_NOT_OPEN、期限切れが 409 QUEST_EXPIRED、100 並列受注で成功 1 件（quickstart #7, #8, SC-004）

### Tests for User Story 3 ⚠️ 先に書いて失敗を確認する

- [X] T051 [P] [US3] `backend/internal/domain/quest/quest_test.go` に `Accept(adventurerID, now)` のテストを追加する: open かつ期限内で成功（status=accepted、assigneeID、acceptedAt=now）、期限 nil でも成功、accepted/completed で `ErrQuestNotOpen`、open かつ deadline <= now で `ErrQuestExpired`
- [X] T052 [P] [US3] `backend/internal/usecase/accept_quest_test.go` を書く: 冒険者が存在しなければ `adventurer.ErrAdventurerNotFound`、クエストが無ければ `ErrQuestNotFound`、ドメインエラーの伝播、リポジトリの `Update` が `ErrConcurrentModification` を返したら `ErrQuestNotOpen` に変換すること
- [X] T053 [P] [US3] `backend/internal/handler/quest_handler_test.go` に `POST /api/quests/{id}/accept` のテストを追加する（body `{"adventurer_id"}` 欠落・不正 UUID で 400 VALIDATION_ERROR、成功で 200 と更新後の Quest、409 QUEST_NOT_OPEN / QUEST_EXPIRED、404 ADVENTURER_NOT_FOUND）
- [X] T054 [P] [US3] `backend/internal/infrastructure/postgres/quest_repository_test.go`（integration）に `Update` のテストを追加する: version 一致で更新され version が +1 されること、古い version で `ErrConcurrentModification`。さらに同一クエストへ 100 goroutine から「FindByID → Accept → Update」を同時実行し、成功がちょうど 1 件・DB の assignee が成功した冒険者であることを確認する（SC-004）

### Implementation for User Story 3

- [X] T055 [US3] `backend/internal/domain/quest/quest.go` に `Accept(adventurerID uuid.UUID, now time.Time) error` を実装し T051 を通す
- [X] T056 [P] [US3] `backend/db/queries/quests.sql` に `UpdateQuest :execrows`（`UPDATE quests SET status, assignee_id, accepted_at, completed_at, version = version + 1 WHERE id = @id AND version = @version`）を追加し、`mise run gen` を実行する
- [X] T057 [US3] `backend/internal/infrastructure/postgres/quest_repository.go` に `Update` を実装する（影響行数 0 → `quest.ErrConcurrentModification`。成功時は Quest の version を +1 する。そのために `backend/internal/domain/quest/quest.go` に公開メソッド `IncrementVersion()` を追加する）。T054 を通す
- [X] T058 [US3] `backend/internal/usecase/accept_quest.go` に `AcceptQuest(questID, adventurerID)` を実装する（冒険者の存在確認 → FindByID → `Accept(adventurerID, clock.Now())` → Update、`ErrConcurrentModification` は `ErrQuestNotOpen` に変換）。T052 を通す
- [X] T059 [US3] `backend/internal/handler/quest_handler.go` に `POST /api/quests/:id/accept` を実装し、`router.go` に登録して T053 を通す
- [X] T060 [US3] `frontend/src/app/quests/[id]/page.tsx` に受注ボタンを実装する: 役割が `adventurer` かつ冒険者を選択済み、クエストが募集中で期限切れでないときに表示。押すと `POST /api/quests/{id}/accept` を呼び、成功で画面を更新、失敗で API の `message` を表示する。`frontend/src/app/page.tsx` の一覧行に詳細へのリンクを付け、一覧 → 詳細 → 受注が 3 操作以内になるようにする（SC-002）

**Checkpoint**: 登録 → 閲覧 → 受注まで動作し、二重受注が起きない

---

## Phase 6: User Story 4 - 冒険者が完了報告する (Priority: P2)

**Goal**: 受注者本人だけが受注済クエストを完了報告でき、「完了」と完了日時が記録される

**Independent Test**: B の完了報告が 403 NOT_ASSIGNEE、A の完了報告で `status=completed`・`completed_at` あり、未受注は 409 QUEST_NOT_ACCEPTED、完了済は 409 QUEST_ALREADY_COMPLETED（quickstart #9〜#11）

### Tests for User Story 4 ⚠️ 先に書いて失敗を確認する

- [X] T061 [P] [US4] `backend/internal/domain/quest/quest_test.go` に `Complete(adventurerID, now)` のテストを追加する: 受注者本人で成功（status=completed、completedAt=now）、期限を過ぎた受注済でも成功、他の冒険者で `ErrNotAssignee`、open で `ErrQuestNotAccepted`、completed で `ErrQuestAlreadyCompleted`
- [X] T062 [P] [US4] `backend/internal/usecase/complete_quest_test.go` を書く: 冒険者・クエストの存在確認、ドメインエラーの伝播、`ErrConcurrentModification` の扱い（再取得せず `ErrQuestAlreadyCompleted` に変換）
- [X] T063 [P] [US4] `backend/internal/handler/quest_handler_test.go` に `POST /api/quests/{id}/complete` のテストを追加する（400 VALIDATION_ERROR、200、403 NOT_ASSIGNEE、409 QUEST_NOT_ACCEPTED / QUEST_ALREADY_COMPLETED、404）

### Implementation for User Story 4

- [X] T064 [US4] `backend/internal/domain/quest/quest.go` に `Complete(adventurerID uuid.UUID, now time.Time) error` を実装し T061 を通す
- [X] T065 [US4] `backend/internal/usecase/complete_quest.go` に `CompleteQuest(questID, adventurerID)` を実装し T062 を通す（リポジトリの `Update` は US3 のものを再利用。US3 未実装で着手する場合は T056・T057 を先に行う）
- [X] T066 [US4] `backend/internal/handler/quest_handler.go` に `POST /api/quests/:id/complete` を実装し、`router.go` に登録して T063 を通す
- [X] T067 [US4] `frontend/src/app/quests/[id]/page.tsx` に完了報告ボタンを実装する: 役割が `adventurer` で、選択中の冒険者が受注者かつ状態が受注済のときに表示。成功で画面を更新し完了日時を表示、失敗で API の `message` を表示する

**Checkpoint**: 全ストーリーが動作する

---

## Phase 7: Polish & Cross-Cutting Concerns

- [X] T068 [P] `backend/db/seed_bulk.sql` に 1,000 件のクエストを `generate_series` で投入する SQL を書き、`mise.toml` に `db:seed:bulk` タスクを追加する（quickstart の性能確認用、SC-003）
- [X] T069 `backend/` で `mise run lint`、`frontend/` で `npm run lint` を実行し、指摘をすべて解消する
- [X] T070 `backend/` で `mise run test`・`mise run test:integration`・`mise run check`（`atlas.sum` の整合性と sqlc 生成物が最新であること）がすべて成功することを確認する
- [ ] T071 quickstart.md の API シナリオ #1〜#11、画面確認 1〜5、性能の目安を手順どおりに実行し、結果が期待どおりであることを確認する
- [X] T072 [P] ルートに `README.md` を作成し、セットアップ・起動・テストの手順を quickstart.md へのリンク付きで簡潔に記載する

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: 依存なし
- **Foundational (Phase 2)**: Setup 完了後。すべてのストーリーをブロックする
- **User Stories (Phase 3〜6)**: Foundational 完了後
- **Polish (Phase 7)**: 全ストーリー完了後

### User Story Dependencies

- **US1 (P1)**: Foundational のみに依存
- **US2 (P1)**: Foundational に加え、US1 の `CreateQuest`/`GetQuest` クエリとリポジトリの `Create`/`FindByID`（T035・T036）に依存する。T043 のテストデータ作成と T047 の `GetQuest` がこれらを使うため、US1 の後に行う
- **US3 (P2)**: Foundational に依存。`FindByID`（T035・T036）を US1 と共有するため、US1 の後が自然
- **US4 (P2)**: US3 の `Update`（T056・T057）を再利用する。US3 の後に行う

### Within Each User Story

- テスト → ドメイン → クエリ生成 → リポジトリ → usecase → handler → 画面
- テストは先に書いて失敗を確認する（憲法 III）
- `backend/internal/domain/quest/quest.go`・`quest_test.go`・`backend/db/queries/quests.sql`・`quest_handler.go`・`quest_handler_test.go` は複数ストーリーで共有するため、ストーリーをまたいだ並列作業はしない

### Parallel Opportunities

- Setup: T004・T005 は他と並列可
- Foundational: T010、T012・T013、T016・T017、T019、T021〜T023、T027・T028 は並列可
- 各ストーリーのテスト 4 本（domain / usecase / handler / integration）は別ファイルなので並列可

---

## Parallel Example: User Story 1

```bash
# テストを並列で書く:
Task: "NewQuest のテーブル駆動テスト in backend/internal/domain/quest/quest_test.go"
Task: "CreateQuest usecase のテスト in backend/internal/usecase/create_quest_test.go"
Task: "POST /api/quests の handler テスト in backend/internal/handler/quest_handler_test.go"
Task: "Create/FindByID の統合テスト in backend/internal/infrastructure/postgres/quest_repository_test.go"

# ドメイン実装とクエリ定義は並列可:
Task: "NewQuest の実装 in backend/internal/domain/quest/quest.go"
Task: "CreateQuest / GetQuest クエリ in backend/db/queries/quests.sql"
```

---

## Implementation Strategy

### MVP First

1. Phase 1: Setup
2. Phase 2: Foundational
3. Phase 3: US1（登録）
4. Phase 4: US2（一覧・詳細）
5. **STOP and VALIDATE**: quickstart #1〜#6 と画面確認 1・5 で「登録して見られる」ことを確認

US1 単体でも API としては検証できるが、画面で価値を確かめられるのは US2 までそろった時点のため、
MVP は US1 + US2 とする。

### Incremental Delivery

1. Setup + Foundational → 冒険者一覧 API が動く
2. US1 → 登録できる
3. US2 → 一覧・詳細で確認できる（MVP）
4. US3 → 受注できる、二重受注が起きない
5. US4 → 完了報告でライフサイクルが閉じる

---

## Notes

- [P] tasks = different files, no dependencies
- sqlc の生成コード（`backend/internal/infrastructure/postgres/sqlcgen/`）は手編集しない。クエリを変えたら `mise run gen`
- スキーマを変えるときは `backend/db/schema.sql` を編集 → `mise run db:diff` → `mise run db:migrate`。既存のマイグレーションファイルは編集しない
- タスクまたは論理的なまとまりごとにコミットする
