# Research: クエスト管理の基本機能

**Feature**: `001-quest-management` | **Date**: 2026-10-11

Technical Context の未確定事項と、主要技術の使い方に関する決定を記録する。

## R1. リポジトリ構成

- **Decision**: ルート直下に `backend/`（Go モジュール）と `frontend/`（Next.js）を置き、
  ルートに `compose.yaml` と `mise.toml` を置くモノレポ構成。
- **Rationale**: バックエンドとフロントエンドで依存・ツールが完全に異なるため、ディレクトリで分離するのが
  最も単純。mise はルートで両方のツールを管理できる。
- **Alternatives considered**: 別リポジトリ（学習用には管理コストが過大）、Go の `web/` 配下に
  フロントを同居（ツール設定が混ざる）。

## R2. バックエンドのレイヤ構成（憲法 II）

- **Decision**: `backend/internal/` 配下を `domain` / `usecase` / `handler` / `infrastructure` に分ける。
  - `domain`: エンティティ・値オブジェクト・ドメインエラー・リポジトリインターフェース。標準ライブラリと
    `github.com/google/uuid` のみに依存。
  - `usecase`: アプリケーションサービス。`domain` のインターフェースと `Clock` インターフェースに依存。
  - `handler`: Echo のハンドラ、リクエスト／レスポンス DTO、ドメインエラー → HTTP の変換。
  - `infrastructure/postgres`: sqlc 生成コードを使ったリポジトリ実装。
  - `cmd/server/main.go` で依存を組み立てる（DI コンテナは使わず手動で注入）。
- **Rationale**: 依存方向が外→内に一方向になり、domain/usecase をフェイク実装で単体テストできる。
- **Alternatives considered**: レイヤを `quest/` などの機能別パッケージに縦割り（機能が 1 つの段階では
  利点が小さい）、DI ライブラリ（wire, fx）の導入（YAGNI、憲法 V）。

## R3. 同時受注の排他制御（FR-009, SC-004）

- **Decision**: `quests.version` 列による楽観的ロック。ドメインのエンティティで状態遷移を検証したうえで、
  リポジトリは `UPDATE ... WHERE id = $1 AND version = $2` で保存する。更新件数 0 なら
  `ErrConcurrentModification` を返し、usecase はそれを「既に受注されている」（409）として扱う。
- **Rationale**: 状態遷移のルールをドメイン層に置いたまま（憲法 II）、DB で原子性を担保できる。
  行ロックを取りっぱなしにしないので単純。
- **Alternatives considered**:
  - `SELECT ... FOR UPDATE`（悲観ロック）: トランザクションを usecase まで引き回す必要があり、
    層の境界が複雑になる。
  - 条件付き UPDATE にルールを埋め込む（`WHERE status = 'open' AND deadline > now()`）: ルールが SQL に
    漏れ、ドメインの単体テストで検証できない。

## R4. マイグレーション（Atlas）と sqlc の連携（憲法 IV）

- **Decision**: Atlas の versioned workflow を使う。
  - あるべき姿のスキーマを `backend/db/schema.sql` に書く。
  - `atlas migrate diff` で `backend/db/migrations/` に差分マイグレーションと `atlas.sum` を生成する。
  - 適用は `atlas migrate apply`。設定は `backend/atlas.hcl`（`env "local"`、diff 用の `dev` は
    Docker の一時 Postgres）。
  - sqlc は `schema: db/schema.sql`、`queries: db/queries/` を読み、`internal/infrastructure/postgres/sqlcgen/`
    に pgx/v5 向けのコードを生成する。生成物はコミットするが手編集しない。
- **Rationale**: スキーマの正が `schema.sql` の 1 か所になり、Atlas と sqlc の両方がそれを読む。
  Prisma の `schema.prisma` → migrate dev に近い運用で、ユーザーの経験とも合う。
- **Alternatives considered**: 手書きの versioned マイグレーションのみ（sqlc の schema 入力が
  マイグレーション群になり、差分の自動生成も使えない）、Atlas の declarative apply のみ
  （履歴が残らず本番運用の練習にならない）。

## R5. 初期データ（冒険者）

- **Decision**: 冒険者の初期データは `backend/db/seed.sql` に置き、開発用タスクで投入する。
  マイグレーションには含めない。
- **Rationale**: マイグレーションはスキーマ変更に限定する（憲法 IV）。冒険者管理は別機能で置き換わる
  予定のため、seed は開発用データとして扱うのが自然。
- **Alternatives considered**: マイグレーションに INSERT を含める（本番にもダミーデータが入る）。

## R6. 「期限切れ」の判定（FR-007a, FR-007b）

- **Decision**: ドメインで `Quest.IsExpired(now)` = `status == open && deadline != nil && !deadline.After(now)`。
  一覧の「期限切れを除く」は SQL 側で
  `NOT (status = 'open' AND deadline IS NOT NULL AND deadline <= $now)` とし、`$now` は usecase の
  `Clock` から渡す。API レスポンスには計算済みの `is_expired` を含める。
- **Rationale**: 現在時刻を引数で渡すので、ドメイン・リポジトリともに時刻を固定してテストできる。
  SQL 側の条件はページングのために必要（アプリ側で絞るとページ件数が崩れる）。
- **Alternatives considered**: DB の `now()` を使う（テストで時刻を固定できない）。

## R7. ページング（FR-006a, SC-003）

- **Decision**: `page`（1 始まり、既定 1）と固定の 1 ページ 20 件で `LIMIT/OFFSET`。総件数は同条件の
  `COUNT(*)` を別クエリで取得し、`total_count` と `total_pages` を返す。範囲外のページは空配列を返す。
  並び順は `created_at DESC, id DESC`。`(created_at DESC, id DESC)` と `status` にインデックスを張る。
- **Rationale**: 1,000 件規模なら OFFSET 方式で十分に 1 秒以内。総ページ数を表示する要件と相性が良い。
- **Alternatives considered**: カーソル方式（総ページ数の表示要件と合わない）。

## R8. 冒険者の識別（FR-013, Clarification Q1）

- **Decision**: 受注・完了報告のリクエストボディに `adventurer_id` を含める。役割（職員／冒険者）の選択は
  フロントエンドの状態（localStorage）だけで扱い、API は役割を検証しない。存在しない冒険者 ID は 404 を返す。
- **Rationale**: 認証なしという決定に沿った最小の方式。将来認証を入れるときは、ボディの
  `adventurer_id` を認証情報からの取得に置き換えるだけで、usecase は変えなくてよい。
- **Alternatives considered**: `X-Adventurer-ID` ヘッダ（認証ヘッダに近い見た目だが、ドキュメント上の
  意味は同じで、利点が小さい）。

## R9. エラー応答（FR-012）

- **Decision**: エラーは `{"error": {"code": "<MACHINE_CODE>", "message": "<日本語メッセージ>"}}` で返す。
  ドメインエラー → HTTP の対応は handler 層の 1 か所に集約する（対応表は `contracts/openapi.yaml`）。
- **Rationale**: フロントは `code` で分岐し、`message` をそのまま表示できる。

## R10. テスト方針（憲法 III）

- **Decision**:
  - domain / usecase: `go test` の単体テスト。リポジトリと `Clock` はフェイク実装。テーブル駆動。
  - infrastructure: Docker Compose の Postgres（テスト用 DB）に対する統合テスト。同時受注 100 並列の
    テスト（SC-004）もここで行う。ビルドタグ `integration` で分離。
  - handler: `httptest` で usecase をフェイクにしたテスト（ステータスコードとエラー形式の確認）。
  - frontend: 本機能では自動テストを必須としない（quickstart の手動確認で検証）。
- **Rationale**: 憲法が求めるのは domain/usecase の単体テスト。DB を伴う排他制御は統合テストでしか
  検証できない。
- **Alternatives considered**: testcontainers-go（Compose で十分なため導入を見送り）。

## R11. ツールのバージョン管理

- **Decision**: `mise.toml` に Go、Node.js（LTS）、Atlas、sqlc を追加し、作業用タスク
  （`db:up`, `db:migrate`, `db:seed`, `gen`, `test`, `dev:*`）も mise tasks で定義する。
  各ツールのバージョンは実装開始時点の最新安定版に固定する。
- **Rationale**: 憲法の技術スタック制約（mise でバージョン管理）に従い、Makefile を別途持たない。

## R12. フロントエンドとバックエンドの接続

- **Decision**: Next.js の `rewrites` で `/api/*` をバックエンド（`http://localhost:8080`）に転送する。
  画面は App Router のクライアントコンポーネントから `/api/*` を呼ぶ。日時は API では UTC の ISO 8601 で
  やり取りし、表示時に日本時間へ変換する。
- **Rationale**: CORS 設定が不要になり、開発時の構成が単純。
- **Alternatives considered**: Echo 側で CORS を許可して直接呼ぶ（設定が増える）。

## R13. ログ

- **Decision**: Echo の RequestID ミドルウェアと、`log/slog` による JSON 形式のアクセスログ・エラーログ。
  500 系はスタック相当の情報（エラーのラップチェーン）をログに出し、レスポンスには出さない。
- **Rationale**: 標準ライブラリで足り、追加依存が不要。
